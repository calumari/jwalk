package jwalk

// Document represents a document, defined as an ordered collection of key-value pairs.
// Each entry in the document is represented by an E.
type Document []Entry

// Array represents an array, defined as a slice of values of any type.
type Array []any

// Entry represents a single entry in a document. It consists of a string key and an
// associated value of any type.
type Entry struct {
	Key   string
	Value any
}
