package main

import (
	"fmt"
	"github.com/egnyte/ax/pkg/backend/docker"
	"github.com/egnyte/ax/pkg/complete"
	"github.com/egnyte/ax/pkg/config"
	"github.com/spf13/cobra"
)

var (
	completionCommand = &cobra.Command{
		Use:    "complete",
		Hidden: true,
	}
)

const (
	// That's right... bash embedded in Go!
	bashCompletion = `
# Extracts --env and -e flags from the command to use it for better code completion results
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

__ax_get_containers()
{
	local output
	if output=$(ax complete docker 2>/dev/null); then
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

func init() {
	completionCommand.AddCommand(&cobra.Command{
		Use: "env",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := config.LoadConfig()
			for envName := range cfg.Environments {
				fmt.Println(envName)
			}
		},
	})
	var suffixOperatorFlag string
	attrsCommand := &cobra.Command{
		Use:  "attrs",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			defaultEnv := ""
			if len(args) > 0 {
				defaultEnv = args[0]
			}
			rc := config.BuildConfig(defaultEnv, "")
			for completion := range complete.GetCompletions(rc) {
				fmt.Printf("%s%s\n", completion, suffixOperatorFlag)
			}
		},
	}
	attrsCommand.Flags().StringVar(&suffixOperatorFlag, "suffix", "", "Suffix to use")
	completionCommand.AddCommand(attrsCommand)

	completionCommand.AddCommand(&cobra.Command{
		Use: "docker",
		Run: func(cmd *cobra.Command, args []string) {
			for _, containerName := range docker.GetRunningContainers("") {
				fmt.Println(containerName)
			}
		},
	})
}
