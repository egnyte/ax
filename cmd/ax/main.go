package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/zefhemel/kingpin"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/docker"
	"github.com/egnyte/ax/pkg/backend/kibana"
	"github.com/egnyte/ax/pkg/backend/stream"
	"github.com/egnyte/ax/pkg/backend/subprocess"
	"github.com/egnyte/ax/pkg/config"
)

var (
	queryCommand    = kingpin.Command("query", "Query logs").Default()
	alertCommand    = kingpin.Command("alert", "Be alerted when logs match a query")
	alertDCommand   = kingpin.Command("alertd", "Be alerted when logs match a query")
	versionCommand  = kingpin.Command("version", "Show the ax version")
	upgrade         = kingpin.Command("upgrade", "Verify if there is a new version available")
	addAlertCommand = alertCommand.Command("add", "Add new alert")
	version         = "v0.3.0"
)

func determineClient(em config.EnvMap) common.Client {
	stat, _ := os.Stdin.Stat()
	var client common.Client
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		client = stream.New(os.Stdin)
	} else if em["backend"] == "docker" {
		client = docker.New(em["pattern"])
	} else if em["backend"] == "kibana" {
		client = kibana.New(em["url"], em["auth"], em["index"])
	} else if em["backend"] == "subprocess" {
		client = subprocess.New(strings.Split(em["command"], " "))
	}
	return client
}

func sigtermContextHandler(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Canceled through SIGTERM (Ctrl-c)")
		cancel()
		// os.Exit(1) -- commented out, implicitly exiting after context cancelation cascaded
	}()

	return ctx
}

func main() {
	cmd := kingpin.Parse()

	rc := config.BuildConfig()
	client := determineClient(rc.Env)

	ctx := sigtermContextHandler(context.Background())

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
		queryMain(ctx, rc, client)
	case "env add":
		config.AddEnv()
	case "env list":
		config.ListEnvs()
	case "env edit":
		config.EditConfig()
	case "alert add":
		addAlertMain(rc, client)
	case "alertd":
		alertMain(ctx, rc)
	case "version":
		println(version)
	case "upgrade":
		if err := upgradeVersion(); err != nil {
			fmt.Println("Upgrade failed.")
		} else {
			fmt.Println("Upgrade has been completed successfully.")
		}
	}
}
