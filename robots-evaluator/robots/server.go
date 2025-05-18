package robots

import (
	"context"

	"github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator"
)

type EvaluatorServer struct {
	robotsevaluator.UnimplementedRobotsEvaluatorServer

	*Evaluator
}

// Assert that Evaluator implements the RobotsEvaluatorServer interface
var _ robotsevaluator.RobotsEvaluatorServer = (*EvaluatorServer)(nil)

// IsAllowed implements the RobotsEvaluatorServer interface
func (e *EvaluatorServer) IsAllowed(ctx context.Context, req *robotsevaluator.IsAllowedRequest) (*robotsevaluator.IsAllowedReply, error) {
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

	return &robotsevaluator.IsAllowedReply{
		IsAllowed: ok,
	}, nil
}
