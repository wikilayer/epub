package epub

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Book is everything an EPUB 3 file says. Identifier, Title, Language,
// ContentsTitle and Modified are required, and so is at least one chapter.
type Book struct {
	// Identifier is the book's unique identifier, such as a "urn:uuid:" URN.
	Identifier string
	Title      string
	// Language is a BCP 47 tag such as "en" or "pt-BR".
	Language string
	Creator  string
	// Date is the publication date as the author gives it, usually
	// YYYY-MM-DD.
	Date string
	// ContentsTitle heads the table of contents page, in the book's language.
	ContentsTitle string
	// Modified is when this edition last changed, written to the second.
	Modified time.Time
	// Stylesheet is CSS shared by every page of the book. It is not read, so
	// keep every url() in it inside the book.
	Stylesheet string
	// Cover names the image, among Images, that a reader shows as the cover.
	Cover    string
	Chapters []Chapter
	Images   []Image
}

// Chapter is one page of the book. Body is an XHTML fragment placed under the
// chapter's heading; it may show an image as "images/" plus its name, and may
// hold inline SVG and MathML, but no script, and its src, srcset, href (outside
// <a>), poster and data attributes may not point outside the book. CSS in a
// style attribute is not read, so keep its url() inside the book too. Children
// are chapters nested under this one in the table of contents and read after
// it.
type Chapter struct {
	Title    string
	Body     string
	Children []Chapter
}

// Image is a picture carried inside the book. Name is a file name of letters,
// digits, '.', '-' and '_', not ending in '.', which chapters refer to as
// "images/" plus the name.
type Image struct {
	Name      string
	MediaType string
	Data      []byte
}

//go:embed templates
var templateFiles embed.FS

var templates = template.Must(template.New("").
	Funcs(template.FuncMap{"x": escapeXML}).
	ParseFS(templateFiles, "templates/*"))

const (
	stylesheetFile = "style.css"
	modifiedLayout = "2006-01-02T15:04:05Z"
	maxNameBytes   = 255
	svgSpace       = "http://www.w3.org/2000/svg"
	mathMLSpace    = "http://www.w3.org/1998/Math/MathML"
)

var plainName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

var resourceAttrs = map[string]bool{"src": true, "href": true, "poster": true, "data": true}

type numbered struct {
	ID         string
	File       string
	Title      string
	Body       string
	Properties string
	Children   []*numbered
}

type bookFile struct {
	name     string
	template string
	data     any
}

// Write writes book to w as an EPUB 3 file. It checks the book first and
// writes nothing when it finds the book incomplete, a chapter's body that is
// not well-formed XML, carries a script, points a resource attribute outside
// the book or shows an image the book does not carry, or an image name a
// reader could not look up, since a reader would refuse that file or show it
// broken. Whether a body is valid XHTML beyond that is left to the caller.
// Once writing has begun, an error from w ends it, and what w already took
// stays there.
func Write(w io.Writer, book Book) error {
	tree, chapters := number(book.Chapters)
	if err := inspect(book, chapters); err != nil {
		return err
	}

	page := map[string]any{"Book": book, "Stylesheet": stylesheetLink(book)}
	files := []bookFile{
		{"META-INF/container.xml", "container.xml", nil},
		{"OEBPS/content.opf", "content.opf", map[string]any{
			"Book":       book,
			"Chapters":   chapters,
			"Modified":   book.Modified.UTC().Format(modifiedLayout),
			"Stylesheet": stylesheetLink(book),
		}},
		{"OEBPS/nav.xhtml", "nav.xhtml", with(page, "Tree", tree)},
	}
	for _, c := range chapters {
		files = append(files, bookFile{"OEBPS/" + c.File, "chapter.xhtml", with(page, "Chapter", c)})
	}

	zw := zip.NewWriter(w)
	mimetype, err := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	if err != nil {
		return err
	}
	if _, err := io.WriteString(mimetype, "application/epub+zip"); err != nil {
		return err
	}
	for _, f := range files {
		var body bytes.Buffer
		if err := templates.ExecuteTemplate(&body, f.template, f.data); err != nil {
			return fmt.Errorf("%s: %w", f.name, err)
		}
		if err := writeDeflated(zw, f.name, body.Bytes()); err != nil {
			return err
		}
	}
	if book.Stylesheet != "" {
		if err := writeDeflated(zw, "OEBPS/"+stylesheetFile, []byte(book.Stylesheet)); err != nil {
			return err
		}
	}
	for _, img := range book.Images {
		if err := writeDeflated(zw, "OEBPS/images/"+img.Name, img.Data); err != nil {
			return err
		}
	}
	return zw.Close()
}

func number(in []Chapter) ([]*numbered, []*numbered) {
	var all []*numbered
	var walk func([]Chapter) []*numbered
	walk = func(in []Chapter) []*numbered {
		out := make([]*numbered, 0, len(in))
		for _, c := range in {
			n := &numbered{Title: c.Title, Body: c.Body}
			all = append(all, n)
			n.ID = fmt.Sprintf("chapter-%d", len(all))
			n.File = n.ID + ".xhtml"
			n.Children = walk(c.Children)
			out = append(out, n)
		}
		return out
	}
	tree := walk(in)
	return tree, all
}

