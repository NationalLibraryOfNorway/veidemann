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
	"net/http"

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/robotsevaluator"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/url"
)

const (
	cancelledByBrowserController = "Cancelled by browser controller"
	blockedByRobots              = "Blocked by robots.txt"
	canceledByBrowserStatusCode  = -5011
)

func canceledRegistrationReply() *browsercontrollerV2.RegisterResourceReply {
	return &browsercontrollerV2.RegisterResourceReply{
		Result: &browsercontrollerV2.RegisterResourceReply_Cancel{Cancel: cancelledByBrowserController},
	}
}

func isCanceledByBrowser(request *browsercontrollerV2.CompleteResourceRequest) bool {
	return request.GetCrawlLog().GetStatusCode() == canceledByBrowserStatusCode ||
		request.GetCrawlLog().GetError().GetMsg() == "CANCELED_BY_BROWSER"
}

// ApiServer implements the gRPC API for the browser controller. It is responsible for handling requests from the browser and forwarding them to the appropriate session. It also handles robots.txt evaluation and logging of crawl logs.
type ApiServer struct {
	browsercontrollerV2.UnimplementedBrowserControllerServer
	sessions        activeSessionLookup
	robotsEvaluator robotsevaluator.RobotsEvaluator
	logWriter       logwriter.LogWriter
}

type activeSessionLookup interface {
	GetActive(int) *session.Session
}

func NewApiServer(sessions activeSessionLookup, robotsEvaluator robotsevaluator.RobotsEvaluator, logWriter logwriter.LogWriter) *ApiServer {
	a := &ApiServer{
		sessions:        sessions,
		robotsEvaluator: robotsEvaluator,
		logWriter:       logWriter,
	}
	return a
}

func (a *ApiServer) RegisterResource(ctx context.Context, request *browsercontrollerV2.RegisterResourceRequest) (*browsercontrollerV2.RegisterResourceReply, error) {
	proxyId := int(request.GetProxyId())

	log := slog.With(
		"proxyId", request.GetProxyId(),
		"method", request.GetMethod(),
		"uri", request.GetUri(),
		"requestId", request.GetRequestId(),
		"crawlExecutionId", request.GetCrawlExecutionId(),
		"jobExecutionId", request.GetJobExecutionId(),
		"collectionRef", request.GetCollectionRef(),
		"component", "server.RegisterResource",
	)

	log.Debug("Register resource request")

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

	sess := a.sessions.GetActive(proxyId)
	if sess == nil {
		log.Debug("Cancelling unavailable session")
		return canceledRegistrationReply(), nil
	}
	if request.GetCrawlExecutionId() != "" && request.GetCrawlExecutionId() != sess.RequestedUrl.GetExecutionId() {
		log.Debug("Cancelling stale resource registration", "activeExecutionId", sess.RequestedUrl.GetExecutionId())
		return canceledRegistrationReply(), nil
	}
	reply := &browsercontrollerV2.RegisterResourceReply{
		Result: &browsercontrollerV2.RegisterResourceReply_Registered{
			Registered: &browsercontrollerV2.ResourceRegistered{
				CrawlExecutionId: sess.RequestedUrl.ExecutionId,
				JobExecutionId:   sess.RequestedUrl.JobExecutionId,
				CollectionRef:    sess.CrawlConfig.CollectionRef,
			},
		},
	}

	if request.RequestId == "" {
		switch request.Method {

		case http.MethodConnect:
			return reply, nil

		case http.MethodOptions:
			normalizedURL := url.Normalize(request.Uri)
			if _, err := sess.CompleteOptionsResource(normalizedURL); err != nil {
				return nil, err
			}
			return reply, nil

		default:
			return canceledRegistrationReply(), nil
		}
	}

	isAllowedByRobots := a.isAllowedByRobots(ctx, sess, request.GetUri())
	if a.sessions.GetActive(proxyId) != sess {
		log.Debug("Cancelling resource registration for inactive session")
		return canceledRegistrationReply(), nil
	}
	if !isAllowedByRobots {
		if _, err := sess.RejectResource(request.GetRequestId()); err != nil {
			return nil, err
		}
		return &browsercontrollerV2.RegisterResourceReply{
			Result: &browsercontrollerV2.RegisterResourceReply_Cancel{Cancel: blockedByRobots},
		}, nil
	}

	found, err := sess.RegisterResource(request.GetRequestId())
	if err != nil {
		return nil, err
	}
	if !found {
		log.Warn("No request found in registry")
	}

	return reply, nil
}

