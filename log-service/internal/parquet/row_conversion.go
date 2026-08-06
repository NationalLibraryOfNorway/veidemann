package parquet

import (
	"encoding/json"
	"fmt"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type crawlLogRow struct {
	WarcID              string `parquet:"warc_id"`
	ExecutionID         string `parquet:"execution_id"`
	JobExecutionID      string `parquet:"job_execution_id"`
	CollectionFinalName string `parquet:"collection_final_name"`
	StatusCode          int32  `parquet:"status_code"`
	Size                int64  `parquet:"size"`
	FetchTimeMs         int64  `parquet:"fetch_time_ms"`
	Retries             int32  `parquet:"retries"`
	RequestedURI        string `parquet:"requested_uri"`
	ResponseURI         string `parquet:"response_uri"`
	DiscoveryPath       string `parquet:"discovery_path"`
	Referrer            string `parquet:"referrer"`
	ContentType         string `parquet:"content_type"`
	BlockDigest         string `parquet:"block_digest"`
	PayloadDigest       string `parquet:"payload_digest"`
	StorageRef          string `parquet:"storage_ref"`
	RecordType          string `parquet:"record_type"`
	WarcRefersTo        string `parquet:"warc_refers_to"`
	IPAddress           string `parquet:"ip_address"`
	Method              string `parquet:"method"`
	TimeStamp           int64  `parquet:"time_stamp"`
	FetchTimeStamp      int64  `parquet:"fetch_time_stamp"`
	ErrorCode           int32  `parquet:"error_code"`
	ErrorMsg            string `parquet:"error_msg"`
	ErrorDetail         string `parquet:"error_detail"`
}

type pageLogRow struct {
	WarcID              string `parquet:"warc_id"`
	ExecutionID         string `parquet:"execution_id"`
	JobExecutionID      string `parquet:"job_execution_id"`
	CollectionFinalName string `parquet:"collection_final_name"`
	URI                 string `parquet:"uri"`
	Referrer            string `parquet:"referrer"`
	Method              string `parquet:"method"`
	Outlinks            string `parquet:"outlinks"`
}

type resourceRow struct {
	PageID        string `parquet:"page_id"`
	WarcID        string `parquet:"warc_id"`
	URI           string `parquet:"uri"`
	Referrer      string `parquet:"referrer"`
	ResourceType  string `parquet:"resource_type"`
	ContentType   string `parquet:"content_type"`
	DiscoveryPath string `parquet:"discovery_path"`
	Method        string `parquet:"method"`
	StatusCode    int32  `parquet:"status_code"`
	FromCache     bool   `parquet:"from_cache"`
	Renderable    bool   `parquet:"renderable"`
	ErrorCode     int32  `parquet:"error_code"`
	ErrorMsg      string `parquet:"error_msg"`
	ErrorDetail   string `parquet:"error_detail"`
}

func crawlLogToRow(crawlLog *logV1.CrawlLog) *crawlLogRow {
	return &crawlLogRow{
		WarcID:              crawlLog.GetWarcId(),
		ExecutionID:         crawlLog.GetExecutionId(),
		JobExecutionID:      crawlLog.GetJobExecutionId(),
		CollectionFinalName: crawlLog.GetCollectionFinalName(),
		StatusCode:          crawlLog.GetStatusCode(),
		Size:                crawlLog.GetSize(),
		FetchTimeMs:         crawlLog.GetFetchTimeMs(),
		Retries:             crawlLog.GetRetries(),
		RequestedURI:        crawlLog.GetRequestedUri(),
		ResponseURI:         crawlLog.GetResponseUri(),
		DiscoveryPath:       crawlLog.GetDiscoveryPath(),
		Referrer:            crawlLog.GetReferrer(),
		ContentType:         crawlLog.GetContentType(),
		BlockDigest:         crawlLog.GetBlockDigest(),
		PayloadDigest:       crawlLog.GetPayloadDigest(),
		StorageRef:          crawlLog.GetStorageRef(),
		RecordType:          crawlLog.GetRecordType(),
		WarcRefersTo:        crawlLog.GetWarcRefersTo(),
		IPAddress:           crawlLog.GetIpAddress(),
		Method:              crawlLog.GetMethod(),
		TimeStamp:           protoTimestampToMillis(crawlLog.GetTimeStamp()),
		FetchTimeStamp:      protoTimestampToMillis(crawlLog.GetFetchTimeStamp()),
		ErrorCode:           crawlLog.GetError().GetCode(),
		ErrorMsg:            crawlLog.GetError().GetMsg(),
		ErrorDetail:         crawlLog.GetError().GetDetail(),
	}
}

func pageLogToRow(pageLog *logV1.PageLog) (*pageLogRow, error) {
	outlinks, err := marshalOutlinks(pageLog.GetOutlink())
	if err != nil {
		return nil, err
	}
	return &pageLogRow{
		WarcID:              pageLog.GetWarcId(),
		ExecutionID:         pageLog.GetExecutionId(),
		JobExecutionID:      pageLog.GetJobExecutionId(),
		CollectionFinalName: pageLog.GetCollectionFinalName(),
		URI:                 pageLog.GetUri(),
		Referrer:            pageLog.GetReferrer(),
		Method:              pageLog.GetMethod(),
		Outlinks:            outlinks,
	}, nil
}

func pageLogResourceToRow(pageID string, resource *logV1.PageLog_Resource) *resourceRow {
	return &resourceRow{
		PageID:        pageID,
		WarcID:        resource.GetWarcId(),
		URI:           resource.GetUri(),
		Referrer:      resource.GetReferrer(),
		ResourceType:  resource.GetResourceType(),
		ContentType:   resource.GetContentType(),
		DiscoveryPath: resource.GetDiscoveryPath(),
		Method:        resource.GetMethod(),
		StatusCode:    resource.GetStatusCode(),
		FromCache:     resource.GetFromCache(),
		Renderable:    resource.GetRenderable(),
		ErrorCode:     resource.GetError().GetCode(),
		ErrorMsg:      resource.GetError().GetMsg(),
		ErrorDetail:   resource.GetError().GetDetail(),
	}
}

func protoTimestampToMillis(timestamp *timestamppb.Timestamp) int64 {
	if timestamp == nil {
		return 0
	}
	return timestamp.AsTime().UnixMilli()
}

func marshalOutlinks(outlinks []string) (string, error) {
	if len(outlinks) == 0 {
		return "", nil
	}
	buf, err := json.Marshal(outlinks)
	if err != nil {
		return "", fmt.Errorf("marshal outlinks: %w", err)
	}
	return string(buf), nil
}
