package epub_test

import (
	"bytes"
	"encoding/xml"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wikilayer/epub"
	"github.com/wikilayer/epub/epubtest"
)

func sampleBook() epub.Book {
	return epub.Book{
		Identifier:    "urn:uuid:5b1e7c1a-6a0e-4c62-9a55-0b7e5f0d2c11",
		Title:         "A Field Guide to Small Rivers",
		Creator:       "Mira Holt",
		Language:      "en",
		Date:          "2026-10-08",
		ContentsTitle: "Contents",
		Modified:      time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC),
		Stylesheet:    "blockquote { font-style: italic; }\n",
		Chapters: []epub.Chapter{
			{Title: "Front Matter", Body: "<p>A guide in two parts.</p>"},
			{Title: "Part One", Body: "<blockquote><p>An epigraph.</p></blockquote>", Children: []epub.Chapter{
				{Title: "Ångström Brook", Body: "<h2>The Source</h2><p>The café by the naïve bridge.</p>"},
				{Title: "The Level Stone", Body: `<p><img src="images/stone.png" alt="A stone"/></p>`},
			}},
			{Title: "Part Two", Children: []epub.Chapter{
				{Title: "The Delta", Body: "<p>The end.</p>"},
			}},
		},
		Images: []epub.Image{{Name: "stone.png", MediaType: "image/png", Data: epubtest.OnePixelPNG()}},
	}
}

func write(t *testing.T, book epub.Book) []byte {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, epub.Write(&out, book))
	return out.Bytes()
}

type packageDocument struct {
	Title string `xml:"metadata>title"`
	Items []struct {
		Href       string `xml:"href,attr"`
		Properties string `xml:"properties,attr"`
	} `xml:"manifest>item"`
}

func readPackage(t *testing.T, file []byte) packageDocument {
	t.Helper()
	var pkg packageDocument
	require.NoError(t, xml.Unmarshal([]byte(epubtest.Files(t, file)["OEBPS/content.opf"]), &pkg))
	return pkg
}

func TestWrite_AcceptsABookWithOnlyWhatIsRequired(t *testing.T) {
	book := sampleBook()
	book.Date = ""
	book.Creator = ""
	book.Stylesheet = ""
	book.Images = nil
	book.Chapters = []epub.Chapter{{Title: "Only", Body: "<p>One page.</p>"}}

	file := write(t, book)

	epubtest.RequireValid(t, file)
	files := epubtest.Files(t, file)
	assert.NotContains(t, files, "OEBPS/style.css")
	assert.NotContains(t, files["OEBPS/chapter-1.xhtml"], "stylesheet",
		"a page links a stylesheet the book does not carry")
}

func TestWrite_SharesTheStylesheetWithEveryPage(t *testing.T) {
	files := epubtest.Files(t, write(t, sampleBook()))

	assert.Equal(t, "blockquote { font-style: italic; }\n", files["OEBPS/style.css"])
	for _, page := range []string{"OEBPS/nav.xhtml", "OEBPS/chapter-1.xhtml", "OEBPS/chapter-6.xhtml"} {
		assert.Contains(t, files[page], `href="style.css"`, "%s is left unstyled", page)
	}
}

func TestWrite_TakesAStoredImageAsTheCover(t *testing.T) {
	book := sampleBook()
	book.Images = append(book.Images, epub.Image{Name: "delta.png", MediaType: "image/png", Data: epubtest.OnePixelPNG()})
	book.Chapters[2].Children[0].Body = `<p><img src="images/delta.png" alt="The delta"/></p>`
	book.Cover = "delta.png"
	file := write(t, book)
	epubtest.RequireValid(t, file)

	var covers []string
	for _, item := range readPackage(t, file).Items {
		if slices.Contains(strings.Fields(item.Properties), "cover-image") {
			covers = append(covers, item.Href)
		}
	}
	assert.Equal(t, []string{"images/delta.png"}, covers,
		"a reader shows the cover the manifest marks, and no other")
}

func TestWrite_NavigationFollowsTheChapterTree(t *testing.T) {
	assert.Equal(t, []string{
		"Front Matter",
		"Part One",
		"  Ångström Brook",
		"  The Level Stone",
		"Part Two",
		"  The Delta",
	}, epubtest.NavTitles(t, write(t, sampleBook())))
}

