# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cerberus ./cmd/server

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates && \
    addgroup -S cerberus && \
    adduser -S cerberus -G cerberus

COPY --from=build /out/cerberus /app/cerberus

ENV SERVER_PORT=8080
EXPOSE 8080

USER cerberus

ENTRYPOINT ["/app/cerberus"]
