FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o auction-simulator .

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/auction-simulator .
# Rename example.env to .env so godotenv.Load() picks it up at runtime
# For local testing, replace 'example.env' with your own .env file if needed:
#   COPY .env .env
COPY example.env .env

RUN mkdir -p results

ENTRYPOINT ["./auction-simulator"]
