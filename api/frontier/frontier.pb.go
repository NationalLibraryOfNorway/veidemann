// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v6.31.0
// source: frontier/v1/frontier.proto

package frontier

import (
	commons "github.com/NationalLibraryOfNorway/veidemann/api/commons"
	config "github.com/NationalLibraryOfNorway/veidemann/api/config"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type CrawlSeedRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	JobExecutionId string               `protobuf:"bytes,1,opt,name=job_execution_id,json=jobExecutionId,proto3" json:"job_execution_id,omitempty"`
	Job            *config.ConfigObject `protobuf:"bytes,5,opt,name=job,proto3" json:"job,omitempty"`
	Seed           *config.ConfigObject `protobuf:"bytes,6,opt,name=seed,proto3" json:"seed,omitempty"`
	// When this seed should stop crawling. Absence of this value indicates no timeout
	Timeout *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=timeout,proto3" json:"timeout,omitempty"`
}

func (x *CrawlSeedRequest) Reset() {
	*x = CrawlSeedRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_frontier_v1_frontier_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CrawlSeedRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CrawlSeedRequest) ProtoMessage() {}

func (x *CrawlSeedRequest) ProtoReflect() protoreflect.Message {
	mi := &file_frontier_v1_frontier_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CrawlSeedRequest.ProtoReflect.Descriptor instead.
func (*CrawlSeedRequest) Descriptor() ([]byte, []int) {
	return file_frontier_v1_frontier_proto_rawDescGZIP(), []int{0}
}

func (x *CrawlSeedRequest) GetJobExecutionId() string {
	if x != nil {
		return x.JobExecutionId
	}
	return ""
}

func (x *CrawlSeedRequest) GetJob() *config.ConfigObject {
	if x != nil {
		return x.Job
	}
	return nil
}

func (x *CrawlSeedRequest) GetSeed() *config.ConfigObject {
	if x != nil {
		return x.Seed
	}
	return nil
}

func (x *CrawlSeedRequest) GetTimeout() *timestamppb.Timestamp {
	if x != nil {
		return x.Timeout
	}
	return nil
}

// The execution id for a seed crawl
type CrawlExecutionId struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *CrawlExecutionId) Reset() {
	*x = CrawlExecutionId{}
	if protoimpl.UnsafeEnabled {
		mi := &file_frontier_v1_frontier_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CrawlExecutionId) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CrawlExecutionId) ProtoMessage() {}

func (x *CrawlExecutionId) ProtoReflect() protoreflect.Message {
	mi := &file_frontier_v1_frontier_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CrawlExecutionId.ProtoReflect.Descriptor instead.
func (*CrawlExecutionId) Descriptor() ([]byte, []int) {
	return file_frontier_v1_frontier_proto_rawDescGZIP(), []int{1}
}

func (x *CrawlExecutionId) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

// Message sent from Harvester to return the harvest result.
// When the fetch is done, a stream of PageHarvest objects are returned:
// The first object contains metrics.
// Subsequent objects contain outlinks until all outlinks are sent.
// Finally the client should complete the request.
type PageHarvest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Msg:
	//
	//	*PageHarvest_Metrics_
	//	*PageHarvest_Outlink
	//	*PageHarvest_Error
	Msg isPageHarvest_Msg `protobuf_oneof:"msg"`
	// Session token from the PageHarvestSpec.
	SessionToken string `protobuf:"bytes,5,opt,name=session_token,json=sessionToken,proto3" json:"session_token,omitempty"`
}

func (x *PageHarvest) Reset() {
	*x = PageHarvest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_frontier_v1_frontier_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PageHarvest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PageHarvest) ProtoMessage() {}

