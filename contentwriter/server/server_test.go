/*
 * Copyright 2021 National Library of Norway.
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

package server

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/database"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/flags"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/writer"
	"github.com/go-redis/redismock/v9"
	"github.com/nlnwa/gowarc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

const bufSize = 1024 * 1024

type serverAndClient struct {
	lis        *bufconn.Listener
	dbMock     *r.Mock
	redisMock  redismock.ClientMock
	server     *grpc.Server
	clientConn *grpc.ClientConn
	client     contentwriterV1.ContentWriterClient
	registry   *warcWriterRegistry
}

func newServerAndClient(settings *flags.Mock) serverAndClient {
	dbMockConn := database.NewMockConnection()
	dbMockConn.GetMock().
		On(r.Table("config").Get("c1")).Return(map[string]interface{}{
		"id": "c1",
		"meta": map[string]interface{}{
			"name": "c1",
		},
		"collection": map[string]interface{}{
			"collectionDedupPolicy": "HOURLY",
			"fileRotationPolicy":    "MONTHLY",
		}}, nil).
		On(r.Table("config").Get("c2")).Return(map[string]interface{}{
		"id": "c2",
		"meta": map[string]interface{}{
			"name": "c2",
		},
		"collection": map[string]interface{}{
			"collectionDedupPolicy": "HOURLY",
			"fileRotationPolicy":    "MONTHLY",
			"compress":              true,
		}}, nil)

	lis := bufconn.Listen(bufSize)

	rdb, mock := redismock.NewClientMock()
	warcWriterRegistry := newWarcWriterRegistry(
		writer.Options{
			WarcDir:     settings.WarcDir(),
			WarcVersion: gowarc.V1_1,
			Flush:       true,
			PoolSize:    settings.WarcWriterPoolSize(),
		},
		database.NewConfigCache(dbMockConn, time.Duration(1)),
		&database.CrawledContentHashCache{
			Client: rdb,
		},
	)
	contentWriterService := &ContentWriterService{
		configCache:        database.NewConfigCache(dbMockConn, time.Duration(1)),
		warcWriterRegistry: warcWriterRegistry,
	}

	grpcServer := grpc.NewServer()
	contentwriterV1.RegisterContentWriterServer(grpcServer, contentWriterService)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			panic(fmt.Errorf("Server exited with error: %v", err))
		}
	}()

	// Set up client
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("localhost", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Errorf("Failed to dial bufnet: %v", err))
	}

	return serverAndClient{
		server:     grpcServer,
		clientConn: conn,
		client:     contentwriterV1.NewContentWriterClient(conn),
		dbMock:     dbMockConn.GetMock(),
		redisMock:  mock,
		lis:        lis,
		registry:   warcWriterRegistry,
	}
}

func (s serverAndClient) close() {
	_ = s.clientConn.Close()
	s.server.GracefulStop()
	s.registry.Close()
}

type writeRequests []*contentwriterV1.WriteRequest

var writeReq1 = writeRequests{
	&contentwriterV1.WriteRequest{Value: &contentwriterV1.WriteRequest_ProtocolHeader{ProtocolHeader: &contentwriterV1.Data{
		RecordNum: 0,
		Data: []byte("GET / HTTP/1.0\r\n" +
			"Host: example.com\r\n" +
			"Accept-Language: en-US,en;q=0.8,ru;q=0.6\r\n" +
			"Referer: http://example.com/foo.html\r\n" +
			"Connection: close\r\n" +
			"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36\r\n" +
			"\r\n",
		),
	}}},
	&contentwriterV1.WriteRequest{Value: &contentwriterV1.WriteRequest_ProtocolHeader{ProtocolHeader: &contentwriterV1.Data{
		RecordNum: 1,
		Data: []byte("HTTP/1.1 200 OK\r\n" +
			"Date: Tue, 19 Sep 2016 17:18:40 GMT\r\n" +
			"Server: Apache/2.0.54 (Ubuntu)\r\n" +
			"Last-Modified: Mon, 16 Jun 2013 22:28:51 GMT\r\n" +
			"ETag: \"3e45-67e-2ed02ec0\"\r\n" +
			"Accept-Ranges: bytes\r\n" +
			"Content-Length: 19\r\n" +
			"Connection: close\r\n" +
			"Content-Type: text/plain\r\n" +
			"\r\n",
		),
	}}},
	&contentwriterV1.WriteRequest{Value: &contentwriterV1.WriteRequest_Payload{Payload: &contentwriterV1.Data{
		RecordNum: 1,
		Data:      []byte("This is the content"),
	}}},
	&contentwriterV1.WriteRequest{Value: &contentwriterV1.WriteRequest_Meta{Meta: &contentwriterV1.WriteRequestMeta{
		ExecutionId: "eid1",
		TargetUri:   "http://www.example.com/foo.html",
		RecordMeta: map[int32]*contentwriterV1.WriteRequestMeta_RecordMeta{
			0: {
				RecordNum:         0,
				Type:              contentwriterV1.RecordType_REQUEST,
				Size:              270,
				RecordContentType: "application/http;msgtype=request",
				BlockDigest:       "sha1:9CA62209A0DE739B1A9DDB119BAFBE63539820FC",
				PayloadDigest:     "sha1:DA39A3EE5E6B4B0D3255BFEF95601890AFD80709",
			},
			1: {
				RecordNum:         1,
				Type:              contentwriterV1.RecordType_RESPONSE,
				Size:              267,
				RecordContentType: "application/http;msgtype=response",
				BlockDigest:       "sha1:4126C2DC27F113BEEC37A46276514CD4300DA10D",
				PayloadDigest:     "sha1:C37FFB221569C553A2476C22C7DAD429F3492977",
			},
		},
		FetchTimeStamp: timestamppb.Now(),
		IpAddress:      "127.0.0.1",
		CollectionRef:  &configV1.ConfigRef{Kind: configV1.Kind_collection, Id: "c1"},
	}}},
}

var writeReq2 = writeRequests{
	&contentwriterV1.WriteRequest{Value: &contentwriterV1.WriteRequest_ProtocolHeader{ProtocolHeader: &contentwriterV1.Data{
		RecordNum: 0,
		Data: []byte("GET / HTTP/1.0\r\n" +
			"Host: example.com\r\n" +
			"Accept-Language: en-US,en;q=0.8,ru;q=0.6\r\n" +
			"Referer: http://example.com/foo.html\r\n" +
			"Connection: close\r\n" +
			"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36\r\n" +
			"\r\n",
		),
	}}},
	&contentwriterV1.WriteRequest{Value: &contentwriterV1.WriteRequest_ProtocolHeader{ProtocolHeader: &contentwriterV1.Data{
		RecordNum: 1,
		Data: []byte("HTTP/1.1 200 OK\r\n" +
			"Date: Tue, 19 Sep 2016 17:18:40 GMT\r\n" +
			"Server: Apache/2.0.54 (Ubuntu)\r\n" +
			"Last-Modified: Mon, 16 Jun 2013 22:28:51 GMT\r\n" +
			"ETag: \"3e45-67e-2ed02ec0\"\r\n" +
			"Accept-Ranges: bytes\r\n" +
			"Content-Length: 19\r\n" +
			"Connection: close\r\n" +
			"Content-Type: text/plain\r\n" +
			"\r\n",
		),
	}}},
	&contentwriterV1.WriteRequest{Value: &contentwriterV1.WriteRequest_Payload{Payload: &contentwriterV1.Data{
		RecordNum: 1,
		Data:      []byte("This is the content"),
	}}},
	&contentwriterV1.WriteRequest{Value: &contentwriterV1.WriteRequest_Meta{Meta: &contentwriterV1.WriteRequestMeta{
		ExecutionId: "eid1",
		TargetUri:   "http://www.example.com/foo.html",
		RecordMeta: map[int32]*contentwriterV1.WriteRequestMeta_RecordMeta{
			0: {
				RecordNum:         0,
				Type:              contentwriterV1.RecordType_REQUEST,
				Size:              270,
				RecordContentType: "application/http;msgtype=request",
				BlockDigest:       "sha1:9CA62209A0DE739B1A9DDB119BAFBE63539820FC",
				PayloadDigest:     "sha1:DA39A3EE5E6B4B0D3255BFEF95601890AFD80709",
			},
			1: {
				RecordNum:         1,
				Type:              contentwriterV1.RecordType_RESPONSE,
				Size:              267,
				RecordContentType: "application/http;msgtype=response",
				BlockDigest:       "sha1:4126C2DC27F113BEEC37A46276514CD4300DA10D",
				PayloadDigest:     "sha1:C37FFB221569C553A2476C22C7DAD429F3492977",
			},
		},
		FetchTimeStamp: timestamppb.Now(),
		IpAddress:      "127.0.0.1",
		CollectionRef:  &configV1.ConfigRef{Kind: configV1.Kind_collection, Id: "c2"},
	}}},
}

func TestContentWriterService_Write(t *testing.T) {
	testSettings := flags.NewMock(t.TempDir(), 1)

	writer.Now = func() time.Time {
		return time.Date(2000, 10, 10, 2, 59, 59, 0, time.UTC)
	}

	serverAndClient := newServerAndClient(testSettings)
	redisMock := serverAndClient.redisMock
	redisMock.ExpectHGet(
		"c1_2000101002",
		"sha1:C37FFB221569C553A2476C22C7DAD429F3492977",
	).RedisNil()

	redisMock.Regexp().ExpectHSet(
		"c1_2000101002",
		"sha1:C37FFB221569C553A2476C22C7DAD429F3492977",
		"(?s).*",
	).SetVal(1)

	redisMock.CustomMatch(func(expectArgs, cmdArgs []interface{}) error {
		// cmdArgs: ["expire", key, ttl]
		if len(cmdArgs) != 4 {
			return fmt.Errorf("unexpected args: %#v", cmdArgs)
		}
		if cmdArgs[0] != "expire" {
			return fmt.Errorf("unexpected command: %v", cmdArgs[0])
		}
		if cmdArgs[1] != "c1_2000101002" {
			return fmt.Errorf("unexpected key: %v", cmdArgs[1])
		}
		// ignore ttl completely
		return nil
	}).ExpectExpireNX("c1_2000101002", 0).SetVal(true)

	ctx := context.Background()
	assert := assert.New(t)

	stream, err := serverAndClient.client.Write(ctx)
	assert.NoError(err)
	for i, r := range writeReq1 {
		err = stream.Send(r)
		assert.NoErrorf(err, "Error sending request #%d", i)
	}
	reply, err := stream.CloseAndRecv()
	assert.NoError(err)
	if reply == nil {
		t.Fatalf("Reply is nil")
	}
	assert.NoError(serverAndClient.redisMock.ExpectationsWereMet())

	assert.Equal(2, len(reply.Meta.RecordMeta))

	fileNamePattern := `c1_2000101002-\d{14}-0001-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+).warc`

	assert.Equal(int32(0), reply.Meta.RecordMeta[0].RecordNum)
	assert.Equal(contentwriterV1.RecordType_REQUEST, reply.Meta.RecordMeta[0].Type)
	assert.Regexp(".{8}-.{4}-.{4}-.{4}-.{12}", reply.Meta.RecordMeta[0].WarcId)
	assert.Equal("sha1:9CA62209A0DE739B1A9DDB119BAFBE63539820FC", reply.Meta.RecordMeta[0].BlockDigest)
	assert.Equal("sha1:DA39A3EE5E6B4B0D3255BFEF95601890AFD80709", reply.Meta.RecordMeta[0].PayloadDigest)
	assert.Equal("c1_2000101002", reply.Meta.RecordMeta[0].CollectionFinalName)
	assert.Equal("", reply.Meta.RecordMeta[0].RevisitReferenceId)
	assert.Regexp("warcfile:"+fileNamePattern+`:\d\d\d$`, reply.Meta.RecordMeta[0].StorageRef)

	assert.Equal(int32(1), reply.Meta.RecordMeta[1].RecordNum)
	assert.Equal(contentwriterV1.RecordType_RESPONSE, reply.Meta.RecordMeta[1].Type)
	assert.Regexp(".{8}-.{4}-.{4}-.{4}-.{12}", reply.Meta.RecordMeta[1].WarcId)
	assert.Equal("sha1:4126C2DC27F113BEEC37A46276514CD4300DA10D", reply.Meta.RecordMeta[1].BlockDigest)
	assert.Equal("sha1:C37FFB221569C553A2476C22C7DAD429F3492977", reply.Meta.RecordMeta[1].PayloadDigest)
	assert.Equal("c1_2000101002", reply.Meta.RecordMeta[1].CollectionFinalName)
	assert.Equal("", reply.Meta.RecordMeta[1].RevisitReferenceId)
	assert.Regexp("warcfile:"+fileNamePattern+`:\d\d\d\d$`, reply.Meta.RecordMeta[1].StorageRef)

	dirHasFilesMatching(t, testSettings.WarcDir(), "^"+fileNamePattern+".open$", 1)
	serverAndClient.close()
	dirHasFilesMatching(t, testSettings.WarcDir(), "^"+fileNamePattern+"$", 1)
}

func TestContentWriterService_Write_Compressed(t *testing.T) {
	testSettings := flags.NewMock(t.TempDir(), 1)

	writer.Now = func() time.Time {
		return time.Date(2000, 10, 10, 2, 59, 59, 0, time.UTC)
	}

	serverAndClient := newServerAndClient(testSettings)
	redisMock := serverAndClient.redisMock
	redisMock.ExpectHGet(
		"c2_2000101002",
		"sha1:C37FFB221569C553A2476C22C7DAD429F3492977",
	).RedisNil()

	redisMock.Regexp().ExpectHSet(
		"c2_2000101002",
		"sha1:C37FFB221569C553A2476C22C7DAD429F3492977",
		"(?s).*",
	).SetVal(1)

	redisMock.CustomMatch(func(expectArgs, cmdArgs []interface{}) error {
		// cmdArgs: ["expire", key, ttl]
		if len(cmdArgs) != 4 {
			return fmt.Errorf("unexpected args: %#v", cmdArgs)
		}
		if cmdArgs[0] != "expire" {
			return fmt.Errorf("unexpected command: %v", cmdArgs[0])
		}
		if cmdArgs[1] != "c2_2000101002" {
			return fmt.Errorf("unexpected key: %v", cmdArgs[1])
		}
		// ignore ttl completely
		return nil
	}).ExpectExpireNX("c2_2000101002", 0).SetVal(true)

	ctx := context.Background()
	assert := assert.New(t)

	stream, err := serverAndClient.client.Write(ctx)
	assert.NoError(err)
	for i, r := range writeReq2 {
		err = stream.Send(r)
		assert.NoErrorf(err, "Error sending request #%d", i)
	}
	reply, err := stream.CloseAndRecv()
	assert.NoError(err)
	if reply == nil {
		t.Fatalf("Reply is nil")
	}
	assert.NoError(err)
	assert.Equal(2, len(reply.GetMeta().GetRecordMeta()))

	assert.NoError(serverAndClient.redisMock.ExpectationsWereMet())

	fileNamePattern := `c2_2000101002-\d{14}-0001-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+).warc.gz`

	assert.Equal(int32(0), reply.Meta.RecordMeta[0].RecordNum)
	assert.Equal(contentwriterV1.RecordType_REQUEST, reply.Meta.RecordMeta[0].Type)
	assert.Regexp(".{8}-.{4}-.{4}-.{4}-.{12}", reply.Meta.RecordMeta[0].WarcId)
	assert.Equal("sha1:9CA62209A0DE739B1A9DDB119BAFBE63539820FC", reply.Meta.RecordMeta[0].BlockDigest)
	assert.Equal("sha1:DA39A3EE5E6B4B0D3255BFEF95601890AFD80709", reply.Meta.RecordMeta[0].PayloadDigest)
	assert.Equal("c2_2000101002", reply.Meta.RecordMeta[0].CollectionFinalName)
	assert.Equal("", reply.Meta.RecordMeta[0].RevisitReferenceId)
	assert.Regexp("warcfile:"+fileNamePattern+`:\d\d\d$`, reply.Meta.RecordMeta[0].StorageRef)

	assert.Equal(int32(1), reply.Meta.RecordMeta[1].RecordNum)
	assert.Equal(contentwriterV1.RecordType_RESPONSE, reply.Meta.RecordMeta[1].Type)
	assert.Regexp(".{8}-.{4}-.{4}-.{4}-.{12}", reply.Meta.RecordMeta[1].WarcId)
	assert.Equal("sha1:4126C2DC27F113BEEC37A46276514CD4300DA10D", reply.Meta.RecordMeta[1].BlockDigest)
	assert.Equal("sha1:C37FFB221569C553A2476C22C7DAD429F3492977", reply.Meta.RecordMeta[1].PayloadDigest)
	assert.Equal("c2_2000101002", reply.Meta.RecordMeta[1].CollectionFinalName)
	assert.Equal("", reply.Meta.RecordMeta[1].RevisitReferenceId)
	assert.Regexp("warcfile:"+fileNamePattern+`:\d\d\d$`, reply.Meta.RecordMeta[1].StorageRef)

	dirHasFilesMatching(t, testSettings.WarcDir(), "^"+fileNamePattern+".open$", 1)
	serverAndClient.close()
	dirHasFilesMatching(t, testSettings.WarcDir(), "^"+fileNamePattern+"$", 1)
}

func TestContentWriterService_WriteRevisit(t *testing.T) {
	testSettings := flags.NewMock(t.TempDir(), 1)

	writer.Now = func() time.Time {
		return time.Date(2000, 10, 10, 2, 59, 59, 0, time.UTC)
	}

	serverAndClient := newServerAndClient(testSettings)

	crawledContent := &contentwriterV1.CrawledContent{
		Date:      timestamppb.New(time.Date(2021, 8, 27, 13, 52, 0, 0, time.UTC)),
		Digest:    "sha1:C37FFB221569C553A2476C22C7DAD429F3492977:c1_2000101002",
		TargetUri: "http://www.example.com",
		WarcId:    "fff232109-0d71-467f-b728-de86be386c6f",
	}
	b, err := proto.Marshal(crawledContent)
	if err != nil {
		t.Fatalf("Failed to marshal CrawledContent: %v", err)
	}

	redisMock := serverAndClient.redisMock
	redisMock.Regexp().ExpectHGet(
		"c1_2000101002",
		"sha1:C37FFB221569C553A2476C22C7DAD429F3492977",
	).SetVal(string(b))

	ctx := context.Background()
	assert := assert.New(t)

	stream, err := serverAndClient.client.Write(ctx)
	assert.NoError(err)
	for i, r := range writeReq1 {
		err = stream.Send(r)
		assert.NoErrorf(err, "Error sending request #%d", i)
	}
	reply, err := stream.CloseAndRecv()
	if reply == nil {
		t.Fatalf("Reply is nil")
	}
	assert.NoError(err)
	assert.NoError(serverAndClient.redisMock.ExpectationsWereMet())
	assert.Equal(2, len(reply.Meta.RecordMeta))

	fileNamePattern := `c1_2000101002-\d{14}-0001-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+).warc`

	assert.Equal(int32(0), reply.Meta.RecordMeta[0].RecordNum)
	assert.Equal(contentwriterV1.RecordType_REQUEST, reply.Meta.RecordMeta[0].Type)
	assert.Regexp(".{8}-.{4}-.{4}-.{4}-.{12}", reply.Meta.RecordMeta[0].WarcId)
	assert.Equal("sha1:9CA62209A0DE739B1A9DDB119BAFBE63539820FC", reply.Meta.RecordMeta[0].BlockDigest)
	assert.Equal("sha1:DA39A3EE5E6B4B0D3255BFEF95601890AFD80709", reply.Meta.RecordMeta[0].PayloadDigest)
	assert.Equal("c1_2000101002", reply.Meta.RecordMeta[0].CollectionFinalName)
	assert.Equal("", reply.Meta.RecordMeta[0].RevisitReferenceId)
	assert.Regexp(`warcfile:`+fileNamePattern+`:\d\d\d`, reply.Meta.RecordMeta[0].StorageRef)

	assert.Equal(int32(1), reply.Meta.RecordMeta[1].RecordNum)
	assert.Equal(contentwriterV1.RecordType_REVISIT, reply.Meta.RecordMeta[1].Type)
	assert.Regexp(".{8}-.{4}-.{4}-.{4}-.{12}", reply.Meta.RecordMeta[1].WarcId)
	assert.Equal("sha1:YO5NSCLIZRCG75SP5WBNAMFKWWQLLCCK", reply.Meta.RecordMeta[1].BlockDigest)
	assert.Equal("sha1:C37FFB221569C553A2476C22C7DAD429F3492977", reply.Meta.RecordMeta[1].PayloadDigest)
	assert.Equal("c1_2000101002", reply.Meta.RecordMeta[1].CollectionFinalName)
	assert.Equal("<urn:uuid:fff232109-0d71-467f-b728-de86be386c6f>", reply.Meta.RecordMeta[1].RevisitReferenceId)
	assert.Regexp(`warcfile:`+fileNamePattern+`:\d\d\d\d`, reply.Meta.RecordMeta[1].StorageRef)

	dirHasFilesMatching(t, testSettings.WarcDir(), "^"+fileNamePattern+".open$", 1)
	serverAndClient.close()
	dirHasFilesMatching(t, testSettings.WarcDir(), "^"+fileNamePattern+"$", 1)
}

func dirHasFilesMatching(t *testing.T, dir string, pattern string, count int) bool {
	p := regexp.MustCompile(pattern)
	var found []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if p.MatchString(d.Name()) {
			found = append(found, d.Name())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Error walking directory %q: %v", dir, err)
	}
	if len(found) == count {
		t.Log("Found matching file:", found, "pattern:", pattern)
		return true
	} else {
		t.Log("Did not find all files", found, "expected:", count, "pattern:", pattern)
		f := ""
		for _, ff := range found {
			f += "\n  " + ff
		}
		return assert.Fail(t, "Wrong number of files in '"+dir+"'", "Expected %d files to match %s, but found %d\nFiles in dir:%s", count, pattern, found, f)
	}
}
