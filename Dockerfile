FROM golang:1.27-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /coord ./cmd/coord

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /coord /coord
EXPOSE 8080
ENTRYPOINT ["/coord"]
