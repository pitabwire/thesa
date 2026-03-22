VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
IMAGE   ?= thesa-bff
REGISTRY ?= ""

LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
UI_DIR  := ui

# Production dart-defines (override via environment or make args)
BFF_BASE_URL     ?= https://api.stawi.org/thesa
OIDC_CLIENT_ID   ?= d6qbqdkpf2t52mcunf3g
OIDC_ISSUER      ?= https://oauth2.stawi.org

.PHONY: build test lint format clean \
        ui-deps ui-generate ui-build ui-build-prod ui-build-dev \
        ui-clean ui-test ui-analyze \
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

## Production web build — tree-shaken, minified, dart-defines baked in.
ui-build-prod: ui-generate
	cd $(UI_DIR) && flutter build web \
		--release \
		--base-href="/" \
		--tree-shake-icons \
		--dart-define=BFF_BASE_URL=$(BFF_BASE_URL) \
		--dart-define=OIDC_CLIENT_ID=$(OIDC_CLIENT_ID) \
		--dart-define=OIDC_ISSUER=$(OIDC_ISSUER) \
		--dart-define=ENV=production
	@echo "Production build complete: $(UI_DIR)/build/web/"

## Development web build — fast iteration, debug info, localhost defaults.
ui-build-dev: ui-generate
	cd $(UI_DIR) && flutter build web \
		--profile \
		--base-href="/" \
		--dart-define=BFF_BASE_URL=http://localhost:8080 \
		--dart-define=OIDC_CLIENT_ID=$(OIDC_CLIENT_ID) \
		--dart-define=OIDC_ISSUER=$(OIDC_ISSUER) \
		--source-maps
	@echo "Dev build complete: $(UI_DIR)/build/web/"

## Default build alias (production)
ui-build: ui-build-prod

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

ui-clean:
	cd $(UI_DIR) && flutter clean
	rm -rf $(UI_DIR)/build
