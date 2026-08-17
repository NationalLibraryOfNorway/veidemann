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
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
)

type CwcSession struct {
	contentwriterV1.ContentWriter_WriteClient
	done            bool
	canceled        bool
	m               sync.Mutex
	sendMu          sync.Mutex
	cwcCtx          context.Context
	ctxCancel       context.CancelFunc
	cancelOnce      sync.Once
	sendSeq         uint64
	sendInFlight    int32
	sendStateMu     sync.Mutex
	protocolHeaders map[int32][]byte
}

func (cwc *CwcSession) cancelContext() {
	cwc.cancelOnce.Do(cwc.ctxCancel)
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
	cwc.sendMu.Lock()
	defer cwc.sendMu.Unlock()
	if sendType != "meta" && sendType != "cancel" {
		cwc.m.Lock()
		terminal := cwc.done
		cwc.m.Unlock()
		if terminal {
			return nil
		}
	}

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
	return rc.getCwcSessionForTerminal(false)
}

func (rc *RecordContext) getCwcSessionForTerminal(terminal bool) (*CwcSession, error) {
	l := LogWithContext(rc.ctx, "PROXY:CWC")

	for {
		rc.mutex.Lock()
		if rc.cwc != nil {
			cwc := rc.cwc
			rc.mutex.Unlock()
			return cwc, nil
		}
		if rc.cwcErr != nil {
			err := rc.cwcErr
			rc.mutex.Unlock()
			return nil, err
		}
		if rc.cwcCreating {
			ready := rc.cwcReady
			rc.mutex.Unlock()
			<-ready
			continue
		}
		if rc.done && !terminal {
			err := rc.terminalErr
			if err == nil {
				err = AlreadyCompleted
			}
			rc.mutex.Unlock()
			return nil, err
		}
		rc.cwcCreating = true
		rc.cwcReady = make(chan struct{})
		rc.mutex.Unlock()
		break
	}

	c, err := rc.openCwcSession()
	if err != nil {
		l.WithError(err).Warn("Error connecting to ContentWriter")
		err = errors.WrapInternalError(err, errors.RuntimeException, "Error connecting to ContentWriter", err.Error())
	}

	rc.mutex.Lock()
	rc.cwc = c
	rc.cwcErr = err
	rc.cwcCreating = false
	close(rc.cwcReady)
	rc.mutex.Unlock()

	return c, err
}

func (rc *RecordContext) openCwcSession() (*CwcSession, error) {
	cwcCtx, cancel := context.WithCancel(context.Background())
	cwc, err := rc.conn.ContentWriterClient().Write(cwcCtx)
	if err != nil {
		cancel()
		return nil, err
	}

	// Ownership of cancel transfers to the successful session. Every terminal
	// path calls cancelContext after CloseAndRecv or when its timeout expires.
	return &CwcSession{
		ContentWriter_WriteClient: cwc,
		cwcCtx:                    cwcCtx,
		ctxCancel:                 cancel,
		protocolHeaders:           make(map[int32][]byte, 2),
	}, nil
}

func (rc *RecordContext) CancelContentWriter(msg string) error {
	_, err := rc.terminateContentWriter(false, msg)
	return err
}

func (rc *RecordContext) SendProtocolHeader(recNum int32, p []byte) error {
	l := LogWithContext(rc.ctx, "PROXY:CWC")

	cwc, err := rc.getCwcSession()
	if err != nil {
		return err
	}

	cwc.m.Lock()
	canceled := cwc.canceled || cwc.done
	cwc.m.Unlock()
	if canceled {
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

	cwc.m.Lock()
	canceled := cwc.canceled || cwc.done
	cwc.m.Unlock()
	if canceled {
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
	return rc.terminateContentWriter(true, "")
}

func (rc *RecordContext) terminateContentWriter(sendMeta bool, cancelMessage string) (*contentwriterV1.WriteReply, error) {
	l := LogWithContext(rc.ctx, "PROXY:CWC")

	var cwc *CwcSession
	var err error
	if sendMeta {
		cwc, err = rc.getCwcSessionForTerminal(true)
		if err != nil {
			return nil, err
		}
	} else {
		for {
			rc.mutex.Lock()
			if !rc.cwcCreating {
				cwc = rc.cwc
				err = rc.cwcErr
				rc.mutex.Unlock()
				break
			}
			ready := rc.cwcReady
			rc.mutex.Unlock()
			<-ready
		}
		if cwc == nil {
			return nil, err
		}
	}

	cwc.m.Lock()
	if cwc.done {
		cwc.m.Unlock()
		return nil, nil
	}
	cwc.done = true
	cwc.canceled = !sendMeta
	cwc.m.Unlock()

	timedOut := atomic.Bool{}
	timer := time.AfterFunc(rc.finalizationTimeout, func() {
		timedOut.Store(true)
		cwc.cancelContext()
	})
	defer func() {
		if !timer.Stop() || timedOut.Load() {
			contentWriterTerminalTimeouts.Inc()
		}
		cwc.cancelContext()
	}()

	var terminalRequest *contentwriterV1.WriteRequest
	if sendMeta {
		terminalRequest = &contentwriterV1.WriteRequest{Value: rc.Meta}
	} else {
		terminalRequest = &contentwriterV1.WriteRequest{
			Value: &contentwriterV1.WriteRequest_Cancel{Cancel: cancelMessage},
		}
	}

	if sendErr := rc.sendWriteRequest(cwc, map[bool]string{true: "meta", false: "cancel"}[sendMeta], -1, nil, terminalRequest); sendErr != nil {
		err = sendErr
		l.WithError(sendErr).Info("Error sending terminal ContentWriter request")
	}

	cwc.sendMu.Lock()
	reply, closeErr := cwc.CloseAndRecv()
	cwc.sendMu.Unlock()
	if closeErr != nil {
		l.WithError(closeErr).Info("Error closing ContentWriter stream")
		if err == nil {
			err = closeErr
		}
	}
	if timedOut.Load() {
		return reply, fmt.Errorf("ContentWriter terminal operation timed out after %s: %w", rc.finalizationTimeout, context.DeadlineExceeded)
	}
	return reply, err
}
