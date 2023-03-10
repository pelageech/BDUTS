FROM golang:1.20.2-alpine AS build
WORKDIR /app
COPY . .

RUN go build -o /app/server

FROM alpine:3.17.2
WORKDIR /app
COPY --from=build /app/server /app/content.mp4 .
CMD ["./server"]
