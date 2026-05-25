FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.work ./
COPY CostHandler_mcp/go.mod CostHandler_mcp/go.sum ./CostHandler_mcp/
COPY CostHandler_agent/go.mod CostHandler_agent/go.sum ./CostHandler_agent/
COPY CostHandler_bot/go.mod CostHandler_bot/go.sum ./CostHandler_bot/

RUN cd CostHandler_mcp && go mod download
RUN cd CostHandler_agent && go mod download
RUN cd CostHandler_bot && go mod download

COPY CostHandler_mcp/ ./CostHandler_mcp/
COPY CostHandler_agent/ ./CostHandler_agent/
COPY CostHandler_bot/ ./CostHandler_bot/

RUN cd CostHandler_bot && CGO_ENABLED=0 go build -o /costhandler ./cmd/main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -u 1000 appuser
USER appuser

WORKDIR /home/appuser

COPY --from=builder /costhandler .

EXPOSE 8080

CMD ["./costhandler"]
