FROM golang:1.12.6-alpine3.9

WORKDIR /app

RUN apk add --no-cache --update build-base

COPY . .

RUN go build -mod=vendor ./...
