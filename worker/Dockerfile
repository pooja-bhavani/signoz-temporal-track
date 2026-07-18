FROM golang:latest AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY go.mod ./
COPY shared/ shared/
COPY worker/ worker/
RUN go mod tidy && CGO_ENABLED=0 go build -o /worker ./worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /worker /worker
CMD ["/worker"]
