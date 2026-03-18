# syntax=docker/dockerfile:1

# ---- Build Go backend ----
FROM golang:1.23-alpine AS go-builder
WORKDIR /src
COPY allin-server/go.mod allin-server/go.sum ./
RUN go mod download
COPY allin-server/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /app/server ./cmd/server

# ---- Build frontend ----
FROM node:22-alpine AS node-builder
WORKDIR /src
COPY allin-web/package*.json ./
RUN npm ci
COPY allin-web/ .
RUN npm run build

# ---- Final image ----
FROM gcr.io/distroless/static-debian12
COPY --from=go-builder /app/server /server
COPY --from=node-builder /src/dist /static
EXPOSE 8080
ENTRYPOINT ["/server"]
