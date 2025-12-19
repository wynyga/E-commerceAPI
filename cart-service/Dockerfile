# Filepath: cart-service/Dockerfile
# (PASTIKAN Anda copy-paste SEMUA ini ke file)

# --- STAGE 1: Build Binary ---
# (Pastikan go.mod cart-service Anda juga go 1.24)
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Salin file go.mod dan go.sum dari cart-service
COPY go.mod go.sum ./

# Download semua dependensi (GORM, Chi, dll)
RUN go mod download

# Salin sisa kode dari cart-service
COPY . .

# Build binary
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/api

# --- STAGE 2: Final Image ---
FROM alpine:latest
WORKDIR /app

# Salin *hanya* binary yang sudah di-build
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/server .

# Port untuk cart-service
EXPOSE 8081

# Perintah untuk menjalankan server
CMD ["./server"]