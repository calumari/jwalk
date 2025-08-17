package jwalk

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Unmarshaler returns a custom JSON unmarshaller for handling operator objects.
// If the first key starts with '$', it invokes the corresponding operator from
// the registry. Otherwise, it unmarshals the object as a map[string]any.
func Unmarshaler(r *OperatorRegistry) *json.Unmarshalers {
	return json.UnmarshalFromFunc(func(dec *jsontext.Decoder, val *any) error {
		if dec.PeekKind() != '{' {
			return json.SkipFunc
		}
		if _, err := dec.ReadToken(); err != nil { // consume opening '{'
			return fmt.Errorf("read opening '{': %w", err)
		}

		// fast path for empty object
		if dec.PeekKind() == '}' {
			if _, err := dec.ReadToken(); err != nil {
				return fmt.Errorf("read closing '}': %w", err)
			}
			return nil
		}

		var firstKey string
		if err := json.UnmarshalDecode(dec, &firstKey); err != nil {
			return fmt.Errorf("read first object key: %w", err)
		}

		// handle operator objects {"$<name>": ...}
		if len(firstKey) > 0 && firstKey[0] == '$' {
			res, err := r.Call(firstKey[1:], dec)
			if err != nil {
				return fmt.Errorf("operator %q: %w", firstKey, err)
			}
			*val = res

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

			return nil
		}

		// fallback: unmarshal as map[string]any
		var firstVal any
		if err := json.UnmarshalDecode(dec, &firstVal); err != nil {
			return fmt.Errorf("read value for key %q: %w", firstKey, err)
		}

		m := map[string]any{firstKey: firstVal}

		// read the remaining key-value pairs
		for dec.PeekKind() != '}' {
			var k string
			if err := json.UnmarshalDecode(dec, &k); err != nil {
				return fmt.Errorf("read object key: %w", err)
			}
			var v any
			if err := json.UnmarshalDecode(dec, &v); err != nil {
				return fmt.Errorf("read value for key %q: %w", k, err)
			}
			m[k] = v
		}

		if _, err := dec.ReadToken(); err != nil { // consume closing '}'
			return fmt.Errorf("read closing '}': %w", err)
		}

		*val = m
		return nil
	})
}
