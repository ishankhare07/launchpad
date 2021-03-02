SHELL := /bin/bash
ISTIO_VERSION ?= 1.8.2
ISTIO_PATH ?= ${PWD}/istio-${ISTIO_VERSION}
ISTIOCTL ?= ${ISTIO_PATH}/bin/istioctl

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

download-istio:
	@if [ -d ${ISTIO_PATH} ]; then\
		exit;\
	else \
		curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${ISTIO_VERSION} sh -; \
		eval ${ISTIOCTL} version; \
	fi
setup-istio:
	eval ${ISTIOCTL} install --set profile=demo -y
	kubectl label namespace default istio-injection=enabled
cleanup-istio:
	eval ${ISTIOCTL} manifest generate --set profile=demo | kubectl delete --ignore-not-found=true -f -
	eval ${ISTIOCTL} analyze
install-kind:
	-brew install kind
create-cluster: install-kind
	kind create cluster --config kind.yaml
cleanup-cluster:
	kind delete cluster --name launchpad-test
setup-cluster: create-cluster download-istio setup-istio
	kubectl apply -f ${ISTIO_PATH}/samples/helloworld/helloworld.yaml
	kubectl run nginx --image=nginx
setup-traffic-split:
	kubectl apply -f dest-rule.yaml
	kubectl apply -f vs.yaml
execute-requests:
	kubectl exec -it nginx -- bash -c  "for x in {1..10};do curl helloworld:5000/hello; done"
clean:
	kind delete cluster --name launchpad-test
