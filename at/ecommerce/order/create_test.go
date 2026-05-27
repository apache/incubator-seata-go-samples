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

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/tm"
)

func TestDeductInventoryCarriesXID(t *testing.T) {
	receivedXID := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedXID = r.Header.Get(constant.XidKey)
		if r.URL.Path != "/deductInventory" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req InventoryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode inventory request: %v", err)
		}
		if req.CommodityCode != "C100001" || req.Count != 2 {
			t.Fatalf("unexpected inventory request: %+v", req)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	inventoryService = server.URL
	ctx := context.Background()
	tm.SetXID(ctx, "xid-inventory-001")

	err := deductInventory(ctx, OrderRequest{
		UserID:        "U100001",
		CommodityCode: "C100001",
		Count:         2,
		Money:         30,
	})
	if err != nil {
		t.Fatalf("deductInventory returned error: %v", err)
	}
	if receivedXID != "xid-inventory-001" {
		t.Fatalf("unexpected xid header: %s", receivedXID)
	}
}

func TestDeductAccountReturnsDownstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "balance not enough", http.StatusBadRequest)
	}))
	defer server.Close()

	accountService = server.URL
	ctx := context.Background()
	tm.SetXID(ctx, "xid-account-001")

	err := deductAccount(ctx, OrderRequest{
		UserID:        "U100001",
		CommodityCode: "C100001",
		Count:         2,
		Money:         100,
	})
	if err == nil {
		t.Fatal("expected deductAccount to fail")
	}
}
