# Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ .
RUN npm run build

# Build Backend
FROM golang:1.22-alpine AS backend-builder
WORKDIR /src
COPY go.mod ./
# If go.sum existed, we'd copy it here: COPY go.sum ./
COPY . .
# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /server-bin ./cmd/server

# Final Runtime Image
FROM alpine:latest
WORKDIR /app

# Install certificates for HTTPS requests (if needed)
RUN apk --no-cache add ca-certificates

# Copy binary and static assets
COPY --from=backend-builder /server-bin ./server
COPY --from=frontend-builder /src/frontend/dist ./static

# Environment variables
ENV STATIC_DIR=./static
ENV LINTER_SERVER_PORT=8080

# Expose port
EXPOSE 8080

# Run
CMD ["./server"]
