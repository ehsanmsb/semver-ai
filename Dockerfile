FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.buildVersion=${VERSION}" -o /release-manager ./cmd/release-manager

FROM alpine:3.23
RUN apk add --no-cache git ca-certificates && adduser -D -u 10001 semver
COPY --from=build /release-manager /usr/local/bin/release-manager
USER semver
WORKDIR /workspace
ENTRYPOINT ["release-manager"]
