FROM golang:1.11.5

WORKDIR /app

COPY . .

RUN go build -mod=vendor ./...
