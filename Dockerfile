FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /typst-render-api ./server

FROM alpine:3.21

RUN apk add --no-cache fontconfig freetype ttf-liberation
RUN fc-cache -f
RUN apk add --no-cache typst

WORKDIR /app

COPY --from=builder /typst-render-api .

EXPOSE 8080

ENV TEMPLATE_DIR=/templates
ENV TYPST_PATH=/usr/bin/typst
ENV PORT=8080

CMD ["./typst-render-api"]
