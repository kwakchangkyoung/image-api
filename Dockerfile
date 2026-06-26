FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o image_api .

FROM scratch

COPY --from=builder /build/image_api /image_api

EXPOSE 8080
ENTRYPOINT ["/image_api"]
