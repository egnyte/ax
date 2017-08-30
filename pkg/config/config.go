package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/yaml.v2"

	"github.com/olekukonko/tablewriter"
	"github.com/zefhemel/ax/pkg/backend/docker"
	"github.com/zefhemel/ax/pkg/backend/kibana"
)

var dataDir string

type EnvMap map[string]string

type Config struct {
	DefaultEnv   string            `yaml:"default"`
	Environments map[string]EnvMap `yaml:"env"`
}

type RuntimeConfig struct {
	ActiveEnv string
	DataDir   string
	Env       EnvMap
}

var (
	activeEnv      = kingpin.Flag("env", "Environment to connect to").Short('e').String()
	dockerFlag     = kingpin.Flag("docker", "Query docker container logs").HintAction(docker.DockerHintAction).String()
	envCommand     = kingpin.Command("env", "Environment management commands")
	envInitCommand = envCommand.Command("add", "Add an environment")
	envListCommand = envCommand.Command("list", "List all environments").Default()
)

func NewConfig() Config {
	return Config{
		Environments: make(map[string]EnvMap),
	}
}

func loadConfig() Config {
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
	return config
}

func configPathName() string {
	return fmt.Sprintf("%s/ax.yaml", dataDir)
}

func BuildConfig() RuntimeConfig {
	config := loadConfig()
	rc := RuntimeConfig{
		DataDir: dataDir,
		Env:     make(EnvMap),
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

func kibanaConfig(reader *bufio.Reader) (EnvMap, error) {
	em := EnvMap{
		"backend": "kibana",
	}
	fmt.Print("URL: ")
	em["url"] = readLine(reader)
	fmt.Println("Attempting to connect to Kibana on ", em["url"])
	resp, err := http.Head(em["url"])
	if err != nil {
		fmt.Printf("Got error connecting to Kibana: %s\n", err)
		return em, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		user, pass := credentials(reader)
		em["auth"] = fmt.Sprintf("Basic %s", b64Encode(fmt.Sprintf("%s:%s", user, pass)))
		fmt.Println("Checking...")
	}
	kibanaClient := kibana.New(em["url"], em["auth"], "")
	indices, err := kibanaClient.ListIndices()
	if err != nil {
		fmt.Println("Could not connect to Kibana to get list of indices successfullly")
		return EnvMap{}, nil
	}
	fmt.Println("List of indices:")
	for _, index := range indices {
		fmt.Println("  ", index)
	}
	fmt.Print("Index: ")
	em["index"] = readLine(reader)
	return em, nil
}

func saveConfig(config Config) {
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
	config := loadConfig()
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Name for new environment: ")
	name := readLine(reader)
	fmt.Print("Choose a backend [kibana]: ")
	backend := readLine(reader)
	var em EnvMap
	var err error
	switch backend {
	case "kibana":
		em, err = kibanaConfig(reader)
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
	saveConfig(config)
}

func ListEnvs() {
	config := loadConfig()
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
