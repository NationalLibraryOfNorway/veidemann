// Copyright 2026 National Library of Norway.
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"io"
)

// UploadFinish sends the complete capture under the caller's deadline.
func UploadFinish(ctx context.Context, client RecorderControlClient, header *FinishHarvestHeader, links []string) (*FinishHarvestReply, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := client.FinishHarvest(ctx)
	if err != nil {
		return nil, err
	}
	err = stream.Send(&FinishHarvestRequest{Value: &FinishHarvestRequest_Header{Header: header}})
	for _, link := range links {
		if err != nil {
			break
		}
		err = stream.Send(&FinishHarvestRequest{Value: &FinishHarvestRequest_Outlink{Outlink: link}})
	}
	if err != nil && err != io.EOF {
		return nil, err
	}
	return stream.CloseAndRecv()
}
