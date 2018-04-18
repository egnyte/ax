package subprocess

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/stream"
)

type SubprocessClient struct {
	command []string
}

func (client *SubprocessClient) Query(ctx context.Context, query common.Query) <-chan common.LogMessage {
	resultChan := make(chan common.LogMessage)
	cmd := exec.Command(client.command[0], client.command[1:]...)
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Could not get stdout pipe: %v", err)
		close(resultChan)
		return resultChan
	}
	stdOutStream := stream.New(stdOut)
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Could not get stderr pipe: %v", err)
		close(resultChan)
		return resultChan
	}
	stdErrStream := stream.New(stdErr)
	if err := cmd.Start(); err != nil {
		fmt.Printf("Could not start process: %s because: %v\n", client.command[0], err)
		close(resultChan)
		return resultChan
	}
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
			case <-ctx.Done():
				close(resultChan)
				cmd.Process.Kill() // Ignoring error, not sure if that's ok
				// Returning to avoid the Wait()
				return
			}

		}
		close(resultChan)
		if err := cmd.Wait(); err != nil {
			fmt.Printf("Process exited with error: %v\n", err)
		}
	}()
	return resultChan
}

func New(command []string) *SubprocessClient {
	return &SubprocessClient{command}
}

var _ common.Client = &SubprocessClient{}
