FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy && go build -o agent-registry .

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Create data directory for embedded storage
RUN mkdir -p /data/agent-registry

COPY --from=builder /app/agent-registry .

EXPOSE 8001

CMD ["./agent-registry"]