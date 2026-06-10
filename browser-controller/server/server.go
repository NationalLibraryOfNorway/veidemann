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
	gerr "errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"runtime/debug"
	"time"

	browsercontrollerV1 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v1"
	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/errors"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/robotsevaluator"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/url"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApiServer is the gRPC api endpoint for the Browser Controller
type ApiServer struct {
	browsercontrollerV1.UnimplementedBrowserControllerServer
	sessions        *session.Registry
	ln              net.Listener
	listenAddr      net.Addr
	addr            string
	grpcServer      *grpc.Server
	robotsEvaluator robotsevaluator.RobotsEvaluator
	logWriter       logwriter.LogWriter
}

// NewApiServer returns a new instance of ApiServer listening on the given port
func NewApiServer(listenInterface string, listenPort int, sessions *session.Registry, robotsEvaluator robotsevaluator.RobotsEvaluator, logWriter logwriter.LogWriter) *ApiServer {
	a := &ApiServer{
		sessions:        sessions,
		addr:            fmt.Sprintf("%s:%d", listenInterface, listenPort),
		robotsEvaluator: robotsEvaluator,
		logWriter:       logWriter,
	}
	return a
}

func (a *ApiServer) Start() error {
	ln, err := net.Listen("tcp", a.addr)
	if err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	a.ln = ln
	a.listenAddr = ln.Addr()

	opts := []grpc.ServerOption{
		grpc.StatsHandler(&myStatsHandler{}),
	}
	a.grpcServer = grpc.NewServer(opts...)
	browsercontrollerV1.RegisterBrowserControllerServer(a.grpcServer, a)

	slog.Info("API server listening", "address", a.addr)
	return a.grpcServer.Serve(ln)
}

func (a *ApiServer) Close() {
	slog.Info("Shutting down API server")

	// Set a timer to fire a hard shutdown if graceful shutdown doesn't return
	t := time.AfterFunc(time.Minute, a.grpcServer.Stop)

	// Do a graceful shutdown
	a.grpcServer.GracefulStop()
	t.Stop()
}

