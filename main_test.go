package main

import (
	"context"
	"os/exec"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestCheckKubectl(t *testing.T) {
	// context
	ctx := context.Background()
	result := checkKubectl(ctx)
	if !result {
		t.Error("checkKubectl failed: kubectl is not installed or not in the system's PATH.")
	}
}

func TestCheckDuplicateLocalPorts(t *testing.T) {
	configData := `
kubeconfig: /path/to/your/kubeconfig.yaml
bastion:
  name: bastion
  connections:
    - local_port: 5434
      remote_host: 10.116.48.59
      remote_port: 5432
    - local_port: 5435
      remote_host: 10.116.48.60
      remote_port: 5432
rules:
  - namespace: your-namespace-1
    pod_name: your-pod-name-1
    local_port: 8080
    remote_port: 80
  - namespace: your-namespace-2
    pod_name: your-pod-name-2
    local_port: 8081
    remote_port: 80
`

	var config Config
	err := yaml.Unmarshal([]byte(configData), &config)
	if err != nil {
		t.Fatalf("Error parsing configuration data for TestCheckDuplicateLocalPorts: %v", err)
	}

	if checkDuplicateLocalPorts(config) {
		t.Error("checkDuplicateLocalPorts failed: found duplicate local ports.")
	}
}

func TestConnectBastion(t *testing.T) {
	bastion := Bastion{Name: "bastion"}
	connection := Connection{
		LocalPort:  5434,
		RemoteHost: "10.116.48.59",
		RemotePort: 5432,
	}

	// context
	ctx := context.Background()

	cmd := connectBastion(ctx, bastion, connection)
	// get gcloud path
	gcloudPath, err := exec.LookPath("gcloud")
	if err != nil {
		t.Fatalf("Error getting gcloud path: %v", err)
	}
	if cmd == nil || cmd.Path != gcloudPath {
		t.Error("connectBastion failed: the command is not properly configured.")
	}
}
