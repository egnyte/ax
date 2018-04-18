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

	"github.com/zefhemel/kingpin"
	yaml "gopkg.in/yaml.v2"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/docker"
	"github.com/egnyte/ax/pkg/backend/kibana"
	"github.com/olekukonko/tablewriter"
)

var dataDir string

type EnvMap map[string]string

type Config struct {
	DefaultEnv   string            `yaml:"default"`
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
	activeEnv      = kingpin.Flag("env", "Environment to connect to").Short('e').HintAction(envHintAction).String()
	dockerFlag     = kingpin.Flag("docker", "Query docker container logs").HintAction(docker.DockerHintAction).String()
	fileFlag       = kingpin.Flag("file", "Query logs in a file").String()
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
	return config
}

func configPathName() string {
	return fmt.Sprintf("%s/ax.yaml", dataDir)
}

func BuildConfig() RuntimeConfig {
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
	if *fileFlag != "" {
		rc.ActiveEnv = fmt.Sprintf("file.%s", *fileFlag)
		rc.Env["backend"] = "file"
		rc.Env["filename"] = *fileFlag
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

func testKibana(em EnvMap) bool {
	kibanaClient := kibana.New(em["url"], em["auth"], "")
	_, err := kibanaClient.ListIndices()
	return err != nil
}

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
	fmt.Print("Choose a backend [kibana]: ")
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
	for k, _ := range config.Environments {
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
