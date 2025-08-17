package jwalk

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/go-json-experiment/json/jsontext"
)

type funcEntry struct {
	fn   reflect.Value
	elem reflect.Type
}

type OperatorRegistry struct {
	mu      sync.RWMutex
	entries map[string]funcEntry
}

func NewOperatorRegistry() *OperatorRegistry {
	return &OperatorRegistry{entries: make(map[string]funcEntry)}
}

var DefaultRegistry = NewOperatorRegistry()

var (
	jsontextDecoderType = reflect.TypeOf((*jsontext.Decoder)(nil))
	errorType           = reflect.TypeOf((*error)(nil)).Elem()
)

func validateFuncSignature(name string, fn any) (reflect.Value, reflect.Type, error) {
	fnVal := reflect.ValueOf(fn)
	if fnVal.Kind() != reflect.Func {
		return fnVal, nil, fmt.Errorf("operator %q invalid function signature (got %T)", name, fn)
	}
	typ := fnVal.Type()
	if typ.NumIn() != 2 || typ.NumOut() != 1 {
		return fnVal, typ, fmt.Errorf("operator %q invalid function signature (expected 2 inputs, 1 output; got %d, %d)", name, typ.NumIn(), typ.NumOut())
	}
	if typ.In(0) != jsontextDecoderType {
		return fnVal, typ, fmt.Errorf("operator %q invalid function signature (first param must be *jsontext.Decoder; got %s)", name, typ.In(0))
	}
	arg := typ.In(1)
	if arg.Kind() != reflect.Pointer || arg.Elem().Kind() == reflect.Invalid {
		return fnVal, typ, fmt.Errorf("operator %q invalid function signature (second param must be pointer to concrete type; got %s)", name, arg)
	}
	if typ.Out(0) != errorType {
		return fnVal, typ, fmt.Errorf("operator %q invalid function signature (return type must be error; got %s)", name, typ.Out(0))
	}
	return fnVal, arg.Elem(), nil
}

func (r *OperatorRegistry) Register(name string, fn any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[name]; exists {
		return fmt.Errorf("operator %q already registered", name)
	}

	fnVal, elemType, err := validateFuncSignature(name, fn)
	if err != nil {
		return err
	}

	r.entries[name] = funcEntry{fn: fnVal, elem: elemType}
	return nil
}

func (r *OperatorRegistry) Call(name string, dec *jsontext.Decoder) (any, error) {
	r.mu.RLock()
	ent, ok := r.entries[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("operator %q not registered", name)
	}

	argv := reflect.New(ent.elem)
	results := ent.fn.Call([]reflect.Value{reflect.ValueOf(dec), argv})

	if len(results) != 1 {
		return nil, fmt.Errorf("operator %q invalid function signature (expected 1 output; got %d)", name, len(results))
	}

	if errVal := results[0].Interface(); errVal != nil {
		return nil, fmt.Errorf("operator %q execution: %w", name, errVal.(error))
	}

	return argv.Elem().Interface(), nil
}

func Register[T any](r *OperatorRegistry, name string, fn func(dec *jsontext.Decoder) (T, error)) error {
	wrapped := func(dec *jsontext.Decoder, val *T) error {
		res, err := fn(dec)
		if err != nil {
			return fmt.Errorf("operator %q execution: %w", name, err)
		}
		*val = res
		return nil
	}
	return r.Register(name, wrapped)
}

func MustRegister[T any](r *OperatorRegistry, name string, fn func(dec *jsontext.Decoder) (T, error)) {
	if err := Register(r, name, fn); err != nil {
		panic(err)
	}
}
