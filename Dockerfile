FROM node:22-alpine AS frontend-builder

WORKDIR /app/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
COPY --from=frontend-builder /app/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/traffic-monitor ./cmd/server

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=backend-builder /out/traffic-monitor /usr/local/bin/traffic-monitor

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:8080/healthz || exit 1

CMD ["traffic-monitor"]
