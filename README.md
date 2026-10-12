# epub

[![Tests](https://github.com/wikilayer/epub/actions/workflows/tests.yml/badge.svg)](https://github.com/wikilayer/epub/actions/workflows/tests.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/wikilayer/epub.svg)](https://pkg.go.dev/github.com/wikilayer/epub)

Writes EPUB 3 books from Go: metadata, a tree of chapters with XHTML bodies, the pictures they
show and a stylesheet go in, and a book that opens on a table of contents nested like the chapters
comes out. The package depends on nothing outside the standard library.

```sh
go get github.com/wikilayer/epub
```

```go
book := epub.Book{
	Identifier:    "urn:uuid:5b1e7c1a-6a0e-4c62-9a55-0b7e5f0d2c11",
	Title:         "A Field Guide to Small Rivers",
	Language:      "en",
	ContentsTitle: "Contents",
	Modified:      time.Now(),
	Chapters: []epub.Chapter{
		{Title: "Part One", Body: "<p>An epigraph.</p>", Children: []epub.Chapter{
			{Title: "The Level Stone", Body: `<p><img src="images/stone.png" alt="A stone"/></p>`},
		}},
	},
	Images: []epub.Image{{Name: "stone.png", MediaType: "image/png", Data: stone}},
}
err := epub.Write(w, book)
```

`Write` checks the whole book before it writes a byte and returns every fault together, so a book
it refuses never reaches a reader half-made. Given chapter bodies that are valid XHTML,
[epubcheck](https://github.com/w3c/epubcheck) accepts what it writes. What a body may hold and
what is refused is in the
[package documentation](https://pkg.go.dev/github.com/wikilayer/epub).

Package [`epubtest`](https://pkg.go.dev/github.com/wikilayer/epub/epubtest) checks a written book
with epubcheck and reads its parts back, for tests of code that builds books. epubcheck needs Java:
`brew install epubcheck` on a Mac, `apt-get install epubcheck` on Debian and Ubuntu.

## Lines of Code

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/wikilayer/epub/main/.github/loc-history-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/wikilayer/epub/main/.github/loc-history-light.svg">
  <img alt="Lines of code over time" src="https://raw.githubusercontent.com/wikilayer/epub/main/.github/loc-history.svg">
</picture>
