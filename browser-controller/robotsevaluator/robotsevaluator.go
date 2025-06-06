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

package robotsevaluator

import (
	"context"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config"
	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/serviceconnections"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

type RobotsEvaluator interface {
	serviceconnections.Connection
	IsAllowed(context.Context, *robotsevaluatorV1.IsAllowedRequest) bool
}

type robotsEvaluator struct {
	*serviceconnections.ClientConn
	robotsevaluatorV1.RobotsEvaluatorClient
}

func New(opts ...serviceconnections.ConnectionOption) RobotsEvaluator {
	return &robotsEvaluator{
		ClientConn: serviceconnections.NewClientConn("RobotsEvaluator", opts...),
	}
}

func (r *robotsEvaluator) Connect() error {
	if err := r.ClientConn.Connect(); err != nil {
		return err
	} else {
		r.RobotsEvaluatorClient = robotsevaluatorV1.NewRobotsEvaluatorClient(r.ClientConn.Connection())
		return nil
	}
}

func (r *robotsEvaluator) IsAllowed(ctx context.Context, request *robotsevaluatorV1.IsAllowedRequest) bool {
	resolvedPoliteness, ignore := resolvePolicy(request.Politeness)
	if ignore {
		return true
	}

	request.Politeness = resolvedPoliteness
	reply, err := r.RobotsEvaluatorClient.IsAllowed(ctx, request)
	if err != nil {
		log.Error().Err(err).Msg("failed to query robots evaluator")
		return true
	}

	return reply.IsAllowed
}

func resolvePolicy(politenessConfig *configV1.ConfigObject) (resolvedPoliteness *configV1.ConfigObject, ignore bool) {
	var resolvedPolicy configV1.PolitenessConfig_RobotsPolicy
	switch politenessConfig.GetPolitenessConfig().GetRobotsPolicy() {
	case configV1.PolitenessConfig_OBEY_ROBOTS_CLASSIC:
		resolvedPolicy = configV1.PolitenessConfig_OBEY_ROBOTS
	case configV1.PolitenessConfig_CUSTOM_ROBOTS_CLASSIC:
		resolvedPolicy = configV1.PolitenessConfig_CUSTOM_ROBOTS
	case configV1.PolitenessConfig_CUSTOM_IF_MISSING_CLASSIC:
		resolvedPolicy = configV1.PolitenessConfig_CUSTOM_IF_MISSING
	default:
		resolvedPolicy = configV1.PolitenessConfig_IGNORE_ROBOTS
	}

	resolvedPoliteness = proto.Clone(politenessConfig).(*configV1.ConfigObject)
	resolvedPoliteness.GetPolitenessConfig().RobotsPolicy = resolvedPolicy
	return resolvedPoliteness, resolvedPolicy == configV1.PolitenessConfig_IGNORE_ROBOTS
}
