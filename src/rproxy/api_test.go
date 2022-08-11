package rproxy

import "testing"

func TestRequestURIToProxyURL(t *testing.T) {
	requestURIToProxyURL("/https%3A%2F%2Fproduction.api-lambda.fundamentei.io")
	requestURIToProxyURL("/https://production.api-lambda.fundamentei.io")
	requestURIToProxyURL("/aHR0cHM6Ly9wcm9kdWN0aW9uLmFwaS1sYW1iZGEuZnVuZGFtZW50ZWkuaW8=")
}
