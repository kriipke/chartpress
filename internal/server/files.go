// internal/server/files.go — the read-only chart file explorer.
//
// GET /charts/{name}/files returns a rendered chart's file tree + contents,
// shaped for the web ChartExplorer ({nodes, files}). The operator only persists
// the packaged archive (charts/{name}.zip), and it re-renders from the CR spec
// on every reconcile, so browsing is READ-ONLY: we fetch that archive and unzip
// it in memory per request rather than storing (or letting anyone edit) the
// expanded files. zipDir makes the chart directory the archive root, so entry
// paths are already "<chart>/<relpath>" — the shape the tree view expects.
package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/kriipke/chartpress/internal/objectstore"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// fileNode is one node of the tree. A node with a non-nil Children slice is a
// folder; a leaf (file) omits Children so the client's Array.isArray check reads
// it as a file.
type fileNode struct {
	Name     string     `json:"name"`
	Children []fileNode `json:"children,omitempty"`
}

type chartFiles struct {
	Name  string            `json:"name"`
	Phase string            `json:"phase"`
	Nodes []fileNode        `json:"nodes"`
	Files map[string]string `json:"files"`
}

// maxArchiveBytes caps how much of an archive we buffer/unzip, bounding memory
// on a hostile or corrupt object. Rendered Helm bundles are kilobytes.
const maxArchiveBytes = 32 << 20 // 32 MiB

// handleChartFiles serves a Ready chart's rendered files. stored is the already
// owner-scoped metadata.name resolved by handleChartByName, so the lookup can
// only reach the caller's own chart.
func (s *Server) handleChartFiles(w http.ResponseWriter, r *http.Request, stored string) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.Downloader == nil {
		http.Error(w, "file browsing unavailable: object storage is not configured", http.StatusServiceUnavailable)
		return
	}
	obj, err := s.Lister.Get(r.Context(), s.Namespace, stored)
	if err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "chart not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get chart: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Defense in depth: stored already embeds the owner hash, but confirm the
	// label matches before serving (mirrors handleChartByName).
	if !s.requestOwner(r).ownsChart(*obj) {
		http.Error(w, "chart not found", http.StatusNotFound)
		return
	}
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	key, _, _ := unstructured.NestedString(obj.Object, "status", "artifactKey")
	observed, _, _ := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	// Same freshness gate as the download URL: only serve files for a Ready status
	// that reflects the spec the API server currently holds.
	if phase != "Ready" || key == "" || observed != obj.GetGeneration() {
		http.Error(w, "chart files are only available once the chart is Ready", http.StatusConflict)
		return
	}

	nodes, files, err := unzipArchive(r.Context(), s.Downloader, key)
	if err != nil {
		http.Error(w, "failed to read chart archive: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(chartFiles{Name: chartDisplayName(*obj), Phase: phase, Nodes: nodes, Files: files})
}

// unzipArchive downloads the archive at key and returns its file tree + contents.
func unzipArchive(ctx context.Context, d objectstore.Downloader, key string) ([]fileNode, map[string]string, error) {
	rc, err := d.Get(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, maxArchiveBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if len(data) > maxArchiveBytes {
		return nil, nil, errArchiveTooLarge
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, err
	}

	files := make(map[string]string, len(zr.File))
	paths := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		p := strings.TrimPrefix(f.Name, "./")
		fr, err := f.Open()
		if err != nil {
			return nil, nil, err
		}
		b, err := io.ReadAll(io.LimitReader(fr, maxArchiveBytes))
		fr.Close()
		if err != nil {
			return nil, nil, err
		}
		files[p] = string(b)
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return buildTree(paths), files, nil
}

// errArchiveTooLarge is returned when an archive exceeds maxArchiveBytes.
var errArchiveTooLarge = &archiveError{"chart archive exceeds the size limit"}

type archiveError struct{ msg string }

func (e *archiveError) Error() string { return e.msg }

// buildTree assembles a nested folder/file tree from slash-separated paths,
// listing folders before files and alphabetically within each.
func buildTree(paths []string) []fileNode {
	root := &treeNode{children: map[string]*treeNode{}}
	for _, p := range paths {
		cur := root
		for _, part := range strings.Split(p, "/") {
			if part == "" {
				continue
			}
			next, ok := cur.children[part]
			if !ok {
				next = &treeNode{name: part, children: map[string]*treeNode{}}
				cur.children[part] = next
				cur.order = append(cur.order, part)
			}
			cur = next
		}
	}
	return root.toNodes()
}

type treeNode struct {
	name     string
	order    []string
	children map[string]*treeNode
}

func (n *treeNode) toNodes() []fileNode {
	var dirs, files []fileNode
	for _, k := range n.order {
		c := n.children[k]
		if len(c.children) > 0 {
			dirs = append(dirs, fileNode{Name: k, Children: c.toNodes()})
		} else {
			files = append(files, fileNode{Name: k})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return append(dirs, files...)
}
