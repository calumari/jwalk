package jwalk

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func Unmarshalers(r *OperatorRegistry) *json.Unmarshalers {
	return json.JoinUnmarshalers(
		Unmarshaler(r),
		DocumentUnmarshaler(r),
		CollectionUnmarshaler(r),
	)
}

// Unmarshaler returns a custom JSON unmarshaller for handling operator objects.
// If the first key starts with '$', it invokes the corresponding operator from
// the registry. Otherwise, it unmarshals the object as a map[string]any.
func Unmarshaler(r *OperatorRegistry) *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *any) error {
		if dec.PeekKind() != '{' {
			return json.SkipFunc
		}
		if _, err := dec.ReadToken(); err != nil { // consume opening '{'
			return fmt.Errorf("read opening '{': %w", err)
		}

		res := make(D, 0)

		// fast path for empty object
		if dec.PeekKind() == '}' {
			if _, err := dec.ReadToken(); err != nil {
				return fmt.Errorf("read closing '}': %w", err)
			}

			*v = res
			return nil
		}

		var firstKey string
		if err := json.UnmarshalDecode(dec, &firstKey); err != nil {
			return fmt.Errorf("read first object key: %w", err)
		}

		// handle operator objects {"$<name>": ...}
		if len(firstKey) > 0 && firstKey[0] == '$' {
			vv, err := r.Call(firstKey[1:], dec)
			if err != nil {
				return fmt.Errorf("operator %q: %w", firstKey, err)
			}

			// skip any extra fields in the object. this is necessary to ensure
			// we don't leave the decoder in an invalid state. if the operator
			// expects to handle extra fields, it should do so itself.
			for dec.PeekKind() != '}' {
				if err := dec.SkipValue(); err != nil {
					return fmt.Errorf("operator %q skip extra fields: %w", firstKey, err)
				}
			}
			if _, err := dec.ReadToken(); err != nil { // consume closing '}'
				return fmt.Errorf("operator %q read closing '}': %w", firstKey, err)
			}

			*v = vv
			return nil
		}

		// fallback: treat as regular object and build a Document
		var firstVal any
		if err := json.UnmarshalDecode(dec, &firstVal); err != nil {
			return fmt.Errorf("read value for key %q: %w", firstKey, err)
		}
		res = append(res, E{Key: firstKey, Value: firstVal})

		for dec.PeekKind() != '}' {
			var k string
			if err := json.UnmarshalDecode(dec, &k); err != nil {
				return fmt.Errorf("read object key: %w", err)
			}
			var vv any
			if err := json.UnmarshalDecode(dec, &vv); err != nil {
				return fmt.Errorf("unmarshal value: %w", err)
			}
			res = append(res, E{Key: k, Value: vv})
		}

		if _, err := dec.ReadToken(); err != nil { // consume closing '}'
			return fmt.Errorf("read closing '}': %w", err)
		}

		*v = res
		return nil
	})
}

func DocumentUnmarshaler(r *OperatorRegistry) *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *D) error {
		if dec.PeekKind() != '{' {
			return json.SkipFunc
		}
		if _, err := dec.ReadToken(); err != nil { // consume opening '{'
			return fmt.Errorf("read opening '{': %w", err)
		}

		res := make(D, 0)

		// fast path for empty object
		if dec.PeekKind() == '}' {
			if _, err := dec.ReadToken(); err != nil {
				return fmt.Errorf("read closing '}': %w", err)
			}

			*v = res
			return nil
		}

		for dec.PeekKind() != '}' {
			var k string
			if err := json.UnmarshalDecode(dec, &k); err != nil {
				return fmt.Errorf("read object key: %w", err)
			}
			var vv any
			if err := json.UnmarshalDecode(dec, &vv); err != nil {
				return fmt.Errorf("unmarshal value: %w", err)
			}
			res = append(res, E{Key: k, Value: vv})
		}

		if _, err := dec.ReadToken(); err != nil { // consume closing '}'
			return fmt.Errorf("read closing '}': %w", err)
		}

		*v = res
		return nil
	})
}

func CollectionUnmarshaler(r *OperatorRegistry) *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *A) error {
		if dec.PeekKind() != '[' {
			return json.SkipFunc
		}
		if _, err := dec.ReadToken(); err != nil { // consume opening '['
			return fmt.Errorf("read opening '[': %w", err)
		}

		res := make(A, 0)

		// fast path for empty array
		if dec.PeekKind() == ']' {
			if _, err := dec.ReadToken(); err != nil {
				return fmt.Errorf("read closing ']': %w", err)
			}

			*v = res
			return nil
		}

		for dec.PeekKind() != ']' {
			var vv any
			if err := json.UnmarshalDecode(dec, &vv); err != nil {
				return fmt.Errorf("unmarshal value: %w", err)
			}
			res = append(res, vv)
		}

		if _, err := dec.ReadToken(); err != nil { // consume closing ']'
			return fmt.Errorf("read closing ']': %w", err)
		}

		*v = res
		return nil
	})
}
