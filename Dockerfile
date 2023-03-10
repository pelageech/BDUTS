FROM golang:1.20.2-alpine AS build
WORKDIR /app
COPY . .

RUN go build -o /app/lb

FROM alpine:3.17.2
WORKDIR /app
COPY --from=build /app/lb .
COPY --from=build /app/resources resources
COPY --from=build /app/MyCertificate.crt /app/MyKey.key .
CMD ["./lb"]
