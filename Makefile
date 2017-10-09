#
# Makefile
#
# Phone Channel 
# TODO: Need some sort of generic makefile for go services
#

REPO=agent-mgmt
# Repository directory inside docker container
REPO_DIR=/go/src/github.com/newtonsystems/agent-mgmt
# Filename of k8s deployment file inside 'local' devops folder
LOCAL_DEPLOYMENT_FILENAME=agent-mgmt-deployment.yml

NEWTON_DIR=/Users/danvir/Masterbox/sideprojects/github/newtonsystems/
CURRENT_BRANCH=`git rev-parse --abbrev-ref HEAD`
CURRENT_RELEASE_VERSION=0.0.1

TIMESTAMP=tmp-$(shell date +%s )

#
# Help for Makefile & Colorised Messages
#
# Powered by https://gist.github.com/prwhite/8168133
GREEN  := $(shell tput -Txterm setaf 2)
RED    := $(shell tput -Txterm setaf 1)
BLUE   := $(shell tput -Txterm setaf 4)
WHITE  := $(shell tput -Txterm setaf 7)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)

INFO=$(GREEN)[INFO] $(RESET)
STAGE=$(BLUE)[INFO] $(RESET)
ERROR=$(RED)[ERROR] $(RESET)
WARN=$(YELLOW)[WARN] $(RESET)

#
# Help Command
#

