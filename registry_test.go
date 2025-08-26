package jwalk

import (
	"bytes"
	"testing"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Register(t *testing.T) {
	t.Run("valid function registration succeeds", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error { return nil }
		err := r.Register("valid", fn)
		require.NoError(t, err)
	})

	t.Run("namespaced function registration succeeds", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *string) error { return nil }
		err := r.Register("ns.directive", fn)
		require.NoError(t, err)
	})

	t.Run("duplicate name returns error", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error { return nil }
		err := r.Register("dup", fn)
		require.NoError(t, err)

		err = r.Register("dup", fn)
		require.Error(t, err)
	})

	t.Run("invalid namespace format returns error", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error { return nil }
		invalid := []string{".bad", "bad.", "a.b.c"}
		for _, name := range invalid {
			err := r.Register(name, fn)
			require.Error(t, err, "expected error for name %s", name)
		}
	})

	t.Run("non function value returns error", func(t *testing.T) {
		r := newRegistry()
		err := r.Register("bad", 7)
		require.Error(t, err)
	})

	t.Run("wrong parameter count returns error", func(t *testing.T) {
		r := newRegistry()
		f1 := func(*jsontext.Decoder) error { return nil } // one param
		err := r.Register("wrong1", f1)
		require.Error(t, err)

		f3 := func(*jsontext.Decoder, *int, *string) error { return nil } // three params
		err = r.Register("wrong3", f3)
		require.Error(t, err)
	})

	t.Run("wrong return count returns error", func(t *testing.T) {
		r := newRegistry()
		f0 := func(*jsontext.Decoder, *int) {} // no return
		err := r.Register("wrong0", f0)
		require.Error(t, err)

		f2 := func(*jsontext.Decoder, *int) (int, error) { return 0, nil } // two returns
		err = r.Register("wrong2", f2)
		require.Error(t, err)
	})

	t.Run("invalid first parameter type returns error", func(t *testing.T) {
		r := newRegistry()
		f := func(i int, v *int) error { return nil }
		err := r.Register("wrongFirst", f)
		require.Error(t, err)
	})

	t.Run("non-pointer second parameter returns error", func(t *testing.T) {
		r := newRegistry()
		f := func(dec *jsontext.Decoder, v int) error { return nil }
		err := r.Register("wrongSecond", f)
		require.Error(t, err)
	})

	t.Run("invalid return type returns error", func(t *testing.T) {
		r := newRegistry()
		f := func(dec *jsontext.Decoder, v *int) int { return 0 }
		err := r.Register("wrongReturn", f)
		require.Error(t, err)
	})
}

func TestRegistry_exec(t *testing.T) {
	t.Run("fully qualified name decodes value", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error {
			var num int
			if err := json.UnmarshalDecode(dec, &num); err != nil {
				return err
			}
			*v = num
			return nil
		}
		err := r.Register("val", fn)
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("123")))
		got, err := r.Exec("val", dec)
		require.NoError(t, err)
		assert.Equal(t, 123, got)
		assert.IsType(t, 0, got)
	})

	t.Run("unique short name resolves directive", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error {
			var num int
			if err := json.UnmarshalDecode(dec, &num); err != nil {
				return err
			}
			*v = num
			return nil
		}
		err := r.Register("ns.val", fn)
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("5")))
		got, err := r.Exec("val", dec) // short name
		require.NoError(t, err)
		assert.Equal(t, 5, got)
	})

	t.Run("bare name resolves when namespaced duplicate exists", func(t *testing.T) {
		r := newRegistry()
		bareFn := func(dec *jsontext.Decoder, v *int) error { *v = 1; return nil }
		otherFn := func(dec *jsontext.Decoder, v *int) error { *v = 2; return nil }
		err := r.Register("val", bareFn) // bare fully qualified name
		require.NoError(t, err)
		err = r.Register("ns.val", otherFn) // namespaced
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err := r.Exec("val", dec) // resolves bare
		require.NoError(t, err)
		assert.Equal(t, 1, got)

		dec2 := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err = r.Exec("ns.val", dec2)
		require.NoError(t, err)
		assert.Equal(t, 2, got)
	})

	t.Run("missing directive returns error", func(t *testing.T) {
		r := newRegistry()
		dec := jsontext.NewDecoder(bytes.NewReader([]byte("1")))
		got, err := r.Exec("missing", dec)
		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("ambiguous short name returns error", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error {
			var n int
			if err := json.UnmarshalDecode(dec, &n); err != nil {
				return err
			}
			*v = n
			return nil
		}
		err := r.Register("a.value", fn)
		require.NoError(t, err)
		err = r.Register("b.value", fn)
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("1")))
		got, err := r.Exec("value", dec)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ambiguous")
		assert.Contains(t, err.Error(), "a.value")
		assert.Contains(t, err.Error(), "b.value")
		assert.Nil(t, got)

		// fully qualified works
		dec2 := jsontext.NewDecoder(bytes.NewReader([]byte("2")))
		got, err = r.Exec("a.value", dec2)
		require.NoError(t, err)
		assert.Equal(t, 2, got)
	})

	t.Run("directive error wraps error", func(t *testing.T) {
		r := newRegistry()
		err := r.Register("fail", func(dec *jsontext.Decoder, v *int) error { return assert.AnError })
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err := r.Exec("fail", dec)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, got)
	})

	t.Run("short name error reports fully qualified name", func(t *testing.T) {
		r := newRegistry()
		failFn := func(dec *jsontext.Decoder, v *int) error { return assert.AnError }
		err := r.Register("ns.err", failFn)
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err := r.Exec("err", dec) // use short name
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "ns.err")
		assert.Nil(t, got)
	})
}

