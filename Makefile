SHELL := /bin/bash

##@ Development
lint:  ## Run lint on the package
	@printf "\033[2m→ Running lint...\033[0m\n"
	golint -set_exit_status

server:  ## Run HTTP server
	@printf "\033[2m→ Running server...\033[0m\n"
	ELASTICSEARCH_URL=http://localhost:9200 LISTEN_ADDR=localhost go run cmd/server/main.go

cluster: ## Run Elasticsearch for development
	@printf "\033[2m→ Launching Elasticsearch...\033[0m\n"
	docker run \
		--name "logging-1" \
		--env "cluster.name=logging" \
		--env "discovery.type=single-node" \
		--env "xpack.security.enabled=false" \
		--publish 9200:9200 \
		--rm \
		docker.elastic.co/elasticsearch/elasticsearch:8.1.3;

##@ Test
test-unit:  ## Run unit tests
	@printf "\033[2m→ Running unit tests...\033[0m\n"
	go test -v -race
test: test-unit

test-integ:  ## Run integration tests
	@printf "\033[2m→ Running integration tests...\033[0m\n"
	ELASTICSEARCH_URL=http://localhost:9200 go run cmd/setup/main.go
	ELASTICSEARCH_URL=http://localhost:9200 go test -v -race *integration_test.go

##@ Other
#------------------------------------------------------------------------------
help:  ## Display help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
#------------- <https://suva.sh/posts/well-documented-makefiles> --------------

.DEFAULT_GOAL := help
.PHONY: help cluster lint server test test-unit test-integ
