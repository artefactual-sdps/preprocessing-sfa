FROM golang:1.26 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/mockapi ./cmd/mockapi

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/mockapi /mockapi

EXPOSE 8080
ENTRYPOINT ["/mockapi"]
