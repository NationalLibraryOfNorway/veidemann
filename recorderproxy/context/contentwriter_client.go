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
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"sync"
	"sync/atomic"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
)

type CwcSession struct {
	contentwriterV1.ContentWriter_WriteClient
	done            bool
	canceled        bool
	m               sync.Mutex
	cwcCtx          context.Context
	ctxCancel       context.CancelFunc
	sendSeq         uint64
	sendInFlight    int32
	sendStateMu     sync.Mutex
	protocolHeaders map[int32][]byte
}

func headerDigest(data []byte) string {
	sum := sha1.Sum(data)
	return fmt.Sprintf("%x", sum[:])
}

func headerPreview(data []byte) string {
	const maxPreview = 160
	if len(data) <= maxPreview {
		return string(data)
	}
	return string(data[:maxPreview]) + "...<truncated>"
}

func (cwc *CwcSession) beginSend() (uint64, int32) {
	seq := atomic.AddUint64(&cwc.sendSeq, 1)
	inFlight := atomic.AddInt32(&cwc.sendInFlight, 1)
	return seq, inFlight
}

func (cwc *CwcSession) endSend() {
	atomic.AddInt32(&cwc.sendInFlight, -1)
}

func (cwc *CwcSession) noteProtocolHeaderAttempt(recNum int32, header []byte) ([]byte, bool) {
	cwc.sendStateMu.Lock()
	defer cwc.sendStateMu.Unlock()

	if prev, ok := cwc.protocolHeaders[recNum]; ok {
		return append([]byte(nil), prev...), true
	}

	cwc.protocolHeaders[recNum] = append([]byte(nil), header...)
	return nil, false
}

func (rc *RecordContext) logConcurrentSend(sendType string, recNum int32, payloadSize int, seq uint64, inFlight int32) {
	l := LogWithContext(rc.ctx, "PROXY:CWC")
	uri := ""
	if rc.Uri != nil {
		uri = rc.Uri.String()
	}

	l.WithField("requestId", rc.RequestId).
		WithField("session", rc.Session()).
		WithField("uri", uri).
		WithField("sendType", sendType).
		WithField("recordNum", recNum).
		WithField("payloadSize", payloadSize).
		WithField("seq", seq).
		WithField("inFlight", inFlight).
		Warn("Concurrent ContentWriter stream send detected")
}

func (rc *RecordContext) logDuplicateProtocolHeader(recNum int32, seq uint64, header, previous []byte) {
	l := LogWithContext(rc.ctx, "PROXY:CWC")
	uri := ""
	if rc.Uri != nil {
		uri = rc.Uri.String()
	}

	l.WithField("requestId", rc.RequestId).
		WithField("session", rc.Session()).
		WithField("uri", uri).
		WithField("recordNum", recNum).
		WithField("seq", seq).
		WithField("sameBytes", bytes.Equal(previous, header)).
		WithField("previousDigest", headerDigest(previous)).
		WithField("currentDigest", headerDigest(header)).
		WithField("previousPreview", headerPreview(previous)).
		WithField("currentPreview", headerPreview(header)).
		Warn("Duplicate ContentWriter protocol header send attempt detected")
}

func (rc *RecordContext) sendWriteRequest(cwc *CwcSession, sendType string, recNum int32, payload []byte, request *contentwriterV1.WriteRequest) error {
	seq, inFlight := cwc.beginSend()
	defer cwc.endSend()

	if inFlight > 1 {
		rc.logConcurrentSend(sendType, recNum, len(payload), seq, inFlight)
	}

	if sendType == "protocolHeader" {
		if prev, duplicate := cwc.noteProtocolHeaderAttempt(recNum, payload); duplicate {
			rc.logDuplicateProtocolHeader(recNum, seq, payload, prev)
		}
	}

	return cwc.Send(request)
}

