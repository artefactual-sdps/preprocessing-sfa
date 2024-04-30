# syntax = docker/dockerfile:1.4

ARG GO_VERSION

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

FROM alpine:3.18.2 AS base
ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g ${GROUP_ID} -S preprocessing
RUN adduser -u ${USER_ID} -S -D preprocessing preprocessing
USER preprocessing
RUN mkdir /home/preprocessing/shared

FROM base AS preprocessing-worker
ENV PYTHONUNBUFFERED=1
USER root
RUN apk add --update --no-cache python3 && \
	ln -sf python3 /usr/bin/python && \
	python3 -m ensurepip
USER preprocessing
RUN pip3 install --no-cache --upgrade pip lxml bagit==v1.8.1
COPY --from=build-preprocessing-worker --link /src/hack/sampledata/xsd/* /
COPY --from=build-preprocessing-worker --link /out/preprocessing-worker /home/preprocessing/bin/preprocessing-worker
CMD ["/home/preprocessing/bin/preprocessing-worker"]
