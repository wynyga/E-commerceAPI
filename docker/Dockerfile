# syntax=docker/dockerfile:1
#
# One Dockerfile for every Go service. Build from the repository root:
#   docker build -f docker/Dockerfile --build-arg SERVICE=order .

FROM golang:1.24 AS build
ARG SERVICE
WORKDIR /src

# Each service's go.mod points at ../../shared, so both must be present.
COPY shared ./shared
COPY services/${SERVICE} ./services/${SERVICE}

WORKDIR /src/services/${SERVICE}
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app .

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
