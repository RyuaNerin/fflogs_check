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
		ResponseHeaderTimeout: 2 * time.Minute,
		TLSHandshakeTimeout:   2 * time.Minute,
		IdleConnTimeout:       2 * time.Minute,
		ExpectContinueTimeout: 2 * time.Minute,
	}
	http.DefaultClient.Timeout = 2 * time.Minute
	http.DefaultClient.Transport = tr
	if conn, err := net.DialTimeout("tcp", "127.0.0.1:50000", time.Second); err == nil {
		conn.Close()

		url, _ := url.Parse("http://127.0.0.1:50000")
		tr.Proxy = http.ProxyURL(url)
	}
}
