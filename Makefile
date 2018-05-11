## Go parameters ##
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=portauthority
BINARY_MAC=$(BINARY_NAME)_mac

## Actions ##
all: clean deps build-mac build-linux

clean:
				$(GOCLEAN)
				rm -f $(BINARY_NAME)
				rm -f $(BINARY_MAC)
run-mac: clean build-mac
				./$(BINARY_MAC)
deps:	| glide
				@echo "Installing dependencies"
				@glide install -v

## File Targets ##
deploy-minikube: clean-minikube
				@echo "Deploying officially built Port Authority"
				@echo "Applying Clair postgres deployment files"
				kubectl apply -f ./minikube/clair/postgres
				kubectl rollout status deployment/clair-postgres-deployment
				sleep 5
				@echo "Applying Clair deployment files"
				kubectl apply -f ./minikube/clair/clair
				kubectl rollout status deployment/clair-deployment
				@echo "Applying portauthority postgres deployment files"
				kubectl apply -f ./minikube/portauthority/postgres
				kubectl rollout status deployment/portauthority-postgres-deployment
				sleep 5
				@echo "Applying portauthority deployment files"
				kubectl apply -f ./minikube/portauthority/portauthority
				kubectl rollout status deployment/portauthority-deployment

## File Targets ##
deploy-minikube-dev: clean docker-build clean-minikube
				@echo "Deploying locally built devloper build of Port Authority"
				@echo "Applying Clair postgres deployment files"
				kubectl apply -f ./minikube/clair/postgres
				kubectl rollout status deployment/clair-postgres-deployment
				sleep 5
				@echo "Applying Clair deployment files"
				kubectl apply -f ./minikube/clair/clair
				kubectl rollout status deployment/clair-deployment
				@echo "Applying portauthority postgres deployment files"
				kubectl apply -f ./minikube/portauthority/postgres
				kubectl rollout status deployment/portauthority-postgres-deployment
				sleep 5
				@echo "Applying portauthority deployment files"
				kubectl apply -f ./minikube/portauthority/portauthority-local
				kubectl rollout status deployment/portauthority-deployment

clean-minikube:
				@echo "Cleaning up previous portauthority deployments (postgres will remain)"
				kubectl delete -f ./minikube/portauthority/portauthority --ignore-not-found
				kubectl delete -f ./minikube/portauthority/portauthority-local --ignore-not-found

clean-minikube-postgres:
				@echo "Cleaning Clair postgres database"
				kubectl delete -f ./minikube/clair/postgres --ignore-not-found
				@echo "Cleaning portauthority postgres database"
				kubectl delete -f ./minikube/portauthority/postgres --ignore-not-found

## Builds ##
build-mac:
				$(GOBUILD) -o $(BINARY_MAC)
build-linux:
				CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)
docker-build:
				docker build -t $(BINARY_NAME) .

## Glide ##
glide:
	@if ! hash glide 2>/dev/null; then curl https://glide.sh/get | sh; fi
