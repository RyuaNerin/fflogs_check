package fflogs

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

var (
	oauthHeaderLock    sync.Mutex
	oauthHeader        = ""
	oauthHeaderExpires time.Time

	clientId     = os.Getenv("FFLOGS_OAUTH2_CLIENT_ID")
	clientSecret = os.Getenv("FFLOGS_OAUTH2_CLIENT_SECRET")
)

func getOAuthHeader() (string, error) {
	oauthHeaderLock.Lock()
	defer oauthHeaderLock.Unlock()

	now := time.Now()
	if oauthHeader != "" && now.Before(oauthHeaderExpires) {
		return oauthHeader, nil
	}

	form := url.Values{
		"grant_type":    []string{"client_credentials"},
		"client_id":     []string{clientId},
		"client_secret": []string{clientSecret},
	}

	req, _ := http.NewRequest(
		"POST",
		"https://www.fflogs.com/oauth/token",
		strings.NewReader(form.Encode()),
	)
	req.Header = http.Header{
		"Content-Type": []string{"application/x-www-form-urlencoded"},
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer resp.Body.Close()

	var token struct {
		Error       string `json:"error"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	err = jsoniter.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if token.Error != "" {
		return "", errors.WithStack(errors.New(token.Error))
	}

	oauthHeader = fmt.Sprintf("Bearer %s", token.AccessToken)
	oauthHeaderExpires = now.Add(time.Duration(token.ExpiresIn) * time.Second)

	return oauthHeader, nil
}

func (inst *instance) callOAuthRequest(tmpl *template.Template, tmplData interface{}, respData interface{}) error {
	inst.bufQueryString.Reset()
	err := tmpl.Execute(&inst.bufQueryString, tmplData)
	if err != nil {
		return errors.WithStack(err)
	}

	queryData := struct {
		Query string `json:"query"`
	}{
		Query: inst.bufQueryString.String(),
	}

	inst.bufPostData.Reset()
	err = jsoniter.NewEncoder(&inst.bufPostData).Encode(&queryData)
	if err != nil {
		return err
	}

	authorization, err := getOAuthHeader()
	if err != nil {
		return err
	}

	req, _ := http.NewRequest(
		"POST",
		"https://ko.fflogs.com/api/v2/client",
		bytes.NewReader(inst.bufPostData.Bytes()),
	)
	req.Header = http.Header{
		"Authorization": []string{authorization},
		"Content-Type":  []string{"application/json; encoding=utf-8"},
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = jsoniter.NewDecoder(resp.Body).Decode(&respData)
	if err != io.EOF && err != nil {
		return err
	}

	return nil
}
