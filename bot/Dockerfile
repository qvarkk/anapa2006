FROM golang:1.27-alpine AS builder
WORKDIR /src
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/bot ./cmd/bot

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/bot .
COPY russian_trusted_root_ca.pem /app/certs/russian_trusted_root_ca.pem
VOLUME /data
ENV TZ=Europe/Moscow
ENTRYPOINT ["/app/bot"]