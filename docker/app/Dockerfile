FROM golang:1.26.1-alpine3.23 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /app/api ./cmd/api && \
    go build -trimpath -ldflags="-s -w" -o /app/exporter ./cmd/exporter

# TODO: add migrations somewhere around here, but that's for production later

FROM alpine:3.24.1 AS run-stage

RUN apk --no-cache add ca-certificates

WORKDIR /

COPY --from=build-stage /app /bin/app

RUN adduser -D -u 10001 greenlight
USER greenlight

CMD ["/bin/app/api"]