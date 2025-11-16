# Filepath: auth-service/Dockerfile
# (PASTIKAN Anda copy-paste SEMUA ini ke file)

# --- STAGE 1: Build Binary ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Salin file go.mod dan go.sum
COPY go.mod go.sum ./

# Download semua dependensi
RUN go mod download

# Salin sisa kode dari auth-service
COPY . .

# Build binary
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/api

# --- STAGE 2: Final Image ---
FROM alpine:latest
WORKDIR /app

# Salin *hanya* binary yang sudah di-build
COPY --from=builder /app/server .

# Port untuk auth-service
EXPOSE 8080

# Perintah untuk menjalankan server
CMD ["./server"]