package config

import (
	"bufio"
	"fmt"

	"github.com/egnyte/ax/pkg/backend/cloudwatch"
)

func cloudwatchConfig(reader *bufio.Reader, existingConfig Config) (EnvMap, error) {
	em := EnvMap{
		"backend": "cloudwatch",
	}
	existingCwEnv := findFirstEnvWhere(existingConfig.Environments, func(em EnvMap) bool {
		return em["backend"] == "cloudwatch"
	})
	if existingCwEnv != nil {
		accessKey := (*existingCwEnv)["accesskey"]
		fmt.Printf("Access Key ID [%s]: ", accessKey)
		em["accesskey"] = readLine(reader)
		if em["accesskey"] == "" {
			em["accesskey"] = (*existingCwEnv)["accesskey"]
			em["accesssecretkey"] = (*existingCwEnv)["accesssecretkey"]
		}
	} else {
		fmt.Print("Access Secret Key: ")
		em["accesssecretkey"] = readLine(reader)
	}
	fmt.Print("Region (us-east-1, us-west-1, us-west-2, eu-west-1, eu-central-1, ap-southeast-1, ap-southeast-2, ap-northeast-1, sa-east-1): ")
	em["region"] = readLine(reader)
	var cwClient *cloudwatch.CloudwatchClient
	var groups []string
	var err error
	fmt.Println("Attempting to connect to Cloudwatch")
	cwClient = cloudwatch.New(em["accesskey"], em["accesssecretkey"], em["region"], "")
	groups, err = cwClient.ListGroups()
	if err != nil {
		fmt.Printf("Got error connecting to Cloudwatch: %s\n", err)
		return em, err
	}
	fmt.Println("List of groups:")
	for _, group := range groups {
		fmt.Println("  ", group)
	}
	fmt.Print("Group: ")
	em["groupname"] = readLine(reader)
	return em, nil
}