func inspect(book Book, chapters []*numbered) error {
	var errs []error
	required := map[string]string{
		"Identifier":    book.Identifier,
		"Title":         book.Title,
		"Language":      book.Language,
		"ContentsTitle": book.ContentsTitle,
	}
	for _, field := range []string{"Identifier", "Title", "Language", "ContentsTitle"} {
		if strings.TrimSpace(required[field]) == "" {
			errs = append(errs, fmt.Errorf("book has no %s", field))
		}
	}
	if book.Modified.IsZero() {
		errs = append(errs, errors.New("book has no Modified time"))
	}
	if len(chapters) == 0 {
		errs = append(errs, errors.New("book has no chapters"))
	}

	images := make(map[string]bool, len(book.Images))
	folded := make(map[string]string, len(book.Images))
	for _, img := range book.Images {
		switch {
		case !plainName.MatchString(img.Name) || strings.HasSuffix(img.Name, ".") || len(img.Name) > maxNameBytes:
			errs = append(errs, fmt.Errorf(
				"image name %q is not a file name of letters, digits, '.', '-' and '_' that ends in no '.'", img.Name))
		case images[img.Name]:
			errs = append(errs, fmt.Errorf("two images are named %q", img.Name))
		case folded[strings.ToLower(img.Name)] != "":
			errs = append(errs, fmt.Errorf("images %q and %q differ only in case", folded[strings.ToLower(img.Name)], img.Name))
		}
		images[img.Name] = true
		folded[strings.ToLower(img.Name)] = img.Name
		if img.MediaType == "" {
			errs = append(errs, fmt.Errorf("image %q has no media type", img.Name))
		}
	}
	if book.Cover != "" && !images[book.Cover] {
		errs = append(errs, fmt.Errorf("cover %q is not among the images", book.Cover))
	}

	for i, c := range chapters {
		at := fmt.Sprintf("chapter %d %q", i+1, c.Title)
		if strings.TrimSpace(c.Title) == "" {
			at = fmt.Sprintf("chapter %d", i+1)
			errs = append(errs, fmt.Errorf("%s has no title", at))
		}
		read, err := readBody(c.Body)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", at, err))
		}
		for _, name := range read.images {
			if !images[name] {
				errs = append(errs, fmt.Errorf("%s: body shows images/%s, which is not among the images", at, name))
			}
		}
		c.Properties = read.properties()
	}
	return errors.Join(errs...)
}

type body struct {
	images []string
	svg    bool
	mathML bool
}

func (b body) properties() string {
	var out []string
	if b.mathML {
		out = append(out, "mathml")
	}
	if b.svg {
		out = append(out, "svg")
	}
	return strings.Join(out, " ")
}

func readBody(fragment string) (body, error) {
	var read body
	d := xml.NewDecoder(strings.NewReader("<body>" + fragment + "</body>"))
	depth := 0
	closed := false
	for {
		token, err := d.Token()
		if errors.Is(err, io.EOF) {
			return read, nil
		}
		if err != nil {
			return read, fmt.Errorf("body is not well-formed XML: %w", err)
		}
		if closed {
			return read, errors.New("body closes the element it is placed in")
		}
		switch t := token.(type) {
		case xml.StartElement:
			depth++
			read.svg = read.svg || t.Name.Space == svgSpace
			read.mathML = read.mathML || t.Name.Space == mathMLSpace
			if t.Name.Local == "script" {
				return read, errors.New("body carries a script, which this package does not declare")
			}
			for _, attr := range t.Attr {
				if t.Name.Local == "a" {
					continue
				}
				var addresses []string
				switch {
				case attr.Name.Local == "srcset":
					for _, candidate := range strings.Split(attr.Value, ",") {
						if fields := strings.Fields(candidate); len(fields) > 0 {
							addresses = append(addresses, fields[0])
						}
					}
				case resourceAttrs[attr.Name.Local]:
					addresses = []string{attr.Value}
				}
				for _, address := range addresses {
					name, err := resource(address)
					if err != nil {
						return read, err
					}
					if name != "" {
						read.images = append(read.images, name)
					}
				}
			}
		case xml.EndElement:
			depth--
			closed = depth == 0
		}
	}
}

func resource(address string) (string, error) {
	target, err := url.Parse(address)
	if err != nil {
		return "", fmt.Errorf("body points at %q, which is not a URL: %w", address, err)
	}
	if target.Scheme != "" || target.Host != "" {
		return "", fmt.Errorf("body loads %s from outside the book, which a book cannot carry", address)
	}
	if target.RawQuery != "" || target.ForceQuery {
		return "", fmt.Errorf("body points at %s with a query, which a book cannot resolve", address)
	}
	name, _ := strings.CutPrefix(target.Path, "images/")
	if name == target.Path {
		return "", nil
	}
	return name, nil
}

func stylesheetLink(book Book) string {
	if book.Stylesheet == "" {
		return ""
	}
	return stylesheetFile
}

func with(page map[string]any, key string, value any) map[string]any {
	out := maps.Clone(page)
	out[key] = value
	return out
}

func writeDeflated(zw *zip.Writer, name string, body []byte) error {
	f, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
	if err != nil {
		return err
	}
	_, err = f.Write(body)
	return err
}

func escapeXML(s string) (string, error) {
	var out strings.Builder
	if err := xml.EscapeText(&out, []byte(s)); err != nil {
		return "", err
	}
	return out.String(), nil
}
