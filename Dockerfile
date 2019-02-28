FROM golang:1.12.0

WORKDIR /app

COPY . .

RUN go build ./...
