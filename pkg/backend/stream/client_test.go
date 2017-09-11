package stream

import (
	"strings"
	"testing"
	"time"

	"github.com/egnyte/ax/pkg/backend/common"
)

func TestFindTimestamp(t *testing.T) {
	sampleData := `{"jstimestamp":1504516581620, "message": "Sup yo"}
{"ts": "2017-08-04T11:16:52.088Z", "message": "Sup yo 2"}
`
	sc := New(strings.NewReader(sampleData))
	for msg := range sc.Query(common.Query{}) {
		//fmt.Printf("%+v\n", msg)
		if msg.Timestamp.Day() != 4 {
			t.Error("Wrong day", msg.Timestamp.Day())
		}
		if msg.Timestamp.Minute() != 16 {
			t.Error("Wrong minute", msg.Timestamp.Minute())
		}
	}
}

func TestFindTimestampInMessage(t *testing.T) {
	sampleData := `{"message": "INFO: This happened at 2017-07-04T14:09:28.184Z so"}
{"message": "2017-06-04 06:52:14,689  INFO    All ELCC processes are running"}
{"message": "[2017-05-04 08:51:14,689]  INFO    All ELCC processes are running"}
{"message": "(2017-06-04 09:25:39,261) INFO    (Processor) End of sync notification sent to server"}
`
	months := []time.Month{7, 6, 5, 6}
	sc := New(strings.NewReader(sampleData))
	counter := 0
	for msg := range sc.Query(common.Query{}) {
		if msg.Timestamp.Month() != months[counter] {
			t.Error("Wrong month", msg.Timestamp.Month(), "expected", months[counter], "in", msg.Attributes["message"])
		}
		message, _ := msg.Attributes["message"].(string)
		if !strings.HasPrefix(message, "INFO") {
			t.Error("Stripping of timestamp failed")
		}

		counter++
	}

}
