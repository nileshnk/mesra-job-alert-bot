FROM golang:1.21.3-alpine3.18

WORKDIR /app

COPY . .

RUN go mod tidy

ENTRYPOINT [ "go", "run", "." ]

EXPOSE 5050