# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23.2

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

# Build worker image
FROM alpine:3.20 AS preprocessing-worker
RUN apk add --update --no-cache libxml2-utils

# Copy the JRE (Eclipse Temurin v11) from the verapdf/cli image
ENV JAVA_HOME=/opt/java/openjdk
ENV PATH="${JAVA_HOME}/bin:${PATH}"
COPY --from=ghcr.io/verapdf/cli:latest --link $JAVA_HOME $JAVA_HOME

ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g ${GROUP_ID} -S preprocessing
RUN adduser -u ${USER_ID} -S -D preprocessing preprocessing

# Make preprocessing the owner of the verapdf log dir
RUN mkdir --parents /var/opt/verapdf/logs && chown -R preprocessing:preprocessing /var/opt/verapdf

USER preprocessing

COPY --from=build-preprocessing-worker --link /out/preprocessing-worker /home/preprocessing/bin/preprocessing-worker
RUN mkdir /home/preprocessing/shared

# Copy the veraPDF application from the verapdf/cli image
COPY --from=ghcr.io/verapdf/cli:latest --link /opt/verapdf/ /opt/verapdf/

CMD ["/home/preprocessing/bin/preprocessing-worker"]
