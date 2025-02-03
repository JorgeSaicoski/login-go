FROM golang:1.21

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main ./cmd/server/main.go

EXPOSE 8080
CMD ["./main"]