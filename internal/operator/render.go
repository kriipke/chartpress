package operator

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"

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
	if err := engine.ExpandSubcharts(chartDir); err != nil {
		return nil, err
	}
	return zipDir(chartDir)
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
