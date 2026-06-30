package operator

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/kriipke/chartpress/internal/engine"
)

// Renderer turns a Spec into the chart archive bytes the operator uploads.
type Renderer interface {
	RenderZip(spec engine.Spec) ([]byte, error)
}

// chartRenderer renders via engine.GenerateChart (reusing Helm's on-disk SaveDir
// layout, including nested subcharts) into a temp dir, then zips the directory.
type chartRenderer struct {
	templatesDir string
}

func (r *chartRenderer) RenderZip(spec engine.Spec) ([]byte, error) {
	tmp, err := os.MkdirTemp("", "chartpress-render-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	chartDir, err := engine.GenerateChart(spec, r.templatesDir, tmp)
	if err != nil {
		return nil, err
	}
	// chartutil.SaveDir packs each subchart dependency as charts/<name>-<ver>.tgz.
	// Expand them to editable charts/<name>/ directories so the downloaded chart
	// has editable subchart sources (design §4 "edit values after download").
	if err := expandPackagedSubcharts(chartDir); err != nil {
		return nil, err
	}
	return zipDir(chartDir)
}

// expandPackagedSubcharts replaces each charts/<name>-<ver>.tgz that
// chartutil.SaveDir emits for a dependency with an expanded charts/<name>/ tree.
func expandPackagedSubcharts(chartDir string) error {
	chartsDir := filepath.Join(chartDir, "charts")
	entries, err := os.ReadDir(chartsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tgz") {
			continue
		}
		tgz := filepath.Join(chartsDir, e.Name())
		if err := untarInto(tgz, chartsDir); err != nil {
			return fmt.Errorf("expand %s: %w", e.Name(), err)
		}
		if err := os.Remove(tgz); err != nil {
			return err
		}
	}
	return nil
}

// untarInto extracts a gzipped tar into destDir, rejecting unsafe (absolute or
// parent-escaping) member paths.
func untarInto(tgzPath, destDir string) error {
	f, err := os.Open(tgzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(hdr.Name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path in archive: %q", hdr.Name)
		}
		target := filepath.Join(destDir, clean)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

// zipDir walks root and returns a zip whose entries are forward-slash paths
// relative to root's parent (so the archive root is the chart directory name).
func zipDir(root string) ([]byte, error) {
	parent := filepath.Dir(root)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(parent, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if walkErr != nil {
		_ = zw.Close()
		return nil, walkErr
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
