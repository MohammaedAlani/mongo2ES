FROM golang:1.20-alpine as builder

WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o mongo-to-elastic ./cmd/app

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/mongo-to-elastic .
COPY --from=builder /app/config/config.json ./config/config.json

EXPOSE 9090 8080
CMD ["./mongo-to-elastic", "-config=./config/config.json"]