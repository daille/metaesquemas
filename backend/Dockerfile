FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
# -mod=mod resuelve y registra go.sum de lo que importa el binario (sin analizar las pruebas, como haría "go mod tidy").
RUN CGO_ENABLED=0 go build -mod=mod -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
# Directorios de datos con propietario uid 1000 (el mismo que el worker) para compartir volúmenes
RUN mkdir -p /out/data/store /out/data/inbox /out/data/jobs

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/api /api
COPY --from=build --chown=1000:1000 /out/data /data
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/api"]
