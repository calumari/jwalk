package jwalk

import (
	"testing"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/stretchr/testify/require"
)

func unmarshal(t *testing.T, r *OperatorRegistry, src string) any {
	t.Helper()
	var out any
	err := json.Unmarshal([]byte(src), &out, json.WithUnmarshalers(Unmarshalers(r)))
	require.NoError(t, err)
	return out
}

func assertD(t *testing.T, v any) D {
	t.Helper()
	d, ok := v.(D)
	require.True(t, ok, "expected D, got %T", v)
	return d
}

func assertA(t *testing.T, v any) A {
	t.Helper()
	a, ok := v.(A)
	require.True(t, ok, "expected A, got %T", v)
	return a
}

func Test_unmarshalValue(t *testing.T) {
	t.Run("empty object returns empty D", func(t *testing.T) {
		r := NewOperatorRegistry()
		d := assertD(t, unmarshal(t, r, `{}`))
		require.Len(t, d, 0)
	})

	t.Run("empty array returns empty A", func(t *testing.T) {
		r := NewOperatorRegistry()
		a := assertA(t, unmarshal(t, r, `[]`))
		require.Len(t, a, 0)
	})

	t.Run("regular object preserves ordering", func(t *testing.T) {
		r := NewOperatorRegistry()
		d := assertD(t, unmarshal(t, r, `{"a":1,"b":2}`))
		require.Equal(t, []E{{Key: "a", Value: float64(1)}, {Key: "b", Value: float64(2)}}, []E(d))
	})

	t.Run("nested array wraps object elements", func(t *testing.T) {
		r := NewOperatorRegistry()
		a := assertA(t, unmarshal(t, r, `[1,{"x":2}]`))
		require.Len(t, a, 2)
		require.Equal(t, float64(1), a[0])
		d := assertD(t, a[1])
		require.Equal(t, "x", d[0].Key)
	})

	t.Run("operator object dispatches and skips extra fields", func(t *testing.T) {
		r := NewOperatorRegistry()
		require.NoError(t, Register(r, "val", func(dec *jsontext.Decoder) (int, error) {
			var num int
			if err := json.UnmarshalDecode(dec, &num); err != nil {
				return 0, err
			}
			return num, nil
		}))
		v := unmarshal(t, r, `{"$val": 42, "ignored": true}`)
		require.Equal(t, 42, v)
	})

	t.Run("primitive value bypassed via SkipFunc", func(t *testing.T) {
		r := NewOperatorRegistry()
		v := unmarshal(t, r, `123`)
		require.Equal(t, float64(123), v)
	})
}

func Test_unmarshalDocument(t *testing.T) {
	t.Run("non-object document decodes into empty D", func(t *testing.T) {
		var d D
		err := json.Unmarshal([]byte(`null`), &d, json.WithUnmarshalers(unmarshalDocument()))
		require.NoError(t, err)
		require.Len(t, d, 0)
	})

	t.Run("unclosed object returns error", func(t *testing.T) {
		var d D
		err := json.Unmarshal([]byte(`{`), &d, json.WithUnmarshalers(unmarshalDocument()))
		require.Error(t, err)
	})

	t.Run("empty object decodes into empty D", func(t *testing.T) {
		var d D
		err := json.Unmarshal([]byte(`{}`), &d, json.WithUnmarshalers(unmarshalDocument()))
		require.NoError(t, err)
		require.Len(t, d, 0)
	})

	t.Run("target D preserves ordering and skips operator dispatch", func(t *testing.T) {
		r := NewOperatorRegistry()
		called := false
		require.NoError(t, Register(r, "val", func(dec *jsontext.Decoder) (int, error) {
			called = true
			var num int
			if err := json.UnmarshalDecode(dec, &num); err != nil {
				return 0, err
			}
			return num, nil
		}))

		// use full unmarshalers (includes operator logic) but target D so operator must not trigger
		var d D
		err := json.Unmarshal([]byte(`{"$val":42,"b":2}`), &d, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)
		require.False(t, called, "operator should not be dispatched when decoding into *D")
		require.Equal(t, []E{{Key: "$val", Value: float64(42)}, {Key: "b", Value: float64(2)}}, []E(d))
	})

	t.Run("nested operator inside D dispatched", func(t *testing.T) {
		r := NewOperatorRegistry()
		require.NoError(t, Register(r, "val", func(dec *jsontext.Decoder) (int, error) {
			var num int
			if err := json.UnmarshalDecode(dec, &num); err != nil {
				return 0, err
			}
			return num, nil
		}))
		var d D
		err := json.Unmarshal([]byte(`{"outer":{"$val":7}}`), &d, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)
		require.Len(t, d, 1)
		innerVal, ok := d[0].Value.(D)
		if ok {
			t.Fatalf("expected operator dispatch inside nested object, got D: %#v", innerVal)
		}
		require.Equal(t, "outer", d[0].Key)
		require.Equal(t, 7, d[0].Value)
	})
}

func Test_unmarshalCollection(t *testing.T) {
	t.Run("non-array collection decodes into empty A", func(t *testing.T) {
		var a A
		err := json.Unmarshal([]byte(`null`), &a, json.WithUnmarshalers(unmarshalCollection()))
		require.NoError(t, err)
		require.Len(t, a, 0)
	})

	t.Run("unclosed array returns error", func(t *testing.T) {
		var a A
		err := json.Unmarshal([]byte(`[`), &a, json.WithUnmarshalers(unmarshalCollection()))
		require.Error(t, err)
	})

	t.Run("empty array decodes into empty A", func(t *testing.T) {
		var a A
		err := json.Unmarshal([]byte(`[]`), &a, json.WithUnmarshalers(unmarshalCollection()))
		require.NoError(t, err)
		require.Len(t, a, 0)
	})

	t.Run("array regular object element decodes as D", func(t *testing.T) {
		var a A
		err := json.Unmarshal([]byte(`[{"a":1}]`), &a, json.WithUnmarshalers(Unmarshalers(NewOperatorRegistry())))
		require.NoError(t, err)
		require.Len(t, a, 1)
		d, ok := a[0].(D)
		require.True(t, ok, "expected D for object element, got %T", a[0])
		require.Equal(t, []E{{Key: "a", Value: float64(1)}}, []E(d))
	})

	t.Run("array operator object element dispatches operator", func(t *testing.T) {
		r := NewOperatorRegistry()
		require.NoError(t, Register(r, "val", func(dec *jsontext.Decoder) (int, error) {
			var num int
			if err := json.UnmarshalDecode(dec, &num); err != nil {
				return 0, err
			}
			return num, nil
		}))
		var a A
		err := json.Unmarshal([]byte(`[1,{"$val":5}]`), &a, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)
		require.Equal(t, float64(1), a[0])
		require.Equal(t, 5, a[1])
	})
}
