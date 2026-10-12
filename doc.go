// Package epub writes EPUB 3 books.
//
// A Book holds the metadata, a tree of chapters whose bodies are XHTML
// fragments, the pictures they show and an optional stylesheet. Write checks
// the book and turns it into an EPUB 3 file with a table of contents nested
// like the chapter tree, which the book opens on, and the chapters read in the
// order the contents list them. Given bodies that are valid XHTML, epubcheck
// accepts the file.
//
// Package epubtest checks a written book with epubcheck and reads its parts
// back, for tests of code that builds books.
package epub
