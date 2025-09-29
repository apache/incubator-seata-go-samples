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

// Package main demonstrates basic GRPC communication with Seata server
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"seata.apache.org/seata-go/pkg/client"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	configPath = flag.String("config", "../config/seata-grpc.yml", "config file path")
)

func main() {
	flag.Parse()
	
	fmt.Println("=== Seata GRPC Communication Basic Client Sample ===")
	fmt.Println("This sample demonstrates basic GRPC communication between client and Seata server")
	
	// Initialize seata client with GRPC protocol
	// The configuration will specify GRPC as the communication protocol
	client.InitPath(*configPath)
	log.Info("Seata client initialized with GRPC protocol")
	
	// TODO: Once new GRPC features are merged, this will demonstrate:
	// 1. Direct GRPC connection to Seata server
	// 2. Enhanced GRPC communication features
	// 3. Improved performance and reliability
	
	// For now, demonstrate basic global transaction that works with current version
	err := tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "BasicGrpcCommunicationSample",
		Timeout: 30 * time.Second,
	}, func(ctx context.Context) error {
		// This transaction will use the configured GRPC protocol to communicate with Seata server
		fmt.Printf("Global transaction started with XID: %s\n", tm.GetXID(ctx))
		fmt.Println("Communication protocol: GRPC")
		
		// TODO: Add more sophisticated GRPC communication examples once new features are available:
		// - Streaming communication
		// - Load balancing
		// - Connection pooling
		// - Monitoring and metrics
		
		// Simulate business logic
		fmt.Println("Executing business logic...")
		time.Sleep(1 * time.Second)
		fmt.Println("Business logic completed successfully")
		
		return nil
	})
	
	if err != nil {
		log.Fatalf("Global transaction failed: %v", err)
	}
	
	fmt.Println("✓ Basic GRPC communication sample completed successfully")
	fmt.Println("✓ This demonstrates the foundation for advanced GRPC features")
}