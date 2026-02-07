# Stage 1: Build
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

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /thesa-bff /thesa-bff
COPY definitions/ /definitions/
COPY specs/ /specs/
COPY config/ /config/

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/thesa-bff"]
CMD ["--config", "/config/config.yaml"]
