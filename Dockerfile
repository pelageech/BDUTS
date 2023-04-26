FROM golang:1.20.2-alpine AS build
WORKDIR /app
ADD go.mod .
COPY . .

RUN go build -o lb .

FROM alpine
WORKDIR /app
COPY --from=build /app/lb /app/lb
COPY --from=build /app/resources resources
CMD ["./lb/BDUTS"]
