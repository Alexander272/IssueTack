# 1️⃣ Фронтенд
FROM node:25-alpine3.23 AS frontend
WORKDIR /build
COPY frontend/package.json frontend/yarn.lock ./
RUN yarn install --frozen-lockfile
COPY frontend/ .
RUN yarn build

# Предкомпрессия статических ассетов
RUN apk add --no-cache brotli && \
    find dist -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" \) -exec gzip -9 -k {} + && \
    find dist -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" \) -exec brotli -q 11 -k {} +

# 2️⃣ Сборка Go
FROM golang:1.25-alpine3.23 AS builder
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=frontend /build/dist ./web/frontend

# Критически важные флаги для Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -extldflags '-static'" -o main cmd/app/main.go

# 3️⃣ Runtime
FROM alpine:3.23.4

# Время + непривилегированный пользователь
RUN apk add --no-cache tzdata && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

COPY --from=builder /build/configs/ /configs/
COPY --from=builder /build/main /bin/main

# Безопасность и документация
USER appuser
EXPOSE 9000
ENTRYPOINT ["/bin/main"]