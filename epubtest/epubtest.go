// Package epubtest checks EPUB files in tests and reads their parts back.
//
// RequireValid runs epubcheck, which has to be on PATH: a test that cannot run
// it fails rather than passing a book nobody checked.
package epubtest

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// RequireValid fails the test unless epubcheck accepts file, and shows its
// report when it does not.
func RequireValid(t testing.TB, file []byte) {
	t.Helper()
	if _, err := exec.LookPath("epubcheck"); err != nil {
		t.Fatal("epubcheck is not installed; skipping would pass a book no reader was promised to open")
	}
	path := filepath.Join(t.TempDir(), "book.epub")
	require.NoError(t, os.WriteFile(path, file, 0o600))
	report, err := exec.Command("epubcheck", path).CombinedOutput()
	require.NoError(t, err, "epubcheck refused the book:\n%s", report)
}

// OnePixelPNG returns the smallest valid PNG, for a test that needs an image.
func OnePixelPNG() []byte {
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewGray(image.Rect(0, 0, 1, 1))); err != nil {
		panic(err)
	}
	return out.Bytes()
}

// Files returns every file in the book by its path inside the archive.
func Files(t testing.TB, file []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(file), int64(len(file)))
	require.NoError(t, err)
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		body, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		out[f.Name] = string(body)
	}
	return out
}

// NavEntry is one line of the table of contents. Depth is 1 for the top level.
type NavEntry struct {
	Depth int
	Title string
	Href  string
}

// Nav returns the table of contents in reading order.
func Nav(t testing.TB, file []byte) []NavEntry {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(Files(t, file)["OEBPS/nav.xhtml"]))
	require.NoError(t, err)
	var entries []NavEntry
	var visit func(n *html.Node, depth int)
	visit = func(n *html.Node, depth int) {
		if n.Type == html.ElementNode && n.Data == "ol" {
			depth++
		}
		if n.Type == html.ElementNode && n.Data == "a" && n.FirstChild != nil {
			entry := NavEntry{Depth: depth, Title: n.FirstChild.Data}
			for _, a := range n.Attr {
				if a.Key == "href" {
					entry.Href = a.Val
				}
			}
			entries = append(entries, entry)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c, depth)
		}
	}
	visit(doc, 0)
	return entries
}

// NavTitles returns the titles of the table of contents, each indented two
// spaces per level below the top.
func NavTitles(t testing.TB, file []byte) []string {
	t.Helper()
	var titles []string
	for _, entry := range Nav(t, file) {
		titles = append(titles, strings.Repeat("  ", entry.Depth-1)+entry.Title)
	}
	return titles
}

// Spine returns the ids of the pages in the order a reader turns them.
func Spine(t testing.TB, file []byte) []string {
	t.Helper()
	var pkg struct {
		Items []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	require.NoError(t, xml.Unmarshal([]byte(Files(t, file)["OEBPS/content.opf"]), &pkg))
	idrefs := make([]string, 0, len(pkg.Items))
	for _, item := range pkg.Items {
		idrefs = append(idrefs, item.IDRef)
	}
	return idrefs
}
