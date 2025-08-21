package jwalk

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Unmarshaler returns a custom JSON unmarshaller that:
//   - Wraps JSON objects as type D (ordered document) rather than map[string]any
//   - Wraps JSON arrays as type A so callers can distinguish from []any
//   - Detects operator objects of the form {"$<name>": <value>[, ...ignored...]}
//     and dispatches to the registered operator implementation. Any extra
//     fields after the operator root field are currently ignored (skipped).
//   - Leaves primitive JSON values (string, number, bool, null) to other
//     unmarshaler logic by returning json.SkipFunc.
//
// Empty objects ({}) produce an empty D; empty arrays ([]) produce an empty A.
func Unmarshaler(r *OperatorRegistry) *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *any) error {
		switch dec.PeekKind() {
		case '{':
			if _, err := dec.ReadToken(); err != nil { // consume '{'
				return fmt.Errorf("read object open: %w", err)
			}

			// handle empty object early.
			if dec.PeekKind() == '}' {
				if _, err := dec.ReadToken(); err != nil { // consume '}'
					return fmt.Errorf("read object close: %w", err)
				}
				*v = D{}
				return nil
			}

			// read first key to decide if operator object.
			var firstKey string
			if err := json.UnmarshalDecode(dec, &firstKey); err != nil {
				return fmt.Errorf("read object first key: %w", err)
			}

			// operator object: {"$<name>": ...}
			if len(firstKey) > 0 && firstKey[0] == '$' {
				vv, err := r.Call(firstKey[1:], dec)
				if err != nil {
					return fmt.Errorf("operator %q call: %w", firstKey, err)
				}
				// skip any remaining fields (currently permissive behavior).
				for dec.PeekKind() != '}' {
					if err := dec.SkipValue(); err != nil {
						return fmt.Errorf("operator %q skip extra field: %w", firstKey, err)
					}
				}
				if _, err := dec.ReadToken(); err != nil { // consume closing '}'
					return fmt.Errorf("operator %q read object close: %w", firstKey, err)
				}
				*v = vv
				return nil
			}

			// regular object: build a D preserving key order.
			res := make(D, 0)
			var firstVal any
			if err := json.UnmarshalDecode(dec, &firstVal); err != nil {
				return fmt.Errorf("read object value for key %q: %w", firstKey, err)
			}
			res = append(res, E{Key: firstKey, Value: firstVal})

			for dec.PeekKind() != '}' {
				var k string
				if err := json.UnmarshalDecode(dec, &k); err != nil {
					return fmt.Errorf("read object key: %w", err)
				}
				var vv any
				if err := json.UnmarshalDecode(dec, &vv); err != nil {
					return fmt.Errorf("read object value: %w", err)
				}
				res = append(res, E{Key: k, Value: vv})
			}

			if _, err := dec.ReadToken(); err != nil { // consume closing '}'
				return fmt.Errorf("read object close: %w", err)
			}
			*v = res
			return nil

		case '[':
			if _, err := dec.ReadToken(); err != nil { // consume '['
				return fmt.Errorf("read array open: %w", err)
			}
			// empty array?
			if dec.PeekKind() == ']' {
				if _, err := dec.ReadToken(); err != nil { // consume closing ']'
					return fmt.Errorf("read array close: %w", err)
				}
				*v = A{}
				return nil
			}
			arr := make(A, 0)
			for dec.PeekKind() != ']' {
				var elem any
				if err := json.UnmarshalDecode(dec, &elem); err != nil {
					return fmt.Errorf("read array element: %w", err)
				}
				arr = append(arr, elem)
			}
			if _, err := dec.ReadToken(); err != nil { // consume closing ']'
				return fmt.Errorf("read array close: %w", err)
			}
			*v = arr
			return nil

		default: // primitive (string, number, bool, null) -> let outer logic handle
			return json.SkipFunc
		}
	})
}
