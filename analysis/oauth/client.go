package oauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
)

type Client struct {
	clientID     string
	clientSecret string

	headerLock    sync.Mutex
	headerValue   string
	headerExpires time.Time
}

func New(oauthID string, oauthSecret string) *Client {
	return &Client{
		clientID:     oauthID,
		clientSecret: oauthSecret,
	}
}

func (c *Client) Reset() {
	c.headerLock.Lock()
	c.headerValue = ""
	c.headerLock.Unlock()
}

func (c *Client) NewRequest(ctx context.Context, method string, urlStr string, body io.Reader) (*http.Request, bool) {
	c.headerLock.Lock()
	defer c.headerLock.Unlock()

	now := time.Now()
	if c.headerValue == "" || now.After(c.headerExpires) {
		form := url.Values{
			"grant_type":    []string{"client_credentials"},
			"client_id":     []string{c.clientID},
			"client_secret": []string{c.clientSecret},
		}

		req, _ := http.NewRequest(
			"POST",
			"https://www.fflogs.com/oauth/token",
			strings.NewReader(form.Encode()),
		)
		req.Header = http.Header{
			"Content-Type": []string{"application/x-www-form-urlencoded"},
		}
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			sentry.CaptureException(err)
			return nil, false
		}
		defer resp.Body.Close()

		var token struct {
			Error       string `json:"error"`
			AccessToken string `json:"access_token"`
			ExpiresIn   int64  `json:"expires_in"`
		}
		err = jsoniter.NewDecoder(resp.Body).Decode(&token)
		if err != nil {
			sentry.CaptureException(err)
			return nil, false
		}
		if token.Error != "" {
			return nil, false
		}

		c.headerValue = fmt.Sprintf("Bearer %s", token.AccessToken)
		c.headerExpires = now.Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		sentry.CaptureException(err)
		return nil, false
	}
	req.Header = http.Header{
		"Authorization": []string{c.headerValue},
		"Content-Type":  []string{"application/json; encoding=utf-8"},
	}
	req = req.WithContext(ctx)

	return req, true
}
