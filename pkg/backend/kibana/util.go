package kibana

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"time"
)

func createMultiSearch(objs ...interface{}) (io.Reader, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, obj := range objs {
		err := encoder.Encode(obj)
		if err != nil {
			return nil, err
		}
	}
	//fmt.Println(buf.String())
	return &buf, nil
}

func safeFilename(name string) string {
	re := regexp.MustCompile(`[^\w\-]`)
	return re.ReplaceAllString(name, "_")
}

func unixMillis(t time.Time) int64 {
	return t.Unix() * 1000
}
