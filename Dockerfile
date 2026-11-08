# Stage 1: Build Web UI
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go Gateway Binary
FROM golang:1.23-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/geogate ./cmd/geogate

# Stage 3: Minimal Runtime Container
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/geogate /app/geogate

EXPOSE 8080
ENTRYPOINT ["/app/geogate"]
CMD ["serve", "-upstream", "http://localhost:3000", "-port", "8080"]
