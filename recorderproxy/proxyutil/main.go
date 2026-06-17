/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	acceptAllCerts = &tls.Config{InsecureSkipVerify: true}
	recorderProxy  *recorderproxy.RecorderProxy
	client         *http.Client
)

func main() {
	flag.BoolP("help", "h", false, "Usage instructions")
	flag.String("log-level", "info", "log level, available levels are panic, fatal, error, warn, info, debug and trace")
	flag.String("log-formatter", "text", "log formatter, available values are text, logfmt and json")
	flag.Bool("log-method", false, "log method name")
	flag.BoolP("keep-running", "r", false, "keep proxy running until ^C")
	flag.String("interface", "", "interface this proxy listens to. No value means all interfaces.")
	flag.IntP("port", "p", 0, "port this proxy listens to. If port is 0, a random port is used")
	flag.StringP("proxy", "x", "", "host:port to secondary proxy")
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)

	os.Setenv("JAEGER_SAMPLER_TYPE", "const")
	os.Setenv("JAEGER_SAMPLER_PARAM", "1")

	err := logger.InitLog(viper.GetString("log-level"), viper.GetString("log-formatter"), viper.GetBool("log-method"))
	if err != nil {
		logger.LogWithComponent("INIT").Errorf("Could not init logger: %s", err)
		flag.Usage()
		os.Exit(1)
	}

	if viper.GetBool("help") {
		flag.Usage()
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	grpcServices := testutil.NewGrpcServiceMock()

	if flag.NArg() == 0 {
		recorderProxy, _ = newProxy(grpcServices)
		defer recorderProxy.Shutdown(ctx)
	} else {
		for _, url := range flag.Args() {
			getWithProxy(ctx, url, grpcServices)
		}
	}

	if viper.GetBool("keep-running") {
		<-ctx.Done()
	}
}

func getWithProxy(ctx context.Context, url string, grpcServices *testutil.GrpcServiceMock) {
	recorderProxy, client = newProxy(grpcServices)
	defer recorderProxy.Shutdown(ctx)

	clientTimeout := 5000 * time.Second

	statusCode, got, err := get(url, client, clientTimeout)
	if grpcServices.DoneBC != nil {
		<-grpcServices.DoneBC
	}
	if grpcServices.DoneCW != nil {
		<-grpcServices.DoneCW
	}

	logger.LogWithComponent("CLIENT").Infof("Status: %v", statusCode)

	if len(got) > 0 {
		logger.LogWithComponent("CLIENT").Printf("Content: %s... (%d bytes)\n\n", got[0:10], len(got))
	}

	if err != nil {
		logger.LogWithComponent("CLIENT").WithError(err).Error("Invalid request")
	}
}

func newProxy(mock *testutil.GrpcServiceMock) (*recorderproxy.RecorderProxy, *http.Client) {
	host := viper.GetString("interface")
	port := viper.GetInt("port")
	nextProxy := viper.GetString("proxy")
	p := recorderproxy.NewRecorderProxy(0, mock.ClientConn, nextProxy)

	ln, err := p.Listen(host, port)
	if err != nil {
		panic(err)
	}
	go func() {
		err := p.Serve(ln)
		if err != nil {
			logger.LogWithComponent("PROXY").WithError(err).Error("Error starting recorder proxy")
		}
	}()

	proxyUrl, _ := url.Parse("http://" + ln.Addr().String())
	tr := &http.Transport{TLSClientConfig: acceptAllCerts, Proxy: http.ProxyURL(proxyUrl), DisableKeepAlives: false}
	client := &http.Client{Transport: tr}

	return p, client
}
