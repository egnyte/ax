package config

import (
	"bufio"
	"fmt"

	"github.com/egnyte/ax/pkg/backend/kibana"
)

func kibanaConfig(reader *bufio.Reader, existingConfig Config) (EnvMap, error) {
	em := EnvMap{
		"backend": "kibana",
	}
	existingKibanaEnv := findFirstEnvWhere(existingConfig.Environments, func(em EnvMap) bool {
		return em["backend"] == "kibana"
	})
	if existingKibanaEnv != nil {
		defaultUrl := (*existingKibanaEnv)["url"]
		fmt.Printf("URL [%s]: ", defaultUrl)
		em["url"] = readLine(reader)
		if em["url"] == "" {
			em["auth"] = (*existingKibanaEnv)["auth"]
			em["url"] = defaultUrl
		}
	} else {
		fmt.Print("URL: ")
		em["url"] = readLine(reader)
	}
	var kibanaClient *kibana.Client
	var indices []string
	var err error
	for {
		fmt.Println("Attempting to connect to Kibana on ", em["url"])
		kibanaClient = kibana.New(em["url"], em["auth"], "")
		indices, err = kibanaClient.ListIndices()
		if err != nil && err.Error() == "Authentication failed" {
			user, pass := credentials(reader)
			em["auth"] = fmt.Sprintf("Basic %s", b64Encode(fmt.Sprintf("%s:%s", user, pass)))
			continue
		} else if err != nil {
			fmt.Printf("Got error connecting to Kibana: %s\n", err)
			return em, err
		}
		break
	}
	fmt.Println("List of indices:")
	for _, index := range indices {
		fmt.Println("  ", index)
	}
	fmt.Print("Index: ")
	em["index"] = readLine(reader)
	return em, nil
}