func (rc *RecordContext) getCwcSession() (*CwcSession, error) {
	if rc.cwc != nil {
		return rc.cwc, nil
	}

	l := LogWithContext(rc.ctx, "PROXY:CWC")

	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	cwcCtx, cancel := context.WithCancel(context.Background())

	cwc, err := rc.conn.ContentWriterClient().Write(cwcCtx)
	if err != nil {
		l.WithError(err).Warn("Error connecting to ContentWriter")
		err = errors.WrapInternalError(err, errors.RuntimeException, "Error connecting to ContentWriter", err.Error())
		cancel()
		return nil, err
	}

	c := &CwcSession{
		ContentWriter_WriteClient: cwc,
		cwcCtx:                    cwcCtx,
		ctxCancel:                 cancel,
		protocolHeaders:           make(map[int32][]byte, 2),
	}
	rc.cwc = c

	go func() {
		<-rc.ctx.Done()
		defer rc.closeSession()
		c.m.Lock()
		defer c.m.Unlock()
		if !c.done {
			c.done = true
			l.Info("ContentWriter client session canceled by client")
			err := rc.sendWriteRequest(c, "cancel", -1, nil, &contentwriterV1.WriteRequest{
				Value: &contentwriterV1.WriteRequest_Cancel{Cancel: "Veidemann recorder proxy lost connection to client"},
			})
			if err != nil {
				l.WithError(err).Warn("Error writing to ContentWriter")
			}
			_, err = c.CloseAndRecv()
			if err != nil {
				l.WithError(err).Warn("Error closing from ContentWriter")
			}
		}
		cancel()
	}()

	return c, nil
}

func (rc *RecordContext) CancelContentWriter(msg string) error {
	defer rc.closeSession()

	if rc.cwc == nil {
		// No ContentWriter session to cancel
		return nil
	}

	l := LogWithContext(rc.ctx, "PROXY:CWC")

	cwc, err := rc.getCwcSession()
	if cwc != nil {
		cwc.canceled = true
	}
	if err != nil {
		return err
	}

	cwc.m.Lock()
	defer cwc.m.Unlock()
	if !cwc.done {
		cwc.done = true
		defer cwc.ctxCancel()

		err = rc.sendWriteRequest(cwc, "cancel", -1, nil, &contentwriterV1.WriteRequest{Value: &contentwriterV1.WriteRequest_Cancel{Cancel: msg}})
		if err != nil {
			l.WithError(err).Info("Error sending ContentWriter cancel")
		}
		_, err := cwc.CloseAndRecv()
		if err != nil {
			l.WithError(err).Info("Error sending ContentWriter cancel")
		}
	}
	return err
}

func (rc *RecordContext) SendProtocolHeader(recNum int32, p []byte) error {
	l := LogWithContext(rc.ctx, "PROXY:CWC")

	cwc, err := rc.getCwcSession()
	if err != nil {
		return err
	}

	if cwc.canceled {
		return nil
	}

	protocolHeaderRequest := &contentwriterV1.WriteRequest{
		Value: &contentwriterV1.WriteRequest_ProtocolHeader{
			ProtocolHeader: &contentwriterV1.Data{
				RecordNum: recNum,
				Data:      p,
			},
		},
	}

	err = rc.sendWriteRequest(cwc, "protocolHeader", recNum, p, protocolHeaderRequest)
	if err != nil {
		l.WithError(err).Info("Error sending ContentWriter protocol header")
	}
	return err
}

func (rc *RecordContext) SendPayload(recNum int32, p []byte) error {
	l := LogWithContext(rc.ctx, "PROXY:CWC")

	cwc, err := rc.getCwcSession()
	if err != nil {
		return err
	}

	if cwc.canceled {
		return nil
	}

	payloadRequest := &contentwriterV1.WriteRequest{
		Value: &contentwriterV1.WriteRequest_Payload{
			Payload: &contentwriterV1.Data{
				RecordNum: recNum,
				Data:      p,
			},
		},
	}

	err = rc.sendWriteRequest(cwc, "payload", recNum, p, payloadRequest)
	if err != nil {
		l.WithError(err).Info("Error sending ContentWriter payload")
	}
	return err
}

func (rc *RecordContext) SendMeta() (reply *contentwriterV1.WriteReply, err error) {
	l := LogWithContext(rc.ctx, "PROXY:CWC")

	cwc, err := rc.getCwcSession()
	if err != nil {
		return nil, err
	}

	if cwc.canceled {
		return nil, nil
	}

	cwc.m.Lock()
	defer cwc.m.Unlock()
	if !cwc.done {
		cwc.done = true
		defer cwc.ctxCancel()

		metaRequest := &contentwriterV1.WriteRequest{
			Value: rc.Meta,
		}

		err = rc.sendWriteRequest(cwc, "meta", -1, nil, metaRequest)
		if err != nil {
			l.WithError(err).Info("Error sending ContentWriter meta")
			return nil, err
		}

		reply, err = cwc.CloseAndRecv()
		if err != nil {
			l.WithError(err).Info("Error receiving ContentWriter meta response")
		}
	}
	return
}
