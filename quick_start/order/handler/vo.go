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

package handler

type Order struct {
	UserID        string `json:"user_id,omitempty"`
	CommodityCode string `json:"commodity_code,omitempty"`
	Count         int64  `json:"count,omitempty"`
	Money         int64  `json:"money,omitempty"`
}

type Message struct {
	Data any    `json:"data,omitempty"`
	Code int64  `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}
