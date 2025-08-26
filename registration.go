package jwalk

import "github.com/go-json-experiment/json/jsontext"

// Registration is a deferred directive registration. Packages that define
// directives expose values of this type so callers opt in explicitly instead of
// relying on import side-effects (init functions).
//
// For example, in a package "dateop":
//
//	var Date = jwalk.NewDirective("date", func(dec *jsontext.Decoder) (time.Time, error) { ... })
//
// Usage:
//
//	r, _ := jwalk.NewRegistry(dateop.Date /* , other directives... */)
//
// This keeps dependencies explicit and avoids global mutation at import time.
type Registration func(r *Registry) error

// NewDirective wraps the generic Register helper into a Registration closure so
// that dependent packages can expose named directives (decoders for sentinel
// objects) without performing side effects at import time.
func NewDirective[T any](name string, fn func(dec *jsontext.Decoder) (T, error)) Registration {
	return func(r *Registry) error {
		return r.Register(name, func(dec *jsontext.Decoder, v *T) error {
			out, err := fn(dec)
			if err != nil {
				return err
			}
			*v = out
			return nil
		})
	}
}

// Group groups multiple registrations into one. This allows fluent usage
// without variadic expansion, e.g.:
//
//	jwalk.Use(r, jwalk.Group(jwalk.TimeDirective, jwalk.DurationDirective), otherDirective)
//
// or with the stdlib helper:
//
//	jwalk.Use(r, jwalk.StdlibBundle(), otherDirective)
func Group(regs ...Registration) Registration {
	return func(r *Registry) error { return Apply(r, regs...) }
}

// Apply applies one or more registrations to an existing registry. Stops at the
// first error and returns it. This mirrors the style of json/v2's
// WithUnmarshalers option allowing concise multi-registration:
//
//	jwalk.Apply(r, jwalk.TimeDirective, jwalk.DurationDirective)
//	jwalk.Apply(r, jwalk.Stdlib()...)                 // slice expansion
//	jwalk.Apply(r, jwalk.StdlibBundle(), custom)      // bundle form
func Apply(r *Registry, regs ...Registration) error {
	for _, reg := range regs {
		if err := reg(r); err != nil {
			return err
		}
	}
	return nil
}

// NewRegistry constructs a new registry and applies the provided registrations.
func NewRegistry(regs ...Registration) (*Registry, error) {
	r := newRegistry()
	if err := Apply(r, regs...); err != nil {
		return nil, err
	}
	return r, nil
}
