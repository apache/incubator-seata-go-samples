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

package httpjson

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func ReadArgs(r *http.Request, expected int) ([]json.RawMessage, error) {
	defer r.Body.Close()

	var args []json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		return nil, err
	}
	if len(args) != expected {
		return nil, fmt.Errorf("expect %d args, got %d", expected, len(args))
	}
	return args, nil
}

func StringArg(args []json.RawMessage, index int) (string, error) {
	var value string
	if err := json.Unmarshal(args[index], &value); err != nil {
		return "", err
	}
	return value, nil
}

func IntArg(args []json.RawMessage, index int) (int, error) {
	var value int
	if err := json.Unmarshal(args[index], &value); err != nil {
		return 0, err
	}
	return value, nil
}

func BoolArg(args []json.RawMessage, index int) (bool, error) {
	var value bool
	if err := json.Unmarshal(args[index], &value); err != nil {
		return false, err
	}
	return value, nil
}

func WriteText(w http.ResponseWriter, statusCode int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(body))
}
