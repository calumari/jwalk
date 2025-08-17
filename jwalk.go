package jwalk

// D represents a document, defined as an ordered collection of key-value pairs.
// Each entry in the document is represented by an E.
type D []E

// A represents an array, defined as a slice of values of any type.
type A []any

// E represents a single entry in a document. It consists of a string key and an
// associated value of any type.
type E struct {
	Key   string
	Value any
}
