package subprocess

import (
	"context"
	"os/exec"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/stream"
)

type SubprocessClient struct {
	command []string
}

func (client *SubprocessClient) Query(ctx context.Context, query common.Query) <-chan common.LogMessage {
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
		stdOutQuery := stdOutStream.Query(ctx, query)
		stdErrQuery := stdErrStream.Query(ctx, query)
		stdOutOpen := true
		stdErrOpen := true
		for stdOutOpen || stdErrOpen {
			select {
			case message, ok := <-stdOutQuery:
				if !ok {
					stdOutOpen = false
					continue
				}
				resultChan <- message
			case message, ok := <-stdErrQuery:
				if !ok {
					stdErrOpen = false
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
