# Сборка координационного сервиса (COORD).
#
# Стадия сборки компилирует два статических бинаря: сервис и healthcheck
# для финального distroless-образа, где нет shell, curl и wget.
FROM golang:1.27-alpine AS build

WORKDIR /src

# Слой зависимостей отдельно от исходников: правка кода не сбрасывает кеш.
COPY go.mod ./
RUN go mod download

COPY . .

ARG VERSION=dev

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/coord ./cmd/coord \
    && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/healthcheck ./cmd/healthcheck

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/coord /coord
COPY --from=build /out/healthcheck /healthcheck

ENV HTTP_ADDR=:8080

EXPOSE 8080

USER nonroot:nonroot

# HEALTHCHECK работает без shell: /healthcheck читает HTTP_ADDR и запрашивает /health.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/healthcheck"]

ENTRYPOINT ["/coord"]
