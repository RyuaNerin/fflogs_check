package share

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

func init() {
	tr := &http.Transport{
		MaxConnsPerHost:       0,
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   64,
		ResponseHeaderTimeout: 10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		ExpectContinueTimeout: 30 * time.Second,
	}
	http.DefaultClient.Timeout = 1 * time.Minute
	http.DefaultClient.Transport = tr
	if conn, err := net.DialTimeout("tcp", "127.0.0.1:50000", time.Second); err == nil {
		conn.Close()

		url, _ := url.Parse("http://127.0.0.1:50000")
		tr.Proxy = http.ProxyURL(url)
	}
}
