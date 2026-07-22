FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/warehouse-routing ./cmd/server

FROM alpine:3.22
RUN adduser -D -H app
USER app
COPY --from=build /out/warehouse-routing /warehouse-routing
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/readyz || exit 1
ENTRYPOINT ["/warehouse-routing"]
