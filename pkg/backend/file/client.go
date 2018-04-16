package file

import (
	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/subprocess"
)

type FileClient struct {
	filename string
}

func (client *FileClient) Query(query common.Query) <-chan common.LogMessage {
	resultChan := make(chan common.LogMessage)
	command := []string{"cat"}
	if query.Follow {
		command = []string{"tail", "-f"}
	}
	command = append(command, client.filename)
	proc := subprocess.New(command)
	go func() {
		for message := range proc.Query(query) {
			resultChan <- message
		}
		close(resultChan)
	}()
	return resultChan
}

func New(filename string) *FileClient {
	return &FileClient{filename}
}

var _ common.Client = &FileClient{}
