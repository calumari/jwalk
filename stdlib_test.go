package jwalk

import (
	"testing"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/require"
)

func TestTimeDirective(t *testing.T) {
	t.Run("valid rfc3339 timestamp decodes correctly", func(t *testing.T) {
		r, err := NewRegistry(TimeDirective)
		require.NoError(t, err)

		ts := "2025-08-26T12:34:56Z"
		var out any
		err = json.Unmarshal([]byte(`{"$std.time":"`+ts+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.Parse(time.RFC3339, ts)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})

	t.Run("valid rfc3339 timestamp with timezone decodes correctly", func(t *testing.T) {
		r, err := NewRegistry(TimeDirective)
		require.NoError(t, err)

		ts := "2025-08-26T12:34:56-08:00"
		var out any
		err = json.Unmarshal([]byte(`{"$std.time":"`+ts+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.Parse(time.RFC3339, ts)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})

	t.Run("fractional seconds decode correctly", func(t *testing.T) {
		r, err := NewRegistry(TimeDirective)
		require.NoError(t, err)

		ts := "2025-08-26T12:34:56.789Z"
		var out any
		err = json.Unmarshal([]byte(`{"$std.time":"`+ts+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.Parse(time.RFC3339, ts)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})

	t.Run("decode error bubbles up", func(t *testing.T) {
		r, err := NewRegistry(TimeDirective)
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$std.time":"not-a-time"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
	})
}

func TestDurationDirective(t *testing.T) {
	t.Run("valid duration string decodes correctly", func(t *testing.T) {
		r, err := NewRegistry(DurationDirective)
		require.NoError(t, err)

		dStr := "2h15m30s"
		var out any
		err = json.Unmarshal([]byte(`{"$std.duration":"`+dStr+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.ParseDuration(dStr)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})

	t.Run("simple duration formats decode correctly", func(t *testing.T) {
		r, err := NewRegistry(DurationDirective)
		require.NoError(t, err)

		tests := []string{
			"1s",
			"5m",
			"3h",
			"24h",
			"100ms",
			"500µs",
			"1000ns",
		}

		for _, tt := range tests {
			t.Run(tt, func(t *testing.T) {
				var out any
				err = json.Unmarshal([]byte(`{"$std.duration":"`+tt+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
				require.NoError(t, err, "failed to decode %s", tt)

				want, err := time.ParseDuration(tt)
				require.NoError(t, err)
				require.Equal(t, want, out)
			})
		}
	})

	t.Run("negative duration decodes correctly", func(t *testing.T) {
		r, err := NewRegistry(DurationDirective)
		require.NoError(t, err)

		dStr := "-1h30m"
		var out any
		err = json.Unmarshal([]byte(`{"$std.duration":"`+dStr+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.ParseDuration(dStr)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})

	t.Run("decode error bubbles up", func(t *testing.T) {
		r, err := NewRegistry(DurationDirective)
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$std.duration":"not-a-duration"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
	})
}

func TestNewTimeDirective(t *testing.T) {
	t.Run("custom name registration works", func(t *testing.T) {
		r, err := NewRegistry(NewTimeDirective("custom.time"))
		require.NoError(t, err)

		ts := "2025-12-31T23:59:59Z"
		var out any
		err = json.Unmarshal([]byte(`{"$custom.time":"`+ts+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.Parse(time.RFC3339, ts)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})

	t.Run("short name resolves with custom namespace", func(t *testing.T) {
		r, err := NewRegistry(NewTimeDirective("myapp.timestamp"))
		require.NoError(t, err)

		ts := "2025-01-15T10:30:00Z"
		var out any
		err = json.Unmarshal([]byte(`{"$timestamp":"`+ts+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.Parse(time.RFC3339, ts)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})
}

func TestDuration(t *testing.T) {
	t.Run("custom name registration works", func(t *testing.T) {
		r, err := NewRegistry(NewDurationDirective("custom.dur"))
		require.NoError(t, err)

		dStr := "45m30s"
		var out any
		err = json.Unmarshal([]byte(`{"$custom.dur":"`+dStr+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.ParseDuration(dStr)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})

	t.Run("short name resolves with custom namespace", func(t *testing.T) {
		r, err := NewRegistry(NewDurationDirective("myapp.timeout"))
		require.NoError(t, err)

		dStr := "30s"
		var out any
		err = json.Unmarshal([]byte(`{"$timeout":"`+dStr+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.ParseDuration(dStr)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})
}

func TestStdlib(t *testing.T) {
	t.Run("bundle applies all registrations", func(t *testing.T) {
		r := newRegistry()
		err := Apply(r, Stdlib()) // Stdlib returns a Bundle
		require.NoError(t, err)

		// time via short alias since only std.time/duration registered and 'time' unambiguous relative to any others
		var out any
		ts := "2025-08-26T08:00:00Z"
		err = json.Unmarshal([]byte(`{"$time":"`+ts+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.Parse(time.RFC3339, ts)
		require.NoError(t, err)
		require.Equal(t, want, out)
	})

	t.Run("duration short name resolves correctly", func(t *testing.T) {
		r := newRegistry()
		err := Apply(r, Stdlib())
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$duration":"1h30m"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		want, err := time.ParseDuration("1h30m")
		require.NoError(t, err)
		require.Equal(t, want, out)
	})

	t.Run("fully qualified names work", func(t *testing.T) {
		r := newRegistry()
		err := Apply(r, Stdlib())
		require.NoError(t, err)

		// Test both fully qualified names
		var outTime any
		err = json.Unmarshal([]byte(`{"$std.time":"2025-01-01T00:00:00Z"}`), &outTime, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		var outDur any
		err = json.Unmarshal([]byte(`{"$std.duration":"5s"}`), &outDur, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)

		wantTime, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
		wantDur, _ := time.ParseDuration("5s")
		require.Equal(t, wantTime, outTime)
		require.Equal(t, wantDur, outDur)
	})
}

func TestStdlibEdgeCases(t *testing.T) {
	t.Run("time directive with non-string value returns error", func(t *testing.T) {
		r, err := NewRegistry(TimeDirective)
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$std.time":123}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
	})

	t.Run("duration directive with non-string value returns error", func(t *testing.T) {
		r, err := NewRegistry(DurationDirective)
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$std.duration":123}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
	})

	t.Run("time directive with empty string returns error", func(t *testing.T) {
		r, err := NewRegistry(TimeDirective)
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$std.time":""}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
	})

	t.Run("duration directive with empty string returns error", func(t *testing.T) {
		r, err := NewRegistry(DurationDirective)
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$std.duration":""}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.Error(t, err)
	})

	t.Run("time directive with invalid RFC3339 format returns error", func(t *testing.T) {
		r, err := NewRegistry(TimeDirective)
		require.NoError(t, err)

		invalidFormats := []string{
			"2025-13-01T00:00:00Z", // invalid month
			"2025-01-32T00:00:00Z", // invalid day
			"2025-01-01T25:00:00Z", // invalid hour
			"2025-01-01T00:60:00Z", // invalid minute
			"2025-01-01T00:00:60Z", // invalid second
			"2025-01-01 00:00:00",  // wrong format (space instead of T)
			"Jan 1, 2025",          // completely wrong format
		}

		for _, format := range invalidFormats {
			var out any
			err = json.Unmarshal([]byte(`{"$std.time":"`+format+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
			require.Error(t, err, "expected error for format %s", format)
		}
	})

	t.Run("duration directive with invalid format returns error", func(t *testing.T) {
		r, err := NewRegistry(DurationDirective)
		require.NoError(t, err)

		invalidFormats := []string{
			"1x",     // invalid unit
			"1.5.5s", // double decimal
			"1h2m3x", // invalid unit at end
			"abc",    // non-numeric
			"1h2q3s", // invalid unit in middle
		}

		for _, format := range invalidFormats {
			var out any
			err = json.Unmarshal([]byte(`{"$std.duration":"`+format+`"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
			require.Error(t, err, "expected error for format %s", format)
		}
	})

	t.Run("zero duration parses correctly", func(t *testing.T) {
		r, err := NewRegistry(DurationDirective)
		require.NoError(t, err)

		var out any
		err = json.Unmarshal([]byte(`{"$std.duration":"0s"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)
		require.Equal(t, time.Duration(0), out)

		// Also test "0" (without unit, which Go allows)
		err = json.Unmarshal([]byte(`{"$std.duration":"0"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)
		require.Equal(t, time.Duration(0), out)
	})

	t.Run("very large duration parses correctly", func(t *testing.T) {
		r, err := NewRegistry(DurationDirective)
		require.NoError(t, err)

		// Test max duration that Go can handle
		var out any
		err = json.Unmarshal([]byte(`{"$std.duration":"9223372036854775807ns"}`), &out, json.WithUnmarshalers(Unmarshalers(r)))
		require.NoError(t, err)
		require.IsType(t, time.Duration(0), out)
	})
}
