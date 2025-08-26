package main

import (
	"fmt"
	"time"

	"github.com/go-json-experiment/json"

	jwalk "github.com/calumari/jwalk"
)

// Example demonstrates the jwalk library for JSON unmarshaling with custom
// directives. It shows how to use built-in stdlib directives ($std.time and
// $std.duration) and how the document structure is preserved while sentinel
// objects (decoded via directives) are processed.
func main() {
	// Build a registry with standard library directives
	r, err := jwalk.NewRegistry(jwalk.Stdlib())
	if err != nil {
		panic(err)
	}

	// Example JSON with time and duration sentinel objects mixed with regular data
	input := []byte(`{
		"name": "example",
		"created": {"$std.time": "2023-10-01T12:00:00Z"},
		"timeout": {"$std.duration": "5m30s"},
		"config": {
			"enabled": true,
			"retry_after": {"$std.duration": "1h"}
		},
		"events": [
			{"$std.time": "2023-10-01T12:05:00Z"},
			{"$std.time": "2023-10-01T12:10:00Z"}
		]
	}`)

	// Unmarshal into a Document (D) which preserves field ordering
	var doc jwalk.D
	err = json.Unmarshal(input, &doc, json.WithUnmarshalers(jwalk.Unmarshalers(r)))
	if err != nil {
		panic(err)
	}

	fmt.Println("=== jwalk Document Structure ===")
	for _, entry := range doc {
		fmt.Printf("Field: %s\n", entry.Key)

		switch v := entry.Value.(type) {
		case time.Time:
			fmt.Printf("  Time: %s\n", v.Format(time.RFC3339))
		case time.Duration:
			fmt.Printf("  Duration: %v\n", v)
		case jwalk.D:
			fmt.Printf("  Nested Document with %d fields\n", len(v))
			for _, nested := range v {
				fmt.Printf("    %s: %v\n", nested.Key, formatValue(nested.Value))
			}
		case jwalk.A:
			fmt.Printf("  Array with %d elements\n", len(v))
			for i, elem := range v {
				fmt.Printf("    [%d]: %v\n", i, formatValue(elem))
			}
		default:
			fmt.Printf("  Value: %v (type: %T)\n", v, v)
		}
		fmt.Println()
	}

	// You can also unmarshal into interface{} for more flexible handling
	fmt.Println("=== Direct Unmarshaling ===")
	var result interface{}
	err = json.Unmarshal(input, &result, json.WithUnmarshalers(jwalk.Unmarshalers(r)))
	if err != nil {
		panic(err)
	}

	// The result maintains the jwalk structure (D, A, E types)
	if d, ok := result.(jwalk.D); ok {
		fmt.Printf("Root document has %d fields\n", len(d))

		// Access specific fields
		for _, entry := range d {
			if entry.Key == "created" {
				if t, ok := entry.Value.(time.Time); ok {
					fmt.Printf("Created timestamp: %s\n", t.Format("2006-01-02 15:04:05"))
				}
			}
		}
	}
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case time.Time:
		return val.Format(time.RFC3339)
	case time.Duration:
		return val.String()
	case jwalk.D:
		return fmt.Sprintf("Document[%d]", len(val))
	case jwalk.A:
		return fmt.Sprintf("Array[%d]", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}
