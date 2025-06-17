package crawler

import (
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	once      sync.Once
	netClient *http.Client
)

func newNetClient() *http.Client {
	once.Do(func() {
		var netTransport = &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 2 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 2 * time.Second,
		}
		netClient = &http.Client{
			Timeout:   time.Second * 2,
			Transport: netTransport,
		}
	})
	return netClient
}
