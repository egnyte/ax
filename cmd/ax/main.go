package main

import (
	"fmt"
	"os"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/zefhemel/ax/pkg/backend/common"
	"github.com/zefhemel/ax/pkg/backend/docker"
	"github.com/zefhemel/ax/pkg/backend/kibana"
	"github.com/zefhemel/ax/pkg/backend/stream"
	"github.com/zefhemel/ax/pkg/backend/subprocess"
	"github.com/zefhemel/ax/pkg/config"
)

var (
	queryCommand = kingpin.Command("query", "Query logs").Default()
)

func main() {
	stat, _ := os.Stdin.Stat()
	cmd := kingpin.Parse()

	rc := config.BuildConfig()
	var client common.Client
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		client = stream.New(os.Stdin)
	} else if rc.Env["backend"] == "docker" {
		client = docker.New(rc.Env["pattern"])
	} else if rc.Env["backend"] == "kibana" {
		client = kibana.New(rc.Env["url"], rc.Env["auth"], rc.Env["index"])
	} else if rc.Env["backend"] == "subprocess" {
		client = subprocess.New(strings.Split(rc.Env["command"], " "))
	}

	switch cmd {
	case "query":
		if client == nil {
			fmt.Println("No env selected, and no default set. Exiting.")
			return
		}
		queryMain(rc, client)
	case "env add":
		config.AddEnv()
	case "env list":
		config.ListEnvs()
	}

}
