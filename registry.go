package jwalk

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-json-experiment/json/jsontext"
)

type funcEntry struct {
	fn   reflect.Value
	elem reflect.Type
}

// Registry holds a set of named directive functions for decoding special JSON
// objects. It is safe for concurrent use.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]funcEntry // full names (may include namespace prefix like ns.name)
	shorts  map[string][]string  // short name -> list of fully qualified names
	sep     string               // namespace separator (default ".")
}

func newRegistry() *Registry { // internal constructor
	return &Registry{
		entries: make(map[string]funcEntry),
		shorts:  make(map[string][]string),
		sep:     ".",
	}
}

var (
	jsontextDecoderType = reflect.TypeOf((*jsontext.Decoder)(nil))
	errorType           = reflect.TypeOf((*error)(nil)).Elem()
)

func validateFuncSignature(name string, fn any) (reflect.Value, reflect.Type, error) {
	fnVal := reflect.ValueOf(fn)
	if fnVal.Kind() != reflect.Func {
		return fnVal, nil, fmt.Errorf("directive %q invalid function signature (got %T)", name, fn)
	}
	typ := fnVal.Type()
	if typ.NumIn() != 2 || typ.NumOut() != 1 {
		return fnVal, typ, fmt.Errorf("directive %q invalid function signature (expected 2 inputs, 1 output; got %d, %d)", name, typ.NumIn(), typ.NumOut())
	}
	if typ.In(0) != jsontextDecoderType {
		return fnVal, typ, fmt.Errorf("directive %q invalid function signature (first param must be *jsontext.Decoder; got %s)", name, typ.In(0))
	}
	arg := typ.In(1)
	if arg.Kind() != reflect.Pointer || arg.Elem().Kind() == reflect.Invalid {
		return fnVal, typ, fmt.Errorf("directive %q invalid function signature (second param must be pointer to concrete type; got %s)", name, arg)
	}
	if typ.Out(0) != errorType {
		return fnVal, typ, fmt.Errorf("directive %q invalid function signature (return type must be error; got %s)", name, typ.Out(0))
	}

	return fnVal, arg.Elem(), nil
}

// Register inserts a directive. Names may include an explicit namespace
// prefix (e.g. "std.time") or be bare ("time"). The registry allows using
// the bare name to resolve a directive only when it is unambiguous across all
// registered directives. Once two directives share the same short name, the
// bare lookup becomes ambiguous and callers must use the fully qualified name.
func (r *Registry) Register(name string, fn any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[name]; exists {
		return fmt.Errorf("directive %q already registered", name)
	}
	// validate namespace form: either bare (no separator) or exactly one
	// separator producing two non-empty components: ns.name
	idx := lastIndex(name, r.sep)
	if idx >= 0 { // namespaced
		if idx == 0 || idx == len(name)-1 || strings.Count(name, r.sep) != 1 { // empty side or multi-level
			return fmt.Errorf("directive %q invalid namespace (expected form ns.name)", name)
		}
	}

	fnVal, elemType, err := validateFuncSignature(name, fn)
	if err != nil {
		return err
	}

	r.entries[name] = funcEntry{fn: fnVal, elem: elemType}

	short := name
	if idx >= 0 { // namespaced
		short = name[idx+1:]
	}
	r.shorts[short] = append(r.shorts[short], name)

	return nil
}

func (r *Registry) Exec(name string, dec *jsontext.Decoder) (any, error) {
	r.mu.RLock()

	displayName := name
	var ambiguous bool
	var matches []string

	ent, ok := r.entries[name]
	if !ok { // attempt short-name resolution only if name has no namespace separator
		if lastIndex(name, r.sep) == -1 { // bare
			matches = r.shorts[name]
			switch len(matches) {
			case 0:
				// no match
			case 1:
				ent, ok = r.entries[matches[0]]
				if ok {
					displayName = matches[0]
				}
			default:
				ambiguous = true
			}
		}
	}
	r.mu.RUnlock()
	if !ok {
		if ambiguous {
			return nil, fmt.Errorf("directive %q is ambiguous; use fully qualified name (%s)", name, strings.Join(matches, ", "))
		}
		return nil, fmt.Errorf("directive %q not registered", name)
	}

	argv := reflect.New(ent.elem)
	results := ent.fn.Call([]reflect.Value{reflect.ValueOf(dec), argv})

	if len(results) != 1 {
		return nil, fmt.Errorf("directive %q invalid function signature (expected 1 output; got %d)", displayName, len(results))
	}
	if errVal := results[0].Interface(); errVal != nil {
		return nil, fmt.Errorf("directive %q execution: %w", displayName, errVal.(error))
	}

	return argv.Elem().Interface(), nil
}

func lastIndex(s, sep string) int {
	if len(sep) != 1 {
		return -1
	}
	b := sep[0]
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}
