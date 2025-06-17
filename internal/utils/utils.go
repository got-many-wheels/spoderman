package utils

import "net/url"

func IsValidUrl(s string) bool {
	_, err := url.Parse(s)
	if err != nil {
		return false
	}
	return true
}
