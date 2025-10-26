FROM node:20-alpine AS frontend-builder

WORKDIR /app/ui

COPY ui/package.json ui/pnpm-lock.yaml ./

RUN npm install -g pnpm && \
    pnpm install --frozen-lockfile

COPY ui/ ./
RUN pnpm run build

FROM golang:1.24-alpine AS backend-builder

# Build arguments for version injection
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG COMMIT_ID=0000000

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

COPY --from=frontend-builder /app/static ./static
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w \
      -X github.com/ysicing/tiga/internal/version.Version=${VERSION} \
      -X github.com/ysicing/tiga/internal/version.BuildTime=${BUILD_TIME} \
      -X github.com/ysicing/tiga/internal/version.CommitID=${COMMIT_ID}" \
    -o tiga cmd/tiga/main.go

FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=backend-builder /app/tiga .

EXPOSE 12306

CMD ["./tiga"]
