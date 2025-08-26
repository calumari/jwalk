package jwalk

import (
	"bytes"
	"testing"

	"github.com/go-json-experiment/json/jsontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDirective(t *testing.T) {
	t.Run("decode wraps value", func(t *testing.T) {
		r, err := NewRegistry(NewDirective("num", func(dec *jsontext.Decoder) (int, error) { return 11, nil }))
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err := r.Exec("num", dec)
		require.NoError(t, err)
		assert.Equal(t, 11, got)
	})

	t.Run("error bubbles up", func(t *testing.T) {
		r, err := NewRegistry(NewDirective("err", func(dec *jsontext.Decoder) (int, error) { return 0, assert.AnError }))
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err := r.Exec("err", dec)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, got)
	})
}

func TestGroup(t *testing.T) {
	t.Run("empty bundle succeeds", func(t *testing.T) {
		r, err := NewRegistry(Group())
		require.NoError(t, err)
		assert.NotNil(t, r)
	})

	t.Run("combines multiple directives", func(t *testing.T) {
		r, err := NewRegistry(Group(
			NewDirective("a", func(dec *jsontext.Decoder) (string, error) { return "A", nil }),
			NewDirective("b", func(dec *jsontext.Decoder) (string, error) { return "B", nil }),
		))
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		gotA, err := r.Exec("a", dec)
		require.NoError(t, err)
		assert.Equal(t, "A", gotA)

		dec = jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		gotB, err := r.Exec("b", dec)
		require.NoError(t, err)
		assert.Equal(t, "B", gotB)
	})

	t.Run("registration error stops processing", func(t *testing.T) {
		called := false
		_, err := NewRegistry(Group(
			Registration(func(r *Registry) error { return assert.AnError }),
			Registration(func(r *Registry) error { called = true; return nil }),
		))
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.False(t, called, "later registration ran after error")
	})
}

func TestApply(t *testing.T) {
	t.Run("empty registration list succeeds", func(t *testing.T) {
		r := newRegistry()
		err := Apply(r)
		assert.NoError(t, err)
	})

	t.Run("applies all registrations", func(t *testing.T) {
		count := 0
		r := newRegistry()
		err := Apply(r,
			Registration(func(r *Registry) error { count++; return nil }),
			Registration(func(r *Registry) error { count++; return nil }),
		)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("registration error stops processing", func(t *testing.T) {
		called := false
		r := newRegistry()
		err := Apply(r,
			Registration(func(r *Registry) error { return assert.AnError }),
			Registration(func(r *Registry) error { called = true; return nil }),
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.False(t, called, "later registration ran after error")
	})
}

func TestNewRegistry(t *testing.T) {
	t.Run("empty build creates empty registry", func(t *testing.T) {
		r, err := NewRegistry()
		require.NoError(t, err)
		assert.NotNil(t, r)

		// verify registry is functional but empty
		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		got, err := r.Exec("missing", dec)
		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("creates registry with multiple registrations", func(t *testing.T) {
		r, err := NewRegistry(
			NewDirective("a", func(dec *jsontext.Decoder) (string, error) { return "A", nil }),
			NewDirective("b", func(dec *jsontext.Decoder) (string, error) { return "B", nil }),
		)
		require.NoError(t, err)

		dec := jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		gotA, err := r.Exec("a", dec)
		require.NoError(t, err)
		assert.Equal(t, "A", gotA)

		dec = jsontext.NewDecoder(bytes.NewReader([]byte("0")))
		gotB, err := r.Exec("b", dec)
		require.NoError(t, err)
		assert.Equal(t, "B", gotB)
	})

	t.Run("registration error returns error", func(t *testing.T) {
		r, err := NewRegistry(Registration(func(r *Registry) error { return assert.AnError }))
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, r)
	})
}
