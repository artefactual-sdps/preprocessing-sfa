# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24.2

FROM golang:${GO_VERSION}-alpine AS build-go
WORKDIR /src
ENV CGO_ENABLED=0
COPY --link go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY --link . .

FROM build-go AS build-preprocessing-worker
ARG VERSION_PATH
ARG VERSION_LONG
ARG VERSION_SHORT
ARG VERSION_GIT_HASH
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	go build \
	-trimpath \
	-ldflags="-X '${VERSION_PATH}.Long=${VERSION_LONG}' -X '${VERSION_PATH}.Short=${VERSION_SHORT}' -X '${VERSION_PATH}.GitCommit=${VERSION_GIT_HASH}'" \
	-o /out/preprocessing-worker \
	./cmd/worker

FROM debian:12-slim AS preprocessing-worker
RUN apt-get update && apt-get install -y --no-install-recommends \
	libxml2-utils \
	openjdk-17-jre-headless \
	&& rm -rf /var/lib/apt/lists/*

ARG USER_ID=1000
ARG GROUP_ID=1000
RUN groupadd --gid ${GROUP_ID} preprocessing && \
	useradd --uid ${USER_ID} --gid preprocessing --create-home preprocessing && \
	mkdir --parents /var/opt/verapdf/logs /home/preprocessing/shared && \
	chown -R preprocessing:preprocessing /var/opt/verapdf /home/preprocessing

USER preprocessing

COPY --from=ghcr.io/verapdf/cli:latest --link /opt/verapdf/ /opt/verapdf/
COPY --from=build-preprocessing-worker --link /out/preprocessing-worker /home/preprocessing/bin/preprocessing-worker

CMD ["/home/preprocessing/bin/preprocessing-worker"]
