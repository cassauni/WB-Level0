FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/order-service ./cmd

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /bin/order-service /app/order-service
COPY web/ /app/web/
COPY migrations/ /app/migrations/
ENV HTTP_ADDR=:8081
EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/app/order-service"]
