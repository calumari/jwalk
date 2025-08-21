package main

import (
	"fmt"
	"time"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	jwalk "github.com/calumari/jwalk"
)

func init() {
	jwalk.MustRegister(jwalk.DefaultRegistry, "date", func(dec *jsontext.Decoder) (time.Time, error) {
		var t time.Time
		err := json.UnmarshalDecode(dec, &t)
		return t, err
	})
}

// Example shows registering a $date operator converting objects of the form
// {"$date": <RFC3339>} into a time.Time.
func main() {
	// input := []byte(`[{"$date":"2023-10-01T12:00:00Z"}]`)
	input := []byte(`{"b":true,"time":{"$date":"2023-10-01T12:00:00Z"},"a":[{"$date":"2023-10-01T12:00:00Z"}],"m":{"o":{"$date":"2023-10-01T12:00:00Z"}}}`)

	// Option 1: traditional unmarshal into *D
	var d jwalk.D
	err := json.Unmarshal(input, &d, json.WithUnmarshalers(jwalk.Unmarshalers(jwalk.DefaultRegistry)))
	if err != nil {
		panic(err)
	}

	// Output: 2023-10-01T12:00:00Z
	fmt.Println(d, d)
}
