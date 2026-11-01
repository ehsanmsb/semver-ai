.PHONY: build test lint fmt validate docker
build:
	go build -trimpath -o bin/release-manager ./cmd/release-manager
test:
	go test -race ./...
lint:
	go vet ./...
	test -z "$$(gofmt -l cmd internal)"
fmt:
	gofmt -w cmd internal
validate:
	go run ./cmd/release-manager validate-config --config examples/.release-manager.yml
docker:
	docker build -t semver-ai:local .