func (a *ApiServer) CompleteResource(ctx context.Context, request *browsercontrollerV2.CompleteResourceRequest) (*browsercontrollerV2.CompleteResourceReply, error) {
	proxyID := int(request.GetProxyId())

	log := slog.With(
		"id", request.GetRequestId(),
		"statusCode", request.GetCrawlLog().GetStatusCode(),
		"method", request.GetCrawlLog().GetMethod(),
		"uri", request.GetCrawlLog().GetRequestedUri(),
		"proxyID", proxyID,
		"cached", request.GetCached(),
		"component", "server.CompleteResource",
	)

	log.Debug("Complete resource request")

	if proxyID == 0 {
		if !request.GetCached() && request.GetCrawlLog().GetWarcId() != "" {
			if err := a.logWriter.WriteCrawlLog(ctx, request.GetCrawlLog()); err != nil {
				log.Error("Failed writing crawlLog", "error", err)
			}
		}
		return &browsercontrollerV2.CompleteResourceReply{}, nil
	}

	sess := a.sessions.GetActive(proxyID)
	if sess == nil {
		if isCanceledByBrowser(request) {
			log.Debug("Discarding late browser-canceled resource for released session")
		} else {
			log.Warn("Missing session", "request", request)
		}
		return &browsercontrollerV2.CompleteResourceReply{}, nil
	}
	if executionID := request.GetCrawlLog().GetExecutionId(); executionID != "" && executionID != sess.RequestedUrl.GetExecutionId() {
		if isCanceledByBrowser(request) {
			log.Debug("Discarding stale browser-canceled resource", "activeExecutionId", sess.RequestedUrl.GetExecutionId())
		} else {
			log.Warn("Discarding stale resource completion", "activeExecutionId", sess.RequestedUrl.GetExecutionId())
		}
		return &browsercontrollerV2.CompleteResourceReply{}, nil
	}

	completion, err := sess.RecordResourceCompletion(request.GetRequestId(), request.GetCrawlLog(), request.GetCached())
	if err != nil {
		return nil, err
	}
	if !completion.Found {
		switch request.GetCrawlLog().GetMethod() {
		case http.MethodOptions, http.MethodConnect:
			log.Warn("Completed connect request")

		default:
			log.Warn("Request not found in registry")
		}
		return &browsercontrollerV2.CompleteResourceReply{}, nil
	}

	return &browsercontrollerV2.CompleteResourceReply{}, nil
}

func (a *ApiServer) isAllowedByRobots(ctx context.Context, sess *session.Session, uri string) bool {
	isAllowedByRobots := a.robotsEvaluator.IsAllowed(ctx, &robotsevaluatorV1.IsAllowedRequest{
		JobExecutionId: sess.RequestedUrl.JobExecutionId,
		ExecutionId:    sess.RequestedUrl.ExecutionId,
		Uri:            uri,
		UserAgent:      sess.UserAgent,
		Politeness:     sess.PolitenessConfig,
		CollectionRef:  sess.CrawlConfig.CollectionRef,
	})

	slog.Debug("Robots evaluator result",
		"uri", uri,
		"jeid", sess.RequestedUrl.JobExecutionId,
		"ceid", sess.RequestedUrl.ExecutionId,
		"policy", sess.PolitenessConfig.GetPolitenessConfig().GetRobotsPolicy(),
		"allowed", isAllowedByRobots)

	return isAllowedByRobots
}
