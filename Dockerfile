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
FROM nginx:alpine AS web
COPY --from=node-builder /src/dist /usr/share/nginx/html
COPY nginx/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
