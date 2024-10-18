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

FROM alpine:3.20 AS preprocessing-worker
RUN apk add --update --no-cache libxml2-utils

ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g ${GROUP_ID} -S preprocessing
RUN adduser -u ${USER_ID} -S -D preprocessing preprocessing

USER preprocessing
COPY --from=build-preprocessing-worker --link /out/preprocessing-worker /home/preprocessing/bin/preprocessing-worker
RUN mkdir /home/preprocessing/shared

CMD ["/home/preprocessing/bin/preprocessing-worker"]
