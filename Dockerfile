FROM golang:1.19

WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /go/src/app/goimg

RUN mkdir -p /go/src/app/data
EXPOSE 8000
VOLUME ["/go/src/app/data"]
CMD ["/go/src/app/goimg"]
