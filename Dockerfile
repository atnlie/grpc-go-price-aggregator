FROM golang:1.21.5-alpine

WORKDIR /app

COPY . ./

RUN go mod verify
RUN go mod download
RUN go mod tidy
RUN go build server/main.go

EXPOSE 8880

CMD ["./main"]