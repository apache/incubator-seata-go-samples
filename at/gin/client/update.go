// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/parnurzeal/gorequest"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
)

func updateData(ctx context.Context) (re error) {
	request := gorequest.New()
	log.Infof("branch transaction begin")
	request.Post(serverIpPort+"/updateDataSuccess").
		Set(constant.XidKey, tm.GetXID(ctx)).
		End(func(response gorequest.Response, body string, errs []error) {
			if response.StatusCode != http.StatusOK {
				re = fmt.Errorf("update data fail")
			}
		})
	return
}

func sampleUpdate(ctx context.Context) {
	if err := tm.WithGlobalTx(ctx, &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx_Update",
		Timeout: time.Second * 30,
	}, updateData); err != nil {
		panic(fmt.Sprintf("tm update data err, %v", err))
	}
}
