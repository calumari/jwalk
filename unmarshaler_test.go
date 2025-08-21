package jwalk

import (
	"testing"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/stretchr/testify/require"
)

func decodeWithUnmarshaler(t *testing.T, r *OperatorRegistry, src string) any {
	t.Helper()
	var out any
	err := json.Unmarshal([]byte(src), &out, json.WithUnmarshalers(Unmarshaler(r)))
	require.NoError(t, err)
	return out
}

func TestUnmarshaler(t *testing.T) {
	r := NewOperatorRegistry()

	t.Run("empty object -> empty D", func(t *testing.T) {
		v := decodeWithUnmarshaler(t, r, `{}`)
		require.IsType(t, D{}, v)
		require.Len(t, v.(D), 0)
	})

	t.Run("empty array -> empty A", func(t *testing.T) {
		v := decodeWithUnmarshaler(t, r, `[]`)
		require.IsType(t, A{}, v)
		require.Len(t, v.(A), 0)
	})

	t.Run("regular object ordering preserved", func(t *testing.T) {
		v := decodeWithUnmarshaler(t, r, `{"a":1,"b":2}`)
		d, ok := v.(D)
		require.True(t, ok)
		require.Equal(t, []E{{Key: "a", Value: float64(1)}, {Key: "b", Value: float64(2)}}, []E(d))
	})

	t.Run("nested array wraps objects", func(t *testing.T) {
		v := decodeWithUnmarshaler(t, r, `[1,{"x":2}]`)
		arr := v.(A)
		require.Len(t, arr, 2)
		require.Equal(t, float64(1), arr[0])
		d, ok := arr[1].(D)
		require.True(t, ok)
		require.Equal(t, "x", d[0].Key)
	})

	t.Run("operator object dispatch + skip extra", func(t *testing.T) {
		r := NewOperatorRegistry()
		err := Register(r, "val", func(dec *jsontext.Decoder) (int, error) {
			var num int
			err := json.UnmarshalDecode(dec, &num)
			return num, err
		})
		require.NoError(t, err)
		v := decodeWithUnmarshaler(t, r, `{"$val": 42, "ignored": true}`)
		require.Equal(t, 42, v)
	})

	t.Run("primitive value bypassed (SkipFunc)", func(t *testing.T) {
		v := decodeWithUnmarshaler(t, r, `123`)
		// default number decoding is float64
		require.Equal(t, float64(123), v)
	})
}