func (x *PageHarvest) ProtoReflect() protoreflect.Message {
	mi := &file_frontier_v1_frontier_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PageHarvest.ProtoReflect.Descriptor instead.
func (*PageHarvest) Descriptor() ([]byte, []int) {
	return file_frontier_v1_frontier_proto_rawDescGZIP(), []int{2}
}

func (m *PageHarvest) GetMsg() isPageHarvest_Msg {
	if m != nil {
		return m.Msg
	}
	return nil
}

func (x *PageHarvest) GetMetrics() *PageHarvest_Metrics {
	if x, ok := x.GetMsg().(*PageHarvest_Metrics_); ok {
		return x.Metrics
	}
	return nil
}

func (x *PageHarvest) GetOutlink() *QueuedUri {
	if x, ok := x.GetMsg().(*PageHarvest_Outlink); ok {
		return x.Outlink
	}
	return nil
}

func (x *PageHarvest) GetError() *commons.Error {
	if x, ok := x.GetMsg().(*PageHarvest_Error); ok {
		return x.Error
	}
	return nil
}

func (x *PageHarvest) GetSessionToken() string {
	if x != nil {
		return x.SessionToken
	}
	return ""
}

type isPageHarvest_Msg interface {
	isPageHarvest_Msg()
}

type PageHarvest_Metrics_ struct {
	// Collected metrics for the page fetched
	Metrics *PageHarvest_Metrics `protobuf:"bytes,2,opt,name=metrics,proto3,oneof"`
}

type PageHarvest_Outlink struct {
	// The outlinks found in the harvested page
	Outlink *QueuedUri `protobuf:"bytes,3,opt,name=outlink,proto3,oneof"`
}

type PageHarvest_Error struct {
	// If the overall page fetch failed. Should not be used for a singel uri failure
	Error *commons.Error `protobuf:"bytes,4,opt,name=error,proto3,oneof"`
}

func (*PageHarvest_Metrics_) isPageHarvest_Msg() {}

func (*PageHarvest_Outlink) isPageHarvest_Msg() {}

func (*PageHarvest_Error) isPageHarvest_Msg() {}

// A specification of the page to fetch.
type PageHarvestSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The URI to fetch
	QueuedUri *QueuedUri `protobuf:"bytes,1,opt,name=queued_uri,json=queuedUri,proto3" json:"queued_uri,omitempty"`
	// The configuration for the fetch
	CrawlConfig *config.ConfigObject `protobuf:"bytes,2,opt,name=crawl_config,json=crawlConfig,proto3" json:"crawl_config,omitempty"`
	// Session token for this request.
	// The Harvester is responsible for setting the same session token in all responses to this request.
	SessionToken string `protobuf:"bytes,3,opt,name=session_token,json=sessionToken,proto3" json:"session_token,omitempty"`
}

func (x *PageHarvestSpec) Reset() {
	*x = PageHarvestSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_frontier_v1_frontier_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PageHarvestSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PageHarvestSpec) ProtoMessage() {}

