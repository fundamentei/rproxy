package rproxy

import "github.com/BurntSushi/toml"

// Config is for representing all the "configurable"s
type Config struct {
	General  general  `toml:"general"`
	Limits   limits   `toml:"limits"`
	Timeouts timeouts `toml:"timeouts"`
}

type general struct {
	// This should be defined if you care about full encryption. Since this proxy is known for the use of `Authorization`
	// header to encrypt the traffic, by adding a salt that's only known between the proxy and the client, we can ensure
	// that nothing will leak
	SharedKeySalt        string   `toml:"sharedKeySalt"`
	IsEncryptedHeaderKey string   `toml:"isEncryptedHeaderKey"`
	AllowedHosts         []string `toml:"allowedHosts"`
	DisallowedHosts      []string `toml:"disallowedHosts"`
	AllowedMethods       []string `toml:"allowedMethods"`
	// If enabled it won't pass through CORS requests. Not implemented yet
	UnsafeCORS bool `toml:"unsafeCORS"`
	// Is the address that the proxy will listen to when running locally
	Listen string `toml:"listen"`
}

type limits struct {
	MaxRequestSizeInKB    uint32 `toml:"maxRequestSizeInKb"`
	MaxResponseSizeInKB   uint32 `toml:"maxResponseSizeInKb"`
	maxIdleConns          int    `toml:"maxIdleConns"`
	maxIdleConnsPerHost   int    `toml:"maxIdleConnsPerHost"`
	maxConnsPerHost       int    `toml:"maxConnsPerHost"`
	maxResponseHeaderInKB int64  `toml:"maxResponseHeaderInKb"`
}

// https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
// https://i.stack.imgur.com/OWegJ.png
type timeouts struct {
	ClientTimeoutMS uint32 `toml:"clientTimeoutMs"`
	// DialerTimeoutMS limits the time spent establishing a TCP connection (if a new one is needed)
	DialerTimeoutMS uint32 `toml:"dialerTimeoutMs"`
	// TLSHandshakeTimeoutMS limits the time spent performing the TLS handshake
	TLSHandshakeTimeoutMS uint32 `toml:"tlsHandshakeTimeoutMs"`
	// ResponseHeaderTimeoutMS limits the time spent reading the headers of the response
	ResponseHeaderTimeoutMS uint32 `toml:"responseHeaderTimeoutMs"`
	// ExpectContinueTimeoutMS limits the time the client will wait between sending the request headers when including an
	// Expect: 100-continue and receiving the go-ahead to send the body
	ExpectContinueTimeoutMS uint32 `toml:"expectContinueTimeoutMs"`
	// IdleConnTimeoutMS limits the amount of time an idle connection is kept in the connection pool
	IdleConnTimeoutMS uint32 `toml:"idleConnTimeoutMs"`
}

// NewConfigFromFile is for parsing the configuration from the specified file
func NewConfigFromFile(filepath string) (*Config, error) {
	cfg := &Config{}
	if _, err := toml.DecodeFile(filepath, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
