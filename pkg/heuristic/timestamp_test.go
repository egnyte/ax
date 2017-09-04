package heuristic

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zefhemel/ax/pkg/backend/common"
	"github.com/zefhemel/ax/pkg/backend/stream"
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

func TestFindTimestamp(t *testing.T) {
	sampleData := `{"jstimestamp":1504530981620, "message": "Sup yo"}
{"ts": "2017-09-04T13:16:52.088Z", "message": "Sup yo 2"}
`
	sc := stream.New(strings.NewReader(sampleData))
	for msg := range sc.Query(common.Query{}) {
		ts, err := GetTimestamp(msg)
		if err != nil {
			t.Error("Could not get timestamp from", msg)
		}
		if ts.Year() != 2017 {
			t.Error("Wrong year", ts.Year())
		}
	}
}

func TestFindTimestampInMessage(t *testing.T) {
	sampleData := `{"message": "2017-09-04 06:52:14,689  INFO    All ELCC processes are running"}
{"message": "This happened at 2017-09-04T14:09:28.184Z so"}
`
	sc := stream.New(strings.NewReader(sampleData))
	for msg := range sc.Query(common.Query{}) {
		ts, err := GetTimestamp(msg)
		if err != nil {
			t.Error("Could not get timestamp from", msg)
		}
		if ts.Year() != 2017 {
			t.Error("Wrong year", ts.Year())
		}
	}

}

func TestFormatRegex(t *testing.T) {
	for _, format := range formatsToTry {
		if !formatToRegex(format).MatchString(format) {
			t.Error("Doesn't match itself", format)
		}
	}
	fmt.Println(formatToRegex(time.ANSIC))
}
