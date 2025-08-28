package jwalk

import (
	"time"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

var (
	// TimeDirective constructs a Directive that decodes values of either form:
	//
	//	{"$std.time": "2006-01-02T15:04:05Z07:00"}                      // RFC3339 (default)
	//	{"$std.time": {"value":"2023-10-05","layout":"2006-01-02"}}     // custom layout
	//
	// When the object form is used, layout is optional and defaults to time.RFC3339.
	StdTimeDirective = NewDirective("std.time", unmarshalTime)

	// DurationDirective constructs a Directive that decodes values of the form:
	//
	//	{"$std.duration": "1h30m"}
	//
	// into a time.Duration using time.ParseDuration.
	StdDurationDirective = NewDirective("std.duration", unmarshalDuration)
)

func unmarshalTime(dec *jsontext.Decoder) (time.Time, error) {
	// Support object with value/layout or plain string.
	if dec.PeekKind() == '{' {
		var aux struct {
			Value  string `json:"value"`
			Layout string `json:"layout"`
		}
		if err := json.UnmarshalDecode(dec, &aux); err != nil {
			return time.Time{}, err
		}
		layout := aux.Layout
		if layout == "" {
			layout = time.RFC3339
		}
		return time.Parse(layout, aux.Value)
	}

	var value string
	if err := json.UnmarshalDecode(dec, &value); err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, value)
}

func unmarshalDuration(dec *jsontext.Decoder) (time.Duration, error) {
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
