package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/BurntSushi/toml"
	"github.com/zefhemel/ax/pkg/backend/docker"
	"github.com/zefhemel/ax/pkg/backend/kibana"
)

var dataDir string

type EnvMap map[string]string

type Config struct {
	DefaultEnv   string            `toml:"default"`
	Environments map[string]EnvMap `toml:"env"`
}

type RuntimeConfig struct {
	ActiveEnv string
	DataDir   string
	Env       EnvMap
}

var (
	activeEnv   = kingpin.Flag("env", "Environment to connect to").Short('e').String()
	dockerFlag  = kingpin.Flag("docker", "Query docker container logs").HintAction(docker.DockerHintAction).String()
	initCommand = kingpin.Command("init", "Initial setup of ax")
)

func BuildConfig() RuntimeConfig {
	var config Config
	_, err := toml.DecodeFile(fmt.Sprintf("%s/ax.toml", dataDir), &config)
	rc := RuntimeConfig{
		DataDir:   dataDir,
		ActiveEnv: config.DefaultEnv,
		Env:       make(EnvMap),
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error occurred reading config: %s", err)
		return rc
	}
	var ok bool
	if config.DefaultEnv != "" {
		rc.Env, ok = config.Environments[config.DefaultEnv]
		if !ok {
			fmt.Println("Undefined active environment:", config.DefaultEnv)
			os.Exit(1)
		}
	}
	if *activeEnv != "" {
		rc.Env, ok = config.Environments[*activeEnv]
		rc.ActiveEnv = *activeEnv
		if !ok {
			fmt.Println("Undefined active environment:", *activeEnv)
			os.Exit(1)
		}
	}
	if *dockerFlag != "" {
		rc.ActiveEnv = fmt.Sprintf("docker.%s", *dockerFlag)
		rc.Env["backend"] = "docker"
		rc.Env["pattern"] = *dockerFlag
	}

	return rc
}

func credentials(reader *bufio.Reader) (string, string) {
	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Password: ")
	bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	password := string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password)
}

func readLine(reader *bufio.Reader) string {
	val, _ := reader.ReadString('\n')
	return strings.TrimSpace(val)
}

func b64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func kibanaConfig(reader *bufio.Reader) {
	fmt.Print("URL: ")
	url := readLine(reader)
	fmt.Println("Attempting to connect to Kibana on ", url)
	resp, err := http.Head(url)
	if err != nil {
		fmt.Printf("Got error connecting to Kibana: %s\n", err)
		return
	}
	if resp.StatusCode == http.StatusUnauthorized {
		user, pass := credentials(reader)
		authHeader := fmt.Sprintf("Basic %s", b64Encode(fmt.Sprintf("%s:%s", user, pass)))
		fmt.Println("Checking...")
		kibanaClient := kibana.New(url, authHeader, "")
		indices, err := kibanaClient.ListIndices()
		if err != nil {
			fmt.Println("Could not connect to Kibana to get list of indices successfullly")
			return
		}
		fmt.Println("List of indices:")
		for _, index := range indices {
			fmt.Println("  ", index)
		}
		fmt.Print("Index: ")
		index := readLine(reader)
		config := Config{
			DefaultEnv: "kibana",
			Environments: map[string]EnvMap{
				"kibana": EnvMap{
					"backend": "kibana",
					"url":     url,
					"auth":    authHeader,
					"index":   index,
				},
			},
		}
		fmt.Println("Config", config)
		f, err := os.Create(fmt.Sprintf("%s/ax.toml", dataDir))
		if err != nil {
			fmt.Println("Couldn't open ax.toml for writing", err)
			return
		}
		encoder := toml.NewEncoder(f)
		err = encoder.Encode(&config)
		if err != nil {
			fmt.Println("Couldn't write to ax.toml", err)
			return
		}
	}
}

func InitSetup() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Choose a backend [kibana]: ")
	backend := readLine(reader)
	switch backend {
	case "kibana":
		kibanaConfig(reader)
	default:
		fmt.Println("Unsupported backend")
		return
	}
}

func init() {
	dataDir = fmt.Sprintf("%s/.config/ax", os.Getenv("HOME"))
	err := os.MkdirAll(dataDir, 0700)
	if err != nil {
		fmt.Println("Could not create", dataDir)
		os.Exit(1)
	}

	// Set up logging
	f, err := os.Create(fmt.Sprintf("%s/ax.log", dataDir))
	if err != nil {
		panic(err)
	}
	log.SetOutput(f)
}