func TestRegistryEdgeCases(t *testing.T) {
	t.Run("concurrent registration and calls are safe", func(t *testing.T) {
		r := newRegistry()

		// This is a basic smoke test for thread safety
		// In a real scenario, you might use race detector (-race flag)
		done := make(chan bool, 2)

		// Goroutine 1: register
		go func() {
			defer func() { done <- true }()
			fn := func(dec *jsontext.Decoder, v *int) error { *v = 42; return nil }
			err := r.Register("concurrent", fn)
			require.NoError(t, err)
		}()

		// Goroutine 2: try to call (might fail initially, that's ok)
		go func() {
			defer func() { done <- true }()
			dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
			_, _ = r.Exec("concurrent", dec) // might error, we don't care
		}()

		// Wait for both goroutines
		<-done
		<-done

		// Verify the registration worked
		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err := r.Exec("concurrent", dec)
		require.NoError(t, err)
		assert.Equal(t, 42, got)
	})

	t.Run("pointer to interface parameter is allowed", func(t *testing.T) {
		r := newRegistry()
		var i any
		f := func(dec *jsontext.Decoder, v *any) error { *v = i; return nil }
		err := r.Register("interface_param", f)
		// Go's reflection actually allows *any (interface{}) as a valid pointer type
		require.NoError(t, err)
	})

	t.Run("complex types as parameters work", func(t *testing.T) {
		type CustomType struct {
			Field string
		}

		r := newRegistry()
		f := func(dec *jsontext.Decoder, v *CustomType) error {
			*v = CustomType{Field: "test"}
			return nil
		}
		err := r.Register("custom", f)
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("null")))
		got, err := r.Exec("custom", dec)
		require.NoError(t, err)
		expected := CustomType{Field: "test"}
		assert.Equal(t, expected, got)
	})

	t.Run("empty string name is valid", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *string) error { *v = "empty"; return nil }
		err := r.Register("", fn)
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("null")))
		got, err := r.Exec("", dec)
		require.NoError(t, err)
		assert.Equal(t, "empty", got)
	})

	t.Run("namespace with single character names", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error { *v = 1; return nil }
		err := r.Register("a.b", fn)
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err := r.Exec("b", dec) // short name
		require.NoError(t, err)
		assert.Equal(t, 1, got)

		dec2 := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err = r.Exec("a.b", dec2) // full name
		require.NoError(t, err)
		assert.Equal(t, 1, got)
	})

	t.Run("multiple dots in name returns error", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error { return nil }
		err := r.Register("a.b.c", fn)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid namespace")
	})

	t.Run("name starting with separator returns error", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error { return nil }
		err := r.Register(".invalid", fn)
		require.Error(t, err)
	})

	t.Run("name ending with separator returns error", func(t *testing.T) {
		r := newRegistry()
		fn := func(dec *jsontext.Decoder, v *int) error { return nil }
		err := r.Register("invalid.", fn)
		require.Error(t, err)
	})
}
