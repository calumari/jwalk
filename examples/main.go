package main

import (
	"fmt"
	"time"

	jwalk "github.com/calumari/jwalk"
)

// This example demonstrates using jwalk with standard library directives for
// decoding JSON "sentinel" objects into Go's native time types. It shows
// different ways to build a directive registry, and how the unmarshaled
// structure preserves ordering while decoding sentinels.

func main() {
	// Build a registry that knows how to handle time and duration sentinels.
	reg, err := jwalk.NewRegistry(jwalk.WithDirective(jwalk.StdTimeDirective), jwalk.WithDirective(jwalk.StdDurationDirective))
	if err != nil {
		panic(err)
	}

	// Example JSON mixing plain fields with sentinel objects.
	//  `$std.time` → Go time.Time
	//  `$std.duration` → Go time.Duration
	input := []byte(`{
		"name": "example",
		"created": {"$std.time": "2023-10-01T12:00:00Z"},
		"timeout": {"$std.duration": "5m30s"},
		"config": {
			"enabled": true,
			"retry_after": {"$std.duration": "1h"}
		},
		"events": [
			{"$std.time": "2023-10-01T12:05:00Z", "other": "data"},
			{"$std.time": "2023-10-01T12:10:00Z"}
		]
	}`)

	// Unmarshal into a `jwalk.Document`, which preserves field ordering
	var doc jwalk.Document
	if err := reg.Unmarshal(input, &doc); err != nil {
		panic(err)
	}

	fmt.Println("Document view:")
	for _, entry := range doc {
		fmt.Printf("Field: %s\n", entry.Key)

		switch v := entry.Value.(type) {
		case time.Time:
			fmt.Printf("  Time: %s\n", v.Format(time.RFC3339))
		case time.Duration:
			fmt.Printf("  Duration: %v\n", v)
		case jwalk.Document:
			fmt.Printf("  Nested Document (%d fields):\n", len(v))
			for _, nested := range v {
				fmt.Printf("    %s: %s\n", nested.Key, formatValue(nested.Value))
			}
		case jwalk.Array:
			fmt.Printf("  Array (%d elements):\n", len(v))
			for i, elem := range v {
				fmt.Printf("    [%d]: %s\n", i, formatValue(elem))
			}
		default:
			fmt.Printf("  Value: %v (type %T)\n", v, v)
		}
		fmt.Println()
	}

	// You can also unmarshal directly into `any`, producing a mixed structure
	var result any
	if err := reg.Unmarshal(input, &result); err != nil {
		panic(err)
	}

	if d, ok := result.(jwalk.Document); ok {
		fmt.Printf("Root document has %d fields\n", len(d))
		for _, entry := range d {
			if entry.Key == "created" {
				if t, ok := entry.Value.(time.Time); ok {
					fmt.Printf("Created: %s\n", t.Format("2006-01-02 15:04:05"))
				}
			}
		}
	}
}

func formatValue(v any) string {
	switch val := v.(type) {
	case time.Time:
		return fmt.Sprintf("Time(%s)", val.Format(time.RFC3339))
	case time.Duration:
		return fmt.Sprintf("Duration(%s)", val)
	case jwalk.Document:
		return fmt.Sprintf("Document[%d]", len(val))
	case jwalk.Array:
		return fmt.Sprintf("Array[%d]", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}
