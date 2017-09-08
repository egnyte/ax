package subprocess

import (
	"os/exec"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/stream"
)

type SubprocessClient struct {
	command []string
}

func (client *SubprocessClient) Query(query common.Query) <-chan common.LogMessage {
	cmd := exec.Command(client.command[0], client.command[1:]...)
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stdOutStream := stream.New(stdOut)
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	stdErrStream := stream.New(stdErr)
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	resultChan := make(chan common.LogMessage)
	go func() {
		stdOutQuery := stdOutStream.Query(query)
		stdErrQuery := stdErrStream.Query(query)
		closed := 0
		for closed < 2 {
			select {
			case message, ok := <-stdOutQuery:
				if !ok {
					closed++
					continue
				}
				resultChan <- message
			case message, ok := <-stdErrQuery:
				if !ok {
					closed++
					continue
				}
				resultChan <- message
			}
		}
		close(resultChan)
		if err := cmd.Wait(); err != nil {
			panic(err)
		}
	}()
	return resultChan
}

func New(command []string) *SubprocessClient {
	return &SubprocessClient{command}
}

var _ common.Client = &SubprocessClient{}
