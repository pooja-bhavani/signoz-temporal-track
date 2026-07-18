FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY shared/ shared/
COPY worker/ worker/
RUN go mod tidy && CGO_ENABLED=0 go build -o /worker ./worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /worker /worker
CMD ["/worker"]
