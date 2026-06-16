/*
 * Copyright 2020 National Library of Norway.
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

package server

import (
	"context"
	"log/slog"

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/robotsevaluator"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/url"
)

// ApiServer implements the gRPC API for the browser controller. It is responsible for handling requests from the browser and forwarding them to the appropriate session. It also handles robots.txt evaluation and logging of crawl logs.
type ApiServer struct {
	browsercontrollerV2.UnimplementedBrowserControllerServer
	sessions        *session.Registry
	robotsEvaluator robotsevaluator.RobotsEvaluator
	logWriter       logwriter.LogWriter
}

func NewApiServer(sessions *session.Registry, robotsEvaluator robotsevaluator.RobotsEvaluator, logWriter logwriter.LogWriter) *ApiServer {
	a := &ApiServer{
		sessions:        sessions,
		robotsEvaluator: robotsEvaluator,
		logWriter:       logWriter,
	}
	return a
}

func (a *ApiServer) RegisterResource(ctx context.Context, request *browsercontrollerV2.RegisterResourceRequest) (*browsercontrollerV2.RegisterResourceReply, error) {
	proxyId := int(request.GetProxyId())

	slog.Debug("Register resource",
		"proxyId", request.GetProxyId(),
		"method", request.GetMethod(),
		"uri", request.GetUri(),
		"requestId", request.GetRequestId(),
		"crawlExecutionId", request.GetCrawlExecutionId(),
		"jobExecutionId", request.GetJobExecutionId(),
		"collectionRef", request.GetCollectionRef())

	if proxyId == 0 {
		return &browsercontrollerV2.RegisterResourceReply{
			Result: &browsercontrollerV2.RegisterResourceReply_Registered{
				Registered: &browsercontrollerV2.ResourceRegistered{
					CrawlExecutionId: request.GetCrawlExecutionId(),
					JobExecutionId:   request.GetJobExecutionId(),
					CollectionRef:    request.GetCollectionRef(),
				},
			},
		}, nil
	}

	sess := a.sessions.Get(proxyId)

	if sess == nil {
		slog.Warn("Cancelling nil session", "proxyId", request.GetProxyId(), "method", request.GetMethod(), "uri", request.GetUri())
		return &browsercontrollerV2.RegisterResourceReply{
			Result: &browsercontrollerV2.RegisterResourceReply_Cancel{Cancel: "Cancelled by browser controller"},
		}, nil
	}

	robotsRequest := &robotsevaluatorV1.IsAllowedRequest{
		JobExecutionId: sess.RequestedUrl.JobExecutionId,
		ExecutionId:    sess.RequestedUrl.ExecutionId,
		Uri:            request.GetUri(),
		UserAgent:      sess.UserAgent,
		Politeness:     sess.PolitenessConfig,
		CollectionRef:  sess.CrawlConfig.CollectionRef,
	}
	isAllowedByRobots := a.robotsEvaluator.IsAllowed(ctx, robotsRequest)

	slog.Debug("Robots evaluator result",
		"uri", request.GetUri(),
		"jeid", sess.RequestedUrl.JobExecutionId,
		"ceid", sess.RequestedUrl.ExecutionId,
		"policy", sess.PolitenessConfig.GetPolitenessConfig().GetRobotsPolicy(),
		"allowed", isAllowedByRobots)

	if !isAllowedByRobots {
		if request.GetRequestId() != "" {
			if req := sess.Requests.GetByRequestId(request.GetRequestId()); req != nil {
				req.GotNew = true
				if err := sess.NotifyRequest(req); err != nil {
					return nil, err
				}
			}
		}
		return &browsercontrollerV2.RegisterResourceReply{
			Result: &browsercontrollerV2.RegisterResourceReply_Cancel{Cancel: "Blocked by robots.txt"},
		}, nil
	}

	var req *requests.Request
	if request.RequestId == "" {
		normalizedURL := url.Normalize(request.Uri)
		switch request.Method {
		case "CONNECT":
			return &browsercontrollerV2.RegisterResourceReply{
				Result: &browsercontrollerV2.RegisterResourceReply_Registered{
					Registered: &browsercontrollerV2.ResourceRegistered{
						CrawlExecutionId: sess.RequestedUrl.ExecutionId,
						JobExecutionId:   sess.RequestedUrl.JobExecutionId,
						CollectionRef:    sess.CrawlConfig.CollectionRef,
					},
				},
			}, nil
		case "OPTIONS":
			req = sess.Requests.GetByUrl(normalizedURL, true)
			if req == nil {
				slog.Info("No new request found", "requestId", request.RequestId, "method", request.Method, "url", normalizedURL, "hasFulfilledRequest", sess.Requests.GetByUrl(normalizedURL, false) != nil)
			} else {
				req.GotNew = true
			}
		default:
			slog.Debug("New request from proxy without ID", "method", request.Method, "uri", request.Uri)
			return &browsercontrollerV2.RegisterResourceReply{
				Result: &browsercontrollerV2.RegisterResourceReply_Cancel{Cancel: "Cancelled by browser controller"},
			}, nil
		}
	} else {
		req = sess.Requests.GetByRequestId(request.RequestId)
		if req == nil {
			slog.Warn("No request found", "requestId", request.RequestId)
		} else {
			req.GotNew = true
			if err := sess.NotifyRequest(req); err != nil {
				return nil, err
			}
		}
	}

	return &browsercontrollerV2.RegisterResourceReply{
		Result: &browsercontrollerV2.RegisterResourceReply_Registered{
			Registered: &browsercontrollerV2.ResourceRegistered{
				CrawlExecutionId: sess.RequestedUrl.ExecutionId,
				JobExecutionId:   sess.RequestedUrl.JobExecutionId,
				CollectionRef:    sess.CrawlConfig.CollectionRef,
			},
		},
	}, nil
}

func (a *ApiServer) CompleteResource(ctx context.Context, request *browsercontrollerV2.CompleteResourceRequest) (*browsercontrollerV2.CompleteResourceReply, error) {
	slog.Debug("Request completed", "statusCode", request.CrawlLog.StatusCode, "method", request.CrawlLog.Method, "uri", request.CrawlLog.RequestedUri)

	proxyId := int(request.ProxyId)

	if proxyId == 0 {
		if !request.Cached && request.GetCrawlLog().GetWarcId() != "" {
			if err := a.logWriter.WriteCrawlLog(ctx, request.GetCrawlLog()); err != nil {
				slog.Error("Failed writing crawlLog for direct session", "error", err)
			}
		}
		return &browsercontrollerV2.CompleteResourceReply{}, nil
	}

	sess := a.sessions.Get(proxyId)

	if sess == nil {
		slog.Warn("Missing session", "warcId", request.GetCrawlLog().GetWarcId(), "method", request.GetCrawlLog().GetMethod(), "uri", request.GetCrawlLog().GetRequestedUri())
		return &browsercontrollerV2.CompleteResourceReply{}, nil
	}

	if sess.Requests == nil {
		slog.Warn("Missing request registry", "warcId", request.GetCrawlLog().GetWarcId(), "method", request.GetCrawlLog().GetMethod(), "uri", request.GetCrawlLog().GetRequestedUri())
		return &browsercontrollerV2.CompleteResourceReply{}, nil
	}

	req := sess.Requests.GetByRequestId(request.RequestId)
	if req == nil {
		switch request.GetCrawlLog().GetMethod() {
		case "OPTIONS", "CONNECT":
		default:
			slog.Error("Missing reqId", "method", request.GetCrawlLog().GetMethod(), "statusCode", request.GetCrawlLog().GetStatusCode(), "uri", request.GetCrawlLog().GetRequestedUri(), "cached", request.Cached)
		}
		return &browsercontrollerV2.CompleteResourceReply{}, nil
	}

	req.CrawlLog = request.GetCrawlLog()
	if request.Cached {
		if initialRequest := sess.Requests.InitialRequest(); initialRequest != nil && initialRequest.RequestId == req.RequestId {
			slog.Info("Aborting fetch")
			_ = sess.AbortFetch()
		}
		req.FromCache = true
	}
	req.GotComplete = true
	if err := sess.NotifyRequest(req); err != nil {
		return nil, err
	}

	return &browsercontrollerV2.CompleteResourceReply{}, nil
}
