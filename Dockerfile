FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/tradejournal-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /
COPY --from=builder /out/tradejournal-api /tradejournal-api

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/tradejournal-api"]
