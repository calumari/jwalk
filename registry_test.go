package jwalk

import (
	"bytes"
	"reflect"
	"testing"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperatorRegistry_Register(t *testing.T) {
	t.Run("non-function returns error", func(t *testing.T) {
		err := NewOperatorRegistry().Register("bad", 7)
		require.Error(t, err)
	})

	t.Run("wrong param count returns error", func(t *testing.T) {
		f1 := func(*jsontext.Decoder) error { return nil } // one param
		err := NewOperatorRegistry().Register("wrong1", f1)
		require.Error(t, err)
		f3 := func(*jsontext.Decoder, *int, *string) error { return nil } // three params
		err = NewOperatorRegistry().Register("wrong3", f3)
		require.Error(t, err)
	})

	t.Run("wrong return count returns error", func(t *testing.T) {
		f0 := func(*jsontext.Decoder, *int) {} // no return
		err := NewOperatorRegistry().Register("wrong0", f0)
		require.Error(t, err)
		f2 := func(*jsontext.Decoder, *int) (int, error) { return 0, nil } // two returns
		err = NewOperatorRegistry().Register("wrong2", f2)
		require.Error(t, err)
	})

	t.Run("wrong first param type returns error", func(t *testing.T) {
		f := func(i int, v *int) error { return nil }
		err := NewOperatorRegistry().Register("wrongFirst", f)
		require.Error(t, err)
	})

	t.Run("second param not pointer returns error", func(t *testing.T) {
		f := func(dec *jsontext.Decoder, v int) error { return nil }
		err := NewOperatorRegistry().Register("wrongSecond", f)
		require.Error(t, err)
	})

	t.Run("wrong return type returns error", func(t *testing.T) {
		f := func(dec *jsontext.Decoder, v *int) int { return 0 }
		err := NewOperatorRegistry().Register("wrongReturn", f)
		require.Error(t, err)
	})

	t.Run("duplicate registration returns error", func(t *testing.T) {
		r := NewOperatorRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error { return nil }
		require.NoError(t, r.Register("dup", fn))
		err := r.Register("dup", fn)
		require.Error(t, err)
	})

	t.Run("missing operator call returns error", func(t *testing.T) {
		r := NewOperatorRegistry()
		dec := jsontext.NewDecoder(bytes.NewReader([]byte("1")))
		_, err := r.Call("missing", dec)
		require.Error(t, err)
	})
}

func TestOperatorRegistry_Call(t *testing.T) {
	t.Run("registered operator decodes value", func(t *testing.T) {
		r := NewOperatorRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error {
			var num int
			if err := json.UnmarshalDecode(dec, &num); err != nil {
				return err
			}
			*v = num
			return nil
		}
		require.NoError(t, r.Register("val", fn))

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("123")))
		res, err := r.Call("val", dec)
		require.NoError(t, err)
		require.Equal(t, 123, res)
		require.IsType(t, 0, res)
	})

	t.Run("operator failure returns wrapped error", func(t *testing.T) {
		r := NewOperatorRegistry()
		require.NoError(t, r.Register("fail", func(dec *jsontext.Decoder, v *int) error { return assert.AnError }))
		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		_, err := r.Call("fail", dec)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestRegister(t *testing.T) {
	r := NewOperatorRegistry()
	t.Run("generic helper decodes value", func(t *testing.T) {
		err := Register(r, "str", func(dec *jsontext.Decoder) (string, error) {
			var s string
			if err := json.UnmarshalDecode(dec, &s); err != nil {
				return "", err
			}
			return s, nil
		})
		require.NoError(t, err)
		dec := jsontext.NewDecoder(bytes.NewReader([]byte(`"hello"`)))
		res, err := r.Call("str", dec)
		require.NoError(t, err)
		require.Equal(t, "hello", res)
		require.Equal(t, reflect.TypeOf(""), reflect.TypeOf(res))
	})
}

func TestMustRegister(t *testing.T) {
	t.Run("duplicate name panics", func(t *testing.T) {
		r := NewOperatorRegistry()
		MustRegister(r, "unique", func(dec *jsontext.Decoder) (int, error) { return 1, nil })
		defer func() {
			if rec := recover(); rec == nil {
				// Should have panicked attempting to re-register same name
				t.Fatal("expected panic from MustRegister duplicate")
			}
		}()
		MustRegister(r, "unique", func(dec *jsontext.Decoder) (int, error) { return 2, nil })
	})
}