# Add help text after each target name starting with '\#\#'
# A category can be added with @category
HELP_FUN = \
    %help; \
    while(<>) { push @{$$help{$$2 // 'options'}}, [$$1, $$3] if /^([a-zA-Z\-]+)\s*:.*\#\#(?:@([a-zA-Z\-]+))?\s(.*)$$/ }; \
    print "usage: make [target]\n\n"; \
    for (sort keys %help) { \
    print "${WHITE}$$_:${RESET}\n"; \
    for (@{$$help{$$_}}) { \
    $$sep = " " x (32 - length $$_->[0]); \
    print "  ${YELLOW}$$_->[0]${RESET}$$sep${GREEN}$$_->[1]${RESET}\n"; \
    }; \
    print "\n"; }

.PHONY: help

help:                        ##@other Show this help.
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)


#
# Compile + Go Dependencies Commands
#
.PHONY: compile update-deps-featuretest update-deps-master install-deps-featuretest install-deps-master add-deps-master add-deps-featuretest

compile:
	@echo "$(INFO) Getting packages and building alpine go binary ..."
	@if [ "$(CURRENT_BRANCH)" != "master" && "$(CURRENT_BRANCH)" != "featuretest" ]; then \
		echo "$(INFO) for branch master " \
		make update-deps-master; \
		make install-deps-master; \
	else \
		echo "$(INFO) for branch $(CURRENT_BRANCH) " \
		make update-deps-$(CURRENT_BRANCH); \
		make install-deps-$(CURRENT_BRANCH); \
	fi
	make build-command


build-command:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./app/main.go

update-deps-featuretest:
	@echo "$(INFO) Updating dependencies for featuretest environment"
	cp featuretest.lock glide.lock
	glide -y featuretest.yaml update --force
	cp glide.lock featuretest.lock

update-deps-master:
	@echo "$(INFO) Updating dependencies for $(BLUE)master$(RESET) environment"
	cp master.lock glide.lock
	glide -y master.yaml update --force
	cp glide.lock master.lock

install-deps-featuretest:
	@echo "$(INFO) Installing dependencies for featuretest environment"
	cp featuretest.lock glide.lock
	glide -y featuretest.yaml install
	cp glide.lock featuretest.lock

install-deps-master:
	@echo "$(INFO) Installing dependencies for $(BLUE)master$(RESET) environment"
	cp master.lock glide.lock
	glide -y master.yaml install
	cp glide.lock master.lock


#
# Main (Build binary)
#
.PHONY: build-bin

# TODO: Should speed this up with voluming vendor/
build-bin:              ##@build Cross compile the go binary executable
	@echo "$(INFO) Building a linux-alpine Go binary locally with a docker container $(BLUE)$(REPO):compile$(RESET)"
	docker build -t $(REPO):compile -f Dockerfile.build .
	docker run --rm -v "${PWD}":$(REPO_DIR) $(REPO):compile
	@echo ""

#
# Run Commands
#
.PHONY: build run

build: build-bin
	docker build -t $(REPO):local .

run: build              ##@dev Build and run service locally
	@echo "$(INFO) Running docker container with tag: $(REPO):local"
	@echo "$(BLUE)"
	@echo "$(INFO) Building docker container locally with tag: $(BLUE)$(REPO):local$(RESET)"

	docker run -it $(REPO):local
	@echo "$(NO_COLOR)"

#
# Development (Hot-Reloaded) Commands
#
.PHONY: build-dev run-dev

build-dev:
	docker build -t $(REPO):dev -f Dockerfile.dev .

run-dev: build-dev    ##@dev Build and run (hot-reload) development docker container (few extra packages for debugging containers) (Normally run this for dev changes)
	@echo ""
	docker run -v "${PWD}":$(REPO_DIR) -p 50000:50000  -it $(REPO):dev

#
# Hot Reload Development
#
.PHONY: serve restart kill before

PID      = /tmp/$(REPO).pid
GO_FILES = $(wildcard app/*.go)
APP_DIR = app/
APP      = ./main
APP_MAIN = app/main.go



kill:
	@if [ -f $(PID) ]; then \
		echo "$(INFO) Killing application: $(PID) " \
		kill `cat $(PID)` || true; \
		ps aux;pgrep main; \
		kill -9 pgrep main || true; \
	fi


restart: kill build-command
	@./main & echo $$! > $(PID)

restart-fast: kill
	@go run $(APP_MAIN) & echo $$! > $(PID)

serve: restart
	@inotifywait -r -m . -e create -e modify | \
		while read path action file; do \
			echo "$(INFO) '$$file' has changed from dir '$$path' via '$$action'"; \
			make restart; \
		done

serve-fast: restart-fast
	@inotifywait -r -m $(GO_FILES) -e create -e modify | \
		while read path action file; do \
			echo "$(INFO) '$$file' has changed from dir '$$path' via '$$action'"; \
			make restart-fast; \
		done




#
# Run Commands (Black Box)
#
.PHONY: run-latest-release run-latest

run-latest-release:     ##@run-black-box Run the current release (When you want to run as service as a black-box)
	@echo "$(INFO) Pulling release docker image for branch: newtonsystems/$(REPO):$(CURRENT_RELEASE_VERSION)"
	@echo "$(BLUE)"
	docker pull newtonsystems/$(REPO):$(CURRENT_RELEASE_VERSION);
	docker run newtonsystems/$(REPO):$(CURRENT_RELEASE_VERSION);
	@echo "$(NO_COLOR)"

run-latest:             ##@run-black-box Run the most up-to-date image for your branch from the docker registry or if the image doesnt exist yet you can specify. (When you want to run as service as a black-box)
	@echo "$(INFO) Running the most up-to-date image"
	@echo "$(INFO) Pulling latest docker image for branch: newtonsystems/$(REPO):$(CURRENT_BRANCH)"

	@docker pull newtonsystems/$(REPO):$(CURRENT_BRANCH); if [ $$? -ne 0 ] ; then \
		echo "$(ERROR) Failed to find image in registry: newtonsystems/$(REPO):$(CURRENT_BRANCH)"; \
		read -r -p "$(GREEN) Specific your own image name or Ctrl+C to exit:$(RESET)   " reply; \
		docker pull newtonsystems/$(REPO):$$reply; \
		docker run newtonsystems/$(REPO):$$reply; \
	else \
		docker run newtonsystems/$(REPO):$(CURRENT_BRANCH) app; \
	fi


#
# minikube
#
.PHONY: mkube-update mkube-run-dev

mkube-update: build-bin      ##@kube Updates service in minikube
	@echo "$(INFO) Deploying $(REPO):$(TIMESTAMP) by replacing image in kubernetes deployment config"
	# TODO: add cluster check  - i.e. is minikube pointed at
	@eval $$(minikube docker-env); docker image build -t newtonsystems/$(REPO):$(TIMESTAMP) .
	kubectl set image -f $(NEWTON_DIR)/devops/k8s/deploy/local/$(LOCAL_DEPLOYMENT_FILENAME) $(REPO)=newtonsystems/$(REPO):$(TIMESTAMP)

mkube-run-dev:               ##@kube Run service in minikube (hot-reload)
	@echo "$(INFO) Running $(REPO):kube-dev (Dev in Minikube) by replacing image in kubernetes deployment config"
	@eval $$(minikube docker-env); docker image build -t newtonsystems/$(REPO):kube-dev -f Dockerfile.dev .
	kubectl replace -f $(NEWTON_DIR)/devops/k8s/deploy/local/$(LOCAL_DEPLOYMENT_FILENAME)
	kubectl set image -f $(NEWTON_DIR)/devops/k8s/deploy/local/$(LOCAL_DEPLOYMENT_FILENAME) $(REPO)=newtonsystems/$(REPO):kube-dev
	make update-deps-master
	make install-deps-master
	@echo "$(INFO) Hooking to logs in minikube ..."
	@kubectl logs -f `kubectl get pods -o wide | grep $(REPO) | grep Running | cut -d ' ' -f1` &
	# Add a liveness probe instead of sleep
	@fswatch $(APP_DIR) | while read; do \
			echo "$(INFO) Detected a change, deleting a pod to restart the service"; \
			kubectl delete pod `kubectl get pods -o wide | grep $(REPO) | grep Running | cut -d ' ' -f1` ; \
			sleep 15; \
			kubectl logs -f `kubectl get pods -o wide | grep $(REPO) | grep Running | cut -d ' ' -f1` & \
		done

