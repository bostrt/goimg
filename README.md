# goimg

An image paste app written in Go.

## Getting Started

### Most Basic Usage

```shell
# go get github.com/bostrt/goimg/...
# mkdir /tmp/data
# goimg --data /tmp/data --db /tmp/data/test.db
```

### Using Docker

```shell
# docker run bostrt/goimg:dev
```

*Still working on a v1.0.0 for `latest` tag. Only `dev` tag available for now.*

### OpenShift 3.x

*TODO*

## Settings

### Command Line Options

```shell
# goimg -h
Usage:
  goimg [flags]

Flags:
  -b, --bind string       [int]:<port> to bind to (default "0.0.0.0:8000")
  -c, --config string     config file
      --data string       path to data directory (default "./data")
      --db string         path to database (default "./test.db")
      --gcinterval int    garbage collection interval in seconds (default 300)
      --gclimit int       garbage collection limit per run (default 100)
  -h, --help              help for goimg
```

### Environment Variables

Environment variables are an alternative to the command line options. Pass the same values to environment variables that you would to command line options.

- `GOIMG_BIND`
- `GOIMG_DATA`
- `GOIMG_DB`
- `GOIMG_GCINTERVAL`
- `GOIMG_GCLIMIT`
- `GOIMG_CONFIG`

#### Example

```shell
# export GOIMG_BIND=0.0.0.0:1234
# goimg
Starting on 0.0.0.0:1234
```

## Development

### Build Source and Run

```shell
# git clone https://github.com/bostrt/goimg
# cd goimg/
# go get -u golang.org/x/vgo
# vgo build
# ./goimg
```

### Docker Build and Run

```shell
# git clone https://github.com/bostrt/goimg
# cd goimg/
# docker build -t mygoimg .
# docker run -p 8000:8000 --rm -it mygoimg
```
