FROM golang:onbuild

RUN mkdir -p ./data
EXPOSE 8000
VOLUME ["./data"]
