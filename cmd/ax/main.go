package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/egnyte/ax/pkg/backend/cloudwatch"
	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/docker"
	"github.com/egnyte/ax/pkg/backend/kibana"
	"github.com/egnyte/ax/pkg/backend/stackdriver"
	"github.com/egnyte/ax/pkg/backend/stream"
	"github.com/egnyte/ax/pkg/backend/subprocess"
	"github.com/egnyte/ax/pkg/config"

	"github.com/spf13/cobra"
)

var (
	version        = "dev"
	defaultEnvFlag string
	dockerFlag     string
	bashScriptFlag bool
	rootCmd        = &cobra.Command{
		Use:   "ax [SEARCH PHRASE]",
		Short: "Ax is a structured log query tool",
		BashCompletionFunction: bashCompletion,
		Args: cobra.ArbitraryArgs,
	}
	versionCommand = &cobra.Command{
		Use:   "version",
		Short: "Check the version of Ax you are running",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
)

func determineClient(em config.EnvMap) common.Client {
	stat, _ := os.Stdin.Stat()
	var client common.Client
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		client = stream.New(os.Stdin)
	} else {
		switch em["backend"] {
		case "docker":
			client = docker.New(em["pattern"])
		case "kibana":
			client = kibana.New(em["url"], em["auth"], em["index"])
		case "cloudwatch":
			client = cloudwatch.New(em["accesskey"], em["accesssecretkey"], em["region"], em["groupname"])
		case "stackdriver":
			client = stackdriver.New(em["credentials"], em["project"], em["log"])
		case "subprocess":
			client = subprocess.New(strings.Split(em["command"], " "))
		}
	}
	return client
}

func sigtermContextHandler(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		switch <-c {
		case os.Interrupt:
			fmt.Println("Canceled through SIGINT (Ctrl-C)")
		case syscall.SIGTERM:
			fmt.Println("Canceled through SIGTERM")
		}
		cancel()
	}()

	return ctx
}

func main() {
	// Global flags
	persistentFlags := rootCmd.PersistentFlags()
	persistentFlags.StringVar(&defaultEnvFlag, "env", "", "Environment to use")
	persistentFlags.Lookup("env").Annotations = map[string][]string{cobra.BashCompCustom: {"__ax_get_envs"}}
	persistentFlags.StringVar(&dockerFlag, "docker", "", "Query docker containers with a certain prefix, use * to query all")
	addCompletionFunc(persistentFlags, "docker", "__ax_get_containers")

	// Bash script generation
	// TODO: Add back zsh support
	rootCmd.Flags().BoolVar(&bashScriptFlag, "completion-script-bash", false, "Generate bash script")

	// Add all commands
	rootCmd.AddCommand(config.EnvCommand(), completionCommand, upgradeCommand, alertCommand, alertDCommand, versionCommand)
	rootCmd.Execute()
}
