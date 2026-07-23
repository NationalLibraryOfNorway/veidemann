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

package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
)

// ProxyError is the struct of recorder proxy error
type ProxyError struct {
	code       ErrorCode
	message    string
	detail     string
	cause      error
	statusCode int
}

func (e *ProxyError) Error() string {
	errMsg := fmt.Sprintf("Code: %s, Msg: %s", e.code, e.message)
	if e.detail != "" {
		errMsg = errMsg + ", Detail: " + e.detail
	}
	if e.cause != nil {
		return errMsg + ", Cause: " + e.cause.Error()
	}
	return errMsg
}

func (e *ProxyError) Unwrap() error {
	return e.cause
}

func (e *ProxyError) Cause() error {
	return e.cause
}

func (e *ProxyError) Code() ErrorCode {
	return e.code
}

func (e *ProxyError) Message() string {
	return e.message
}

func (e *ProxyError) Detail() string {
	return e.detail
}

func (e *ProxyError) HttpStatusCode() int {
	return e.statusCode
}

// Cause returns the cause error of this error
func Cause(err error) error {
	if err == nil {
		return nil
	}

	for {
		type causer interface {
			Cause() error
		}

		if c, ok := err.(causer); ok {
			next := c.Cause()
			if next == nil {
				return err
			}
			err = next
			continue
		}

		next := stderrors.Unwrap(err)
		if next == nil {
			return err
		}

		err = next
	}
}

// Code returns the error code
func Code(err error) ErrorCode {
	if err == nil {
		return RuntimeException
	}

	type coder interface {
		Code() ErrorCode
	}

	var c coder
	if !stderrors.As(err, &c) {
		return RuntimeException
	}

	return c.Code()
}

// Message returns the error message
func Message(err error) string {
	if err == nil {
		return ""
	}

	type messenger interface {
		Message() string
	}

	var m messenger
	if !stderrors.As(err, &m) {
		return err.Error()
	}

	return m.Message()
}

// Detail returns the error detail message
func Detail(err error) string {
	type det interface {
		Detail() string
	}

	d, ok := err.(det)
	if !ok {
		return err.Error()
	}
	return d.Detail()
}

// HttpStatusCode returns the http status code which will be sent to client
func HttpStatusCode(err error) int {
	type st interface {
		HttpStatusCode() int
	}

	s, ok := err.(st)
	if !ok {
		return http.StatusServiceUnavailable
	}
	return s.HttpStatusCode()
}

// Error constructs a new error
func Error(code ErrorCode, message, detail string) error {
	return &ProxyError{
		code:       code,
		message:    message,
		detail:     detail,
		statusCode: http.StatusServiceUnavailable,
	}
}

// Wrap waps an error with an error and a message
func Wrap(err error, code ErrorCode, message, detail string) error {
	if err == nil {
		return nil
	}
	return &ProxyError{
		code:       code,
		message:    message,
		cause:      err,
		detail:     detail,
		statusCode: http.StatusServiceUnavailable,
	}
}

// Error constructs a new error
func ErrorInternal(code ErrorCode, message, detail string) error {
	return &ProxyError{
		code:       code,
		message:    message,
		detail:     detail,
		statusCode: http.StatusBadGateway,
	}
}

// Wrap waps an error with an error and a message
func WrapInternalError(err error, code ErrorCode, message, detail string) error {
	if err == nil {
		return nil
	}
	return &ProxyError{
		code:       code,
		message:    message,
		cause:      err,
		detail:     detail,
		statusCode: http.StatusBadGateway,
	}
}

func AsCommonsError(err error) *commonsV1.Error {
	return &commonsV1.Error{
		Code:   Code(err).Int32(),
		Msg:    Message(err),
		Detail: Detail(err),
	}
}

type BrowserControllerCancelError struct {
	Reason string
}

func (e *BrowserControllerCancelError) Error() string {
	if e.Reason == "" {
		return "cancelled by browser controller"
	}
	return "cancelled by browser controller: " + e.Reason
}

func IsBrowserControllerCancel(err error) bool {
	var e *BrowserControllerCancelError
	return stderrors.As(err, &e)
}
