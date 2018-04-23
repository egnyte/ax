package heuristic

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/egnyte/ax/pkg/backend/common"
)

var ErrorCouldNotParse = errors.New("Could not parse timestamp")

type TimestampParser func(interface{}) *time.Time
type LogTimestampParser func(common.LogMessage) *time.Time
type MessageTimestampParser func(message string) *time.Time

var formatsToTry []string = []string{
	time.RFC3339,
	time.ANSIC,
	time.UnixDate,
	time.RubyDate,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC3339Nano,
	"2006-01-02 15:04:05,000",
}

var formatsToTryRx []*regexp.Regexp = []*regexp.Regexp{
	// time.RFC3339,
	regexp.MustCompile(`\d+-\d+-\d+T\d+:\d+:\d+(\.\d+)?(Z\d*:?\d*)?`),
	// time.ANSIC,
	regexp.MustCompile(`[A-Za-z_]+ [A-Za-z_]+ +\d+ \d+:\d+:\d+ \d+`),
	// time.UnixDate,
	regexp.MustCompile(`[A-Za-z_]+ [A-Za-z_]+ +\d+ \d+:\d+:\d+( [A-Za-z_]+ \d+)?`),
	// TODO: Update these
	// time.RubyDate,
	regexp.MustCompile(`[A-Za-z_]+ [A-Za-z_]+ \d+ \d+:\d+:\d+ [\-\+]\d+ \d+`),
	// time.RFC822,
	regexp.MustCompile(`\d+ [A-Za-z_]+ \d+ \d+:\d+ [A-Za-z_]+`),
	// time.RFC822Z,
	regexp.MustCompile(`\d+ [A-Za-z_]+ \d+ \d+:\d+ -\d+`),
	// time.RFC850,
	regexp.MustCompile(`[A-Za-z_]+, \d+-[A-Za-z_]+-\d+ \d+:\d+:\d+ [A-Za-z_]+`),
	// time.RFC1123,
	regexp.MustCompile(`[A-Za-z_]+, \d+ [A-Za-z_]+ \d+ \d+:\d+:\d+ [A-Za-z_]+`),
	// time.RFC1123Z,
	regexp.MustCompile(`[A-Za-z_]+, \d+ [A-Za-z_]+ \d+ \d+:\d+:\d+ -\d+`),
	// time.RFC3339Nano,
	regexp.MustCompile(`\d+-\d+-\d+[A-Za-z_]+\d+:\d+:\d+\.\d+[A-Za-z_]+\d+:\d+`),
	// "2006-01-02 15:04:05",
	regexp.MustCompile(`\d+-\d+-\d+ \d+:\d+:\d+(,\d+)?`),
}

func epochMsToTime(i int64) *time.Time {
	now := time.Now()
	ts := time.Unix(i/1000, 0)
	if ts.Year() < 2000 || ts.Year() > now.Year() {
		return nil
	}
	return &ts
}

func epochToTime(i int64) *time.Time {
	now := time.Now()
	ts := time.Unix(i, 0)
	if ts.Year() < 2000 || ts.Year() > now.Year() {
		return epochMsToTime(i)
	}
	return &ts
}

func GuessTimestampParseFunc(exampleV interface{}) TimestampParser {
	switch exampleVal := exampleV.(type) {
	case float64:
		t := epochToTime(int64(exampleVal))
		if t != nil {
			return func(v interface{}) *time.Time {
				if val, ok := v.(float64); ok {
					return epochToTime(int64(val))
				} else {
					return nil
				}
			}
		}
	case string:
		// Try to parse as an int
		i, err := strconv.ParseInt(exampleVal, 10, 64)
		if err == nil {
			t := epochToTime(i)
			if t != nil {
				return func(v interface{}) *time.Time {
					if val, ok := v.(string); ok {
						i, err := strconv.ParseInt(val, 10, 64)
						if err != nil {
							return nil
						}
						return epochToTime(i)
					} else {
						return nil
					}
				}
			}
		}
		for _, encoding := range formatsToTry {
			_, err := parseTime(encoding, exampleVal)
			if err == nil {
				return func(v interface{}) *time.Time {
					if val, ok := v.(string); ok {
						ts, err := parseTime(encoding, val)
						if err != nil {
							return nil
						}
						return &ts
					}
					return nil
				}
			}
		}
	}
	return nil
}

func ParseTimestamp(v interface{}) (*time.Time, error) {
	fn := GuessTimestampParseFunc(v)
	if fn == nil {
		return nil, ErrorCouldNotParse
	}
	return fn(v), nil
}

// Currently unused
func formatToRegex(format string) *regexp.Regexp {
	rformat := regexp.QuoteMeta(format)
	r := regexp.MustCompile(`[A-Za-z_]+`)
	rformat = r.ReplaceAllString(rformat, `[A-Za-z_]+`)
	r = regexp.MustCompile(`\d+`)
	rformat = r.ReplaceAllString(rformat, `\d+`)
	//fmt.Println("Regex for", format, rformat)
	return regexp.MustCompile(rformat)
}

// Same as time.Parse except handling ,XXX case
// https://github.com/golang/go/issues/6189
var formatReplace *regexp.Regexp = regexp.MustCompile(`,\d+`)

func parseTime(format, s string) (time.Time, error) {
	if strings.HasSuffix(format, ",000") {
		s = formatReplace.ReplaceAllString(s, "")
		format = formatReplace.ReplaceAllString(format, "")
	}
	return time.Parse(format, s)
}

func findTimestampInMessage(exampleMessage common.LogMessage) LogTimestampParser {
	//fmt.Println("Message", message)
	message, _ := exampleMessage.Attributes["message"].(string)
	for i, formatRx := range formatsToTryRx {
		if times := formatRx.FindString(message); times != "" {
			_, err := parseTime(formatsToTry[i], times)
			if err != nil {
				continue
			}
			return func(lm common.LogMessage) *time.Time {
				message, _ := lm.Attributes["message"].(string)
				if times := formatRx.FindString(message); times != "" {
					ts, err := parseTime(formatsToTry[i], times)
					if err != nil {
						return nil
					}
					wrapperRegExp := regexp.MustCompile(fmt.Sprintf(`[\[\(]?%s[\]\)]?\s*`, formatRx))
					message = wrapperRegExp.ReplaceAllString(message, "")
					lm.Attributes["message"] = message
					return &ts
				} else {
					return nil
				}
			}
		}
	}
	return nil
}

func FindTimestampFunc(exampleMessage common.LogMessage) LogTimestampParser {
	for k, v := range exampleMessage.Attributes {
		if fn := GuessTimestampParseFunc(v); fn != nil {
			return func(lm common.LogMessage) *time.Time {
				return fn(lm.Attributes[k])
			}
		}
	}
	return findTimestampInMessage(exampleMessage)
}

//func GetTimestamp(lm common.LogMessage) (*time.Time, error) {
//	if fn := FindTimestampFunc(lm); fn != nil {
//		return fn(lm), nil
//	}
//	if fn := findTimestampInMessage(lm); fn != nil {
//		return fn(lm), nil
//	}
//	return nil, errors.New("Not found")
//}
