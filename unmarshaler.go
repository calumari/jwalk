package jwalk

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Unmarshalers returns the full set of jwalk unmarshalers allowing decoding
// into:
//   - any/interface{} -> objects as D, arrays as A, operator objects dispatched
//   - *D              -> direct ordered object decoding
//   - *A              -> direct array decoding
func Unmarshalers(r *OperatorRegistry) *json.Unmarshalers {
	return json.JoinUnmarshalers(
		unmarshalValue(r), // *any (objects, arrays, operators)
		unmarshalDocument(),
		unmarshalCollection(),
	)
}

// Unmarshaler returns a custom JSON unmarshaller that:
//   - Wraps JSON objects as type D (ordered document) rather than map[string]any
//   - Wraps JSON arrays as type A so callers can distinguish from []any
//   - Detects operator objects of the form {"$<name>": <value>[, ...ignored...]}
//     and dispatches to the registered operator implementation. Any extra
//     fields after the operator root field are currently ignored (skipped).
//   - Leaves primitive JSON values (string, number, bool, null) to other
//     unmarshalValue logic by returning json.SkipFunc.
//
// Empty objects ({}) produce an empty D; empty arrays ([]) produce an empty A.
func unmarshalValue(r *OperatorRegistry) *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *any) error {
		switch dec.PeekKind() {
		case '{':
			// object, potentially operator
			val, wasOperator, err := decodeObject(dec, r, true)
			if err != nil {
				return err
			}
			if wasOperator {
				*v = val
			} else {
				*v = val.(D)
			}
			return nil
		case '[':
			arr, err := decodeArray(dec, r)
			if err != nil {
				return err
			}
			*v = arr
			return nil
		default:
			return json.SkipFunc
		}
	})
}

// DocumentUnmarshaler provides decoding of a JSON object into a *D when the
// target type is *D (ordered key preservation). Operator objects are NOT
// interpreted here; that only happens when decoding into interface{} via
// Unmarshaler. This separation lets callers opt-in to operator semantics only
// when decoding into interface{} graphs.
func unmarshalDocument() *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *D) error {
		if dec.PeekKind() != '{' {
			return json.SkipFunc
		}
		val, _, err := decodeObject(dec, nil, false)
		if err != nil {
			return err
		}
		*v = val.(D)
		return nil
	})
}

// CollectionUnmarshaler provides decoding of a JSON array into an *A when the
// target type is *A.
func unmarshalCollection() *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *A) error {
		if dec.PeekKind() != '[' {
			return json.SkipFunc
		}
		arr, err := decodeArray(dec, nil)
		if err != nil {
			return err
		}
		*v = arr
		return nil
	})
}

// decodeObject decodes a JSON object into a D or operator value. If
// handleOperators is true and the first key is an operator, it returns the
// operator value (any) with wasOperator=true. Otherwise it returns D.
func decodeObject(dec *jsontext.Decoder, r *OperatorRegistry, handleOperators bool) (val any, wasOperator bool, err error) {
	if _, err = dec.ReadToken(); err != nil { // '{'
		return nil, false, fmt.Errorf("read object open: %w", err)
	}
	if dec.PeekKind() == '}' { // empty
		if _, err = dec.ReadToken(); err != nil { // '}'
			return nil, false, fmt.Errorf("read object close: %w", err)
		}
		return D{}, false, nil
	}
	// read first key
	var firstKey string
	if err = json.UnmarshalDecode(dec, &firstKey); err != nil {
		return nil, false, fmt.Errorf("read object first key: %w", err)
	}
	if handleOperators && len(firstKey) > 0 && firstKey[0] == '$' && r != nil {
		vv, opErr := r.Call(firstKey[1:], dec)
		if opErr != nil {
			return nil, false, fmt.Errorf("operator %q call: %w", firstKey, opErr)
		}
		for dec.PeekKind() != '}' { // skip remaining fields
			if opErr = dec.SkipValue(); opErr != nil {
				return nil, false, fmt.Errorf("operator %q skip extra field: %w", firstKey, opErr)
			}
		}
		if _, opErr = dec.ReadToken(); opErr != nil {
			return nil, false, fmt.Errorf("operator %q read object close: %w", firstKey, opErr)
		}
		return vv, true, nil
	}
	// regular object path
	var firstVal any
	if err = json.UnmarshalDecode(dec, &firstVal); err != nil {
		return nil, false, fmt.Errorf("read object value for key %q: %w", firstKey, err)
	}
	res := D{{Key: firstKey, Value: firstVal}}
	for dec.PeekKind() != '}' {
		var k string
		if err = json.UnmarshalDecode(dec, &k); err != nil {
			return nil, false, fmt.Errorf("read object key: %w", err)
		}
		var vv any
		if err = json.UnmarshalDecode(dec, &vv); err != nil {
			return nil, false, fmt.Errorf("read object value: %w", err)
		}
		res = append(res, E{Key: k, Value: vv})
	}
	if _, err = dec.ReadToken(); err != nil { // '}'
		return nil, false, fmt.Errorf("read object close: %w", err)
	}
	return res, false, nil
}

// decodeArray decodes a JSON array into A.
func decodeArray(dec *jsontext.Decoder, _ *OperatorRegistry) (A, error) {
	if _, err := dec.ReadToken(); err != nil { // '['
		return nil, fmt.Errorf("read array open: %w", err)
	}
	if dec.PeekKind() == ']' { // empty
		if _, err := dec.ReadToken(); err != nil {
			return nil, fmt.Errorf("read array close: %w", err)
		}
		return A{}, nil
	}
	arr := make(A, 0)
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
