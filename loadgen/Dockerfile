FROM golang:latest AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY go.mod ./
COPY loadgen/ loadgen/
RUN go mod tidy && CGO_ENABLED=0 go build -o /loadgen ./loadgen

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /loadgen /loadgen
CMD ["/loadgen"]
