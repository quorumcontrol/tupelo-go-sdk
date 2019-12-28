FROM golang:1.13.5-alpine3.10

WORKDIR /app

RUN apk add --no-cache --update build-base

COPY . .

RUN go build -mod=vendor ./...
