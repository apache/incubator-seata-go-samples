/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"errors"
	"net/http"
)

type APIResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}

type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

func NewConflictError(message string) error {
	return &ConflictError{Message: message}
}

type DownstreamError struct {
	StatusCode int
	Message    string
}

func (e *DownstreamError) Error() string {
	return e.Message
}

func NewDownstreamError(statusCode int, message string) error {
	return &DownstreamError{StatusCode: statusCode, Message: message}
}

func StatusCodeForError(err error) int {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return http.StatusBadRequest
	}

	var conflictErr *ConflictError
	if errors.As(err, &conflictErr) {
		return http.StatusConflict
	}

	var downstreamErr *DownstreamError
	if errors.As(err, &downstreamErr) {
		if downstreamErr.StatusCode >= 400 && downstreamErr.StatusCode < 500 {
			return downstreamErr.StatusCode
		}
		return http.StatusBadGateway
	}

	return http.StatusInternalServerError
}

func PublicErrorMessage(err error) string {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Error()
	}

	var conflictErr *ConflictError
	if errors.As(err, &conflictErr) {
		return conflictErr.Error()
	}

	var downstreamErr *DownstreamError
	if errors.As(err, &downstreamErr) {
		if downstreamErr.StatusCode >= 400 && downstreamErr.StatusCode < 500 {
			return "dependent service rejected the request"
		}
		return "dependent service is unavailable"
	}

	return "internal server error"
}
