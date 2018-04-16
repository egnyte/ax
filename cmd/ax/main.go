package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/zefhemel/kingpin"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/docker"
	"github.com/egnyte/ax/pkg/backend/file"
	"github.com/egnyte/ax/pkg/backend/kibana"
	"github.com/egnyte/ax/pkg/backend/stream"
	"github.com/egnyte/ax/pkg/backend/subprocess"
	"github.com/egnyte/ax/pkg/config"
)

var (
	queryCommand    = kingpin.Command("query", "Query logs").Default()
	alertCommand    = kingpin.Command("alert", "Be alerted when logs match a query")
	alertDCommand   = kingpin.Command("alertd", "Be alerted when logs match a query")
	addAlertCommand = alertCommand.Command("add", "Add new alert")
)

func determineClient(em config.EnvMap) common.Client {
	stat, _ := os.Stdin.Stat()
	var client common.Client
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		client = stream.New(os.Stdin)
	} else if em["backend"] == "docker" {
		client = docker.New(em["pattern"])
	} else if em["backend"] == "file" {
		client = file.New(em["filename"])
	} else if em["backend"] == "kibana" {
		client = kibana.New(em["url"], em["auth"], em["index"])
	} else if em["backend"] == "subprocess" {
		client = subprocess.New(strings.Split(em["command"], " "))
	}
	return client
}

func main() {
	cmd := kingpin.Parse()

	rc := config.BuildConfig()
	client := determineClient(rc.Env)

	switch cmd {
	case "query":
		if client == nil {
			if len(rc.Config.Environments) == 0 {
				// Assuming first time use
				fmt.Println("Welcome to ax! It looks like this is the first time running, so let's start with creating a new environment.")
				config.AddEnv()
				return
			}
			fmt.Println("No default environment set, please use the --env flag to set one. Exiting.")
			return
		}
		queryMain(rc, client)
	case "env add":
		config.AddEnv()
	case "env list":
		config.ListEnvs()
	case "env edit":
		config.EditConfig()
	case "alert add":
		addAlertMain(rc, client)
	case "alertd":
		alertMain(rc)
	}

}
