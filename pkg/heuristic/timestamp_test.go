package heuristic

import (
	"testing"

	"github.com/zefhemel/ax/pkg/backend/common"
)

func TestParseTimestampUnix(t *testing.T) {
	var i interface{}
	common.MustJsonDecode("1504523700", &i)
	ts, err := ParseTimestamp(i)
	if err != nil {
		t.Error(err)
	}
	if ts.Year() != 2017 {
		t.Error("Wrong year")
	}
	common.MustJsonDecode("1504523168849", &i)
	ts, err = ParseTimestamp(i)
	if err != nil {
		t.Error(err)
	}
	if ts.Year() != 2017 {
		t.Error("Wrong year")
	}
	common.MustJsonDecode(`"1504523168849"`, &i)
	ts, err = ParseTimestamp(i)
	if err != nil {
		t.Error(err)
	}
	if ts.Year() != 2017 {
		t.Error("Wrong year")
	}
	// Extreme outside of range
	common.MustJsonDecode(`"1568849"`, &i)
	ts, err = ParseTimestamp(i)
	if err == nil {
		t.Error("Should not be accepted")
	}
	// Extreme outside of range
	common.MustJsonDecode(`"1568849000000000000"`, &i)
	ts, err = ParseTimestamp(i)
	if err == nil {
		t.Error("Should not be accepted")
	}
}

func TestParseTimestampString(t *testing.T) {
	var i interface{}
	common.MustJsonDecode(`"2017-09-04T11:49:24Z"`, &i)
	ts, err := ParseTimestamp(i)
	if err != nil {
		t.Error(err)
	}
	if ts.Year() != 2017 {
		t.Error("Wrong year")
	}
}

func TestFormatRegex(t *testing.T) {
	for _, format := range formatsToTry {
		if !formatToRegex(format).MatchString(format) {
			t.Error("Doesn't match itself", format)
		}
	}
}
