// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v6.31.0
// source: browsercontroller/v1/browsercontroller.proto

package browsercontroller

import (
	config "github.com/NationalLibraryOfNorway/veidemann/api/config"
	log "github.com/NationalLibraryOfNorway/veidemann/api/log"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type NotifyActivity_Activity int32

const (
	NotifyActivity_DATA_RECEIVED     NotifyActivity_Activity = 0
	NotifyActivity_ALL_DATA_RECEIVED NotifyActivity_Activity = 1
)

// Enum value maps for NotifyActivity_Activity.
var (
	NotifyActivity_Activity_name = map[int32]string{
		0: "DATA_RECEIVED",
		1: "ALL_DATA_RECEIVED",
	}
	NotifyActivity_Activity_value = map[string]int32{
		"DATA_RECEIVED":     0,
		"ALL_DATA_RECEIVED": 1,
	}
)

func (x NotifyActivity_Activity) Enum() *NotifyActivity_Activity {
	p := new(NotifyActivity_Activity)
	*p = x
	return p
}

func (x NotifyActivity_Activity) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (NotifyActivity_Activity) Descriptor() protoreflect.EnumDescriptor {
	return file_browsercontroller_v1_browsercontroller_proto_enumTypes[0].Descriptor()
}

func (NotifyActivity_Activity) Type() protoreflect.EnumType {
	return &file_browsercontroller_v1_browsercontroller_proto_enumTypes[0]
}

func (x NotifyActivity_Activity) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use NotifyActivity_Activity.Descriptor instead.
func (NotifyActivity_Activity) EnumDescriptor() ([]byte, []int) {
	return file_browsercontroller_v1_browsercontroller_proto_rawDescGZIP(), []int{1, 0}
}

type RegisterNew struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ProxyId          int32             `protobuf:"varint,1,opt,name=proxy_id,json=proxyId,proto3" json:"proxy_id,omitempty"`
	Uri              string            `protobuf:"bytes,2,opt,name=uri,proto3" json:"uri,omitempty"`
	CrawlExecutionId string            `protobuf:"bytes,3,opt,name=crawl_execution_id,json=crawlExecutionId,proto3" json:"crawl_execution_id,omitempty"`
	JobExecutionId   string            `protobuf:"bytes,4,opt,name=job_execution_id,json=jobExecutionId,proto3" json:"job_execution_id,omitempty"`
	CollectionRef    *config.ConfigRef `protobuf:"bytes,5,opt,name=collection_ref,json=collectionRef,proto3" json:"collection_ref,omitempty"`
	Method           string            `protobuf:"bytes,6,opt,name=method,proto3" json:"method,omitempty"`                        // The HTTP method (GET, POST, HEAD, ...)
	RequestId        string            `protobuf:"bytes,7,opt,name=request_id,json=requestId,proto3" json:"request_id,omitempty"` // The browsers internal request id.
}

func (x *RegisterNew) Reset() {
	*x = RegisterNew{}
	if protoimpl.UnsafeEnabled {
		mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RegisterNew) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RegisterNew) ProtoMessage() {}

func (x *RegisterNew) ProtoReflect() protoreflect.Message {
	mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RegisterNew.ProtoReflect.Descriptor instead.
func (*RegisterNew) Descriptor() ([]byte, []int) {
	return file_browsercontroller_v1_browsercontroller_proto_rawDescGZIP(), []int{0}
}

func (x *RegisterNew) GetProxyId() int32 {
	if x != nil {
		return x.ProxyId
	}
	return 0
}

func (x *RegisterNew) GetUri() string {
	if x != nil {
		return x.Uri
	}
	return ""
}

func (x *RegisterNew) GetCrawlExecutionId() string {
	if x != nil {
		return x.CrawlExecutionId
	}
	return ""
}

func (x *RegisterNew) GetJobExecutionId() string {
	if x != nil {
		return x.JobExecutionId
	}
	return ""
}

func (x *RegisterNew) GetCollectionRef() *config.ConfigRef {
	if x != nil {
		return x.CollectionRef
	}
	return nil
}

func (x *RegisterNew) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *RegisterNew) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