// Implements BrowserController
func (a *ApiServer) Do(stream browsercontrollerV1.BrowserController_DoServer) (err error) {
	// Ensure that bugs in implementation is logged and handled
	defer func() {
		if r := recover(); r != nil {
			var fetchError errors.FetchError
			switch v := r.(type) {
			case errors.FetchError:
				fetchError = v
			case error:
				fetchError = errors.New(-5, "Runtime error", v.Error())
			default:
				fetchError = errors.New(-5, "Runtime error", fmt.Sprintf("%s", v))
			}
			slog.Error("API Server recovered from panic", "error", fetchError)

			// Add stacktrace to error
			fetchError.CommonsError().Detail += "\n" + string(debug.Stack())
			err = fetchError
		}
	}()

	var span opentracing.Span
	defer func() {
		if span != nil {
			span.Finish()
		}
	}()
	var sess *session.Session
	var req *requests.Request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		request, err := Recv(ctx, stream.Recv)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			switch {
			case gerr.Is(err, context.DeadlineExceeded):
				return status.Errorf(codes.DeadlineExceeded, "Deadline exceeded while waiting for proxy request: %v", err)
			case gerr.Is(err, context.Canceled):
				return status.Errorf(codes.Canceled, "Browser controller canceled request: %v", err)
			default:
				return status.Errorf(codes.Unknown, "Unknown error while waiting for proxy request: %v", err)
			}
		}

		switch v := request.Action.(type) {
		case *browsercontrollerV1.DoRequest_New:
			if v.New.ProxyId == 0 {
				sess, err = a.sessions.NewDirectSession(v.New.Uri, v.New.CrawlExecutionId, v.New.JobExecutionId)
				if err != nil {
					return fmt.Errorf("could not create session for 0-proxy %w", err)
				}
				if err = Send(stream.Send, &browsercontrollerV1.DoReply{
					Action: &browsercontrollerV1.DoReply_New{
						New: &browsercontrollerV1.NewReply{
							CrawlExecutionId: v.New.CrawlExecutionId,
							JobExecutionId:   v.New.JobExecutionId,
							CollectionRef:    v.New.CollectionRef,
						},
					},
				}); err != nil {
					return err
				}
				span, _ = opentracing.StartSpanFromContext(stream.Context(), "new direct request",
					opentracing.Tag{Key: "http.method", Value: v.New.GetMethod()},
					opentracing.Tag{Key: "http.url", Value: v.New.Uri},
				)
				continue
			}

			sess = a.sessions.Get(int(v.New.ProxyId))
			if sess == nil {
				slog.Warn("Cancelling nil session", "proxy", v.New.ProxyId, "method", v.New.Method, "url", v.New.Uri)
				if err = Send(stream.Send, &browsercontrollerV1.DoReply{
					Action: &browsercontrollerV1.DoReply_Cancel{
						Cancel: "Cancelled by browser controller",
					},
				}); err != nil {
					return err
				}
				continue
			} else {
				cancel()
				span, ctx = opentracing.StartSpanFromContext(sess.Context(), "new request",
					opentracing.Tag{Key: "http.method", Value: v.New.GetMethod()},
					opentracing.Tag{Key: "http.url", Value: v.New.Uri},
					opentracing.Tag{Key: "proxy.id", Value: v.New.ProxyId},
				)
			}

			slog.Debug("Check robots",
				"url", v.New.Uri,
				"jeid", sess.RequestedUrl.JobExecutionId,
				"ceid", sess.RequestedUrl.ExecutionId,
				"policy", sess.PolitenessConfig.GetPolitenessConfig().GetRobotsPolicy())

			robotsRequest := &robotsevaluatorV1.IsAllowedRequest{
				JobExecutionId: sess.RequestedUrl.JobExecutionId,
				ExecutionId:    sess.RequestedUrl.ExecutionId,
				Uri:            v.New.Uri,
				UserAgent:      sess.UserAgent,
				Politeness:     sess.PolitenessConfig,
				CollectionRef:  sess.CrawlConfig.CollectionRef,
			}

			isAllowed := a.robotsEvaluator.IsAllowed(ctx, robotsRequest)
			if !isAllowed {
				slog.Info("URI blocked by robots.txt", "url", v.New.Uri, "ceid", sess.RequestedUrl.ExecutionId)
				req = sess.Requests.GetByRequestId(v.New.RequestId)
				if req == nil {
					slog.Warn("No request found", "requestId", v.New.RequestId)
				} else {
					req.GotNew = true
					if err := sess.Notify(req.RequestId); err != nil {
						return err
					}
				}
				if err = Send(stream.Send, &browsercontrollerV1.DoReply{
					Action: &browsercontrollerV1.DoReply_Cancel{
						Cancel: "Blocked by robots.txt",
					},
				}); err != nil {
					return err
				}
				continue
			}

			if v.New.RequestId == "" {
				switch v.New.Method {
				case "CONNECT":
					reply := &browsercontrollerV1.DoReply{
						Action: &browsercontrollerV1.DoReply_New{
							New: &browsercontrollerV1.NewReply{
								CrawlExecutionId: sess.RequestedUrl.ExecutionId,
								JobExecutionId:   sess.RequestedUrl.JobExecutionId,
								CollectionRef:    sess.CrawlConfig.CollectionRef,
							},
						},
					}
					if err = Send(stream.Send, reply); err != nil {
						return err
					}
					continue
				case "OPTIONS":
					Url := url.Normalize(v.New.Uri)
					req = sess.Requests.GetByUrl(Url, true)
					if req == nil {
						slog.Debug("No new request found",
							"requestId", v.New.RequestId,
							"method", v.New.Method,
							"url", Url,
							"hasFulfilledRequest", sess.Requests.GetByUrl(Url, false) != nil)
					} else {
						req.GotNew = true
					}

				default:
					// The request was not intercepted. Probably from a subsystem in browser e.g. a service worker
					// We cancel this request at the moment
					// TODO: revisit this to see if we can do anything smarter
					slog.Debug("New request from proxy without ID", "method", v.New.Method, "url", v.New.Uri)
					if err = Send(stream.Send, &browsercontrollerV1.DoReply{
						Action: &browsercontrollerV1.DoReply_Cancel{
							Cancel: "Cancelled by browser controller",
						},
					}); err != nil {
						return err
					}
					continue
				}
			} else {
				req = sess.Requests.GetByRequestId(v.New.RequestId)
				if req == nil {
					slog.Warn("No request found", "requestId", v.New.RequestId)
				} else {
					req.GotNew = true
					if err := sess.Notify(req.RequestId); err != nil {
						return err
					}
				}
			}

			reply := &browsercontrollerV1.NewReply{
				CrawlExecutionId: sess.RequestedUrl.ExecutionId,
				JobExecutionId:   sess.RequestedUrl.JobExecutionId,
				CollectionRef:    sess.CrawlConfig.CollectionRef,
			}
			replacementScript := sess.GetReplacementScript(v.New.Uri)
			if replacementScript != nil {
				reply.ReplacementScript = replacementScript
			}
			if err := Send(stream.Send, &browsercontrollerV1.DoReply{Action: &browsercontrollerV1.DoReply_New{New: reply}}); err != nil {
				return err
			}
		case *browsercontrollerV1.DoRequest_Notify:
			if sess == nil {
				slog.Warn("Notify without session", "activity", v.Notify.GetActivity())
				return status.Errorf(codes.Canceled, "Session is cancelled")
			}
			if req != nil {
				if err := sess.Notify(req.RequestId); err != nil {
					return err
				}
			}
		case *browsercontrollerV1.DoRequest_Completed:
			slog.Debug("Request completed",
				"statusCode", v.Completed.CrawlLog.StatusCode,
				"method", v.Completed.CrawlLog.Method,
				"requestedUri", v.Completed.CrawlLog.RequestedUri)
			if sess == nil || (sess.Id != 0 && req == nil) {
				slog.Info("Missing session", "warcId", v.Completed.CrawlLog.WarcId, "method", v.Completed.CrawlLog.Method, "requestedUri", v.Completed.CrawlLog.RequestedUri)
			}
			if req == nil {
				if sess.Id == 0 {
					if !v.Completed.Cached && v.Completed.CrawlLog.GetWarcId() != "" {
						if err := a.logWriter.WriteCrawlLog(stream.Context(), v.Completed.CrawlLog); err != nil {
							slog.Error("Failed writing crawlLog for direct session", "error", err)
						}
					}
				} else {
					switch v.Completed.CrawlLog.Method {
					case "OPTIONS":
					case "CONNECT":
					default:
						slog.Error("Missing reqId",
							"method", v.Completed.CrawlLog.Method,
							"statusCode", v.Completed.CrawlLog.StatusCode,
							"requestedUri", v.Completed.CrawlLog.RequestedUri,
							"cached", v.Completed.Cached)

					}
				}
			} else {
				req.CrawlLog = v.Completed.CrawlLog
				if v.Completed.Cached {
					if sess.Requests.InitialRequest().RequestId == req.RequestId {
						slog.Debug("Aborting fetch")
						_ = sess.AbortFetch()
					}
					req.FromCache = true
				}
				req.GotComplete = true
				if err := sess.Notify(req.RequestId); err != nil {
					return err
				}
			}
		}
	}
}
