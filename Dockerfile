# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION:-dev} -X main.commit=${GIT_COMMIT:-} -X main.date=${BUILD_DATE:-}" -o /out/orch ./cmd/orch

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/orch /usr/local/bin/orch
USER nonroot:nonroot
EXPOSE 8080
ENV ORCH_ADDR=:8080
ENTRYPOINT ["/usr/local/bin/orch"]


