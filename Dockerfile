FROM golang:1.24-alpine AS builder
RUN apk add --no-cache zip make
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build-lambda
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o updo .

FROM scratch
COPY --from=builder /app/updo /updo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/updo"]
