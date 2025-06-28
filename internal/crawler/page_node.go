package crawler

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type Secret struct {
	ID       string
	Hostname string
	Key      string
	Value    string
}

type pageNode struct {
	targetUrl      string
	foundUrls      []string
	foundSecrets   []Secret
	depth          int
	payload        []byte
	secretPatterns map[string]string
}

func newPageNode(targetUrl string, buf []byte) *pageNode {
	commonPatterns := map[string]string{
		"jwt":   `e[yw][A-Za-z0-9-_]+\.(?:e[yw][A-Za-z0-9-_]+)?\.[A-Za-z0-9-_]{2,}(?:(?:\.[A-Za-z0-9-_]{2,}){2})?`,
		"email": `\b([\w\.-]{5,30})@[\w\.-]+\.([A-Za-z]{2,3})\b`,
	}
	return &pageNode{
		targetUrl:      targetUrl,
		payload:        buf,
		secretPatterns: commonPatterns,
		foundUrls:      []string{},
		foundSecrets:   []Secret{},
	}
}

func (node *pageNode) extractAndExtends(hostname string) error {
	// look inside the current page
	if _, err := node.extractUrls(); err != nil {
		return err
	}
	// and for secrets after
	pStr := string(node.payload)
	for key, pattern := range node.secretPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// TODO: should be strict or ignore?
			continue
		}
		matches := re.FindAllStringSubmatch(pStr, -1)
		for _, match := range matches {
			if len(match) > 0 && match[0] != "" {
				node.foundSecrets = append(
					node.foundSecrets,
					Secret{ID: fmt.Sprintf("%s:%s", hostname, match[0]), Hostname: hostname, Key: key, Value: match[0]},
				)
			}
		}
	}
	return nil
}

func (node *pageNode) parseUrl(baseUrl *url.URL, foundUrl string) string {
	href := strings.TrimSpace(foundUrl)
	parsedHref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return baseUrl.ResolveReference(parsedHref).String()
}

func (node *pageNode) extractUrls() ([]string, error) {
	baseURL, err := url.Parse(node.targetUrl)
	if err != nil {
		return node.foundUrls, err
	}
	tokenizer := html.NewTokenizer(bytes.NewReader(node.payload))
	for {
		switch tok := tokenizer.Next(); tok {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return node.foundUrls, nil
			}
			return node.foundUrls, err
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			switch token.Data {
			case "a":
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						node.foundUrls = append(node.foundUrls, node.parseUrl(baseURL, attr.Val))
					}
				}
			case "link":
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						node.foundUrls = append(node.foundUrls, node.parseUrl(baseURL, attr.Val))
					}
				}
			case "script":
				for _, attr := range token.Attr {
					if attr.Key == "src" && strings.HasSuffix(attr.Val, ".js") {
						node.foundUrls = append(node.foundUrls, node.parseUrl(baseURL, attr.Val))
					}
				}
			}
		}
	}
}
