[![Build Status](https://travis-ci.org/target/portauthority.svg?branch=master)](https://travis-ci.org/target/portauthority/builds)

## Introduction

Port Authority is an API service that delivers component based vulnerability assessments for Docker images at time of build and in run-time environments. Port Authority also provides Developers additional customizable offerings to assist with the automated audit and governance of their containers.

The Port Authority API is capable of scanning public or private individual images as well as entire private Docker registries like [Docker Hub](https://hub.docker.com), [Google Container Registry](https://cloud.google.com/container-registry/) or [Artifactory](https://jfrog.com/artifactory/). Port Authority integrates with Kubernetes to continuously discover running containers and inventory those deployed images for scanning.

In the backend Port Authority utilizes the open source static analysis tool [Clair](https://github.com/coreos/clair) by CoreOS to scan images and identify vulnerabilities. For enforcement, Port Authority provides a webhook that when leveraged by a [Kubernetes](https://github.com/kubernetes/kubernetes) admission controller will allow or deny deployments based on customizable policies.


## Getting Started <img align="right" width="300" src="imgs/ahab-small.png">

### Setup and Start Minikube
1. Install [Minikube](https://github.com/kubernetes/minikube)
2. Start Minikube:

   `minikube start`

**NOTE:** Supported Kubernetes versions (1.6.x - 1.9.x). Supported Clair versions v2.x.x.

### Build and Deploy to Minikube
1. Use Minikube Docker:

   `eval $(minikube docker-env)`

2. Deploy official Port Authority stack:

   `make deploy-minikube`

(Optional). Local developer build stack:

1. Use Minikube Docker:

   `eval $(minikube docker-env)`

2. Get all Glide dependancies:

   `make deps`

3. Deploy official Port Authority stack:

   `make deploy-minikube-dev`

## Optional Configuration
Different configuration adjustments can be made to the Port Authority deployment here: [minikube/portauthority/portauthority/config.yml](minikube/portauthority/portauthority/config.yml)

:white_check_mark: Add Docker Credentials used by the K8s Crawler scan feature

```
### Environment variables defined below are mapped to credentials used by the Kubernetes Crawler API (/v1/crawler/k8s)
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

:white_check_mark: Enable the [Kubernetes Admission Controller](docs/webhook-example/README.md) and change webhooks default behavior
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

**Credit [Renee French](http://reneefrench.blogspot.com/) for original golang gopher