type NotifyActivity struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Activity NotifyActivity_Activity `protobuf:"varint,1,opt,name=activity,proto3,enum=veidemann.api.browsercontroller.v1.NotifyActivity_Activity" json:"activity,omitempty"`
}

func (x *NotifyActivity) Reset() {
	*x = NotifyActivity{}
	if protoimpl.UnsafeEnabled {
		mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NotifyActivity) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotifyActivity) ProtoMessage() {}

func (x *NotifyActivity) ProtoReflect() protoreflect.Message {
	mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NotifyActivity.ProtoReflect.Descriptor instead.
func (*NotifyActivity) Descriptor() ([]byte, []int) {
	return file_browsercontroller_v1_browsercontroller_proto_rawDescGZIP(), []int{1}
}

func (x *NotifyActivity) GetActivity() NotifyActivity_Activity {
	if x != nil {
		return x.Activity
	}
	return NotifyActivity_DATA_RECEIVED
}

type Completed struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CrawlLog *log.CrawlLog `protobuf:"bytes,1,opt,name=crawl_log,json=crawlLog,proto3" json:"crawl_log,omitempty"`
	Cached   bool          `protobuf:"varint,2,opt,name=cached,proto3" json:"cached,omitempty"`
}

func (x *Completed) Reset() {
	*x = Completed{}
	if protoimpl.UnsafeEnabled {
		mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Completed) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Completed) ProtoMessage() {}

func (x *Completed) ProtoReflect() protoreflect.Message {
	mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Completed.ProtoReflect.Descriptor instead.
func (*Completed) Descriptor() ([]byte, []int) {
	return file_browsercontroller_v1_browsercontroller_proto_rawDescGZIP(), []int{2}
}

func (x *Completed) GetCrawlLog() *log.CrawlLog {
	if x != nil {
		return x.CrawlLog
	}
	return nil
}

func (x *Completed) GetCached() bool {
	if x != nil {
		return x.Cached
	}
	return false
}

type DoRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Action:
	//
	//	*DoRequest_New
	//	*DoRequest_Notify
	//	*DoRequest_Completed
	Action isDoRequest_Action `protobuf_oneof:"action"`
}

func (x *DoRequest) Reset() {
	*x = DoRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DoRequest) ProtoMessage() {}

func (x *DoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DoRequest.ProtoReflect.Descriptor instead.
func (*DoRequest) Descriptor() ([]byte, []int) {
	return file_browsercontroller_v1_browsercontroller_proto_rawDescGZIP(), []int{3}
}

func (m *DoRequest) GetAction() isDoRequest_Action {
	if m != nil {
		return m.Action
	}
	return nil
}

func (x *DoRequest) GetNew() *RegisterNew {
	if x, ok := x.GetAction().(*DoRequest_New); ok {
		return x.New
	}
	return nil
}

func (x *DoRequest) GetNotify() *NotifyActivity {
	if x, ok := x.GetAction().(*DoRequest_Notify); ok {
		return x.Notify
	}
	return nil
}

func (x *DoRequest) GetCompleted() *Completed {
	if x, ok := x.GetAction().(*DoRequest_Completed); ok {
		return x.Completed
	}
	return nil
}

type isDoRequest_Action interface {
	isDoRequest_Action()
}

type DoRequest_New struct {
	New *RegisterNew `protobuf:"bytes,1,opt,name=new,proto3,oneof"`
}

type DoRequest_Notify struct {
	Notify *NotifyActivity `protobuf:"bytes,2,opt,name=notify,proto3,oneof"`
}

type DoRequest_Completed struct {
	Completed *Completed `protobuf:"bytes,3,opt,name=completed,proto3,oneof"`
}

func (*DoRequest_New) isDoRequest_Action() {}

func (*DoRequest_Notify) isDoRequest_Action() {}

func (*DoRequest_Completed) isDoRequest_Action() {}

type NewReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CrawlExecutionId  string                `protobuf:"bytes,1,opt,name=crawl_execution_id,json=crawlExecutionId,proto3" json:"crawl_execution_id,omitempty"`
	JobExecutionId    string                `protobuf:"bytes,2,opt,name=job_execution_id,json=jobExecutionId,proto3" json:"job_execution_id,omitempty"`
	CollectionRef     *config.ConfigRef     `protobuf:"bytes,4,opt,name=collection_ref,json=collectionRef,proto3" json:"collection_ref,omitempty"`
	ReplacementScript *config.BrowserScript `protobuf:"bytes,5,opt,name=replacement_script,json=replacementScript,proto3" json:"replacement_script,omitempty"`
}

