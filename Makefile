VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
IMAGE   ?= thesa-bff
REGISTRY ?= ""

LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
UI_DIR  := ui

.PHONY: build test lint format clean \
        ui-deps ui-generate ui-build ui-clean ui-test ui-analyze \
        docker-build docker-push

## ── Go targets ──

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/thesa-bff ./cmd/bff

test:
	go test -race ./...

format:
	gofmt -w .
	goimports -w . 2>/dev/null || true

lint:
	golangci-lint run ./...

## ── Flutter UI targets ──

ui-deps:
	cd $(UI_DIR) && flutter pub get

ui-generate: ui-deps
	cd $(UI_DIR) && dart run build_runner build --delete-conflicting-outputs

ui-analyze: ui-deps
	cd $(UI_DIR) && flutter analyze --no-fatal-infos

ui-test: ui-generate
	cd $(UI_DIR) && flutter test

ui-build: ui-generate
	cd $(UI_DIR) && flutter build web --release --base-href="/"

ui-clean:
	cd $(UI_DIR) && flutter clean
	rm -rf $(UI_DIR)/build

## ── Docker targets ──

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

## ── Cleanup ──

clean: ui-clean
	rm -rf bin/
