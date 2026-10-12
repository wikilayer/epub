# Changelog

Notable changes to `epub` are recorded here in the format of
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The package stays below 1.0 while its API
settles, so a minor release may change it; each entry says what an upgrade asks of you. The
[README](https://github.com/wikilayer/epub#readme) shows how to use it.

## [0.2.0] - 2026-10-12

### Changed

- A fault in a chapter names it by its title as well as its place in reading order, as in
  `chapter 3 "The Level Stone": body is not well-formed XML`, so it can be found in the source the
  book was made from. Code that matched the old `chapter 3:` text has to match the new one.
- `srcset` is checked like `src`: a candidate that loads from outside the book, carries a query or
  shows a picture the book does not carry is refused. A book that passed 0.1.0 with such a
  `srcset` was one epubcheck refused, and now `Write` refuses it first.

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

[0.2.0]: https://github.com/wikilayer/epub/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/wikilayer/epub/releases/tag/v0.1.0
