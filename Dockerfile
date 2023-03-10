FROM golang:1.20.2-alpine AS build
WORKDIR /app
COPY . .

RUN go build -o /app/lb

CMD ["./lb"]

