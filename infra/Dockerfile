# Build context is ./backend (see infra/render.yaml), so paths below are
# relative to the Go module root.
#
#   docker build -f infra/Dockerfile -t yuno-failover-api backend
#   docker run --rm -p 8080:8080 -e API_KEY=dev-key yuno-failover-api

FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static binary so it runs on distroless/static (no libc).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server .

# distroless/static: no shell or package manager; :nonroot runs as uid 65532.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
USER nonroot:nonroot
ENV GIN_MODE=release
EXPOSE 8080
ENTRYPOINT ["/server"]
