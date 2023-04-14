package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"gopkg.in/yaml.v3"
)

type Connection struct {
	LocalPort  int    `yaml:"local_port"`
	RemoteHost string `yaml:"remote_host"`
	RemotePort int    `yaml:"remote_port"`
}

type Bastion struct {
	Name        string       `yaml:"name"`
	Zone        string       `yaml:"zone"`
	Connections []Connection `yaml:"connections"`
}

type Rule struct {
	Namespace  string `yaml:"namespace"`
	App        string `yaml:"app"`
	LocalPort  int    `yaml:"local_port"`
	RemotePort int    `yaml:"remote_port"`
}

type GCloudConfig struct {
	Project    string `yaml:"project"`
	ConfigPath string `yaml:"config_path"`
}

type Config struct {
	Kubeconfig  string       `yaml:"kubeconfig"`
	GCloud      GCloudConfig `yaml:"gcloud"`
	Bastion     Bastion      `yaml:"bastion"`
	Rules       []Rule       `yaml:"rules"`
	Environment string       `yaml:"environment"`
}

func checkKubectl(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "kubectl", "version", "--client")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func checkDuplicateLocalPorts(config Config) bool {
	localPorts := make(map[int]bool)

	for _, rule := range config.Rules {
		if localPorts[rule.LocalPort] {
			return true
		}
		localPorts[rule.LocalPort] = true
	}

	for _, connection := range config.Bastion.Connections {
		if localPorts[connection.LocalPort] {
			return true
		}
		localPorts[connection.LocalPort] = true
	}

	return false
}

func connectBastion(ctx context.Context, bastion Bastion, connection Connection) *exec.Cmd {
	sshCmd := exec.CommandContext(ctx, "gcloud", "compute", "ssh", bastion.Name, "--zone", bastion.Zone, "--", "-L", fmt.Sprintf("localhost:%d:%s:%d", connection.LocalPort, connection.RemoteHost, connection.RemotePort), "-t")
	sshCmd.Stderr = os.Stderr
	return sshCmd
}

// checkPortAvailable checks if the port on local machine is available
func checkPortAvailable(port int) bool {
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	if err := cmd.Run(); err != nil {
		return true
	}
	return false
}

