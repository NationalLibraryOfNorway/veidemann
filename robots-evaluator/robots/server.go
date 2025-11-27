package robots

import (
	"context"

	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
)

type EvaluatorServer struct {
	robotsevaluatorV1.UnimplementedRobotsEvaluatorServer

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

	ok, err := e.Evaluator.IsAllowed(ctx, &AllowedRequest{
		RobotsPolicy:       policy,
		MinValiditySeconds: minValiditySeconds,
		Uri:                uri,
		CustomRobots:       custom,
		UserAgent:          userAgent,
	})
	if err != nil {
		return nil, err
	}

	return &robotsevaluatorV1.IsAllowedReply{
		IsAllowed: ok,
	}, nil
}
