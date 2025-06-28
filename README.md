# spoderman

<p align="center">
    <img height="230" src="https://art.pixilart.com/c60c6f8f7dfd0c1.png">
</p>

A Dead simple, configurable and fast web crawler to grab specific data using regular expression.

## Features

- Quick asl.
- Support crawling multiple target at once & multiple target urls input with a file.
- Support local file scan.
- Configurable crawling settings in YAML format.

## Installation

```bash
go install github.com/got-many-wheels/spoderman@latest
```

## Usages

#### Single url

```bash
spoderman crawl -u http://127.0.0.1:8080
```

#### Multiple urls

```bash
spoderman crawl -f ./examples/urls.txt
```

#### Supported options

```bash
spoderman crawl help

NAME:
   spoderman crawl - Start the crawling process

USAGE:
   spoderman crawl [options]

OPTIONS:
   --depth int, -d int                    Maximum depth for crawling. Higher values crawl deeper into link trees. (default: 2)
   --workers int, -w int                  Number of concurrent workers to crawl URLs in parallel. (default: 10)
   --base, -b                             Restrict crawling to the base domain only. (default: false)
   --url string, -u string                Target Url.
   --url-file string, -f string           Target urls file, separated by line break.
   --allowedDomains string, -a string     Domain whitelist, separated by commas.
   --disallowedDomains string, -a string  Domain blacklist, separated by commas.
   --config string, -i string             Set config file, defaults to set flag values or empty
   --output string                        Output location for secret results.
   --help, -h                             show help

GLOBAL OPTIONS:
   --verbose  Enable verbose logging (default: false)
```

#### Example usage:

```bash
spoderman crawl -u http://127.0.0.1:8080 --depth 3 --workers 20 --verbose --base
```

#### With custom settings

You can use your own crawling settings by providing `-i <path to setting>` flag when using the crawl command. Here are the possible options that you can configure:

```yaml
verbose: true
depth: 3
workers: 10
base: false

# output path for secret founds
output: "./.out/"

# both of this works with wildcards, (eg; *domain.com, *.domain.*, etc)
allowedDomains: []
disallowedDomains: []

# regex patterns to find on the web
rules:
  - name: authorization_bearer
    pattern: bearer\s*[a-zA-Z0-9_\-\.=:_\+\/]+
```
