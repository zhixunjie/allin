# syntax=docker/dockerfile:1

# ---- Build Go backend ----
FROM golang:1.25-alpine AS go-builder
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/server ./base/

# ---- Build frontend ----
FROM node:22-alpine AS node-builder
WORKDIR /src
COPY ui-web/package*.json ./
RUN npm ci
COPY ui-web/ .
RUN npm run build

# ---- Server runtime ----
FROM gcr.io/distroless/static-debian12 AS server
WORKDIR /app
COPY --from=go-builder /bin/server .
COPY server/base/config.docker.yaml config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/server"]

# ---- Nginx + built frontend ----
# nginx:alpine 官方镜像会在启动时用 envsubst 处理 /etc/nginx/templates/*.template
# 运行时需设置环境变量 BACKEND_URL（如 server:8080 或 server.railway.internal:8080）
FROM nginx:alpine AS web
COPY --from=node-builder /src/dist /usr/share/nginx/html
COPY nginx/nginx.conf /etc/nginx/templates/default.conf.template
EXPOSE 80
