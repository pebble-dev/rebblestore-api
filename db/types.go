package db

import (
	"time"
)

// JSONTime is a dummy time object that is meant to allow Go's JSON module to
// properly de-serialize the JSON time format.
type JSONTime struct {
	time.Time
}

// UnmarshalJSON allows for the custom time format within the application JSON
// to be decoded into Go's native time format.
func (self *JSONTime) UnmarshalJSON(b []byte) (err error) {
	s := string(b)

	// Return an empty time.Time object if it didn't exist in the first place.
	if s == "null" {
		self.Time = time.Time{}
		return
	}

	t, err := time.Parse("\"2006-01-02T15:04:05.999Z\"", s)
	if err != nil {
		t = time.Time{}
	}
	self.Time = t
	return
}
