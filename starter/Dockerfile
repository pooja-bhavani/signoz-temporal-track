FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY shared/ shared/
COPY starter/ starter/
RUN go mod tidy && CGO_ENABLED=0 go build -o /starter ./starter

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /starter /starter
CMD ["/starter"]
