FROM --platform=linux/amd64 golang:1.22-bookwork as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build

FROM --platform=linux/amd64 debian:bookwork-slim

WORKDIR /app

COPY --from=builder /app/probec /app

ENTRYPOINT ["/app/probec"]
