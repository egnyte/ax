package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/imdario/mergo"
	"github.com/zefhemel/kingpin"
	"gopkg.in/yaml.v2"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var dataDir string

type EnvMap map[string]string

type Config struct {
	DefaultEnv   string            `yaml:"default"`
	Colors       ColorConfig       `yaml:"colors"`
	Environments map[string]EnvMap `yaml:"env"`
	Alerts       []AlertConfig     `yaml:"alerts"`
}

type AlertConfig struct {
	Env      string                `yaml:"env"`
	Name     string                `yaml:"name"`
	Selector common.QuerySelectors `yaml:"selector"`
	Service  AlertServiceConfig    `yaml:"service"`
}

type AlertServiceConfig map[string]string

type RuntimeConfig struct {
	ActiveEnv string
	DataDir   string
	Env       EnvMap
	Config    Config
}

var (
	//activeEnv      = kingpin.Flag("env", "Environment to connect to").Short('e').HintAction(envHintAction).String()
	// dockerFlag     = kingpin.Flag("docker", "Query docker container logs").HintAction(docker.DockerHintAction).String()
	listPlainFlag  bool
	envCommand     = kingpin.Command("env", "Environment management commands")
	envInitCommand = envCommand.Command("add", "Add an environment")
	envEditCommand = envCommand.Command("edit", "Edit your environment configuration file in a text editor")
	envListCommand = envCommand.Command("list", "List all environments").Default()
)

func NewConfig() Config {
	return Config{
		Environments: make(map[string]EnvMap),
		Alerts:       make([]AlertConfig, 0),
	}
}

func EnvCommand() *cobra.Command {

	listCommand := &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			ListEnvs()
		},
	}

	addCommand := &cobra.Command{
		Use: "add",
		Run: func(cmd *cobra.Command, args []string) {
			AddEnv()
		},
	}
	editCommand := &cobra.Command{
		Use: "edit",
		Run: func(cmd *cobra.Command, args []string) {
			EditConfig()
		},
	}

	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Environment management",
		Run: func(cmd *cobra.Command, args []string) {
			listCommand.Run(cmd, args)
		},
	}
	envCmd.AddCommand(listCommand, addCommand, editCommand)

	return envCmd
}

func LoadConfig() Config {
	config := NewConfig()
	buf, err := ioutil.ReadFile(configPathName())
	if err != nil {
		return config
	}
	err = yaml.UnmarshalStrict(buf, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not unmarshall config: %s", err)
		return config
	}
	if config.Environments == nil {
		config.Environments = make(map[string]EnvMap)
	}
	if config.Alerts == nil {
		config.Alerts = make([]AlertConfig, 0)
	}
	if err := mergo.Merge(&config.Colors, defaultColorConfig); err != nil {
		panic("Could not set default colors")
	}
	return config
}

func configPathName() string {
	return fmt.Sprintf("%s/ax.yaml", dataDir)
}

func BuildConfig(activeEnv string, dockerFlag string) RuntimeConfig {
	config := LoadConfig()
	rc := RuntimeConfig{
		DataDir: dataDir,
		Env:     make(EnvMap),
		Config:  config,
	}
	var ok bool
	if config.DefaultEnv != "" {
		rc.Env, ok = config.Environments[config.DefaultEnv]
		if !ok {
			fmt.Println("Undefined active environment:", config.DefaultEnv)
			os.Exit(1)
		}
	}
	if activeEnv != "" {
		rc.Env, ok = config.Environments[activeEnv]
		rc.ActiveEnv = activeEnv
		if !ok {
			fmt.Println("Undefined active environment:", activeEnv)
			os.Exit(1)
		}
	}
	if dockerFlag != "" {
		rc.ActiveEnv = fmt.Sprintf("docker.%s", dockerFlag)
		rc.Env["backend"] = "docker"

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

func findFirstEnvWhere(environments map[string]EnvMap, whereFunc func(EnvMap) bool) *EnvMap {
	for _, m := range environments {
		if whereFunc(m) {
			return &m
		}
	}
	return nil
}

func SaveConfig(config Config) {
	f, err := os.Create(fmt.Sprintf("%s/ax.yaml", dataDir))
	if err != nil {
		fmt.Println("Couldn't open ax.yaml for writing", err)
		return
	}
	defer f.Close()
	buf, err := yaml.Marshal(&config)
	if err != nil {
		fmt.Println("Couldn't write to ax.yaml", err)
	}
	_, err = f.Write(buf)
	if err != nil {
		fmt.Println("Couldn't write to ax.yaml", err)
	}
}

func AddEnv() {
	config := LoadConfig()
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Name for new environment [default]: ")
	name := readLine(reader)
	if name == "" {
		name = "default"
	}
	fmt.Print("Choose a backend (kibana,cloudwatch,stackdriver) [kibana]: ")
	backend := readLine(reader)
	if backend == "" {
		backend = "kibana"
	}
	var em EnvMap
	var err error
	switch backend {
	case "kibana":
		em, err = kibanaConfig(reader, config)
		if err != nil {
			return
		}
	case "cloudwatch":
		em, err = cloudwatchConfig(reader, config)
		if err != nil {
			return
		}
	case "stackdriver":
		em, err = stackdriverConfig(reader, config)
		if err != nil {
			return
		}
	default:
		fmt.Println("Unsupported backend")
		return
	}
	if config.DefaultEnv == "" {
		config.DefaultEnv = name
	}
	config.Environments[name] = em
	SaveConfig(config)
}

func envHintAction() []string {
	config := LoadConfig()
	results := make([]string, 0, len(config.Environments))
	for k := range config.Environments {
		results = append(results, k)
	}
	return results
}
func ListEnvs() {
	config := LoadConfig()
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"D", "Name", "Backend", "Index"})
	for k, v := range config.Environments {
		def := ""
		if config.DefaultEnv == k {
			def = "*"
		}
		table.Append([]string{def, k, v["backend"], v["index"]})
	}
	table.Render() // Send output
}

func EditConfig() {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}
	cmd := exec.Command(editor, configPathName())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting editor", err)
		return
	}
	if err := cmd.Wait(); err != nil {
		fmt.Println("Error waiting for editor", err)
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
	logPath := fmt.Sprintf("%s/ax.log", dataDir)
	f, err := os.Create(logPath)
	if err != nil {
		fmt.Printf("Could not use %s for logging, logging to stdout instead\n", logPath)
		return // Skips log.SetOutput, defaults to stdout
	}
	log.SetOutput(f)
}
