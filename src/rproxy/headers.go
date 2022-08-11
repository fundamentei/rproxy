package rproxy

import "net/http"

var (
	hContentLength = http.CanonicalHeaderKey("Content-Length")
	hAuthorization = http.CanonicalHeaderKey("Authorization")
)
