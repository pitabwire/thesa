# Stage 1: Build Go BFF
FROM golang:1.26 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w \
      -X github.com/pitabwire/frame/version.Version=${VERSION} \
      -X github.com/pitabwire/frame/version.Commit=${COMMIT}" \
    -o /thesa-bff ./cmd/bff

# Stage 2: Runtime
FROM cgr.dev/chainguard/static:latest

COPY --from=builder /thesa-bff /thesa-bff
COPY definitions/ /definitions/
COPY specs/ /specs/
COPY config/ /config/

USER 65532:65532
EXPOSE 80

ENTRYPOINT ["/thesa-bff"]
CMD ["--config", "/config/config.yaml"]
