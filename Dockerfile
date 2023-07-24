FROM golang:alpine

WORKDIR /app

COPY . .

RUN go build -o app cmd/stori/main.go

CMD ["./app"]