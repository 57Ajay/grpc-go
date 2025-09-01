
# -------------------------------
# Stage 1: Build the Go binary
# -------------------------------
FROM golang:1.25-alpine3.22 AS builder

# Install build tools (git often needed for private deps)
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy module files and download dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary (static build)
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./server


# -------------------------------
# Stage 2: Final minimal image
# -------------------------------
FROM alpine:latest

WORKDIR /app

# Copy only the compiled binary
COPY --from=builder /app/server .

# Copy TLS certs (if you need them inside the container)
COPY server.crt server.key ./

# Expose the gRPC port
EXPOSE 50051

# Run the server
ENTRYPOINT ["./server"]