func TestWrite_ReadsTheChaptersInTheOrderOfItsContents(t *testing.T) {
	file := write(t, sampleBook())
	var spine []string
	for _, id := range epubtest.Spine(t, file) {
		if id != "nav" {
			spine = append(spine, id+".xhtml")
		}
	}
	var contents []string
	for _, entry := range epubtest.Nav(t, file) {
		contents = append(contents, entry.Href)
	}

	require.Len(t, spine, 6, "a part that holds only an epigraph is a page of the book all the same")
	assert.Equal(t, contents, spine, "turning the pages reads the chapters in another order than the contents list them")
}

func TestWrite_OpensOnItsTableOfContents(t *testing.T) {
	file := write(t, sampleBook())

	assert.Equal(t, "nav", epubtest.Spine(t, file)[0],
		"a reader leafs through the book and finds no contents page, only a list behind the reader's own menu")
	assert.Contains(t, epubtest.Files(t, file)["OEBPS/nav.xhtml"], "<h1>Contents</h1>",
		"the contents page is headed in the book's own language, not with the book's title a second time")
}

func TestWrite_KeepsAccentedNamesAsWritten(t *testing.T) {
	files := epubtest.Files(t, write(t, sampleBook()))
	var all strings.Builder
	for _, body := range files {
		all.WriteString(body)
	}
	for _, name := range []string{"Ångström", "café", "naïve"} {
		assert.Contains(t, all.String(), name, "a name lost its accent on the way into the book")
	}
}

func TestWrite_EscapesWhatItWritesAsText(t *testing.T) {
	book := sampleBook()
	book.Title = `Salt & "Pepper" <Volume 2>`
	book.Creator = "Q & A"
	book.Chapters[0].Title = "Fish & Chips"

	file := write(t, book)

	epubtest.RequireValid(t, file)
	assert.Equal(t, `Salt & "Pepper" <Volume 2>`, readPackage(t, file).Title,
		"a reader lists the book under another title than it was given")
	assert.Equal(t, "Fish & Chips", epubtest.NavTitles(t, file)[0])
}

func TestWrite_DeclaresInlineSVGAndMathML(t *testing.T) {
	book := sampleBook()
	book.Chapters[0].Body = `<p><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><rect width="1" height="1"/></svg></p>`
	book.Chapters[2].Children[0].Body = `<p><math xmlns="http://www.w3.org/1998/Math/MathML"><mi>x</mi></math></p>`

	file := write(t, book)

	epubtest.RequireValid(t, file)
	opf := epubtest.Files(t, file)["OEBPS/content.opf"]
	assert.Contains(t, opf, `href="chapter-1.xhtml" media-type="application/xhtml+xml" properties="svg"`)
	assert.Contains(t, opf, `href="chapter-6.xhtml" media-type="application/xhtml+xml" properties="mathml"`)
}

func TestWrite_TellsAPictureFromTextThatNamesOne(t *testing.T) {
	book := sampleBook()
	book.Chapters[0].Body = `<p title="images/elsewhere.png">See <img src="images/stone.png" alt="the stone"/> ` +
		`and <a href="https://example.com/rivers">the rivers</a>.</p>`

	epubtest.RequireValid(t, write(t, book))
}