func (x *PageHarvestSpec) ProtoReflect() protoreflect.Message {
	mi := &file_frontier_v1_frontier_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PageHarvestSpec.ProtoReflect.Descriptor instead.
func (*PageHarvestSpec) Descriptor() ([]byte, []int) {
	return file_frontier_v1_frontier_proto_rawDescGZIP(), []int{3}
}

func (x *PageHarvestSpec) GetQueuedUri() *QueuedUri {
	if x != nil {
		return x.QueuedUri
	}
	return nil
}

func (x *PageHarvestSpec) GetCrawlConfig() *config.ConfigObject {
	if x != nil {
		return x.CrawlConfig
	}
	return nil
}

func (x *PageHarvestSpec) GetSessionToken() string {
	if x != nil {
		return x.SessionToken
	}
	return ""
}

type CountResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Count int64 `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
}

func (x *CountResponse) Reset() {
	*x = CountResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_frontier_v1_frontier_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CountResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CountResponse) ProtoMessage() {}

func (x *CountResponse) ProtoReflect() protoreflect.Message {
	mi := &file_frontier_v1_frontier_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CountResponse.ProtoReflect.Descriptor instead.
func (*CountResponse) Descriptor() ([]byte, []int) {
	return file_frontier_v1_frontier_proto_rawDescGZIP(), []int{4}
}

func (x *CountResponse) GetCount() int64 {
	if x != nil {
		return x.Count
	}
	return 0
}

type PageHarvest_Metrics struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The number of uri's downloaded. The requested uri + embedded resources
	UriCount int32 `protobuf:"varint,1,opt,name=uri_count,json=uriCount,proto3" json:"uri_count,omitempty"`
	// Byte count for the resources downloaded. Includes embedded resources
	BytesDownloaded int64 `protobuf:"varint,2,opt,name=bytes_downloaded,json=bytesDownloaded,proto3" json:"bytes_downloaded,omitempty"`
}

func (x *PageHarvest_Metrics) Reset() {
	*x = PageHarvest_Metrics{}
	if protoimpl.UnsafeEnabled {
		mi := &file_frontier_v1_frontier_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PageHarvest_Metrics) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PageHarvest_Metrics) ProtoMessage() {}

func (x *PageHarvest_Metrics) ProtoReflect() protoreflect.Message {
	mi := &file_frontier_v1_frontier_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PageHarvest_Metrics.ProtoReflect.Descriptor instead.
func (*PageHarvest_Metrics) Descriptor() ([]byte, []int) {
	return file_frontier_v1_frontier_proto_rawDescGZIP(), []int{2, 0}
}

func (x *PageHarvest_Metrics) GetUriCount() int32 {
	if x != nil {
		return x.UriCount
	}
	return 0
}

func (x *PageHarvest_Metrics) GetBytesDownloaded() int64 {
	if x != nil {
		return x.BytesDownloaded
	}
	return 0
}

var File_frontier_v1_frontier_proto protoreflect.FileDescriptor

var file_frontier_v1_frontier_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x66, 0x72,
	0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x19, 0x76, 0x65,
	0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e,
	0x74, 0x69, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x1a, 0x1a, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x73,
	0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x19, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x76, 0x31, 0x2f, 0x72,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b,
	0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70,
	0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xe6, 0x01, 0x0a, 0x10, 0x43, 0x72,
	0x61, 0x77, 0x6c, 0x53, 0x65, 0x65, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x28,
	0x0a, 0x10, 0x6a, 0x6f, 0x62, 0x5f, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x5f,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x6a, 0x6f, 0x62, 0x45, 0x78, 0x65,
	0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x37, 0x0a, 0x03, 0x6a, 0x6f, 0x62, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e,
	0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x76, 0x31, 0x2e,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x52, 0x03, 0x6a, 0x6f,
	0x62, 0x12, 0x39, 0x0a, 0x04, 0x73, 0x65, 0x65, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x25, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x52, 0x04, 0x73, 0x65, 0x65, 0x64, 0x12, 0x34, 0x0a, 0x07,
	0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x07, 0x74, 0x69, 0x6d, 0x65, 0x6f,
	0x75, 0x74, 0x22, 0x22, 0x0a, 0x10, 0x43, 0x72, 0x61, 0x77, 0x6c, 0x45, 0x78, 0x65, 0x63, 0x75,
	0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0xd3, 0x02, 0x0a, 0x0b, 0x50, 0x61, 0x67, 0x65, 0x48,
	0x61, 0x72, 0x76, 0x65, 0x73, 0x74, 0x12, 0x4a, 0x0a, 0x07, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d,
	0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x48, 0x61, 0x72, 0x76, 0x65, 0x73, 0x74, 0x2e,
	0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x48, 0x00, 0x52, 0x07, 0x6d, 0x65, 0x74, 0x72, 0x69,
	0x63, 0x73, 0x12, 0x40, 0x0a, 0x07, 0x6f, 0x75, 0x74, 0x6c, 0x69, 0x6e, 0x6b, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e,
	0x51, 0x75, 0x65, 0x75, 0x65, 0x64, 0x55, 0x72, 0x69, 0x48, 0x00, 0x52, 0x07, 0x6f, 0x75, 0x74,
	0x6c, 0x69, 0x6e, 0x6b, 0x12, 0x37, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x45,
	0x72, 0x72, 0x6f, 0x72, 0x48, 0x00, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x23, 0x0a,
	0x0d, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x54, 0x6f, 0x6b,
	0x65, 0x6e, 0x1a, 0x51, 0x0a, 0x07, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x12, 0x1b, 0x0a,
	0x09, 0x75, 0x72, 0x69, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x08, 0x75, 0x72, 0x69, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x29, 0x0a, 0x10, 0x62, 0x79,
	0x74, 0x65, 0x73, 0x5f, 0x64, 0x6f, 0x77, 0x6e, 0x6c, 0x6f, 0x61, 0x64, 0x65, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x0f, 0x62, 0x79, 0x74, 0x65, 0x73, 0x44, 0x6f, 0x77, 0x6e, 0x6c,
	0x6f, 0x61, 0x64, 0x65, 0x64, 0x42, 0x05, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x22, 0xc5, 0x01, 0x0a,
	0x0f, 0x50, 0x61, 0x67, 0x65, 0x48, 0x61, 0x72, 0x76, 0x65, 0x73, 0x74, 0x53, 0x70, 0x65, 0x63,
	0x12, 0x43, 0x0a, 0x0a, 0x71, 0x75, 0x65, 0x75, 0x65, 0x64, 0x5f, 0x75, 0x72, 0x69, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x2e, 0x51, 0x75, 0x65, 0x75, 0x65, 0x64, 0x55, 0x72, 0x69, 0x52, 0x09, 0x71, 0x75, 0x65, 0x75,
	0x65, 0x64, 0x55, 0x72, 0x69, 0x12, 0x48, 0x0a, 0x0c, 0x63, 0x72, 0x61, 0x77, 0x6c, 0x5f, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x76, 0x65,
	0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x4f, 0x62, 0x6a, 0x65,
	0x63, 0x74, 0x52, 0x0b, 0x63, 0x72, 0x61, 0x77, 0x6c, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12,
	0x23, 0x0a, 0x0d, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x54,
	0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x25, 0x0a, 0x0d, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x32, 0xc1, 0x05, 0x0a, 0x08,
	0x46, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x12, 0x67, 0x0a, 0x09, 0x43, 0x72, 0x61, 0x77,
	0x6c, 0x53, 0x65, 0x65, 0x64, 0x12, 0x2b, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e,
	0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x76,
	0x31, 0x2e, 0x43, 0x72, 0x61, 0x77, 0x6c, 0x53, 0x65, 0x65, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x2b, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43,
	0x72, 0x61, 0x77, 0x6c, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x22,
	0x00, 0x12, 0x53, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x4e, 0x65, 0x78, 0x74, 0x50, 0x61, 0x67, 0x65,
	0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x2a, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65,
	0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x48, 0x61, 0x72, 0x76, 0x65, 0x73, 0x74,
	0x53, 0x70, 0x65, 0x63, 0x22, 0x00, 0x12, 0x53, 0x0a, 0x0d, 0x50, 0x61, 0x67, 0x65, 0x43, 0x6f,
	0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x12, 0x26, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d,
	0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x48, 0x61, 0x72, 0x76, 0x65, 0x73, 0x74, 0x1a,
	0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x28, 0x01, 0x12, 0x5d, 0x0a, 0x17, 0x42,
	0x75, 0x73, 0x79, 0x43, 0x72, 0x61, 0x77, 0x6c, 0x48, 0x6f, 0x73, 0x74, 0x47, 0x72, 0x6f, 0x75,
	0x70, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x28,
	0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66,
	0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x75, 0x6e, 0x74,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x55, 0x0a, 0x0f, 0x51, 0x75,
	0x65, 0x75, 0x65, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x12, 0x16, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x28, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e,
	0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x76,
	0x31, 0x2e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x12, 0x76, 0x0a, 0x1b, 0x51, 0x75, 0x65, 0x75, 0x65, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x46,
	0x6f, 0x72, 0x43, 0x72, 0x61, 0x77, 0x6c, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x2b, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x61,
	0x77, 0x6c, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x1a, 0x28, 0x2e,
	0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72,
	0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x74, 0x0a, 0x1b, 0x51, 0x75, 0x65,
	0x75, 0x65, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x46, 0x6f, 0x72, 0x43, 0x72, 0x61, 0x77, 0x6c, 0x48,
	0x6f, 0x73, 0x74, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x12, 0x29, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65,
	0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x61, 0x77, 0x6c, 0x48, 0x6f, 0x73, 0x74, 0x47, 0x72,
	0x6f, 0x75, 0x70, 0x1a, 0x28, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e,
	0x43, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42,
	0x73, 0x0a, 0x23, 0x6e, 0x6f, 0x2e, 0x6e, 0x62, 0x2e, 0x6e, 0x6e, 0x61, 0x2e, 0x76, 0x65, 0x69,
	0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x66, 0x72, 0x6f, 0x6e, 0x74,
	0x69, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x42, 0x0f, 0x46, 0x72, 0x6f, 0x6e, 0x74, 0x69, 0x65, 0x72,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x01, 0x5a, 0x39, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x4e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x4c, 0x69,
	0x62, 0x72, 0x61, 0x72, 0x79, 0x4f, 0x66, 0x4e, 0x6f, 0x72, 0x77, 0x61, 0x79, 0x2f, 0x76, 0x65,
	0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x66, 0x72, 0x6f, 0x6e,
	0x74, 0x69, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_frontier_v1_frontier_proto_rawDescOnce sync.Once
	file_frontier_v1_frontier_proto_rawDescData = file_frontier_v1_frontier_proto_rawDesc
)

func file_frontier_v1_frontier_proto_rawDescGZIP() []byte {
	file_frontier_v1_frontier_proto_rawDescOnce.Do(func() {
		file_frontier_v1_frontier_proto_rawDescData = protoimpl.X.CompressGZIP(file_frontier_v1_frontier_proto_rawDescData)
	})
	return file_frontier_v1_frontier_proto_rawDescData
}

var file_frontier_v1_frontier_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_frontier_v1_frontier_proto_goTypes = []interface{}{
	(*CrawlSeedRequest)(nil),      // 0: veidemann.api.frontier.v1.CrawlSeedRequest
	(*CrawlExecutionId)(nil),      // 1: veidemann.api.frontier.v1.CrawlExecutionId
	(*PageHarvest)(nil),           // 2: veidemann.api.frontier.v1.PageHarvest
	(*PageHarvestSpec)(nil),       // 3: veidemann.api.frontier.v1.PageHarvestSpec
	(*CountResponse)(nil),         // 4: veidemann.api.frontier.v1.CountResponse
	(*PageHarvest_Metrics)(nil),   // 5: veidemann.api.frontier.v1.PageHarvest.Metrics
	(*config.ConfigObject)(nil),   // 6: veidemann.api.config.v1.ConfigObject
	(*timestamppb.Timestamp)(nil), // 7: google.protobuf.Timestamp
	(*QueuedUri)(nil),             // 8: veidemann.api.frontier.v1.QueuedUri
	(*commons.Error)(nil),         // 9: veidemann.api.commons.v1.Error
	(*emptypb.Empty)(nil),         // 10: google.protobuf.Empty
	(*CrawlHostGroup)(nil),        // 11: veidemann.api.frontier.v1.CrawlHostGroup
}
var file_frontier_v1_frontier_proto_depIdxs = []int32{
	6,  // 0: veidemann.api.frontier.v1.CrawlSeedRequest.job:type_name -> veidemann.api.config.v1.ConfigObject
	6,  // 1: veidemann.api.frontier.v1.CrawlSeedRequest.seed:type_name -> veidemann.api.config.v1.ConfigObject
	7,  // 2: veidemann.api.frontier.v1.CrawlSeedRequest.timeout:type_name -> google.protobuf.Timestamp
	5,  // 3: veidemann.api.frontier.v1.PageHarvest.metrics:type_name -> veidemann.api.frontier.v1.PageHarvest.Metrics
	8,  // 4: veidemann.api.frontier.v1.PageHarvest.outlink:type_name -> veidemann.api.frontier.v1.QueuedUri
	9,  // 5: veidemann.api.frontier.v1.PageHarvest.error:type_name -> veidemann.api.commons.v1.Error
	8,  // 6: veidemann.api.frontier.v1.PageHarvestSpec.queued_uri:type_name -> veidemann.api.frontier.v1.QueuedUri
	6,  // 7: veidemann.api.frontier.v1.PageHarvestSpec.crawl_config:type_name -> veidemann.api.config.v1.ConfigObject
	0,  // 8: veidemann.api.frontier.v1.Frontier.CrawlSeed:input_type -> veidemann.api.frontier.v1.CrawlSeedRequest
	10, // 9: veidemann.api.frontier.v1.Frontier.GetNextPage:input_type -> google.protobuf.Empty
	2,  // 10: veidemann.api.frontier.v1.Frontier.PageCompleted:input_type -> veidemann.api.frontier.v1.PageHarvest
	10, // 11: veidemann.api.frontier.v1.Frontier.BusyCrawlHostGroupCount:input_type -> google.protobuf.Empty
	10, // 12: veidemann.api.frontier.v1.Frontier.QueueCountTotal:input_type -> google.protobuf.Empty
	1,  // 13: veidemann.api.frontier.v1.Frontier.QueueCountForCrawlExecution:input_type -> veidemann.api.frontier.v1.CrawlExecutionId
	11, // 14: veidemann.api.frontier.v1.Frontier.QueueCountForCrawlHostGroup:input_type -> veidemann.api.frontier.v1.CrawlHostGroup
	1,  // 15: veidemann.api.frontier.v1.Frontier.CrawlSeed:output_type -> veidemann.api.frontier.v1.CrawlExecutionId
	3,  // 16: veidemann.api.frontier.v1.Frontier.GetNextPage:output_type -> veidemann.api.frontier.v1.PageHarvestSpec
	10, // 17: veidemann.api.frontier.v1.Frontier.PageCompleted:output_type -> google.protobuf.Empty
	4,  // 18: veidemann.api.frontier.v1.Frontier.BusyCrawlHostGroupCount:output_type -> veidemann.api.frontier.v1.CountResponse
	4,  // 19: veidemann.api.frontier.v1.Frontier.QueueCountTotal:output_type -> veidemann.api.frontier.v1.CountResponse
	4,  // 20: veidemann.api.frontier.v1.Frontier.QueueCountForCrawlExecution:output_type -> veidemann.api.frontier.v1.CountResponse
	4,  // 21: veidemann.api.frontier.v1.Frontier.QueueCountForCrawlHostGroup:output_type -> veidemann.api.frontier.v1.CountResponse
	15, // [15:22] is the sub-list for method output_type
	8,  // [8:15] is the sub-list for method input_type
	8,  // [8:8] is the sub-list for extension type_name
	8,  // [8:8] is the sub-list for extension extendee
	0,  // [0:8] is the sub-list for field type_name
}

func init() { file_frontier_v1_frontier_proto_init() }
func file_frontier_v1_frontier_proto_init() {
	if File_frontier_v1_frontier_proto != nil {
		return
	}
	file_frontier_v1_resources_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_frontier_v1_frontier_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CrawlSeedRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_frontier_v1_frontier_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CrawlExecutionId); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_frontier_v1_frontier_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PageHarvest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_frontier_v1_frontier_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PageHarvestSpec); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_frontier_v1_frontier_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CountResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_frontier_v1_frontier_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PageHarvest_Metrics); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_frontier_v1_frontier_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*PageHarvest_Metrics_)(nil),
		(*PageHarvest_Outlink)(nil),
		(*PageHarvest_Error)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_frontier_v1_frontier_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_frontier_v1_frontier_proto_goTypes,
		DependencyIndexes: file_frontier_v1_frontier_proto_depIdxs,
		MessageInfos:      file_frontier_v1_frontier_proto_msgTypes,
	}.Build()
	File_frontier_v1_frontier_proto = out.File
	file_frontier_v1_frontier_proto_rawDesc = nil
	file_frontier_v1_frontier_proto_goTypes = nil
	file_frontier_v1_frontier_proto_depIdxs = nil
}
