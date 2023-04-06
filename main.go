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

	"gopkg.in/yaml.v2"
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

type Config struct {
	Kubeconfig string  `yaml:"kubeconfig"`
	Bastion    Bastion `yaml:"bastion"`
	Rules      []Rule  `yaml:"rules"`
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

func main() {
	// Parse command line arguments
	confFile := flag.String("conf", "", "Path to the configuration file")
	flag.Parse()

	if *confFile == "" {
		flag.Usage()
		os.Exit(1)
	}

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

	// Set the KUBECONFIG environment variable
	os.Setenv("KUBECONFIG", config.Kubeconfig)

	// Check if there are duplicate local ports
	if checkDuplicateLocalPorts(config) {
		fmt.Println("Error: there are duplicate local ports in the configuration file.")
		os.Exit(1)
	}

	// Listen for SIGINT and SIGTERM signals
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Cancel the context when the program is interrupted
	go func() {
		<-ch
		fmt.Println("Interrupted. Exiting...")
		// Cancel the context
		cancel()
		<-ch
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
		fmt.Printf("Connecting to remote host %s via bastion server %s\n", connection.RemoteHost, config.Bastion.Name)
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
