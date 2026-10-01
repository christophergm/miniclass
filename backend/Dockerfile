FROM golang:1.26.4-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/miniclass-api ./cmd/api

FROM alpine:3.22.2
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 miniclass
COPY --from=build --chown=miniclass:miniclass /out/miniclass-api /usr/local/bin/miniclass-api

USER miniclass
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/miniclass-api"]
