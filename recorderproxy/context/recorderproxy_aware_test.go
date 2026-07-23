/*
 * Copyright 2019 National Library of Norway.
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

package context

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
)

func Test_recordProxyDataAware(t *testing.T) {
	ctx1 := context.Background()
	ctx2 := RecordProxyDataAware(ctx1)
	ctx3 := RecordProxyDataAware(ctx2)

	if ctx1.Value(ctxKeyRecorderProxyAware) != nil {
		t.Error("Not expected to RecordProxyAware")
	}

	if ctx2.Value(ctxKeyRecorderProxyAware) == nil {
		t.Error("Expected to RecordProxyAware")
	}

	if !reflect.DeepEqual(ctx2, ctx3) {
		t.Error("Expected to get same context")
	}

	if !reflect.DeepEqual(ctx2.Value(ctxKeyRecorderProxyAware), ctx3.Value(ctxKeyRecorderProxyAware)) {
		t.Error("Expected to get same data")
	}
}

func TestGetHost(t *testing.T) {
	if got := GetHost(context.Background()); got != "" {
		t.Fatalf("GetHost() = %v, want empty", got)
	}

	ctx := RecordProxyDataAware(context.Background())
	if got := GetHost(ctx); got != "" {
		t.Fatalf("GetHost() = %v, want empty", got)
	}

	ctx = RecordProxyDataAware(context.Background())
	SetHost(ctx, "foo")
	if got := GetHost(ctx); got != "foo" {
		t.Fatalf("GetHost() = %v, want foo", got)
	}
}

func TestGetUrl(t *testing.T) {
	uri, _ := url.Parse("http://www.example.com")
	if got := GetUri(context.Background()); got != nil {
		t.Fatalf("GetUrl() = %v, want nil", got)
	}

	ctx := RecordProxyDataAware(context.Background())
	if got := GetUri(ctx); got != nil {
		t.Fatalf("GetUrl() = %v, want nil", got)
	}

	ctx = RecordProxyDataAware(context.Background())
	SetUri(ctx, uri)
	if got := GetUri(ctx); got != uri {
		t.Fatalf("GetUrl() = %v, want %v", got, uri)
	}
}

func TestGetRecordContext(t *testing.T) {
	if got := GetRecordContext(context.Background()); got != nil {
		t.Fatalf("GetRecordContext() = %v, want nil", got)
	}

	ctx := RecordProxyDataAware(context.Background())
	if got := GetRecordContext(ctx); got != nil {
		t.Fatalf("GetRecordContext() = %v, want nil", got)
	}

	want := &RecordContext{}
	ctx = RecordProxyDataAware(context.Background())
	SetRecordContext(ctx, want)
	if got := GetRecordContext(ctx); got != want {
		t.Fatalf("GetRecordContext() = %v, want %v", got, want)
	}
}

func TestResetRequestStateClearsRequestScopedValues(t *testing.T) {
	ctx := RecordProxyDataAware(context.Background())
	uri, _ := url.Parse("http://www.example.com")

	SetHost(ctx, "example.com")
	SetPort(ctx, "443")
	SetUri(ctx, uri)
	SetRequestId(ctx, "req-1")
	SetIp(ctx, "127.0.0.1")
	SetCrawlExecutionId(ctx, "ceid")
	SetJobExecutionId(ctx, "jeid")
	SetCollectionRef(ctx, nil)
	SetRecordContext(ctx, &RecordContext{})

	ResetRequestState(ctx, false)

	if got := GetHost(ctx); got != "" {
		t.Fatalf("GetHost() = %q, want empty", got)
	}
	if got := GetPort(ctx); got != "" {
		t.Fatalf("GetPort() = %q, want empty", got)
	}
	if got := GetUri(ctx); got != nil {
		t.Fatalf("GetUri() = %v, want nil", got)
	}
	if got := GetRequestId(ctx); got != "" {
		t.Fatalf("GetRequestId() = %q, want empty", got)
	}
	if got := GetIp(ctx); got != "" {
		t.Fatalf("GetIp() = %q, want empty", got)
	}
	if got := GetRecordContext(ctx); got != nil {
		t.Fatalf("GetRecordContext() = %v, want nil", got)
	}
	if got := GetCrawlExecutionId(ctx); got != "" {
		t.Fatalf("GetCrawlExecutionId() = %q, want empty", got)
	}
	if got := GetJobExecutionId(ctx); got != "" {
		t.Fatalf("GetJobExecutionId() = %q, want empty", got)
	}
}

func TestResetRequestStatePreservesSessionMetadata(t *testing.T) {
	ctx := RecordProxyDataAware(context.Background())
	uri, _ := url.Parse("http://www.example.com")
	collectionRef := &configV1.ConfigRef{Id: "col1"}

	SetHost(ctx, "example.com")
	SetPort(ctx, "443")
	SetUri(ctx, uri)
	SetRequestId(ctx, "req-1")
	SetCrawlExecutionId(ctx, "ceid")
	SetJobExecutionId(ctx, "jeid")
	SetCollectionRef(ctx, collectionRef)

	ResetRequestState(ctx, true)

	if got := GetHost(ctx); got != "example.com" {
		t.Fatalf("GetHost() = %q, want example.com", got)
	}
	if got := GetPort(ctx); got != "443" {
		t.Fatalf("GetPort() = %q, want 443", got)
	}
	if got := GetUri(ctx); got == nil || got.String() != uri.String() {
		t.Fatalf("GetUri() = %v, want %v", got, uri)
	}
	if got := GetRequestId(ctx); got != "" {
		t.Fatalf("GetRequestId() = %q, want empty", got)
	}
	if got := GetCrawlExecutionId(ctx); got != "ceid" {
		t.Fatalf("GetCrawlExecutionId() = %q, want ceid", got)
	}
	if got := GetJobExecutionId(ctx); got != "jeid" {
		t.Fatalf("GetJobExecutionId() = %q, want jeid", got)
	}
	if got := GetCollectionRef(ctx); got == nil || got.Id != "col1" {
		t.Fatalf("GetCollectionRef() = %v, want col1", got)
	}
}

func TestNewRequestContextPreservesSessionMetadataWithoutSharingRequestState(t *testing.T) {
	parent := RecordProxyDataAware(context.Background())
	uri, _ := url.Parse("https://www.example.com")
	collectionRef := &configV1.ConfigRef{Id: "col1"}
	parentRecordContext := &RecordContext{}

	SetHost(parent, "example.com")
	SetPort(parent, "443")
	SetUri(parent, uri)
	SetCrawlExecutionId(parent, "ceid")
	SetJobExecutionId(parent, "jeid")
	SetCollectionRef(parent, collectionRef)
	SetRequestId(parent, "req-parent")
	SetIp(parent, "127.0.0.1")
	SetRecordContext(parent, parentRecordContext)

	child := NewRequestContext(parent, true)

	if got := GetHost(child); got != "example.com" {
		t.Fatalf("GetHost() = %q, want example.com", got)
	}
	if got := GetPort(child); got != "443" {
		t.Fatalf("GetPort() = %q, want 443", got)
	}
	if got := GetUri(child); got == nil || got.String() != uri.String() {
		t.Fatalf("GetUri() = %v, want %v", got, uri)
	}
	if got := GetCrawlExecutionId(child); got != "ceid" {
		t.Fatalf("GetCrawlExecutionId() = %q, want ceid", got)
	}
	if got := GetJobExecutionId(child); got != "jeid" {
		t.Fatalf("GetJobExecutionId() = %q, want jeid", got)
	}
	if got := GetCollectionRef(child); got == nil || got.Id != "col1" {
		t.Fatalf("GetCollectionRef() = %v, want col1", got)
	}
	if got := GetRequestId(child); got != "" {
		t.Fatalf("GetRequestId() = %q, want empty", got)
	}
	if got := GetIp(child); got != "" {
		t.Fatalf("GetIp() = %q, want empty", got)
	}
	if got := GetRecordContext(child); got != nil {
		t.Fatalf("GetRecordContext() = %v, want nil", got)
	}

	childRecordContext := &RecordContext{}
	SetHost(child, "other.example.com")
	SetRequestId(child, "req-child")
	SetRecordContext(child, childRecordContext)

	if got := GetHost(parent); got != "example.com" {
		t.Fatalf("GetHost() on parent = %q, want example.com", got)
	}
	if got := GetRequestId(parent); got != "req-parent" {
		t.Fatalf("GetRequestId() on parent = %q, want req-parent", got)
	}
	if got := GetRecordContext(parent); got != parentRecordContext {
		t.Fatalf("GetRecordContext() on parent = %v, want %v", got, parentRecordContext)
	}
	if got := GetUri(child); got == uri {
		t.Fatal("GetUri() on child returned parent pointer, want copied URL")
	}
}

func TestNewRequestContextDropsSessionMetadataWhenNotPreserved(t *testing.T) {
	parent := RecordProxyDataAware(context.Background())
	uri, _ := url.Parse("https://www.example.com")

	SetHost(parent, "example.com")
	SetPort(parent, "443")
	SetUri(parent, uri)
	SetCrawlExecutionId(parent, "ceid")
	SetJobExecutionId(parent, "jeid")

	child := NewRequestContext(parent, false)

	if got := GetHost(child); got != "" {
		t.Fatalf("GetHost() = %q, want empty", got)
	}
	if got := GetPort(child); got != "" {
		t.Fatalf("GetPort() = %q, want empty", got)
	}
	if got := GetUri(child); got != nil {
		t.Fatalf("GetUri() = %v, want nil", got)
	}
	if got := GetCrawlExecutionId(child); got != "" {
		t.Fatalf("GetCrawlExecutionId() = %q, want empty", got)
	}
	if got := GetJobExecutionId(child); got != "" {
		t.Fatalf("GetJobExecutionId() = %q, want empty", got)
	}
}
