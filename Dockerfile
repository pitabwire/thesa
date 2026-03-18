# Stage 1: Build Flutter web UI
FROM ghcr.io/cirruslabs/flutter:latest AS flutter-builder

WORKDIR /ui

# Copy dependency files first for caching.
COPY ui/pubspec.yaml ./
COPY ui/packages/ packages/
RUN flutter pub get

# Copy the rest of the UI source.
COPY ui/ .

# Generate code (Riverpod, Drift, Freezed, JSON serialization).
RUN dart run build_runner build --delete-conflicting-outputs

# Build for web with production environment.
# The .env.production is bundled as an asset; BFF_BASE_URL is empty for same-origin.
RUN flutter build web --release --base-href="/"

# Stage 2: Build Go BFF
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Cache dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /thesa-bff ./cmd/bff

# Stage 3: Runtime
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /thesa-bff /thesa-bff
COPY --from=flutter-builder /ui/build/web /ui/
COPY definitions/ /definitions/
COPY specs/ /specs/
COPY config/ /config/

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/thesa-bff"]
CMD ["--config", "/config/config.yaml"]
