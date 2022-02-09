package share

import (
	"context"
	"net/url"
)

func IsContextClosedError(err error) bool {
	switch e := err.(type) {
	case *url.Error:
		err = e.Err
	}

	switch err {
	case context.Canceled:
	case context.DeadlineExceeded:
	default:
		return false
	}

	return true
}
