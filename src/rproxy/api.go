package rproxy

import (
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func realIP(r *http.Request) string {
	// The default is the originating IP, but we try to find better options because this is almost never the right IP
	if parts := strings.Split(r.RemoteAddr, ":"); len(parts) == 2 {
		return parts[0]
	}
	// We'll take the address from "X-Forwarded-For" if it's there
	if xff := strings.Trim(r.Header.Get("X-Forwarded-For"), ","); len(xff) > 0 {
		addrs := strings.Split(xff, ",")
		last := addrs[len(addrs)-1]
		if ip := net.ParseIP(last); ip != nil {
			return ip.String()
		}
	}
	// Parse X-Real-Ip header if it's there
	if xri := r.Header.Get("X-Real-Ip"); len(xri) > 0 {
		if ip := net.ParseIP(xri); ip != nil {
			return ip.String()
		}
	}
	return ""
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
