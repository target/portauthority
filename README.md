[![Build Status]()]()

## Introduction

Port Authority is an API service that uses the fantastic vulnerability static analysis tool [CoreOS Clair](https://github.com/coreos/clair) to provide component based vulnerability assessments of Docker images and assists in the auditing and governance of container deployments from build-time (Docker registries) to run-time (Kubernetes).

The Port Authority API allows for the scanning of public or private individual images or entire private Docker registries like ([Docker Hub](https://hub.docker.com), [Google Container Registry](https://cloud.google.com/container-registry/) and [Artifactory](https://jfrog.com/artifactory/)).

It has close integrations with Kubernetes that will help discover running containers and scan their source images. For enforcement, it provides a webhook that when leveraged by a Kubernetes admission controller will allow or deny deployments based on customizable policies.

## Getting Started

### Setup and Start Minikube
1. Install [Minikube](https://github.com/kubernetes/minikube)
2. Start Minikube:

   `minikube start`

**NOTE:** Supported Kubernetes versions (1.6.x - 1.9.x). Suported Clair versions v2.x.x.

### Build and Deploy to Minikube
1. Use Minikube Docker:

   `eval $(minikube docker-env)`

2. Get all Glide dependancies:

   `make deps`

3. Build & deploy:

   `make deploy-minikube`

## Optional Configuration
Different configuration adjustments can be made to the port authority deployment here: [minikube/portauthority/portauthority/config.yml](minikube/portauthority/portauthority/config.yml)

:white_check_mark: Add Docker Credentials used by the K8s Crawler scan feature

```
### Environment variables defined below are mapped to credentials used by the Kubneretes Crawler API (/v1/crawler/k8s)
### A 'Scan: true' flag will invoke their usage
k8scrawlcredentials:
  # Use "" for basic auth on registries that do not require a username and password
  - url: "docker.io" #basic auth is empty UN and PW
    username: "DOCKER_USER"
    password: "DOCKER_PASS"
  - url: "gcr.io" #basic auth is empty UN and PW
    username: "GCR_USER"
    password: "GCR_PASS"
```

:white_check_mark: Enable the [Kubernetes Admission Controler](docs/webhook-example/README.md) and change webhooks default behavior
```
# Setting imagewebhookdefaultblock to true will set the imagewebhooks endpoint default behavior to block any images with policy violations.
# If it is set to false a user can change enable the behavior by setting the portauthority-webhook deployment annotation to true
imagewebhookdefaultblock: false
```


## Docs

Port Authority is an API service.  See our complete [_API Documentation_](docs/README.md) for further configuration, usage, Postman collections and more.

## Contributing

We always welcome new PRs! See [_Contributing_](CONTRIBUTING.md) for further instructions.

## Bugs and Feature Requests

Found something that doesn't seem right or have a feature request? [Please open a new issue](issues/new/).

## Copyright and License

[![license](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE.txt)

&copy;2018 Target Brands, Inc.
