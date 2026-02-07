VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
IMAGE   ?= thesa-bff
REGISTRY ?= ""

LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build test lint docker-build docker-push clean

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/thesa-bff ./cmd/bff

test:
	go test -race ./...

lint:
	golangci-lint run ./...

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(IMAGE):$(VERSION) \
		-t $(IMAGE):latest \
		.

docker-push:
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)
	docker push $(REGISTRY)/$(IMAGE):latest

clean:
	rm -rf bin/
