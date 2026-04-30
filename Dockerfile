# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26.2-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS="${TARGETOS:-linux}" \
    GOARCH="${TARGETARCH:-$(go env GOARCH)}" \
    go build -trimpath -ldflags="-s -w" -o /out/lab-slackapp .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/lab-slackapp /lab-slackapp

USER nonroot:nonroot
ENTRYPOINT ["/lab-slackapp"]
