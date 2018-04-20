package config

import (
	"bufio"
	"fmt"

	"github.com/egnyte/ax/pkg/backend/stackdriver"
)

func testStackdriver(em EnvMap) bool {
	cwClient := stackdriver.New(em["credentials"], em["project"], em["log"])
	_, err := cwClient.ListLogs()
	return err != nil
}

func stackdriverConfig(reader *bufio.Reader, existingConfig Config) (EnvMap, error) {
	em := EnvMap{
		"backend": "stackdriver",
	}
	existingSdEnv := findFirstEnvWhere(existingConfig.Environments, func(em EnvMap) bool {
		return em["backend"] == "stackdriver"
	})
	if existingSdEnv != nil {
		credentialsPath := (*existingSdEnv)["credentials"]
		fmt.Printf("Path to credentials file (JSON) [%s]: ", credentialsPath)
		em["credentials"] = readLine(reader)
		if em["credentials"] == "" {
			em["credentials"] = (*existingSdEnv)["credentials"]
		}
	} else {
		fmt.Print("Path to credentials file (JSON): ")
		em["credentials"] = readLine(reader)
	}
	fmt.Print("GCP Project name: ")
	em["project"] = readLine(reader)
	var sdClient *stackdriver.StackdriverClient
	var logs []string
	var err error
	for {
		fmt.Println("Attempting to connect to Stackdriver")
		sdClient = stackdriver.New(em["credentials"], em["project"], "")
		logs, err = sdClient.ListLogs()
		if err != nil {
			fmt.Printf("Got error connecting to Stackdriver: %s\n", err)
			return em, err
		}
		break
	}
	fmt.Println("List of logs:")
	for _, log := range logs {
		fmt.Println("  ", log)
	}
	fmt.Print("Log: ")
	em["log"] = readLine(reader)
	return em, nil
}
