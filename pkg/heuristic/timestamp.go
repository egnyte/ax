package heuristic

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/zefhemel/ax/pkg/backend/common"
)

const bottomEpochBarrier = 100000000
const topSecondEpochBarrier = 10000000000
const topMsEpochBarrier = 3000000000000

var ErrorCouldNotParse = errors.New("Could not parse timestamp")

type TimestampParser func(interface{}) time.Time
type LogTimestampParser func(common.LogMessage) time.Time

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
	"2006-01-02 15:04:05",
}

var formatsToTryRx []*regexp.Regexp

func GuessTimestampParseFunc(v interface{}) TimestampParser {
	switch val := v.(type) {
	case float64:
		// Assuming this is a Unix timestamp
		if val < bottomEpochBarrier {
			return nil
		}
		if val > topMsEpochBarrier {
			return nil
		}
		if val > topSecondEpochBarrier {
			// in millis
			return func(v interface{}) time.Time {
				return time.Unix(int64(v.(float64))/1000, 0)
			}
		} else {
			return func(v interface{}) time.Time {
				return time.Unix(int64(v.(float64)), 0)
			}
		}
	case string:
		// Try to parse as an int
		i, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			if i < bottomEpochBarrier {
				break
			}
			if i > topMsEpochBarrier {
				break
			}
			// Assuming this is a Unix timestamp
			if i > topSecondEpochBarrier {
				// in millis
				return func(v interface{}) time.Time {
					i, _ := strconv.ParseInt(v.(string), 10, 64)
					return time.Unix(i/1000, 0)
				}
			} else {
				return func(v interface{}) time.Time {
					i, _ := strconv.ParseInt(v.(string), 10, 64)
					return time.Unix(i, 0)
				}
			}
		}
		for _, encoding := range formatsToTry {
			_, err := time.Parse(encoding, val)
			if err == nil {
				return func(v interface{}) time.Time {
					ts, _ := time.Parse(encoding, v.(string))
					return ts
				}
			}
		}
	}
	return nil
}

func ParseTimestamp(v interface{}) (time.Time, error) {
	fn := GuessTimestampParseFunc(v)
	if fn == nil {
		return time.Now(), ErrorCouldNotParse
	}
	return fn(v), nil
}

func formatToRegex(format string) *regexp.Regexp {
	rformat := regexp.QuoteMeta(format)
	r := regexp.MustCompile(`[A-Za-z_]+`)
	rformat = r.ReplaceAllString(rformat, `[A-Za-z_]+`)
	r = regexp.MustCompile(`\d+`)
	rformat = r.ReplaceAllString(rformat, `\d+`)
	fmt.Println("Regex for", format, rformat)
	return regexp.MustCompile(rformat)
}

func findTimestampInMessage(message string) (time.Time, error) {
	fmt.Println("Message", message)
	for i, formatRx := range formatsToTryRx {
		if times := formatRx.FindString(message); times != "" {
			fmt.Println("Potential match", times)
			ts, err := time.Parse(formatsToTry[i], times)
			if err != nil {
				continue
			}
			return ts, nil
		}
	}
	return time.Now(), errors.New("Not found")
}

func FindTimestampFunc(exampleMessage common.LogMessage) LogTimestampParser {
	for k, v := range exampleMessage.Attributes {
		if fn := GuessTimestampParseFunc(v); fn != nil {
			return func(lm common.LogMessage) time.Time {
				return fn(lm.Attributes[k])
			}
		}
	}
	return nil
}

func GetTimestamp(lm common.LogMessage) (time.Time, error) {
	if fn := FindTimestampFunc(lm); fn != nil {
		return fn(lm), nil
	}
	msg, _ := lm.Attributes["message"].(string)
	return findTimestampInMessage(msg)
}

func init() {
	formatsToTryRx = make([]*regexp.Regexp, len(formatsToTry))
	for i, format := range formatsToTry {
		formatsToTryRx[i] = formatToRegex(format)
	}
}
