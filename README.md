# jwalk

`jwalk` is a small, focused helper for decoding "extended JSON" objects. These are JSON objects whose first key is an operator, such as `$date`. It builds on Go's experimental [`encoding/json/v2`](https://pkg.go.dev/encoding/json/v2) package.

It provides a simple, thread-safe registry that maps operator names to decoding functions. You can then plug that registry into a `json.Unmarshal` call via a custom unmarshaler so that objects whose first key starts with a `$` are treated specially and can decode directly into arbitrary Go values.

## Why?

Many JSON ecosystems, such as MongoDB Extended JSON and certain configuration formats, use special objects, called sentinel objects, such as `{"$date": "2025-08-17T12:00:00Z"}` to represent non-primitive types.

The experimental `encoding/json/v2` package exposes low-level hooks for custom decoding. `jwalk` provides the plumbing to register operators and decode these sentinel objects directly into Go types.

## Install

```bash
go get github.com/calumari/jwalk@latest
```

This requires a Go version new enough to use the `github.com/go-json-experiment/json` module. No build tags or `GOEXPERIMENT` environment variables are needed when using the external module.

## Quick Start

See the [example](examples/main.go) for a full example of how to use `jwalk`.

## Roadmap and Ideas

* Provide a built-in `Unmarshaler(r *OperatorRegistry)` helper
* Predefined common operators (date/time) in a small subpackage
* Future operators could include ObjectID, Regex, and other common sentinel objects

Feedback and pull requests are welcome.

