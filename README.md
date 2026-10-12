# epub

[![Tests](https://github.com/wikilayer/epub/actions/workflows/tests.yml/badge.svg)](https://github.com/wikilayer/epub/actions/workflows/tests.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/wikilayer/epub.svg)](https://pkg.go.dev/github.com/wikilayer/epub)

Writes EPUB 3 books from Go. You give it the metadata, a tree of chapters with XHTML bodies and the
pictures they show; it writes a book that opens on a table of contents nested like your chapters.
When the chapter bodies are valid XHTML,
[epubcheck](https://github.com/w3c/epubcheck), the reference validator readers and stores rely on,
accepts the book. The package depends on nothing outside the standard library; only its test
helpers in `epubtest` bring in testify and `golang.org/x/net/html`.

```sh
go get github.com/wikilayer/epub
```

```go
func writeGuide(path string, stone []byte) error {
	book := epub.Book{
		Identifier:    "urn:uuid:5b1e7c1a-6a0e-4c62-9a55-0b7e5f0d2c11",
		Title:         "A Field Guide to Small Rivers",
		Creator:       "Mira Holt",
		Language:      "en",
		ContentsTitle: "Contents",
		Modified:      time.Now(),
		Stylesheet:    "img { max-width: 100%; }",
		Cover:         "stone.png",
		Chapters: []epub.Chapter{
			{Title: "Part One", Body: "<p>An epigraph.</p>", Children: []epub.Chapter{
				{Title: "The Level Stone", Body: `<p><img src="images/stone.png" alt="A stone"/></p>`},
			}},
		},
		Images: []epub.Image{{Name: "stone.png", MediaType: "image/png", Data: stone}},
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := epub.Write(f, book); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
```

## The book it writes

- `Identifier` is the book's unique identifier; a `urn:uuid:` URN with a fresh UUID per edition is
  the usual choice.
- `Language` is a BCP 47 tag such as `en` or `pt-BR`, and marks every page.
- `ContentsTitle` heads the table of contents page. You supply it, in the book's language, since
  the package carries no translations.
- `Modified` is when this edition last changed.
- `Creator` and `Date` appear only when set; `Date` is written as you give it, usually `YYYY-MM-DD`.
- `Stylesheet` is CSS text, written once as `style.css` and linked from every page.
- `Cover` is the `Name` of one of the `Images`, which readers then show as the cover.

The book opens on its table of contents, which lists every chapter nested as the `Chapters` tree
is. Turning the pages reads the chapters in the same order: a chapter, then its children, then its
next sibling. A chapter with no body of its own, such as a part that only gathers others, is still
a page with its heading. Titles and metadata are escaped as text; a chapter body goes in as you
wrote it.

## What a chapter body may hold

A body is an XHTML fragment that goes under the chapter's `<h1>`. It has to be well-formed XML:
every element closed and no HTML entities beyond the five XML defines, so write `&#160;`, not
`&nbsp;`. It shows a picture as `images/` followed by the picture's `Name`. Inline SVG and MathML
are welcome, and the package declares them in the book's manifest as EPUB requires. Scripts are
not, and a book carries what it shows: the package refuses a `src`, `href` (outside `<a>`),
`poster` or `data` attribute that points outside the book, but it does not read CSS, so a
`url(...)` in a `style` attribute or in `Stylesheet` must stay inside the book too. Whether the
markup is valid XHTML beyond that, such as a `<div>` inside a `<p>`, the package does not check;
epubcheck does.

## What it refuses

`Write` checks the whole book before it writes a byte, and returns every fault it finds together:

- an empty `Identifier`, `Title`, `Language` or `ContentsTitle`, or a zero `Modified`;
- a book without chapters, or a chapter without a title;
- a chapter body that is not well-formed, closes the section it is placed in, or carries a script;
- a body whose `src`, `href`, `poster` or `data` attribute points outside the book or carries a
  query, or shows an `images/` name the book does not carry (links in `<a href>` are yours and go
  unchecked);
- an image name with anything but letters, digits, `.`, `-` and `_`, ending in `.` or longer than
  255 bytes, two names that differ only in case, or an image without a media type;
- a `Cover` that names no image.

A refused book leaves the writer untouched. A writer that fails partway through, such as a full
disk, returns its error and keeps whatever it had taken by then.

## Testing code that builds books

Package [`epubtest`](https://pkg.go.dev/github.com/wikilayer/epub/epubtest) runs epubcheck on a
written book and reads its parts back: the files, the table of contents and the reading order. Its
`RequireValid` fails the test when epubcheck is not installed rather than skipping it. epubcheck
needs Java; `brew install epubcheck` installs both on a Mac, and `apt-get install epubcheck` on
Debian and Ubuntu.

Changes between releases are in
[CHANGELOG.md](https://github.com/wikilayer/epub/blob/main/CHANGELOG.md).
