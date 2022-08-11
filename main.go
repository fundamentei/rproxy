package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"fundamentei.io/rproxy/src/rproxy"
)

var (
	// https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html#configuration-envvars-runtime
	isRunningInLambda = strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda")
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfgFile := rproxy.IfTrueElse(isRunningInLambda, "config.production.toml", "config.toml")
	cfg, err := rproxy.NewConfigFromFile(cfgFile)
	if err != nil {
		return err
	}

	warnIfMissingSharedKeySalt(cfg)

	// Make sure the provided config is valid by trying to parse it
	if _, _, err := net.SplitHostPort(cfg.General.Listen); err != nil {
		return err
	}

	listener, err := net.Listen("tcp", cfg.General.Listen)
	if err != nil {
		panic(err)
	}

	printListenInfo(cfg, listener.Addr())
	handler := rproxy.NewHandler(cfg)
	return http.Serve(listener, handler)
}

func warnIfMissingSharedKeySalt(cfg *rproxy.Config) {
	if cfg.General.SharedKeySalt == "" {
		log.Println(
			"WARNING: A shared key salt wasn't provided, what this means is that the traffic won't be fully encrypted. " +
				"Why? Because this proxy is known for its usage of the `Authorization` header to encrypt the responses, " +
				"meaning that the person authenticated will be able to leak the data. Also, it won't be always that the " +
				"`Authorization` header will be present, and you'll still want to encrypt all parts of the public " +
				"traffic as well. This is a security risk, and you should consider adding a salt.",
		)
	}
}

func printListenInfo(cfg *rproxy.Config, addr net.Addr) {
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		ip := tcpAddr.IP.String()
		if ip == "::" {
			ip = "127.0.0.1"
		}

		httpURL := fmt.Sprintf("http://%s:%d", ip, tcpAddr.Port)
		httpbinURL := "https://httpbin.org/json"

		log.Printf("Listening on %s", httpURL)
		log.Println("Try making a request to any of the following URLs:")
		for exampleID, exampleURL := range []string{
			httpbinURL,
			url.QueryEscape(httpbinURL),
			base64.RawURLEncoding.EncodeToString([]byte(httpbinURL)),
		} {
			log.Printf("\t%d. %s/%s", exampleID+1, httpURL, exampleURL)
		}
	}
}
