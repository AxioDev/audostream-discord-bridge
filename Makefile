GOFILES=$(shell find ./ -type f -name '*.go')
LATEST_TAG=$(shell git describe --tags `git rev-list --tags --max-count=1`)

audostream-discord-bridge: $(GOFILES)
goreleaser build --clean --snapshot

release: $(GOFILES)
	goreleaser release --clean

dev: $(GOFILES)
goreleaser build --clean --single-target --snapshot && ./dist/audostream-discord-bridge_linux_amd64_v1/audostream-discord-bridge

dev-race: $(GOFILES)
go run -race ./cmd/audostream-discord-bridge

dev-profile: $(GOFILES)
goreleaser build --skip=validate --clean --single-target --snapshot && ./dist/audostream-discord-bridge_linux_amd64_v1/audostream-discord-bridge -cpuprofile cpu.prof

test-chart: SHELL:=/bin/bash 
test-chart:
	go test ./test &
	until pidof test.test; do continue; done;
psrecord --plot docs/test-cpu-memory.png $$(pidof audostream-discord-bridge.test)

lint:
	golangci-lint run

format:
	go fmt ./...

clean:
	rm -rf dist
	rm -rf LICENSES.zip LICENSES

.PHONY: audostream-discord-bridge release dev dev-profile dev-race test-chart clean lint format
