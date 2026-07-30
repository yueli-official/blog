FROM golang:1.25.12-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY api ./api
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/blog-api ./cmd/blog
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/blog-healthcheck ./cmd/healthcheck
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/blog-bindingcheck ./cmd/bindingcheck
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/blog-bootstrap ./cmd/bootstrap

FROM migrate/migrate:v4.19.1 AS migration
COPY manifest/sql/migrations /migrations

FROM gcr.io/distroless/static-debian12:nonroot AS bindingcheck
COPY --from=build /out/blog-bindingcheck /usr/local/bin/blog-bindingcheck
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/blog-bindingcheck"]

FROM gcr.io/distroless/static-debian12:nonroot AS bootstrap
COPY --from=build /out/blog-bootstrap /usr/local/bin/blog-bootstrap
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/blog-bootstrap"]

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/blog-api /usr/local/bin/blog-api
COPY --from=build /out/blog-healthcheck /usr/local/bin/blog-healthcheck
COPY --chown=nonroot:nonroot manifest ./manifest

ENV GF_SERVER_ADDRESS=0.0.0.0:8085
ENV OTEL_TRACES_EXPORTER=none

EXPOSE 8085
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/blog-api"]
