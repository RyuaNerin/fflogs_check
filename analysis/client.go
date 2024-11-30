package analysis

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"ffxiv_check/share"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

const (
	maxRetries = 3
)

var (
	fflogsProxyPoolURL           string
	fflogsProxyPoolAuthorization string
)

func init() {
	godotenv.Load(".env")

	fflogsProxyPoolURL = os.Getenv("FFLOGS_PROXY_POOL_URL")
	fflogsProxyPoolAuthorization = os.Getenv("FFLOGS_PROXY_POOL_AUTHORIZATION")
}

func CallGraphQL(ctx context.Context, tmpl *template.Template, tmplData interface{}, respData interface{}) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = callGraphQLInner(ctx, tmpl, tmplData, respData)

		if err == nil {
			break
		}
		if share.IsContextClosedError(err) {
			return err
		}
		if i+1 < maxRetries {
			select {
			case <-time.After(3 * time.Second):
			case <-ctx.Done():
			}
		}
	}
	return err
}

func callGraphQLInner(ctx context.Context, tmpl *template.Template, tmplData interface{}, respData interface{}) error {
	sb := StrBufPool.Get().(*strings.Builder)
	defer StrBufPool.Put(sb)

	sb.Reset()
	err := tmpl.Execute(sb, tmplData)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return err
	}

	queryData := struct {
		Query string `json:"query"`
	}{
		Query: sb.String(),
	}

	buf := BytBufPool.Get().(*bytes.Buffer)
	defer BytBufPool.Put(buf)

	buf.Reset()
	err = jsoniter.NewEncoder(buf).Encode(&queryData)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return err
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fflogsProxyPoolURL, bytes.NewBuffer(buf.Bytes()))

	req.Header.Set("Content-Type", "application/json; encoding=utf-8")
	req.Header.Set("Authorization", fflogsProxyPoolAuthorization)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if !share.IsContextClosedError(err) {
			sentry.CaptureException(err)
			fmt.Printf("%+v\n", errors.WithStack(err))
		}
		return err
	}
	defer resp.Body.Close()

	err = jsoniter.NewDecoder(resp.Body).Decode(&respData)
	if err != io.EOF && err != nil {
		sentry.CaptureException(err)
		fmt.Printf("%+v\n", errors.WithStack(err))
		return err
	}

	return nil
}
