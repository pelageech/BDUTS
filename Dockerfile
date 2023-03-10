FROM golang:1.20.2-alpine AS build
WORKDIR /app
COPY . .
CMD ["go", "run", "main.go"]

