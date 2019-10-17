FROM golang:1.13.1-alpine3.10

WORKDIR /app

RUN apk add --no-cache --update build-base git

COPY . .

RUN go build -mod=vendor ./...
