# goimg
An image paste app written in Go.

## Most Basic Usage
```shell
# go get github.com/bostrt/goimg/...
# mkdir /tmp/data
# goimg --data /tmp/data --db /tmp/data/test.db
```

## Build Source and Run

```shell
# git clone https://github.com/bostrt/goimg
# cd goimg/
# go get -u golang.org/x/vgo
# vgo build
# ./goimg
```

## Docker Build and Run

```shell
# git clone https://github.com/bostrt/goimg
# cd goimg/
# docker build -t mygoimg .
# docker run -p 8000:8000 --rm -it mygoimg
```
Open `http://localhost:8000/` in your browser.

## OpenShift 3.x 

*TODO*
