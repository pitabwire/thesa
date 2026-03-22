VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
IMAGE   ?= thesa-bff
REGISTRY ?= ""

LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
UI_DIR  := ui

# Dart-defines — development defaults, override for production via env or make args.
# e.g. make ui-build-prod BFF_BASE_URL=https://api.stawi.org/thesa
BFF_BASE_URL     ?= http://localhost:8080
OIDC_CLIENT_ID   ?= d6qbqdkpf2t52mcunf3g
OIDC_ISSUER      ?= https://oauth2.stawi.org

.PHONY: build test lint format clean \
        flutter-setup \
        ui-deps ui-generate ui-drift-worker ui-build ui-build-prod ui-build-dev \
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

## ── Flutter SDK setup ──

FLUTTER_CHANNEL ?= stable
FLUTTER_HOME    ?= $(HOME)/flutter

## Install Flutter SDK (stable channel) if not already present.
## Idempotent — skips download when flutter is already on PATH.
flutter-setup:
	@if command -v flutter >/dev/null 2>&1; then \
		echo "Flutter already installed: $$(flutter --version | head -1)"; \
	else \
		echo "Installing Flutter ($(FLUTTER_CHANNEL) channel)..."; \
		FLUTTER_URL=$$(curl -s https://storage.googleapis.com/flutter_infra_release/releases/releases_linux.json \
			| python3 -c "import json,sys; r=json.load(sys.stdin); print('https://storage.googleapis.com/flutter_infra_release/releases/' + [x for x in r['releases'] if x['hash']==r['current_release']['$(FLUTTER_CHANNEL)']][0]['archive'])"); \
		curl -sL "$$FLUTTER_URL" | tar xJ -C $(dir $(FLUTTER_HOME)); \
		export PATH="$(FLUTTER_HOME)/bin:$(FLUTTER_HOME)/bin/cache/dart-sdk/bin:$$PATH"; \
		flutter precache --web; \
		echo "Flutter installed to $(FLUTTER_HOME)"; \
	fi

## ── Flutter UI targets ──

ui-deps: flutter-setup
	cd $(UI_DIR) && flutter pub get

ui-generate: ui-deps
	cd $(UI_DIR) && dart run build_runner build --delete-conflicting-outputs

ui-analyze: ui-deps
	cd $(UI_DIR) && flutter analyze --no-fatal-infos

ui-test: ui-generate
	cd $(UI_DIR) && flutter test

## Compile Drift web worker and fetch sqlite3.wasm for offline-first caching.
## Both files land in web/ so flutter build copies them into the output.
SQLITE3_WASM_URL ?= https://github.com/simolus3/sqlite3.dart/releases/latest/download/sqlite3.wasm

ui-drift-worker: ui-deps
	cd $(UI_DIR) && dart compile js -O2 -o web/drift_worker.js web/drift_worker.dart
	@if [ ! -f $(UI_DIR)/web/sqlite3.wasm ]; then \
		echo "Downloading sqlite3.wasm..."; \
		curl -sL -o $(UI_DIR)/web/sqlite3.wasm $(SQLITE3_WASM_URL); \
	fi
	@echo "Drift web worker ready"

## Development web build (default) — profile mode, source maps, localhost BFF.
ui-build-dev: ui-generate ui-drift-worker
	cd $(UI_DIR) && flutter build web \
		--profile \
		--base-href="/" \
		--dart-define=BFF_BASE_URL=$(BFF_BASE_URL) \
		--dart-define=OIDC_CLIENT_ID=$(OIDC_CLIENT_ID) \
		--dart-define=OIDC_ISSUER=$(OIDC_ISSUER) \
		--source-maps
	@echo "Dev build complete: $(UI_DIR)/build/web/"

## Production web build — release mode, tree-shaken, minified.
## Requires BFF_BASE_URL to be set (e.g. via Cloudflare env vars).
ui-build-prod: ui-generate ui-drift-worker
	cd $(UI_DIR) && flutter build web \
		--release \
		--base-href="/" \
		--tree-shake-icons \
		--dart-define=BFF_BASE_URL=$(BFF_BASE_URL) \
		--dart-define=OIDC_CLIENT_ID=$(OIDC_CLIENT_ID) \
		--dart-define=OIDC_ISSUER=$(OIDC_ISSUER) \
		--dart-define=ENV=production
	@echo "Production build complete: $(UI_DIR)/build/web/"

## Default build alias (development)
ui-build: ui-build-dev

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
