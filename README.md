
# devcli

devcli implements a command-line interface for the development of Kubernetes applications.

It provides a simple way to forward local ports to remote ports on Kubernetes pods.
It also provides a way to forward local ports to remote ports on a bastion server
that is used to connect to pods. The configuration file contains the name of the
bastion server, the name of the zone where the bastion server is located, the
connections that will be forwarded through the bastion server, and the rules for
port forwarding.

## Features
1. Map staging apps to localhost 
2. Map staging databases and cache to localhost


## Use

```
devcli -conf config.yaml
```

## Install

```
go install github.com/okcredit/devcli@latest
```

## Configure

1. Copy `config-template.yaml` to `config.yaml` and add required mapping details


## Prerequisites

1. Install gcloud cli tool (google cloud sdk)

For mac users
- You can use brew to install gcloud (https://formulae.brew.sh/cask/google-cloud-sdk)

```
brew install --cask google-cloud-sdk
```

```
gcloud init
```

Run 

```
brew info google-cloud-sdk
```

To add gcloud components to your PATH, add this to your profile:

  for bash users
    source "$(brew --prefix)/share/google-cloud-sdk/path.bash.inc"

  for zsh users
    source "$(brew --prefix)/share/google-cloud-sdk/path.zsh.inc"

  for fish users
    source "$(brew --prefix)/share/google-cloud-sdk/path.fish.inc"

Example: 
```
source "/opt/homebrew/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/path.zsh.inc"
```


2. Install kubectl 

```
gcloud components install kubectl
```


3. Install gcloud auth credentials plugin for gke
```
gcloud components install gke-gcloud-auth-plugin
```
