package jwalk

import (
	"testing"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unmarshal(t *testing.T, r *Registry, src string) any {
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
		r := newRegistry()
		got := unmarshal(t, r, `{}`)
		d := assertD(t, got)
		require.Len(t, d, 0)
	})

	t.Run("empty array returns empty A", func(t *testing.T) {
		r := newRegistry()
		got := unmarshal(t, r, `[]`)
		a := assertA(t, got)
		require.Len(t, a, 0)
	})

	t.Run("regular object preserves ordering", func(t *testing.T) {
		r := newRegistry()
		got := unmarshal(t, r, `{"a":1,"b":2}`)
		d := assertD(t, got)
		want := []E{{Key: "a", Value: float64(1)}, {Key: "b", Value: float64(2)}}
		require.Equal(t, want, []E(d))
	})

	t.Run("nested array wraps object elements", func(t *testing.T) {
		r := newRegistry()
		got := unmarshal(t, r, `[1,{"x":2}]`)
		a := assertA(t, got)
		require.Len(t, a, 2)
		require.Equal(t, float64(1), a[0])

		d := assertD(t, a[1])
		require.Equal(t, "x", d[0].Key)
	})

	t.Run("sentinel object dispatches and skips extra fields", func(t *testing.T) {
		r, err := NewRegistry(func(r *Registry) error {
			return NewDirective("val", func(dec *jsontext.Decoder) (int, error) {
				var num int
				if err := json.UnmarshalDecode(dec, &num); err != nil {
					return 0, err
				}
				return num, nil
			})(r)
		})
		require.NoError(t, err)

		got := unmarshal(t, r, `{"$val": 42, "ignored": true}`)
		require.Equal(t, 42, got)
	})

	t.Run("primitive value returns primitive", func(t *testing.T) {
		r := newRegistry()
		got := unmarshal(t, r, `123`)
		require.Equal(t, float64(123), got)
	})

	t.Run("sentinel object directive error surfaces", func(t *testing.T) {
		r, err := NewRegistry(func(r *Registry) error {
			return NewDirective("val", func(dec *jsontext.Decoder) (int, error) {
				return 0, assert.AnError
			})(r)
		})
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$val":1}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
	})

	t.Run("sentinel object ambiguous short name returns error", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error { *v = 1; return nil }
		err := r.Register("a.value", fn)
		require.NoError(t, err)
		err = r.Register("b.value", fn)
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$value":1}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ambiguous")
	})

	t.Run("sentinel object unique short name resolves namespaced", func(t *testing.T) {
		r := newRegistry()
		// only namespaced directive; resolve via short name
		err := r.Register("ns.num", func(dec *jsontext.Decoder, v *int) error {
			var n int
			if err := json.UnmarshalDecode(dec, &n); err != nil {
				return err
			}
			*v = n
			return nil
		})
		require.NoError(t, err)

		got := unmarshal(t, r, `{"$num": 7}`)
		require.Equal(t, 7, got)
	})

	t.Run("sentinel object bare preferred when bare and namespaced coexist", func(t *testing.T) {
		r := newRegistry()
		// bare sets value 1; namespaced sets value 2; expect 1
		err := r.Register("num", func(dec *jsontext.Decoder, v *int) error { *v = 1; return nil })
		require.NoError(t, err)
		err = r.Register("ns.num", func(dec *jsontext.Decoder, v *int) error { *v = 2; return nil })
		require.NoError(t, err)

		got := unmarshal(t, r, `{"$num": null}`) // value ignored by our funcs
		require.Equal(t, 1, got)
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

	t.Run("target D preserves ordering and skips directive dispatch", func(t *testing.T) {
		called := false
		r, err := NewRegistry(func(r *Registry) error {
			return NewDirective("val", func(dec *jsontext.Decoder) (int, error) {
				called = true
				var num int
				if err := json.UnmarshalDecode(dec, &num); err != nil {
					return 0, err
				}
				return num, nil
			})(r)
		})
		require.NoError(t, err)

		// use full unmarshalers (includes directive logic) but target D so directive must not trigger
		var d D
		err = json.Unmarshal([]byte(`{"$val":42,"b":2}`), &d, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)
		require.False(t, called, "directive dispatched when decoding into *D")

		want := []E{{Key: "$val", Value: float64(42)}, {Key: "b", Value: float64(2)}}
		require.Equal(t, want, []E(d))
	})

	t.Run("nested directive inside D dispatched", func(t *testing.T) {
		r, err := NewRegistry(func(r *Registry) error {
			return NewDirective("val", func(dec *jsontext.Decoder) (int, error) {
				var num int
				if err := json.UnmarshalDecode(dec, &num); err != nil {
					return 0, err
				}
				return num, nil
			})(r)
		})
		require.NoError(t, err)

		var d D
		err = json.Unmarshal([]byte(`{"outer":{"$val":7}}`), &d, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)
		require.Len(t, d, 1)

		innerVal, ok := d[0].Value.(D)
		if ok {
			t.Fatalf("expected directive dispatch inside nested object, got D: %#v", innerVal)
		}
		require.Equal(t, "outer", d[0].Key)
		require.Equal(t, 7, d[0].Value)
	})

	t.Run("multiple object fields preserve order", func(t *testing.T) {
		var d D
		err := json.Unmarshal([]byte(`{"c":3,"a":1,"b":2}`), &d, json.WithUnmarshalers(unmarshalDocument()))
		require.NoError(t, err)

		want := []E{
			{Key: "c", Value: float64(3)},
			{Key: "a", Value: float64(1)},
			{Key: "b", Value: float64(2)},
		}
		require.Equal(t, want, []E(d))
	})

	t.Run("nested objects become D", func(t *testing.T) {
		var d D
		err := json.Unmarshal([]byte(`{"nested":{"x":1}}`), &d, json.WithUnmarshalers(Unmarshalers(newRegistry())))
		require.NoError(t, err)
		require.Len(t, d, 1)

		nested, ok := d[0].Value.(D)
		require.True(t, ok, "expected nested object to be D")
		require.Len(t, nested, 1)
		require.Equal(t, "x", nested[0].Key)
		require.Equal(t, float64(1), nested[0].Value)
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
		err := json.Unmarshal([]byte(`[{"a":1}]`), &a, json.WithUnmarshalers(Unmarshalers(newRegistry())))
		require.NoError(t, err)
		require.Len(t, a, 1)

		d, ok := a[0].(D)
		require.True(t, ok, "expected D for object element, got %T", a[0])
		want := []E{{Key: "a", Value: float64(1)}}
		require.Equal(t, want, []E(d))
	})

	t.Run("array sentinel object element dispatches directive", func(t *testing.T) {
		r, err := NewRegistry(func(r *Registry) error {
			return NewDirective("val", func(dec *jsontext.Decoder) (int, error) {
				var num int
				if err := json.UnmarshalDecode(dec, &num); err != nil {
					return 0, err
				}
				return num, nil
			})(r)
		})
		require.NoError(t, err)

		var a A
		err = json.Unmarshal([]byte(`[1,{"$val":5}]`), &a, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)
		require.Equal(t, float64(1), a[0])
		require.Equal(t, 5, a[1])
	})

	t.Run("mixed array elements decode correctly", func(t *testing.T) {
		var a A
		err := json.Unmarshal([]byte(`[1,"hello",true,null,{"x":2},[3,4]]`), &a, json.WithUnmarshalers(Unmarshalers(newRegistry())))
		require.NoError(t, err)
		require.Len(t, a, 6)

		require.Equal(t, float64(1), a[0])
		require.Equal(t, "hello", a[1])
		require.Equal(t, true, a[2])
		require.Equal(t, nil, a[3])

		// object becomes D
		d, ok := a[4].(D)
		require.True(t, ok)
		require.Equal(t, "x", d[0].Key)
		require.Equal(t, float64(2), d[0].Value)

		// nested array becomes A
		nested, ok := a[5].(A)
		require.True(t, ok)
		require.Len(t, nested, 2)
		require.Equal(t, float64(3), nested[0])
		require.Equal(t, float64(4), nested[1])
	})

	t.Run("deeply nested arrays preserve structure", func(t *testing.T) {
		var a A
		err := json.Unmarshal([]byte(`[[1,[2,3]],4]`), &a, json.WithUnmarshalers(Unmarshalers(newRegistry())))
		require.NoError(t, err)
		require.Len(t, a, 2)

		// first element is nested array
		level1, ok := a[0].(A)
		require.True(t, ok)
		require.Len(t, level1, 2)
		require.Equal(t, float64(1), level1[0])

		// second element of first array is also array
		level2, ok := level1[1].(A)
		require.True(t, ok)
		require.Len(t, level2, 2)
		require.Equal(t, float64(2), level2[0])
		require.Equal(t, float64(3), level2[1])

		// second top-level element is primitive
		require.Equal(t, float64(4), a[1])
	})
}

func TestUnmarshalers(t *testing.T) {
	t.Run("returns non-nil unmarshaler set", func(t *testing.T) {
		r := newRegistry()
		got := Unmarshalers(r)
		require.NotNil(t, got)
	})

	t.Run("works with nil registry", func(t *testing.T) {
		got := Unmarshalers(nil)
		require.NotNil(t, got)

		// should still handle basic document/collection unmarshaling
		var d D
		err := json.Unmarshal([]byte(`{"key":"value"}`), &d, json.WithUnmarshalers(got))
		require.NoError(t, err)
		require.Len(t, d, 1)
		require.Equal(t, "key", d[0].Key)
		require.Equal(t, "value", d[0].Value)
	})
}

func TestUnmarshalEdgeCases(t *testing.T) {
	t.Run("sentinel with empty name returns error", func(t *testing.T) {
		r := newRegistry()
		var out any
		err := json.Unmarshal([]byte(`{"$":1}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
	})

	t.Run("sentinel object with only dollar key and no other keys", func(t *testing.T) {
		r, err := NewRegistry(func(r *Registry) error {
			return NewDirective("test", func(dec *jsontext.Decoder) (string, error) {
				return "result", nil
			})(r)
		})
		require.NoError(t, err)

		got := unmarshal(t, r, `{"$test":null}`)
		require.Equal(t, "result", got)
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		r := newRegistry()
		var out any
		err := json.Unmarshal([]byte(`{malformed`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
	})

	t.Run("deeply nested sentinel objects work", func(t *testing.T) {
		r, err := NewRegistry(func(r *Registry) error {
			return NewDirective("inner", func(dec *jsontext.Decoder) (int, error) {
				return 42, nil
			})(r)
		})
		require.NoError(t, err)

		got := unmarshal(t, r, `{"level1":{"level2":{"$inner":null}}}`)
		d := assertD(t, got)
		require.Len(t, d, 1)
		require.Equal(t, "level1", d[0].Key)

		level1 := assertD(t, d[0].Value)
		require.Len(t, level1, 1)
		require.Equal(t, "level2", level1[0].Key)
		require.Equal(t, 42, level1[0].Value)
	})

	t.Run("sentinel in array with mixed elements", func(t *testing.T) {
		r, err := NewRegistry(func(r *Registry) error {
			return NewDirective("num", func(dec *jsontext.Decoder) (int, error) {
				return 99, nil
			})(r)
		})
		require.NoError(t, err)

		got := unmarshal(t, r, `[1,"text",{"$num":null},true]`)
		a := assertA(t, got)
		require.Len(t, a, 4)
		require.Equal(t, float64(1), a[0])
		require.Equal(t, "text", a[1])
		require.Equal(t, 99, a[2]) // directive result
		require.Equal(t, true, a[3])
	})

	t.Run("string values decode correctly", func(t *testing.T) {
		r := newRegistry()
		got := unmarshal(t, r, `"just a string"`)
		require.Equal(t, "just a string", got)
	})

	t.Run("boolean values decode correctly", func(t *testing.T) {
		r := newRegistry()
		got := unmarshal(t, r, `true`)
		require.Equal(t, true, got)

		got = unmarshal(t, r, `false`)
		require.Equal(t, false, got)
	})

	t.Run("null value decodes correctly", func(t *testing.T) {
		r := newRegistry()
		got := unmarshal(t, r, `null`)
		require.Nil(t, got)
	})
}