func main() {
	// Parse command line arguments
	confFile := flag.String("conf", "", "Path to the configuration file")
	flag.Parse()

	if *confFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Read environment type from cmd line argument
	environment := flag.String("env", "", "Environment type (dev, staging, prod)")
	flag.Parse()

	// Create a context that will be used to cancel the port-forward commands
	// when the program is interrupted
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Check if kubectl is installed and configured
	if !checkKubectl(ctx) {
		fmt.Println("Error: kubectl is not installed or not in the system's PATH.")
		os.Exit(1)
	}

	// Read and parse the configuration file
	configData, err := os.ReadFile(*confFile)
	if err != nil {
		fmt.Println("Error reading configuration file:", err)
		os.Exit(1)
	}

	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		fmt.Println("Error parsing configuration file:", err)
		os.Exit(1)
	}

	// check if environment is set
	if config.Environment == "" && *environment == "" {
		fmt.Println("Error: environment is not set in the configuration file or passed as a command line argument.")
		os.Exit(1)
	} else if *environment != "" {
		config.Environment = *environment
	}

	// get zone of the bastion instance using gcloud
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "instances", "describe", config.Bastion.Name, "--format", "value(zone)")
	cmd.Stderr = os.Stderr
	zone, err := cmd.Output()
	if err != nil {
		fmt.Println("Error getting zone of the bastion instance:", err)
		os.Exit(1)
	} else {
		config.Bastion.Zone = string(zone)
	}

	// Set the KUBECONFIG environment variable
	if config.Kubeconfig == "" {
		fmt.Println("kubeconfig is not set in the configuration file.")
		// get default kubeconfig path from home directory
		fmt.Println("Using default kubeconfig path: $HOME/.kube/config")
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error getting home directory:", err)
			os.Exit(1)
		}
		config.Kubeconfig = fmt.Sprintf("%s/.kube/config", home)
	}
	os.Setenv("KUBECONFIG", config.Kubeconfig)

	gcloudProjectName := config.GCloud.Project
	gcloudConfigPath := config.GCloud.ConfigPath

	// Set the CLOUDSDK_CONFIG environment variable
	if gcloudConfigPath == "" {
		fmt.Println("gcloud config path is not set in the configuration file.")
		// get default gcloud config path from home directory
		fmt.Println("Using default gcloud config path: $HOME/.config/gcloud")
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error getting home directory:", err)
			os.Exit(1)
		}
		gcloudConfigPath = fmt.Sprintf("%s/.config/gcloud", home)
	}
	os.Setenv("CLOUDSDK_CONFIG", gcloudConfigPath)

	// check if the project is set
	if gcloudProjectName == "" {
		fmt.Println("Error: project is not set in the configuration file.")
		os.Exit(1)
	}

	// Check if there are duplicate local ports
	if checkDuplicateLocalPorts(config) {
		fmt.Println("Error: there are duplicate local ports in the configuration file.")
		os.Exit(1)
	}

	// check if the port on local machine is available
	for _, rule := range config.Rules {
		if !checkPortAvailable(rule.LocalPort) {
			fmt.Printf("Error: port %d is not available on local machine.\n", rule.LocalPort)
			os.Exit(1)
		}
	}

	// set gcloud project
	cmd = exec.CommandContext(ctx, "gcloud", "config", "set", "project", gcloudProjectName)
	if err := cmd.Run(); err != nil {
		fmt.Println("Error setting gcloud project:", err)
		os.Exit(1)
	}

	// get cluster list and set the first cluster as the default cluster
	var defaultClusterName string
	cmd = exec.CommandContext(ctx, "gcloud", "container", "clusters", "list", "--format", "value(name)")
	if out, err := cmd.Output(); err != nil {
		fmt.Println("Error getting cluster list:", err)
		os.Exit(1)
	} else {
		defaultClusterName = string(out)
		cmd = exec.CommandContext(ctx, "gcloud", "config", "set", "container/cluster", defaultClusterName)
		if err := cmd.Run(); err != nil {
			fmt.Println("Error setting gcloud cluster:", err)
			os.Exit(1)
		}
	}

	// get cluster region
	var defaultClusterRegion string
	cmd = exec.CommandContext(ctx, "gcloud", "container", "clusters", "list", "--format", "value(location)")
	if out, err := cmd.Output(); err != nil {
		fmt.Println("Error getting cluster region:", err)
		os.Exit(1)
	} else {
		defaultClusterRegion = string(out)
		cmd = exec.CommandContext(ctx, "gcloud", "config", "set", "compute/region", defaultClusterRegion)
		if err := cmd.Run(); err != nil {
			fmt.Println("Error setting gcloud region:", err)
			os.Exit(1)
		}
	}

	// get credentials for the default cluster
	cmd = exec.CommandContext(ctx, "gcloud", "container", "clusters", "get-credentials", defaultClusterName)
	if err := cmd.Run(); err != nil {
		fmt.Println("Error getting cluster credentials:", err)
		os.Exit(1)
	}

	// Listen for SIGINT and SIGTERM signals
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Cancel the context when the program is interrupted
	go func() {
		<-ch
		fmt.Println("Interrupted. Exiting gracefully...")
		// Cancel the context
		cancel()
		<-ch
		fmt.Println("Interrupted again. Exiting immediately...")
		os.Exit(1)
	}()

	// Run the kubectl port-forward command for each rule
	var wg sync.WaitGroup
	for _, rule := range config.Rules {
		wg.Add(1)
		go func(rule Rule) {
			defer wg.Done()

			// get first pod using workload name
			cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", rule.Namespace, "-l", fmt.Sprintf("app=%s", rule.App), "-o", "jsonpath={.items[0].metadata.name}")
			if out, err := cmd.Output(); err != nil {
				fmt.Printf("Error getting pod name for app %s: %v\n", rule.App, err)
			} else {
				podName := string(out)
				cmd = exec.CommandContext(ctx, "kubectl", "port-forward", fmt.Sprintf("--namespace=%s", rule.Namespace), podName, fmt.Sprintf("%d:%d", rule.LocalPort, rule.RemotePort))
				cmd.Stderr = os.Stderr
				fmt.Printf("Running kubectl port-forward for pod %s\n", podName)
				if err := cmd.Run(); err != nil {
					// If the context was canceled, don't print an error
					if ctx.Err() != nil {
						return
					}
					fmt.Printf("Error running kubectl port-forward for pod %s: %v\n", podName, err)
				}
			}
		}(rule)
	}

	// Connect to the bastion server and forward the connections
	for _, connection := range config.Bastion.Connections {
		cmd := connectBastion(ctx, config.Bastion, connection)
		fmt.Printf("Connecting to remote host %s via bastion server on port %d\n", connection.RemoteHost, connection.LocalPort)
		go func(connection Connection) {
			if err := cmd.Run(); err != nil {
				// If the context was canceled, don't print an error
				if ctx.Err() != nil {
					return
				}
				fmt.Printf("Error connecting to the remote host %s via bastion server %s: %v\n", connection.RemoteHost, config.Bastion.Name, err)
			}
		}(connection)
	}
	wg.Wait()
}
