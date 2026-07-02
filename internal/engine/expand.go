// internal/engine/expand.go
package engine

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExpandSubcharts replaces each charts/<name>-<ver>.tgz that chartutil.SaveDir
// emits for a dependency with an expanded charts/<name>/ tree, so the generated
// chart has editable subchart sources.
func ExpandSubcharts(chartDir string) error {
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
