package stream

import (
	"bufio"

	"io"

	"encoding/json"
	"strings"

	"time"

	"github.com/zefhemel/ax/pkg/backend/common"
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
		return common.LogMessage{
			Message:    strings.TrimSpace(line),
			Timestamp:  time.Now(),
			Attributes: obj,
		}
	}
	message, _ := obj["message"].(string)
	delete(obj, "message")
	return common.LogMessage{
		Message:    message,
		Timestamp:  time.Now(), // TODO: Fix this
		Attributes: obj,
	}
}

func (client *Client) Query(q common.Query) <-chan common.LogMessage {
	resultChan := make(chan common.LogMessage)
	reader := bufio.NewReader(client.reader)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				//fmt.Println("Error: ", err)
				break
			}
			message := parseLine(line)
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
