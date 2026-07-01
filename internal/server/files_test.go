package server

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"
)

func TestBuildTree(t *testing.T) {
	// Folders sort before files; both alphabetical within a level.
	paths := []string{
		"mychart/values.yaml",
		"mychart/Chart.yaml",
		"mychart/templates/_helpers.tpl",
		"mychart/charts/api/Chart.yaml",
	}
	got := buildTree(paths)
	want := []fileNode{{
		Name: "mychart",
		Children: []fileNode{
			{Name: "charts", Children: []fileNode{
				{Name: "api", Children: []fileNode{{Name: "Chart.yaml"}}},
			}},
			{Name: "templates", Children: []fileNode{{Name: "_helpers.tpl"}}},
			{Name: "Chart.yaml"},
			{Name: "values.yaml"},
		},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildTree mismatch:\n got=%#v\nwant=%#v", got, want)
	}
}

// fakeDownloader serves a fixed byte payload as the object body.
type fakeDownloader struct {
	data []byte
	key  string
	err  error
}

func (f *fakeDownloader) Get(_ context.Context, key string) (io.ReadCloser, error) {
	f.key = key
	if f.err != nil {
		return nil, f.err
	}
	return io.NopCloser(bytes.NewReader(f.data)), nil
}

func TestUnzipArchive(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	entries := map[string]string{
		"c/Chart.yaml":             "name: c\n",
		"c/templates/deploy.yaml":  "kind: Deployment\n",
		"c/charts/api/values.yaml": "replicas: 2\n",
	}
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	d := &fakeDownloader{data: buf.Bytes()}
	nodes, files, err := unzipArchive(context.Background(), d, "charts/c.zip")
	if err != nil {
		t.Fatalf("unzipArchive: %v", err)
	}
	if d.key != "charts/c.zip" {
		t.Errorf("downloader got key %q, want charts/c.zip", d.key)
	}
	if len(files) != len(entries) {
		t.Errorf("got %d files, want %d", len(files), len(entries))
	}
	for name, content := range entries {
		if files[name] != content {
			t.Errorf("file %q = %q, want %q", name, files[name], content)
		}
	}
	// Single top-level folder "c".
	if len(nodes) != 1 || nodes[0].Name != "c" || len(nodes[0].Children) == 0 {
		t.Fatalf("unexpected tree root: %#v", nodes)
	}
}
