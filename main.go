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
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
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
	cfgFile := "config.toml"
	// If we're on AWS Lambda, loads the config from the appropriate file.
	if isRunningInLambda {
		if _, err := os.Stat("config.production.toml"); err == nil {
			cfgFile = "config.production.toml"
		} else {
			warnAboutMissingProductionConfigFile()
		}
	}
	cfg, err := rproxy.NewConfigFromFile(cfgFile)
	if err != nil {
		return err
	}

	warnIfMissingSharedKey(cfg)
	proxy := rproxy.NewHandler(cfg)

	if isRunningInLambda {
		lambda.Start(httpadapter.NewV2(proxy).ProxyWithContext)
		return nil
	}

	// Make sure the provided config is valid by trying to parse it
	if _, _, err := net.SplitHostPort(cfg.General.Listen); err != nil {
		return err
	}

	l, err := net.Listen("tcp", cfg.General.Listen)
	if err != nil {
		return err
	}

	printListenInfo(cfg, l.Addr())
	return http.Serve(l, proxy)
}

func warnAboutMissingProductionConfigFile() {
	log.Println(
		"WARNING: Looks like you're running on production and `config.production.toml` is missing. You should consider " +
			"adding it instead of `config.toml` which is mainly used for development with exposed keys. Also, if you're " +
			"running on AWS Lambda and the file is present in your local FS, make sure your deployment step isn't " +
			"configured to exclude it from the package.",
	)
}

func warnIfMissingSharedKey(cfg *rproxy.Config) {
	if cfg.General.SharedKey == "" {
		log.Println(
			"WARNING: A shared key wasn't provided, what this means is that the traffic won't be fully encrypted. " +
				"Why? Because this proxy is known for its usage of the `Authorization` header to encrypt the responses, " +
				"meaning that the person authenticated will be able to leak the data. Also, it won't be always that the " +
				"`Authorization` header will be present, and you'll still want to encrypt all parts of the public " +
				"traffic as well. This is a security risk, and you should consider adding a shared key.",
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
