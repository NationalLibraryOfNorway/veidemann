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

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"github.com/google/uuid"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
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
		logger.LogWithPhase("INIT").Errorf("Could not init logger: %s", err)
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
	defer grpcServices.Close()

	p, client := newProxy(grpcServices)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		_ = p.Shutdown(shutdownCtx)
	}()
	control := recorderproxy.NewControl(grpcServices, p)
	recovered, err := control.RecoverSlot(ctx, &recorderV1.RecoverSlotRequest{ProxyId: 1})
	if err != nil {
		panic(err)
	}
	token := recovered.NextToken
	urls := flag.Args()
	if len(urls) == 0 || viper.GetBool("keep-running") {
		urls = append(urls, "")
	}
	for _, uri := range urls {
		if ctx.Err() != nil {
			break
		}
		id := uuid.NewString()
		begin, err := control.BeginHarvest(ctx, &recorderV1.BeginHarvestRequest{
			AdmissionToken: token, ProxyId: 1, AttemptId: id, RequestedUri: uri, CrawlExecutionId: "proxyutil",
			CollectionRef: &configV1.ConfigRef{Kind: configV1.Kind_collection, Id: "proxyutil"},
		})
		if err != nil {
			panic(err)
		}
		token = begin.NextToken
		if uri != "" {
			getWithProxy(ctx, uri, id, client)
		} else if viper.GetBool("keep-running") {
			logger.LogWithPhase("HARVEST").Infof("Listening for requests with X-Veidemann-Attempt-Id: %s", id)
			<-ctx.Done()
		}
		finishCtx, cancelFinish := context.WithTimeout(context.WithoutCancel(ctx), 35*time.Second)
		reply, err := control.FinishPage(finishCtx, &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: id}, nil)
		cancelFinish()
		if err != nil || !reply.GetReusable() {
			logger.LogWithPhase("HARVEST").WithError(err).Errorf("Finish harvest: %v", reply)
			return
		}
		logger.LogWithPhase("HARVEST").Info(reply)
	}
}

func getWithProxy(ctx context.Context, uri, id string, client *http.Client) {
	statusCode, got, err := get(ctx, uri, id, client, 5000*time.Second)

	logger.LogWithPhase("http_client").Infof("Status: %v", statusCode)

	if len(got) > 0 {
		logger.LogWithPhase("http_client").Printf("Content: %s... (%d bytes)\n\n", got[:min(10, len(got))], len(got))
	}

	if err != nil {
		logger.LogWithPhase("http_client").WithError(err).Error("Invalid request")
	}
}

func newProxy(mock *testutil.GrpcServiceMock) (*recorderproxy.RecorderProxy, *http.Client) {
	host := viper.GetString("interface")
	port := viper.GetInt("port")
	nextProxy := viper.GetString("proxy")
	p := recorderproxy.NewRecorderProxy(1, mock.ContentWriterClient, mock.DnsResolverClient, nextProxy)

	ln, err := p.Listen(host, port)
	if err != nil {
		panic(err)
	}
	go func() {
		err := p.Serve(ln)
		if err != nil {
			logger.LogWithPhase("connection").WithError(err).Error("Error starting recorder proxy")
		}
	}()

	proxyUrl, _ := url.Parse("http://" + ln.Addr().String())
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, Proxy: http.ProxyURL(proxyUrl), DisableKeepAlives: false}
	client := &http.Client{Transport: tr}

	return p, client
}
