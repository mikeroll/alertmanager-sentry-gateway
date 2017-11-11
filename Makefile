NAME=alertmanager-sentry-gateway
VERSION=0.1.0
COMMIT=$(shell git rev-parse --verify HEAD)

PACKAGES=$(shell go list ./...)
BUILD_FLAGS=-ldflags "-X main.VERSION=$(VERSION) -X main.COMMIT=$(COMMIT)"

.PHONY: build test container clean

build: vendor
	go build $(BUILD_FLAGS) sentry-gateway.go

test: vendor
	go test -v $(PACKAGES)
	go vet $(PACKAGES)

container:
	GOARCH=amd64 GOOS=linux go build $(BUILD_FLAGS) sentry-gateway.go
	docker build -t summerwind/$(NAME):latest -t summerwind/$(NAME):$(VERSION) .
	rm -rf sentry-gateway

clean:
	rm -rf sentry-gateway
	rm -rf dist

dist:
	mkdir -p dist
	
	GOARCH=amd64 GOOS=darwin go build $(BUILD_FLAGS) sentry-gateway.go
	tar -czf dist/${NAME}_darwin_amd64.tar.gz sentry-gateway
	rm -rf sentry-gateway
	
	GOARCH=amd64 GOOS=linux go build $(BUILD_FLAGS) sentry-gateway.go
	tar -czf dist/${NAME}_linux_amd64.tar.gz sentry-gateway
	rm -rf sentry-gateway
	
	GOARCH=arm64 GOOS=linux go build $(BUILD_FLAGS) sentry-gateway.go
	tar -czf dist/${NAME}_linux_arm64.tar.gz sentry-gateway
	rm -rf sentry-gateway
	
	GOARCH=arm GOOS=linux go build $(BUILD_FLAGS) sentry-gateway.go
	tar -czf dist/${NAME}_linux_arm.tar.gz sentry-gateway
	rm -rf sentry-gateway

vendor:
	glide install
