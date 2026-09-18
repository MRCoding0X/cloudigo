FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/cloudigo-api ./cmd/api

FROM alpine:3.20
RUN adduser -D -H appuser
WORKDIR /app
COPY --from=build /out/cloudigo-api ./cloudigo-api
RUN mkdir -p /app/storage/uploads /app/storage/temp && chown -R appuser /app
USER appuser
EXPOSE 8080
ENTRYPOINT ["./cloudigo-api"]
