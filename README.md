# jwalk

`jwalk` is a lightweight helper for decoding "extended JSON" or *sentinel objects* in Go, built on top of the experimental [`encoding/json/v2`](https://pkg.go.dev/encoding/json/v2).

## Why?

JSON ecosystems like MongoDB Extended JSON often represent non-primitive types using *sentinel objects*, for example:

```json
{"$date": "2025-08-17T12:00:00Z"}
```

The experimental `encoding/json/v2` package exposes low-level hooks for custom decoding. `jwalk` provides the plumbing to register *directives* and automatically decode these sentinel objects into Go types.

## Install

```bash
go get github.com/calumari/jwalk@latest
```

## Quick Start

```go
package main

import (
	"fmt"
	"github.com/calumari/jwalk"
	"github.com/go-json-experiment/json"
)

func main() {
	reg, err := jwalk.NewRegistry(jwalk.Stdlib())
	if err != nil { /* handle error */}

	data := []byte(`{"created": {"$std.time": "2023-10-01T12:00:00Z"}`)

	var doc jwalk.D
	err = json.Unmarshal(data, &doc, json.WithUnmarshalers(jwalk.Unmarshalers(reg)))
	if err != nil { /* handle error */}

	fmt.Println(doc)
	// Output: map[created:2023-10-01 12:00:00 +0000 UTC]
}
```

See [examples/main.go](examples/main.go) for a full example.
