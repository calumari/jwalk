package jwalk

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Unmarshalers returns the full set of jwalk unmarshalers. These allow decoding
// into:
//   - any / interface{}: objects as Document, arrays as Array, and sentinel objects
//     dispatched through registered directives
//   - *Document: ordered object decoding
//   - *Array: ordered array decoding
func Unmarshalers(reg *Registry) *json.Unmarshalers {
	return json.JoinUnmarshalers(
		unmarshalValue(reg), // *any (objects, arrays, directives)
		unmarshalDocument(),
		unmarshalCollection(),
	)
}

// unmarshalValue returns an unmarshaler for *any. It:
//
//   - Wraps JSON objects as Document instead of map[string]any
//   - Wraps JSON arrays as Array - Detects sentinel objects {"$<name>": <value>[, ...]}
//     and invokes the corresponding directive if registered
//   - Leaves primitive values (string, number, bool, null) to other unmarshalers
//
// Empty objects decode as an empty Document, and empty arrays as an empty
// Array.
func unmarshalValue(reg *Registry) *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *any) error {
		switch dec.PeekKind() {
		case '{':
			// object (possibly a directive sentinel)
			val, wasDirective, err := unmarshalObject(dec, reg, true)
			if err != nil {
				return err
			}

			if wasDirective {
				*v = val
			} else {
				*v = val.(Document)
			}
			return nil

		case '[':
			// array
			arr, err := unmarshalArray(dec, reg)
			if err != nil {
				return err
			}
			*v = arr
			return nil

		default:
			// let other unmarshalers handle primitives
			return json.SkipFunc
		}
	})
}

// unmarshalDocument decodes a JSON object into *Document, preserving key order.
// Directive sentinel objects are not interpreted here; that only when decoding
// into interface{} via unmarshalValue. This allows callers to opt in to
// directive semantics selectively.
func unmarshalDocument() *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *Document) error {
		if dec.PeekKind() != '{' {
			return json.SkipFunc
		}

		val, _, err := unmarshalObject(dec, nil, false)
		if err != nil {
			return err
		}

		*v = val.(Document)
		return nil
	})
}

// unmarshalCollection decodes a JSON array into *Array.
func unmarshalCollection() *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *Array) error {
		if dec.PeekKind() != '[' {
			return json.SkipFunc
		}

		arr, err := unmarshalArray(dec, nil)
		if err != nil {
			return err
		}

		*v = arr
		return nil
	})
}

// unmarshalObject decodes a JSON object. It returns:
//
//   - (val, true, nil) if allowDirective is true, the first key starts with "$", and
//     the registry successfully dispatches the directive.
//   - (Document, false, nil) otherwise, preserving key order.
func unmarshalObject(dec *jsontext.Decoder, reg *Registry, allowDirective bool) (val any, wasDirective bool, err error) {
	if _, err = dec.ReadToken(); err != nil { // '{'
		return nil, false, fmt.Errorf("read object open: %w", err)
	}

	if dec.PeekKind() == '}' { // empty
		if _, err = dec.ReadToken(); err != nil { // '}'
			return nil, false, fmt.Errorf("read object close: %w", err)
		}
		return Document{}, false, nil
	}

	// read first key
	var firstKey string
	if err = json.UnmarshalDecode(dec, &firstKey); err != nil {
		return nil, false, fmt.Errorf("read object first key: %w", err)
	}

	if allowDirective && firstKey != "" && firstKey[0] == '$' {
		// Pass full sentinel (still accepted) so handler context includes it.
		vv, err := reg.InvokeDirective(firstKey[1:], dec)
		if err != nil {
			// registry already provided context in error
			return nil, false, err
		}

		// skip any extra fields after the directive root field
		for dec.PeekKind() != '}' {
			if err = dec.SkipValue(); err != nil {
				return nil, false, fmt.Errorf("directive %q skip extra field: %w", firstKey, err)
			}
		}
		if _, err = dec.ReadToken(); err != nil {
			return nil, false, fmt.Errorf("directive %q read object close: %w", firstKey, err)
		}

		return vv, true, nil
	}

	// regular object path
	var firstVal any
	if err = json.UnmarshalDecode(dec, &firstVal); err != nil {
		return nil, false, fmt.Errorf("read object value for key %q: %w", firstKey, err)
	}

	res := Document{{Key: firstKey, Value: firstVal}}

	for dec.PeekKind() != '}' {
		var k string
		if err = json.UnmarshalDecode(dec, &k); err != nil {
			return nil, false, fmt.Errorf("read object key: %w", err)
		}

		var vv any
		if err = json.UnmarshalDecode(dec, &vv); err != nil {
			return nil, false, fmt.Errorf("read object value: %w", err)
		}

		res = append(res, Entry{Key: k, Value: vv})
	}

	if _, err = dec.ReadToken(); err != nil { // '}'
		return nil, false, fmt.Errorf("read object close: %w", err)
	}

	return res, false, nil
}

// unmarshalArray decodes a JSON array into Array.
func unmarshalArray(dec *jsontext.Decoder, _ *Registry) (Array, error) {
	if _, err := dec.ReadToken(); err != nil { // '['
		return nil, fmt.Errorf("read array open: %w", err)
	}

	if dec.PeekKind() == ']' { // empty
		if _, err := dec.ReadToken(); err != nil {
			return nil, fmt.Errorf("read array close: %w", err)
		}
		return Array{}, nil
	}

	arr := make(Array, 0)

	for dec.PeekKind() != ']' {
		var elem any
		if err := json.UnmarshalDecode(dec, &elem); err != nil {
			return nil, fmt.Errorf("read array element: %w", err)
		}
		arr = append(arr, elem)
	}

	if _, err := dec.ReadToken(); err != nil { // ']'
		return nil, fmt.Errorf("read array close: %w", err)
	}

	return arr, nil
}
