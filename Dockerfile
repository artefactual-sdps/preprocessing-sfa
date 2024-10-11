# syntax=docker/dockerfile:1

# Use the latest bagit-python commit (as of 2024-10-15) because the last release
# is from 2020-06-29 and emits deprecation warnings in Python 3.13.
ARG BAGIT_TAG=56a79001e4cf68cf999fac343bfbfb69f4f73097
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

FROM python:3.13-alpine3.20 AS build-bagit
ARG BAGIT_TAG
ENV PYTHONUNBUFFERED=1
RUN apk add --update --no-cache git
# Install to a venv to keep the compiled artifacts and dependencies together.
RUN python -m venv /venv
ENV PATH="/venv/bin:$PATH"
RUN pip install --no-cache --upgrade \
	pip \
	lxml \
	git+https://github.com/LibraryOfCongress/bagit-python.git@${BAGIT_TAG}

FROM python:3.13-alpine3.20 AS preprocessing-worker
RUN apk add --update --no-cache libxml2-utils

# Copy the bagit-build venv and add its bin directory to the path.
COPY --from=build-bagit --chown=preprocessing:preprocessing /venv /venv
ENV PATH="/venv/bin:$PATH"

ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g ${GROUP_ID} -S preprocessing
RUN adduser -u ${USER_ID} -S -D preprocessing preprocessing

USER preprocessing
COPY --from=build-preprocessing-worker --link /src/hack/sampledata/xsd/* /
COPY --from=build-preprocessing-worker --link /out/preprocessing-worker /home/preprocessing/bin/preprocessing-worker
RUN mkdir /home/preprocessing/shared

CMD ["/home/preprocessing/bin/preprocessing-worker"]
