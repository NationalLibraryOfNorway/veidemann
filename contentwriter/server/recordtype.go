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
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/nlnwa/gowarc"
)

func ToGowarcRecordType(recordType contentwriterV1.RecordType) gowarc.RecordType {
	switch recordType {
	case contentwriterV1.RecordType_WARCINFO:
		return gowarc.Warcinfo
	case contentwriterV1.RecordType_RESPONSE:
		return gowarc.Response
	case contentwriterV1.RecordType_RESOURCE:
		return gowarc.Resource
	case contentwriterV1.RecordType_REQUEST:
		return gowarc.Request
	case contentwriterV1.RecordType_METADATA:
		return gowarc.Metadata
	case contentwriterV1.RecordType_REVISIT:
		return gowarc.Revisit
	case contentwriterV1.RecordType_CONVERSION:
		return gowarc.Conversion
	case contentwriterV1.RecordType_CONTINUATION:
		return gowarc.Continuation
	default:
		return 0
	}
}

func FromGowarcRecordType(recordType gowarc.RecordType) contentwriterV1.RecordType {
	switch recordType {
	case gowarc.Warcinfo:
		return contentwriterV1.RecordType_WARCINFO
	case gowarc.Response:
		return contentwriterV1.RecordType_RESPONSE
	case gowarc.Resource:
		return contentwriterV1.RecordType_RESOURCE
	case gowarc.Request:
		return contentwriterV1.RecordType_REQUEST
	case gowarc.Metadata:
		return contentwriterV1.RecordType_METADATA
	case gowarc.Revisit:
		return contentwriterV1.RecordType_REVISIT
	case gowarc.Conversion:
		return contentwriterV1.RecordType_CONVERSION
	case gowarc.Continuation:
		return contentwriterV1.RecordType_CONTINUATION
	default:
		return 0
	}
}
