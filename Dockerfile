# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS build
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o taskapp ./cmd/tasks/main.go

FROM alpine:3.22
WORKDIR /app
COPY --from=build /app/taskapp .
COPY migrations ./migrations
RUN adduser -D appuser
USER appuser
EXPOSE 8080
CMD ["./taskapp"] 