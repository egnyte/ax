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

const (
	bashCompletion = `
__ax_parse_env_flag()
{
	local envflag
	envflag=$(echo "${words[@]}" | sed -E 's/.*(--env[= ]+|-e +)([A-Za-z_-]+).*/\2/')
	if [[ $envflag = *" "* ]]; then
	    __ax_debug "No env found"
	    envflag=""
    else
		__ax_debug "Env: ${envflag}"
	fi
	echo ${envflag}
}

__ax_get_envs()
{
	local output
	if output=$(ax complete env 2>/dev/null); then
		COMPREPLY=( $(compgen -W "${output[*]}" -- "$cur") )
    fi
}

__ax_get_attrs_where()
{
	__ax_get_attrs_with_suffix "="
}

__ax_get_attrs_where2()
{
	__ax_get_attrs_with_suffix ":"
}

__ax_get_attrs_select()
{
	__ax_get_attrs_with_suffix ""
}


__ax_get_attrs_with_suffix()
{
	local output envflag suffix
	suffix=$1
	envflag=$(__ax_parse_env_flag)
	if output=$(ax complete attrs ${envflag} --suffix=${suffix} 2>/dev/null); then
		__ax_debug "Completion results: ${output[*]}"
		COMPREPLY=( $(compgen -W "${output[*]}" -- "$cur") )
    fi
}
`
)

var (
	version = "dev"
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

var rootCmd = &cobra.Command{
	Use:   "ax",
	Short: "Ax is a structured log query tool",
	BashCompletionFunction: bashCompletion,
	Args: cobra.MinimumNArgs(0),
}

var (
	defaultEnvFlag string
	dockerFlag     string
	bashScriptFlag bool
)

func init() {
	// Global flags
	persistentFlags := rootCmd.PersistentFlags()
	persistentFlags.StringVar(&defaultEnvFlag, "env", "", "Default environment to use")
	persistentFlags.Lookup("env").Annotations = map[string][]string{cobra.BashCompCustom: {"__ax_get_envs"}}
	persistentFlags.StringVar(&dockerFlag, "docker", "", "Docker container prefix")
	persistentFlags.BoolVar(&bashScriptFlag, "completion-script-bash", false, "Generate bash script")

	// Commands
	rootCmd.AddCommand(&cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	})

	rootCmd.AddCommand(config.EnvCommand())
	rootCmd.AddCommand(completionCommand)
}

func main() {
	rootCmd.Execute()
}

func oldMain() {
	rc := config.BuildConfig("", "")
	client := determineClient(rc.Env)

	cmd := "legacy"

	switch cmd {
	case "query":
	case "env add":
		config.AddEnv()
	case "env list":
		config.ListEnvs()
	case "env edit":
		config.EditConfig()
	case "alert add":
		addAlertMain(rc, client)
	case "alertd":
		alertMain(context.Background(), rc)
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
