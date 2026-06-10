# Многостадийная сборка: фронт (Vite) → бэкенд (Go) → минимальный рантайм.
# Итог — один образ: Go-сервер отдаёт SPA из /app/web и обслуживает /ws + /api.

# 1) Сборка фронта.
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2) Сборка Go-бинаря (статический, CGO off → бежит на alpine/scratch).
FROM golang:1.26-alpine AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

# 3) Рантайм.
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=server /bin/server /app/server
COPY --from=web /web/dist /app/web
ENV STATIC_DIR=/app/web
EXPOSE 8080
CMD ["/app/server", "-addr", ":8080"]
