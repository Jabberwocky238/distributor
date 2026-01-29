FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o distributor .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/distributor .
COPY config.yaml .
EXPOSE 8081
CMD ["./distributor"]
