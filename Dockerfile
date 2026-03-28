FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o auction-simulator .

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/auction-simulator .
COPY example.env .

RUN mkdir -p results

ENTRYPOINT ["./auction-simulator"]
