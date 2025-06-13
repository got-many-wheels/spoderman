# spoderman

<p align="center">
    <img height="230" src="https://art.pixilart.com/c60c6f8f7dfd0c1.png">
</p>

A Dead simple and fast web crawler that can be specified to crawl certain websites and depths.

## Installation

```bash
go install github.com/got-many-wheels/spoderman@latest
```

## Usages

Single url

```bash
# run the binary and enter a URL when prompted
spoderman
```

Multiple urls

```bash
cat examples/input.txt | spoderman
```

### Flags

You can customize the crawler's behavior using the following flags:

| Flag            | Type | Default | Description                                                             |
| --------------- | ---- | ------- | ----------------------------------------------------------------------- |
| `-depth`        | int  | `1`     | Maximum depth for crawling. Higher values crawl deeper into link trees. |
| `-workersCount` | int  | `10`    | Number of concurrent workers to crawl URLs in parallel.                 |
| `-verbose`      | bool | `false` | Enables detailed logs for each crawling operation.                      |
| `-base`         | bool | `false` | Restrict crawling to the base domain only (same host as initial URL).   |

Example usage:

```bash
spoderman -depth 3 -workersCount 20 -verbose -base
```
