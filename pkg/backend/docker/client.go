package docker

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/subprocess"
)

type DockerClient struct {
	containerPattern string
}

func GetRunningContainers(pattern string) []string {
	flags := []string{"ps", "--format", "{{.Names}}"}
	if pattern != "" {
		flags = append(flags, "--filter", fmt.Sprintf("name=%s", pattern))
	}
	allContainers, err := exec.Command("docker", flags...).Output()
	if err != nil {
		log.Printf("Retrieving all containers failed: %v\n", err)
		return []string{}
	}
	return strings.Split(strings.TrimSpace(string(allContainers)), "\n")
}

func DockerHintAction() []string {
	return GetRunningContainers("")
}

func (client *DockerClient) Query(ctx context.Context, query common.Query) <-chan common.LogMessage {
	resultChan := make(chan common.LogMessage)
	runningCommands := 0
	for _, containerName := range GetRunningContainers(client.containerPattern) {
		command := []string{"docker", "logs", "--tail", fmt.Sprintf("%d", query.MaxResults)}
		if query.Follow {
			command = append(command, "-f")
		}
		command = append(command, containerName)
		client := subprocess.New(command)
		runningCommands++
		go func() {
			for message := range client.Query(ctx, query) {
				message.Attributes["@container"] = containerName
				resultChan <- message
			}
			runningCommands--
			if runningCommands == 0 {
				close(resultChan)
			}
		}()
	}
	return resultChan
}

func New(containerPattern string) *DockerClient {
	return &DockerClient{containerPattern}
}

var _ common.Client = &DockerClient{}