func (x *NewReply) Reset() {
	*x = NewReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NewReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewReply) ProtoMessage() {}

func (x *NewReply) ProtoReflect() protoreflect.Message {
	mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewReply.ProtoReflect.Descriptor instead.
func (*NewReply) Descriptor() ([]byte, []int) {
	return file_browsercontroller_v1_browsercontroller_proto_rawDescGZIP(), []int{4}
}

func (x *NewReply) GetCrawlExecutionId() string {
	if x != nil {
		return x.CrawlExecutionId
	}
	return ""
}

func (x *NewReply) GetJobExecutionId() string {
	if x != nil {
		return x.JobExecutionId
	}
	return ""
}

func (x *NewReply) GetCollectionRef() *config.ConfigRef {
	if x != nil {
		return x.CollectionRef
	}
	return nil
}

func (x *NewReply) GetReplacementScript() *config.BrowserScript {
	if x != nil {
		return x.ReplacementScript
	}
	return nil
}

type DoReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Action:
	//
	//	*DoReply_New
	//	*DoReply_Cancel
	Action isDoReply_Action `protobuf_oneof:"action"`
}

func (x *DoReply) Reset() {
	*x = DoReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DoReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DoReply) ProtoMessage() {}

func (x *DoReply) ProtoReflect() protoreflect.Message {
	mi := &file_browsercontroller_v1_browsercontroller_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DoReply.ProtoReflect.Descriptor instead.
func (*DoReply) Descriptor() ([]byte, []int) {
	return file_browsercontroller_v1_browsercontroller_proto_rawDescGZIP(), []int{5}
}

func (m *DoReply) GetAction() isDoReply_Action {
	if m != nil {
		return m.Action
	}
	return nil
}

func (x *DoReply) GetNew() *NewReply {
	if x, ok := x.GetAction().(*DoReply_New); ok {
		return x.New
	}
	return nil
}

func (x *DoReply) GetCancel() string {
	if x, ok := x.GetAction().(*DoReply_Cancel); ok {
		return x.Cancel
	}
	return ""
}

type isDoReply_Action interface {
	isDoReply_Action()
}

type DoReply_New struct {
	New *NewReply `protobuf:"bytes,1,opt,name=new,proto3,oneof"`
}

type DoReply_Cancel struct {
	Cancel string `protobuf:"bytes,4,opt,name=cancel,proto3,oneof"` // Roll back the request. The message should contain the reason for canceling the request.
}

func (*DoReply_New) isDoReply_Action() {}

func (*DoReply_Cancel) isDoReply_Action() {}

var File_browsercontroller_v1_browsercontroller_proto protoreflect.FileDescriptor

var file_browsercontroller_v1_browsercontroller_proto_rawDesc = []byte{
	0x0a, 0x2c, 0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c,
	0x6c, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x63, 0x6f,
	0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x22,
	0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x62, 0x72,
	0x6f, 0x77, 0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x1a, 0x19, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x16, 0x6c,
	0x6f, 0x67, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x94, 0x02, 0x0a, 0x0b, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x65, 0x72, 0x4e, 0x65, 0x77, 0x12, 0x19, 0x0a, 0x08, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x5f, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x49, 0x64,
	0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x69, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75,
	0x72, 0x69, 0x12, 0x2c, 0x0a, 0x12, 0x63, 0x72, 0x61, 0x77, 0x6c, 0x5f, 0x65, 0x78, 0x65, 0x63,
	0x75, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10,
	0x63, 0x72, 0x61, 0x77, 0x6c, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64,
	0x12, 0x28, 0x0a, 0x10, 0x6a, 0x6f, 0x62, 0x5f, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f,
	0x6e, 0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x6a, 0x6f, 0x62, 0x45,
	0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x49, 0x0a, 0x0e, 0x63, 0x6f,
	0x6c, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x72, 0x65, 0x66, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x22, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x52, 0x65, 0x66, 0x52, 0x0d, 0x63, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x52, 0x65, 0x66, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x1d, 0x0a,
	0x0a, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x09, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x22, 0x9f, 0x01, 0x0a,
	0x0e, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x41, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x12,
	0x57, 0x0a, 0x08, 0x61, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x3b, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c,
	0x6c, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x41, 0x63, 0x74,
	0x69, 0x76, 0x69, 0x74, 0x79, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x52, 0x08,
	0x61, 0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x22, 0x34, 0x0a, 0x08, 0x41, 0x63, 0x74, 0x69,
	0x76, 0x69, 0x74, 0x79, 0x12, 0x11, 0x0a, 0x0d, 0x44, 0x41, 0x54, 0x41, 0x5f, 0x52, 0x45, 0x43,
	0x45, 0x49, 0x56, 0x45, 0x44, 0x10, 0x00, 0x12, 0x15, 0x0a, 0x11, 0x41, 0x4c, 0x4c, 0x5f, 0x44,
	0x41, 0x54, 0x41, 0x5f, 0x52, 0x45, 0x43, 0x45, 0x49, 0x56, 0x45, 0x44, 0x10, 0x01, 0x22, 0x60,
	0x0a, 0x09, 0x43, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x12, 0x3b, 0x0a, 0x09, 0x63,
	0x72, 0x61, 0x77, 0x6c, 0x5f, 0x6c, 0x6f, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e,
	0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x6c,
	0x6f, 0x67, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x61, 0x77, 0x6c, 0x4c, 0x6f, 0x67, 0x52, 0x08,
	0x63, 0x72, 0x61, 0x77, 0x6c, 0x4c, 0x6f, 0x67, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x61, 0x63, 0x68,
	0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x63, 0x61, 0x63, 0x68, 0x65, 0x64,
	0x22, 0xf7, 0x01, 0x0a, 0x09, 0x44, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x43,
	0x0a, 0x03, 0x6e, 0x65, 0x77, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2f, 0x2e, 0x76, 0x65,
	0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x62, 0x72, 0x6f, 0x77,
	0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x4e, 0x65, 0x77, 0x48, 0x00, 0x52, 0x03,
	0x6e, 0x65, 0x77, 0x12, 0x4c, 0x0a, 0x06, 0x6e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x32, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72,
	0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x41,
	0x63, 0x74, 0x69, 0x76, 0x69, 0x74, 0x79, 0x48, 0x00, 0x52, 0x06, 0x6e, 0x6f, 0x74, 0x69, 0x66,
	0x79, 0x12, 0x4d, 0x0a, 0x09, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74,
	0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x6c, 0x65,
	0x74, 0x65, 0x64, 0x48, 0x00, 0x52, 0x09, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x64,
	0x42, 0x08, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x84, 0x02, 0x0a, 0x08, 0x4e,
	0x65, 0x77, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x2c, 0x0a, 0x12, 0x63, 0x72, 0x61, 0x77, 0x6c,
	0x5f, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x10, 0x63, 0x72, 0x61, 0x77, 0x6c, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74,
	0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x28, 0x0a, 0x10, 0x6a, 0x6f, 0x62, 0x5f, 0x65, 0x78, 0x65,
	0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0e, 0x6a, 0x6f, 0x62, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12,
	0x49, 0x0a, 0x0e, 0x63, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x72, 0x65,
	0x66, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d,
	0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x76,
	0x31, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x66, 0x52, 0x0d, 0x63, 0x6f, 0x6c,
	0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x66, 0x12, 0x55, 0x0a, 0x12, 0x72, 0x65,
	0x70, 0x6c, 0x61, 0x63, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61,
	0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x76, 0x31,
	0x2e, 0x42, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52, 0x11,
	0x72, 0x65, 0x70, 0x6c, 0x61, 0x63, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x53, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x22, 0x6f, 0x0a, 0x07, 0x44, 0x6f, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x40, 0x0a, 0x03,
	0x6e, 0x65, 0x77, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x76, 0x65, 0x69, 0x64,
	0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x62, 0x72, 0x6f, 0x77, 0x73, 0x65,
	0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x4e,
	0x65, 0x77, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x48, 0x00, 0x52, 0x03, 0x6e, 0x65, 0x77, 0x12, 0x18,
	0x0a, 0x06, 0x63, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00,
	0x52, 0x06, 0x63, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x42, 0x08, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x32, 0x7b, 0x0a, 0x11, 0x42, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x43, 0x6f, 0x6e,
	0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x12, 0x66, 0x0a, 0x02, 0x64, 0x6f, 0x12, 0x2d, 0x2e,
	0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x62, 0x72,
	0x6f, 0x77, 0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x2e, 0x44, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2b, 0x2e, 0x76,
	0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x62, 0x72, 0x6f,
	0x77, 0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x2e, 0x76,
	0x31, 0x2e, 0x44, 0x6f, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x28, 0x01, 0x30, 0x01, 0x42,
	0x8e, 0x01, 0x0a, 0x2c, 0x6e, 0x6f, 0x2e, 0x6e, 0x62, 0x2e, 0x6e, 0x6e, 0x61, 0x2e, 0x76, 0x65,
	0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x62, 0x72, 0x6f, 0x77,
	0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x42, 0x18, 0x42, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c,
	0x6c, 0x65, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x01, 0x5a, 0x42, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x4e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x61,
	0x6c, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x79, 0x4f, 0x66, 0x4e, 0x6f, 0x72, 0x77, 0x61, 0x79,
	0x2f, 0x76, 0x65, 0x69, 0x64, 0x65, 0x6d, 0x61, 0x6e, 0x6e, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x62,
	0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x6c, 0x65, 0x72,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_browsercontroller_v1_browsercontroller_proto_rawDescOnce sync.Once
	file_browsercontroller_v1_browsercontroller_proto_rawDescData = file_browsercontroller_v1_browsercontroller_proto_rawDesc
)

func file_browsercontroller_v1_browsercontroller_proto_rawDescGZIP() []byte {
	file_browsercontroller_v1_browsercontroller_proto_rawDescOnce.Do(func() {
		file_browsercontroller_v1_browsercontroller_proto_rawDescData = protoimpl.X.CompressGZIP(file_browsercontroller_v1_browsercontroller_proto_rawDescData)
	})
	return file_browsercontroller_v1_browsercontroller_proto_rawDescData
}

var file_browsercontroller_v1_browsercontroller_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_browsercontroller_v1_browsercontroller_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_browsercontroller_v1_browsercontroller_proto_goTypes = []interface{}{
	(NotifyActivity_Activity)(0), // 0: veidemann.api.browsercontroller.v1.NotifyActivity.Activity
	(*RegisterNew)(nil),          // 1: veidemann.api.browsercontroller.v1.RegisterNew
	(*NotifyActivity)(nil),       // 2: veidemann.api.browsercontroller.v1.NotifyActivity
	(*Completed)(nil),            // 3: veidemann.api.browsercontroller.v1.Completed
	(*DoRequest)(nil),            // 4: veidemann.api.browsercontroller.v1.DoRequest
	(*NewReply)(nil),             // 5: veidemann.api.browsercontroller.v1.NewReply
	(*DoReply)(nil),              // 6: veidemann.api.browsercontroller.v1.DoReply
	(*config.ConfigRef)(nil),     // 7: veidemann.api.config.v1.ConfigRef
	(*log.CrawlLog)(nil),         // 8: veidemann.api.log.v1.CrawlLog
	(*config.BrowserScript)(nil), // 9: veidemann.api.config.v1.BrowserScript
}
var file_browsercontroller_v1_browsercontroller_proto_depIdxs = []int32{
	7,  // 0: veidemann.api.browsercontroller.v1.RegisterNew.collection_ref:type_name -> veidemann.api.config.v1.ConfigRef
	0,  // 1: veidemann.api.browsercontroller.v1.NotifyActivity.activity:type_name -> veidemann.api.browsercontroller.v1.NotifyActivity.Activity
	8,  // 2: veidemann.api.browsercontroller.v1.Completed.crawl_log:type_name -> veidemann.api.log.v1.CrawlLog
	1,  // 3: veidemann.api.browsercontroller.v1.DoRequest.new:type_name -> veidemann.api.browsercontroller.v1.RegisterNew
	2,  // 4: veidemann.api.browsercontroller.v1.DoRequest.notify:type_name -> veidemann.api.browsercontroller.v1.NotifyActivity
	3,  // 5: veidemann.api.browsercontroller.v1.DoRequest.completed:type_name -> veidemann.api.browsercontroller.v1.Completed
	7,  // 6: veidemann.api.browsercontroller.v1.NewReply.collection_ref:type_name -> veidemann.api.config.v1.ConfigRef
	9,  // 7: veidemann.api.browsercontroller.v1.NewReply.replacement_script:type_name -> veidemann.api.config.v1.BrowserScript
	5,  // 8: veidemann.api.browsercontroller.v1.DoReply.new:type_name -> veidemann.api.browsercontroller.v1.NewReply
	4,  // 9: veidemann.api.browsercontroller.v1.BrowserController.do:input_type -> veidemann.api.browsercontroller.v1.DoRequest
	6,  // 10: veidemann.api.browsercontroller.v1.BrowserController.do:output_type -> veidemann.api.browsercontroller.v1.DoReply
	10, // [10:11] is the sub-list for method output_type
	9,  // [9:10] is the sub-list for method input_type
	9,  // [9:9] is the sub-list for extension type_name
	9,  // [9:9] is the sub-list for extension extendee
	0,  // [0:9] is the sub-list for field type_name
}

func init() { file_browsercontroller_v1_browsercontroller_proto_init() }
func file_browsercontroller_v1_browsercontroller_proto_init() {
	if File_browsercontroller_v1_browsercontroller_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_browsercontroller_v1_browsercontroller_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RegisterNew); i {
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
		file_browsercontroller_v1_browsercontroller_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NotifyActivity); i {
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
		file_browsercontroller_v1_browsercontroller_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Completed); i {
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
		file_browsercontroller_v1_browsercontroller_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DoRequest); i {
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
		file_browsercontroller_v1_browsercontroller_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NewReply); i {
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
		file_browsercontroller_v1_browsercontroller_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DoReply); i {
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
	file_browsercontroller_v1_browsercontroller_proto_msgTypes[3].OneofWrappers = []interface{}{
		(*DoRequest_New)(nil),
		(*DoRequest_Notify)(nil),
		(*DoRequest_Completed)(nil),
	}
	file_browsercontroller_v1_browsercontroller_proto_msgTypes[5].OneofWrappers = []interface{}{
		(*DoReply_New)(nil),
		(*DoReply_Cancel)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_browsercontroller_v1_browsercontroller_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_browsercontroller_v1_browsercontroller_proto_goTypes,
		DependencyIndexes: file_browsercontroller_v1_browsercontroller_proto_depIdxs,
		EnumInfos:         file_browsercontroller_v1_browsercontroller_proto_enumTypes,
		MessageInfos:      file_browsercontroller_v1_browsercontroller_proto_msgTypes,
	}.Build()
	File_browsercontroller_v1_browsercontroller_proto = out.File
	file_browsercontroller_v1_browsercontroller_proto_rawDesc = nil
	file_browsercontroller_v1_browsercontroller_proto_goTypes = nil
	file_browsercontroller_v1_browsercontroller_proto_depIdxs = nil
}
