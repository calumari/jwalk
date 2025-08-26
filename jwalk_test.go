package jwalk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestD(t *testing.T) {
	t.Run("empty document", func(t *testing.T) {
		var d D
		require.Len(t, d, 0)
		require.Nil(t, d) // zero value of D is nil slice
	})

	t.Run("initialized document is not nil", func(t *testing.T) {
		d := D{}
		require.Len(t, d, 0)
		require.NotNil(t, d) // D{} creates a non-nil empty slice
	})

	t.Run("single entry document", func(t *testing.T) {
		d := D{{Key: "key", Value: "value"}}
		require.Len(t, d, 1)
		require.Equal(t, "key", d[0].Key)
		require.Equal(t, "value", d[0].Value)
	})

	t.Run("multiple entry document preserves order", func(t *testing.T) {
		d := D{
			{Key: "first", Value: 1},
			{Key: "second", Value: 2},
			{Key: "third", Value: 3},
		}
		require.Len(t, d, 3)
		require.Equal(t, "first", d[0].Key)
		require.Equal(t, "second", d[1].Key)
		require.Equal(t, "third", d[2].Key)
	})

	t.Run("document can contain any value types", func(t *testing.T) {
		nested := D{{Key: "nested", Value: "value"}}
		arr := A{1, 2, 3}
		d := D{
			{Key: "string", Value: "text"},
			{Key: "number", Value: 42},
			{Key: "boolean", Value: true},
			{Key: "null", Value: nil},
			{Key: "document", Value: nested},
			{Key: "array", Value: arr},
		}
		require.Len(t, d, 6)
		require.Equal(t, "text", d[0].Value)
		require.Equal(t, 42, d[1].Value)
		require.Equal(t, true, d[2].Value)
		require.Equal(t, nil, d[3].Value)
		require.Equal(t, nested, d[4].Value)
		require.Equal(t, arr, d[5].Value)
	})
}

func TestA(t *testing.T) {
	t.Run("empty array", func(t *testing.T) {
		var a A
		require.Len(t, a, 0)
		require.Nil(t, a) // zero value of A is nil slice
	})

	t.Run("initialized array is not nil", func(t *testing.T) {
		a := A{}
		require.Len(t, a, 0)
		require.NotNil(t, a) // A{} creates a non-nil empty slice
	})

	t.Run("single element array", func(t *testing.T) {
		a := A{"element"}
		require.Len(t, a, 1)
		require.Equal(t, "element", a[0])
	})

	t.Run("multiple element array preserves order", func(t *testing.T) {
		a := A{"first", "second", "third"}
		require.Len(t, a, 3)
		require.Equal(t, "first", a[0])
		require.Equal(t, "second", a[1])
		require.Equal(t, "third", a[2])
	})

	t.Run("array can contain any value types", func(t *testing.T) {
		nested := D{{Key: "key", Value: "value"}}
		arr := A{1, 2}
		a := A{
			"string",
			42,
			true,
			nil,
			nested,
			arr,
		}
		require.Len(t, a, 6)
		require.Equal(t, "string", a[0])
		require.Equal(t, 42, a[1])
		require.Equal(t, true, a[2])
		require.Equal(t, nil, a[3])
		require.Equal(t, nested, a[4])
		require.Equal(t, arr, a[5])
	})
}

func TestE(t *testing.T) {
	t.Run("entry with string value", func(t *testing.T) {
		e := E{Key: "name", Value: "John"}
		require.Equal(t, "name", e.Key)
		require.Equal(t, "John", e.Value)
	})

	t.Run("entry with complex value", func(t *testing.T) {
		complexValue := D{{Key: "nested", Value: 42}}
		e := E{Key: "complex", Value: complexValue}
		require.Equal(t, "complex", e.Key)
		require.Equal(t, complexValue, e.Value)
	})

	t.Run("entry with nil value", func(t *testing.T) {
		e := E{Key: "null_field", Value: nil}
		require.Equal(t, "null_field", e.Key)
		require.Nil(t, e.Value)
	})

	t.Run("empty key is allowed", func(t *testing.T) {
		e := E{Key: "", Value: "value"}
		require.Equal(t, "", e.Key)
		require.Equal(t, "value", e.Value)
	})
}
