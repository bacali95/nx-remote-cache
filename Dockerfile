FROM oven/bun:1.3.14-alpine AS web-build
WORKDIR /src/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# go:embed requires web/dist to exist before `go build` — it's a build
# artifact (gitignored), produced by the web-build stage above.
COPY --from=web-build /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
COPY --from=build --chown=nonroot:nonroot /out/data /data
USER nonroot:nonroot
ENV CACHE_DIR=/data
EXPOSE 3000
ENTRYPOINT ["/server"]
