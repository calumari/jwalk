package jwalk

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Registry holds a set of named directive functions for decoding special JSON
// objects. It is safe for concurrent use.
//
// Directives may be registered with fully qualified names (e.g. "std.time") or
// with bare names (e.g. "time"). Bare names can be used for lookup only if they
// are unambiguous. Once two directives share the same short name, callers must
// use the fully qualified name.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]*Directive // full names (may include namespace prefix, e.g. ns.name)
	shorts  map[string][]string   // short name -> list of fully qualified names
	sepByte byte                  // single-character namespace separator (default '.')
}

// RegistryOption represents a registry construction option.
//
// RegistryOption do not mutate the registry directly; instead they collect
// directives or configuration that is applied during construction by
// NewRegistry.
type RegistryOption func(*RegistryOptions) error

func WithDirective(d *Directive) RegistryOption {
	return func(o *RegistryOptions) error {
		o.Directives = append(o.Directives, d)
		return nil
	}
}

// RegistryOptions accumulates directives and other configuration during
// NewRegistry construction.
type RegistryOptions struct {
	Directives []*Directive
}

// NewRegistry constructs a Registry and applies any provided options (e.g.
// directive registrations, bundles).
//
// It returns the initialized Registry, or an error if any registration fails.
func NewRegistry(opts ...RegistryOption) (*Registry, error) {
	cfg := &RegistryOptions{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	reg := newRegistry()
	for _, d := range cfg.Directives {
		if err := reg.Register(d); err != nil {
			return nil, err
		}
	}
	return reg, nil
}

// newRegistry constructs an empty Registry with default settings.
func newRegistry() *Registry {
	return &Registry{
		entries: make(map[string]*Directive),
		shorts:  make(map[string][]string),
		sepByte: '.',
	}
}

var defaultRegistry atomic.Pointer[Registry]

func init() {
	defaultRegistry.Store(newRegistry())
}

// DefaultRegistry returns the global default Registry.
func DefaultRegistry() *Registry {
	return defaultRegistry.Load()
}

// SetDefaultRegistry replaces the global default Registry.
func SetDefaultRegistry(reg *Registry) {
	defaultRegistry.Store(reg)
}

// Register inserts a directive into the Registry.
//
// Names may include an explicit namespace prefix (e.g. "std.time") or be bare
// ("time"). The short form can only be used for lookup if it is unambiguous. If
// two or more directives share the same short name, callers must use the fully
// qualified name.
func (r *Registry) Register(d *Directive) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := d.name
	if _, exists := r.entries[name]; exists {
		return fmt.Errorf("directive %q already registered", name)
	}

	// validate namespace form: either bare (no separator) or exactly one
	// separator producing two non-empty components (ns.name).
	idx := strings.LastIndexByte(name, r.sepByte)
	if idx >= 0 { // namespaced
		if idx == len(name)-1 || strings.IndexByte(name, r.sepByte) != idx {
			return fmt.Errorf("directive %q invalid namespace (expected ns.name)", name)
		}
	}

	r.entries[name] = d
	if idx >= 0 {
		short := name[idx+1:]
		r.shorts[short] = append(r.shorts[short], name)
	}
	return nil
}

// InvokeDirective looks up and executes a directive by name.
//
// Both fully qualified and bare names are supported. Bare lookup succeeds only
// if unambiguous. If no directive matches, or if multiple directives share the
// same short name, an error is returned.
func (r *Registry) InvokeDirective(name string, dec *jsontext.Decoder) (any, error) {
	r.mu.RLock()
	var ambiguous bool
	var matches []string
	ent, ok := r.entries[name]
	if !ok {
		if strings.LastIndexByte(name, r.sepByte) == -1 {
			matches = r.shorts[name]
			switch len(matches) {
			case 0:
				// no match
			case 1:
				ent, ok = r.entries[matches[0]]
			default:
				ambiguous = true
			}
		}
	}
	r.mu.RUnlock()

	if !ok {
		if ambiguous {
			return nil, fmt.Errorf("directive %q ambiguous (%s)", name, strings.Join(matches, ", "))
		}
		return nil, fmt.Errorf("directive %q not registered", name)
	}

	v, err := ent.call(dec)
	if err != nil {
		return nil, fmt.Errorf("directive %q: %w", ent.name, err)
	}

	return v, nil
}

// Unmarshal decodes JSON input using the Registry’s unmarshalers.
//
// This is a convenience wrapper over json.Unmarshal that ensures jwalk-specific
// object/array/directive handling is available.
func (r *Registry) Unmarshal(in []byte, out any, opts ...json.Options) error {
	return json.Unmarshal(in, out, append([]json.Options{json.WithUnmarshalers(Unmarshalers(r))}, opts...)...)
}

// Directive describes a directive handler bound to a specific name.
type Directive struct {
	name string
	call func(dec *jsontext.Decoder) (any, error)
}

type Unmarshaler[T any] func(dec *jsontext.Decoder) (T, error)

// NewDirective constructs a Directive given a name and a typed decode function.
//
// Example:
//
//	d := jwalk.NewDirective("std.time", func(dec *jsontext.Decoder) (time.Time, error) {
//	    var s string
//	    if err := json.UnmarshalDecode(dec, &s); err != nil {
//	        return time.Time{}, err
//	    }
//	    return time.Parse(time.RFC3339, s)
//	})
func NewDirective[T any](name string, unmarshaler Unmarshaler[T]) *Directive {
	wrapper := func(dec *jsontext.Decoder) (any, error) {
		return unmarshaler(dec)
	}
	return &Directive{name: name, call: wrapper}
}
