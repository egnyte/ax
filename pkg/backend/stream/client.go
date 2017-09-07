package stream

import (
	"bufio"

	"io"

	"encoding/json"
	"strings"

	"time"

	"github.com/zefhemel/ax/pkg/backend/common"
	"github.com/zefhemel/ax/pkg/heuristic"
)

type Client struct {
	reader io.Reader
}

func New(file io.Reader) *Client {
	return &Client{file}
}

func parseLine(line string) common.LogMessage {
	decoder := json.NewDecoder(strings.NewReader(line))
	obj := make(map[string]interface{})
	err := decoder.Decode(&obj)
	if err != nil {
		obj["message"] = strings.TrimSpace(line)
		return common.LogMessage{
			Timestamp:  time.Now(),
			Attributes: obj,
		}
	}
	return common.LogMessage{
		Timestamp:  time.Now(), // TODO: Fix this
		Attributes: obj,
	}
}

func (client *Client) Query(q common.Query) <-chan common.LogMessage {
	resultChan := make(chan common.LogMessage)
	reader := bufio.NewReader(client.reader)
	go func() {
		var ltFunc heuristic.LogTimestampParser
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				// TODO: erro != EOF break?
				if err == io.EOF {
					break
				}
				//fmt.Println("Error: ", err)
				break
			}
			message := parseLine(line)
			if ltFunc == nil {
				ltFunc = heuristic.FindTimestampFunc(message)
			}
			if ltFunc != nil {
				ts := ltFunc(message)
				if ts != nil {
					message.Timestamp = *ts
				} else {
					ltFunc = heuristic.FindTimestampFunc(message)
					if ltFunc != nil {
						ts := ltFunc(message)
						message.Timestamp = *ts
					}
				}
			}
			if common.MatchesQuery(message, q) {
				message.Attributes = common.Project(message.Attributes, q.SelectFields)
				resultChan <- message
			}
		}
		close(resultChan)
	}()

	return resultChan
}

var _ common.Client = &Client{}
