// Copyright © 2018 National Library of Norway
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jobexecution

import (
	"testing"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	reportV1 "github.com/NationalLibraryOfNorway/veidemann/api/report/v1"
)

func TestCreateJobExecutionsListRequest(t *testing.T) {
	tests := []struct {
		name    string
		opt     options
		want    *reportV1.JobExecutionsListRequest
		wantErr bool
	}{
		{
			"1",
			options{pageSize: 10},
			&reportV1.JobExecutionsListRequest{PageSize: 10},
			false,
		},
		{
			"2",
			options{ids: []string{"id1", "id2"}, pageSize: 10},
			&reportV1.JobExecutionsListRequest{Id: []string{"id1", "id2"}, PageSize: 10},
			false,
		},
		{
			"3",
			options{filters: []string{"jobId=jobId1"}, pageSize: 10},
			&reportV1.JobExecutionsListRequest{
				QueryTemplate: &frontierV1.JobExecutionStatus{JobId: "jobId1"},
				QueryMask:     &commonsV1.FieldMask{Paths: []string{"jobId"}},
				PageSize:      10,
			},
			false,
		},
		{
			"4",
			options{filters: []string{"jobId=jobId1"}, pageSize: 10, watch: true},
			&reportV1.JobExecutionsListRequest{
				QueryTemplate: &frontierV1.JobExecutionStatus{JobId: "jobId1"},
				QueryMask:     &commonsV1.FieldMask{Paths: []string{"jobId"}},
				Watch:         true,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createJobExecutionsListRequest(&tt.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateJobExecutionsListRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.IsType(t, &reportV1.JobExecutionsListRequest{}, got)
			if !proto.Equal(got, tt.want) {
				t.Errorf("CreateJobExecutionsListRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
