package rproxy

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

func realIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-Ip")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

// requestURIToProxyURL is for fetching the destination URL from the request. It allows the following inputs:
// /https%3A%2F%2Fproduction.api-lambda.fundamentei.io
// /https://production.api-lambda.fundamentei.io
// /aHR0cHM6Ly9wcm9kdWN0aW9uLmFwaS1sYW1iZGEuZnVuZGFtZW50ZWkuaW8=
func requestURIToProxyURL(requestURI string) (*url.URL, error) {
	// Removes the leading slash from destination URL that may come up with the request
	if strings.HasPrefix(requestURI, "/") {
		requestURI = strings.TrimPrefix(requestURI, "/")
	}
	d, err := base64.RawStdEncoding.DecodeString(requestURI)
	if err != nil {
		decodedURI, err := url.QueryUnescape(requestURI)
		if err != nil {
			return nil, err
		}
		return url.Parse(decodedURI)
	}
	return url.Parse(string(d))
}
