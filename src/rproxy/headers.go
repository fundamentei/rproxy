package rproxy

import "net/http"

var (
	hContentEncoding = http.CanonicalHeaderKey("Content-Encoding")
	hContentLength   = http.CanonicalHeaderKey("Content-Length")
	hAuthorization   = http.CanonicalHeaderKey("Authorization")
	hXForwardedFor   = http.CanonicalHeaderKey("X-Forwarded-For")
)
