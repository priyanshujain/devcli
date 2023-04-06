
# devcli

devcli is a tool to connect to remote hosts for development usecases.

## Features
1. Map staging apps to localhost 
2. Map staging databases and cache to localhost


## Use

```
devcli -conf config.yaml
```

## Install
```
go get github.com/okcredit/devcli
```

```
go install github.com/okcredit/devcli
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


3. Set auth credentials config for kubectl
```
gcloud components install gke-gcloud-auth-plugin
```

Add this to your profile:
```
export USE_GKE_GCLOUD_AUTH_PLUGIN=True
```

4. Set credentials for kubectl for given cluster
```
gcloud container clusters get-credentials --region=asia-south1 okc-cluster
```