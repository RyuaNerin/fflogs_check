package share

import (
	"net/http"
	"os"

	"github.com/getsentry/sentry-go"
)

func init() {
	err := sentry.Init(
		sentry.ClientOptions{
			Dsn:           os.Getenv("SENTRY_DSN"),
			HTTPTransport: new(http.Transport),
		},
	)
	if err != nil {
		panic(err)
	}
}
