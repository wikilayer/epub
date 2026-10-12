# Changelog

Notable changes to `epub` are recorded here in the format of
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The package stays below 1.0 while its API
settles, so a minor release may change it; each entry says what an upgrade asks of you. The
[README](https://github.com/wikilayer/epub#readme) shows how to use it.

## [0.1.0] - 2026-10-12

### Added

- `Write` turns a `Book` into an EPUB 3 file: metadata, a table of contents nested like the
  chapter tree and opened first, chapters read in the order the contents list them, pictures, a
  cover and an optional stylesheet. Given chapter bodies that are valid XHTML, epubcheck accepts
  the file.
- `Write` checks the book before writing anything and returns every fault at once: missing
  metadata, an untitled chapter, a body that is not well-formed, carries a script, points a
  resource attribute outside the book or shows a picture the book does not carry, an image name a
  reader could not look up, or a cover that names no image. Inline SVG and MathML in a body are
  declared in the manifest.
- Package `epubtest` checks a book with epubcheck and reads its files, table of contents and
  reading order back, for tests of code that builds books.

[0.1.0]: https://github.com/wikilayer/epub/releases/tag/v0.1.0
