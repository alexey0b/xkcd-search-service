# Variables
container_runtime := $(shell which docker || which podman)
minikube_runtime := $(shell which minikube)
kubectl_runtime := $(shell which kubectl)

$(info using ${container_runtime})

# Docker Compose Commands

.PHONY: build-images
build-images:
	${container_runtime} compose build

.PHONY: up
up: down
	${container_runtime} compose up --build -d

.PHONY: down
down:
	${container_runtime} compose down

.PHONY: clean
clean:
	${container_runtime} compose down -v

# Minikube Commands

.PHONY: k8s-init
k8s-init:
	${minikube_runtime} start --driver=docker --network-plugin=cni
	${minikube_runtime} addons enable ingress
	${minikube_runtime} addons enable metrics-server
	${minikube_runtime} addons enable dashboard

.PHONY: k8s-stop
k8s-stop:
	${minikube_runtime} stop

.PHONY: k8s-clean
k8s-clean:
	${minikube_runtime} delete

.PHONY: k8s-load-images
k8s-load-images: build-images
	${minikube_runtime} image load \
		frontend:latest \
		api:latest \
		search:latest \
		update:latest \
		words:latest \

.PHONY: k8s-start
k8s-start: k8s-load-images
	${kubectl_runtime} apply -f k8s/namespace.yaml
	${kubectl_runtime} apply -f k8s/ -R

.PHONY: k8s-delete
k8s-delete:
	${kubectl_runtime} delete -f k8s/ -R --ignore-not-found=true

.PHONY: k8s-restart
k8s-restart: k8s-delete k8s-start

.PHONY: k8s-port-forward
k8s-port-forward:
	@echo "Access: http://kubernetes.docker.internal:3000"
	${kubectl_runtime} port-forward -n ingress-nginx svc/ingress-nginx-controller 3000:80

.PHONY: k8s-dashboard
k8s-dashboard:
	${minikube_runtime} dashboard

# Testing Commands

.PHONY: unit-tests
unit-tests:
	make -C search-services unit-tests

.PHONY: integration-tests
integration-tests:
	make clean
	make up
	@echo wait cluster to start && sleep 10
	${container_runtime} run --rm --network=host tests:latest || (make clean && exit 1)
	make clean
	@echo "test finished"

# Code Quality Commands

.PHONY: lint
lint:
	make -C search-services lint

.PHONY: cover
cover:
	make -C search-services cover 
	mv search-services/cover.out .
	mv search-services/cover.html .
	
.PHONY: security
security:
	make -C search-services security

.PHONY: proto
proto:
	make -C search-services protobuf

# Development Tools Installation

.PHONY: tools
tools:
	go install github.com/yoheimuta/protolint/cmd/protolint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.7.2
	@echo "checking protobuf compiler, if it fails follow guide at https://protobuf.dev/installation/"
	@command -v protoc || (echo "protoc not found" && exit 1)