func TestWrite_RefusesABookItCannotMakeValid(t *testing.T) {
	for name, tc := range map[string]struct {
		spoil func(*epub.Book)
		says  string
	}{
		"no identifier":     {func(b *epub.Book) { b.Identifier = "" }, "no Identifier"},
		"no title":          {func(b *epub.Book) { b.Title = " " }, "no Title"},
		"no language":       {func(b *epub.Book) { b.Language = "" }, "no Language"},
		"no contents title": {func(b *epub.Book) { b.ContentsTitle = "" }, "no ContentsTitle"},
		"no modified time":  {func(b *epub.Book) { b.Modified = time.Time{} }, "no Modified"},
		"no chapters":       {func(b *epub.Book) { b.Chapters = nil }, "no chapters"},
		"untitled chapter":  {func(b *epub.Book) { b.Chapters[1].Children[0].Title = "" }, "chapter 3 has no title"},
		"unclosed tag":      {func(b *epub.Book) { b.Chapters[0].Body = "<p>Open" }, `chapter 1 "Front Matter": body is not well-formed XML`},
		"html entity":       {func(b *epub.Book) { b.Chapters[0].Body = "<p>a&nbsp;b</p>" }, `chapter 1 "Front Matter": body is not well-formed XML`},
		"body that leaves its section": {func(b *epub.Book) {
			b.Chapters[0].Body = "<p>One</p></body><body><p>Two</p>"
		}, `chapter 1 "Front Matter": body closes the element it is placed in`},
		"script": {func(b *epub.Book) {
			b.Chapters[0].Body = "<script>alert(1)</script>"
		}, `chapter 1 "Front Matter": body carries a script`},
		"picture from another address": {func(b *epub.Book) {
			b.Chapters[0].Body = `<p><img src="https://example.com/a.png" alt=""/></p>`
		}, `chapter 1 "Front Matter": body loads https://example.com/a.png from outside the book`},
		"picture from another host by a scheme-relative address": {func(b *epub.Book) {
			b.Chapters[0].Body = `<p><img src="//example.com/a.png" alt=""/></p>`
		}, `chapter 1 "Front Matter": body loads //example.com/a.png from outside the book`},
		"picture with a query": {func(b *epub.Book) {
			b.Chapters[0].Body = `<p><img src="images/stone.png?v=2" alt=""/></p>`
		}, `chapter 1 "Front Matter": body points at images/stone.png?v=2 with a query`},
		"SVG image from another address": {func(b *epub.Book) {
			b.Chapters[0].Body = `<svg xmlns="http://www.w3.org/2000/svg"><image href="https://example.com/a.png" width="1" height="1"/></svg>`
		}, `chapter 1 "Front Matter": body loads https://example.com/a.png from outside the book`},
		"one of a srcset from another address": {func(b *epub.Book) {
			b.Chapters[0].Body = `<p><img src="images/stone.png" srcset="images/stone.png 1x, https://example.com/a.png 2x" alt=""/></p>`
		}, `chapter 1 "Front Matter": body loads https://example.com/a.png from outside the book`},
		"one of a srcset the book does not carry": {func(b *epub.Book) {
			b.Chapters[0].Body = `<p><img src="images/stone.png" srcset="images/missing.png 2x" alt=""/></p>`
		}, `chapter 1 "Front Matter": body shows images/missing.png`},
		"image in a folder": {func(b *epub.Book) { b.Images[0].Name = "../stone.png" }, "is not a file name"},
		"image name a URL would have to escape": {func(b *epub.Book) {
			b.Images[0].Name = "level stone.png"
			b.Chapters[1].Children[1].Body = `<p><img src="images/level stone.png" alt="A stone"/></p>`
		}, "is not a file name"},
		"image name ending in a dot": {func(b *epub.Book) { b.Images[0].Name = "stone." }, "is not a file name"},
		"image names that differ only in case": {func(b *epub.Book) {
			b.Images = append(b.Images, epub.Image{Name: "Stone.png", MediaType: "image/png", Data: epubtest.OnePixelPNG()})
		}, `images "stone.png" and "Stone.png" differ only in case`},
		"picture the book does not carry": {func(b *epub.Book) {
			b.Chapters[0].Body = `<p><img src="images/missing.png" alt=""/></p>`
		}, `chapter 1 "Front Matter": body shows images/missing.png, which is not among the images`},
		"image without type": {func(b *epub.Book) { b.Images[0].MediaType = "" }, "has no media type"},
		"two images, one name": {func(b *epub.Book) {
			b.Images = append(b.Images, b.Images[0])
		}, `two images are named "stone.png"`},
		"cover that is not an image": {func(b *epub.Book) { b.Cover = "missing.png" }, `cover "missing.png"`},
	} {
		t.Run(name, func(t *testing.T) {
			book := sampleBook()
			tc.spoil(&book)
			var out bytes.Buffer

			err := epub.Write(&out, book)

			require.ErrorContains(t, err, tc.says)
			assert.Zero(t, out.Len(), "a refused book left half a file behind")
		})
	}
}

func TestWrite_NamesEveryFaultAtOnce(t *testing.T) {
	book := sampleBook()
	book.Title = ""
	book.Cover = "missing.png"

	err := epub.Write(io.Discard, book)

	require.ErrorContains(t, err, "no Title")
	require.ErrorContains(t, err, `cover "missing.png"`)
}
