# Multi-stage build for a small final image

# 1) Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app

# For HTTPS requests inside the app
RUN apk add --no-cache ca-certificates

# Cache deps
COPY go.mod ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
ENV CGO_ENABLED=0
RUN go build -o /server ./main.go

# 2) Run stage
FROM alpine:3.20
WORKDIR /app

# Copy binary and static files
COPY --from=builder /server /app/server
COPY public /app/public

ENV PORT=8080
EXPOSE 8080

CMD ["./server"]


