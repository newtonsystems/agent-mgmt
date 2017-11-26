#
# Makefile
#

CURRENT_RELEASE_VERSION=0.0.1
REPO=agent-mgmt
LOCAL_DEPLOYMENT_FILENAME=agent-mgmt-deployment.yml
GO_MAIN=./app/main.go
GO_PORT=50000

TIMESTAMP=$(shell date +%s )
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

help:         ##@other Show this help.
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

#
# Arg Utilities for Makefile
#

ifeq (get,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "run"
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(RUN_ARGS):;@:)
endif

# ------------------------------------------------------------------------------
.PHONY: update master build get

update:       ##@build Updates dependencies for your go application
	bash -c "mkubectl.sh --update-deps"

install:      ##@build Install dependencies for your go application
	bash -c "mkubectl.sh --install-deps"

get:        ##@build Add dependency for your go application
	bash -c "mkubectl.sh --get-deps $(RUN_ARGS)"

build:        ##@compile Builds executable cross compiled for alpine docker
	bash -c "mkubectl.sh --compile-inside-docker ${REPO} ${GO_MAIN}"


# ------------------------------------------------------------------------------
# CircleCI support
.PHONY: check preparedb

check:        ##@circleci Needed for running circleci tests
	@echo "$(INFO) Running tests"
	# https://splice.com/blog/lesser-known-features-go-test/
	# Running go test <package1> <package2> <package3> will run packages in parallel
  # we dont want this because of our db access/destruction
	# therefore we set parrallel to 1:    -p 1
	go test -v -p 1 ./app/models ./app/service ./app


# ------------------------------------------------------------------------------

# ------------------------------------------------------------------------------
# Non docker local development (can be useful for super fast local/debugging)
.PHONY: run-conn run-build-bin clean

run-conn:          ##@devlocal Run locally (outside docker) (but connect to minikube linkerd etc)
	@echo "$(INFO) Running go service outside of docker ...."
	go run ${GO_MAIN} --conn.local

build-bin:      ##@devlocal Builds binary locally (outside docker)
	bash -c "REPO=${REPO} GO_MAIN=${GO_MAIN} mkubectl.sh --compile"

clean:
	@rm -rf vendor/
	@rm -f ${REPO}
	@rm -f Gopkg.toml Gopkg.lock
	@rm -rf _build/
	@rm -f .testfailures .testsuccesses


# ------------------------------------------------------------------------------
# Minikube (Normal Development)
.PHONY: run swap-hot-local swap-latest swap-latest-release

run:                    ##@dev Alias for swap-hot-local
	@make REPO=${REPO} GO_MAIN=${GO_MAIN} swap-hot-local

# Tests inside docker (why needed for replica tests - outside docker you cant connect to DNS)  move
test:                   ##@dev Run tests inside minikube
	@bash -c "mkubectl.sh --hot-reload-test ${REPO} ${LOCAL_DEPLOYMENT_FILENAME}"

swap-hot-local:         ##@dev Swaps $(REPO) deployment in minikube with hot-reloadable docker image (You must make sure you are running i.e. infra-minikube.sh --create)
	@bash -c "mkubectl.sh --hot-reload-deployment ${REPO} ${LOCAL_DEPLOYMENT_FILENAME} ${GO_PORT}"

swap-latest:            ##@dev Swaps $(REPO) deployment in minikube with the latest image for branch from dockerhub (You must make sure you are running i.e. infra-minikube.sh --create)
	@bash -c "mkubectl.sh --swap-deployment-with-latest-image ${REPO} ${LOCAL_DEPLOYMENT_FILENAME}"

swap-latest-release:    ##@dev Swaps $(REPO) deployment in minikube with the latest release image for from dockerhub (You must make sure you are running i.e. infra-minikube.sh --create)
	@bash -c "mkubectl.sh --swap-deployment-with-latest-release-image ${REPO} ${LOCAL_DEPLOYMENT_FILENAME} ${CURRENT_RELEASE_VERSION}"
