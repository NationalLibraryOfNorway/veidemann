package robots

import (
	"context"
	"log/slog"

	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
)

type EvaluatorServer struct {
	*Evaluator
}

// Assert that Evaluator implements the RobotsEvaluatorServer interface
var _ robotsevaluatorV1.RobotsEvaluatorServer = (*EvaluatorServer)(nil)

// IsAllowed implements the RobotsEvaluatorServer interface
func (e *EvaluatorServer) IsAllowed(ctx context.Context, req *robotsevaluatorV1.IsAllowedRequest) (*robotsevaluatorV1.IsAllowedReply, error) {
	uri := req.GetUri()
	userAgent := req.GetUserAgent()
	politenessConfig := req.GetPoliteness().GetPolitenessConfig()
	custom := politenessConfig.GetCustomRobots()
	minValiditySeconds := politenessConfig.GetMinimumRobotsValidityDurationS()
	policy := politenessConfig.GetRobotsPolicy()

	collectionId := req.GetCollectionRef().GetId()
	executionId := req.GetExecutionId()
	jobExecutionId := req.GetJobExecutionId()

	slog.Debug("IsAllowed request",
		"uri", uri,
		"userAgent", userAgent,
		"robotsPolicy", policy,
		"minValiditySeconds", minValiditySeconds,
		"customRobotsProvided", custom != "",
		"collectionId", collectionId,
		"executionId", executionId,
		"jobExecutionId", jobExecutionId,
	)

	ok, err := e.Evaluator.IsAllowed(ctx, &AllowedRequest{
		RobotsPolicy:       policy,
		MinValiditySeconds: minValiditySeconds,
		Uri:                uri,
		CustomRobots:       custom,
		UserAgent:          userAgent,
		CollectionId:       collectionId,
		ExecutionId:        executionId,
		JobExecutionId:     jobExecutionId,
	})
	if err != nil {
		slog.Error("IsAllowed error",
			"uri", uri,
			"userAgent", userAgent,
			"robotsPolicy", policy,
			"error", err,
		)
		return nil, err
	}

	return &robotsevaluatorV1.IsAllowedReply{
		IsAllowed: ok,
	}, nil
}
