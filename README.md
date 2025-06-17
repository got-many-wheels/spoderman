# spoderman

<p align="center">
    <img height="230" src="https://art.pixilart.com/c60c6f8f7dfd0c1.png">
</p>

A Dead simple and fast web crawler that can be specified to crawl certain websites and depths.

## Features

- Quick asl.
- Support crawling multiple target at once & multiple target urls input with a file.
- Support local file scan.

## Installation

```bash
go install github.com/got-many-wheels/spoderman@latest
```

## Usages

All supported options:

```bash
NAME:
   spoderman - Dead simple website crawler

USAGE:
   spoderman [global options] [command [command options]]

COMMANDS:
   crawl    Start the crawling process
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --verbose   Enable verbose logging (default: false)
   --help, -h  show help
```

### Crawling Usage

#### Single url

```bash
# run the binary and enter a URL when prompted
spoderman crawl -u http://127.0.0.1:8080
```

#### Multiple urls

```bash
spoderman crawl -f ./examples/urls.txt
```

Options:
- --depth int, -d int           Maximum depth for crawling. Higher values crawl deeper into link trees. (default: 1)
- --workers int, -w int         Number of concurrent workers to crawl URLs in parallel. (default: 10)
- --base, -b                    Restrict crawling to the base domain only. (default: true)
- --url string, -u string       Target Url.
- --url-file string, -f string  Target urls file, separated by line break.
- --help, -h                    show help

#### Example usage:

```bash
spoderman crawl -u http://127.0.0.1:8080 --depth 3 --workers 20 --verbose --base
```
