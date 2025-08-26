package jwalk

import (
	"time"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Standard library directive registrations. These are provided as exported
// variables so callers can opt-in explicitly:
//   r, _ := jwalk.NewRegistry(jwalk.TimeDirective, jwalk.DurationDirective)
// or via the helper (Group of both):
//   r, _ := jwalk.NewRegistry(jwalk.Stdlib())
//
// Each directive accepts a string JSON value and parses it into the target
// type. They are intentionally conservative and do not guess alternative
// formats beyond the canonical form for each type.

// internal decode helpers so we can reuse for custom names.
func decodeTime(dec *jsontext.Decoder) (time.Time, error) {
	var s string
	if err := json.UnmarshalDecode(dec, &s); err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, s)
}

func decodeDuration(dec *jsontext.Decoder) (time.Duration, error) {
	var s string
	if err := json.UnmarshalDecode(dec, &s); err != nil {
		return 0, err
	}
	return time.ParseDuration(s)
}

// NewTimeDirective returns a Registration parsing an RFC3339 timestamp into
// time.Time under a custom directive name.
func NewTimeDirective(name string) Registration {
	return NewDirective(name, decodeTime)
}

// NewDurationDirective returns a Registration parsing a Go duration string into
// time.Duration under a custom directive name.
func NewDurationDirective(name string) Registration {
	return NewDirective(name, decodeDuration)
}

// Default stdlib directive registrations using canonical names.
var (
	TimeDirective     = NewTimeDirective("std.time")
	DurationDirective = NewDurationDirective("std.duration")
)

func Stdlib() Registration {
	return Group(TimeDirective, DurationDirective)
}
