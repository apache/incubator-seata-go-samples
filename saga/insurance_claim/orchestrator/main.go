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
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"seata.apache.org/seata-go-samples/saga/insurance_claim/internal/app"
	"seata.apache.org/seata-go/pkg/client"
	engcfg "seata.apache.org/seata-go/pkg/saga/statemachine/engine/config"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/core"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/invoker"
)

func main() {
	var (
		seataConf    string
		engineConf   string
		businessKey  string
		claimID      string
		claimantID   string
		assessmentID string
		surveyorID   string
		bankAccount  string
		payoutAmount int
		failTransfer bool
	)

	flag.StringVar(&seataConf, "seataConf", "seatago.yaml", "path to the seata-go client config")
	flag.StringVar(&engineConf, "engineConf", "config.yaml", "path to the Saga engine config")
	flag.StringVar(&businessKey, "businessKey", "insurance-claim-saga-demo", "business key")
	flag.StringVar(&claimID, "claimId", "claim-1001", "insurance claim ID")
	flag.StringVar(&claimantID, "claimantId", "claimant-9001", "claimant ID")
	flag.StringVar(&assessmentID, "assessmentId", "assessment-7001", "damage assessment ID")
	flag.StringVar(&surveyorID, "surveyorId", "surveyor-3001", "surveyor ID")
	flag.StringVar(&bankAccount, "bankAccount", "6222020202020202", "bank account")
	flag.IntVar(&payoutAmount, "payoutAmount", 1500, "payout amount")
	flag.BoolVar(&failTransfer, "failTransfer", false, "simulate a bank transfer failure")
	flag.Parse()

	seataConf, err := resolveSamplePath(seataConf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to locate the seata-go client config: %v\n", err)
		os.Exit(1)
	}
	engineConf, err = resolveSamplePath(engineConf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to locate the Saga engine config: %v\n", err)
		os.Exit(1)
	}

	client.InitPath(seataConf)

	engine, err := newStateMachineEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create the state machine engine: %v\n", err)
		os.Exit(1)
	}

	cfgIface := engine.GetStateMachineConfig()
	cfg, ok := cfgIface.(*engcfg.DefaultStateMachineConfig)
	if !ok {
		fmt.Fprintf(os.Stderr, "unexpected state machine config type: %T\n", cfgIface)
		os.Exit(1)
	}

	settings := app.LoadSettings()
	runtimeEngineConf, cleanup, err := prepareRuntimeEngineConfig(engineConf, settings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare the runtime engine config: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := cfg.LoadConfig(runtimeEngineConf); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load the engine config: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize the engine config: %v\n", err)
		os.Exit(1)
	}

	registerHTTPClients(cfgIface, settings)

	db, err := app.OpenDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := app.EnsureBusinessSchema(db); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize the business schema: %v\n", err)
		os.Exit(1)
	}
	if err := app.EnsureSagaStoreSchema(db); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ensure the Saga store schema: %v\n", err)
		os.Exit(1)
	}
	if err := app.ResetClaimData(db, businessKey, claimID); err != nil {
		fmt.Fprintf(os.Stderr, "failed to reset the sample data: %v\n", err)
		os.Exit(1)
	}

	params := map[string]any{
		"businessKey":  businessKey,
		"claimId":      claimID,
		"claimantId":   claimantID,
		"assessmentId": assessmentID,
		"surveyorId":   surveyorID,
		"bankAccount":  bankAccount,
		"payoutAmount": payoutAmount,
		"failTransfer": failTransfer,
	}

	instance, err := engine.StartWithBusinessKey(context.Background(), "InsuranceClaimSaga", "", businessKey, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start the Saga: %v\n", err)
		os.Exit(1)
	}

	snapshot, err := app.LoadSnapshot(db, claimID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load the insurance claim snapshot: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("mode=saga businessKey=%s xid=%s status=%s compensationStatus=%s\n",
		businessKey, instance.ID(), instance.Status(), instance.CompensationStatus())
	fmt.Println(app.FormatSnapshot(snapshot))
}

func registerHTTPClients(cfgIface any, settings app.Settings) {
	cfg := cfgIface.(*engcfg.DefaultStateMachineConfig)
	httpInvoker, ok := cfg.ServiceInvokerManager().ServiceInvoker("http").(*invoker.HTTPInvoker)
	if !ok {
		panic("http invoker is not initialized")
	}

	clientConfig := &http.Client{}
	httpInvoker.RegisterClient("identityService", invoker.NewHTTPClient("identityService", settings.IdentityBaseURL(), clientConfig))
	httpInvoker.RegisterClient("assessmentService", invoker.NewHTTPClient("assessmentService", settings.AssessmentBaseURL(), clientConfig))
	httpInvoker.RegisterClient("fundsService", invoker.NewHTTPClient("fundsService", settings.FundsBaseURL(), clientConfig))
	httpInvoker.RegisterClient("surveyorService", invoker.NewHTTPClient("surveyorService", settings.SurveyorBaseURL(), clientConfig))
	httpInvoker.RegisterClient("transferService", invoker.NewHTTPClient("transferService", settings.TransferBaseURL(), clientConfig))
}

func newStateMachineEngine() (*core.ProcessCtrlStateMachineEngine, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(os.TempDir()); err != nil {
		return nil, err
	}
	defer func() {
		_ = os.Chdir(wd)
	}()
	return core.NewProcessCtrlStateMachineEngine()
}

func resolveSamplePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, fileMustExist(path)
	}

	candidates := []string{
		path,
		filepath.Join("saga", "insurance_claim", path),
	}
	for _, candidate := range candidates {
		if err := fileMustExist(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("the file does not exist: %s", path)
}

func fileMustExist(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("the path refers to a directory: %s", path)
	}
	return nil
}

func prepareRuntimeEngineConfig(engineConf string, settings app.Settings) (string, func(), error) {
	raw, err := os.ReadFile(engineConf)
	if err != nil {
		return "", nil, err
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", nil, err
	}
	cfg["store_dsn"] = settings.MySQLDSN()
	cfg["state_machine_resources"] = []string{filepath.Join(filepath.Dir(engineConf), "statelang", "*.json")}

	file, err := os.CreateTemp("", "insurance-claim-saga-*.yaml")
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(cfg); err != nil {
		_ = os.Remove(file.Name())
		_ = encoder.Close()
		return "", nil, err
	}
	if err := encoder.Close(); err != nil {
		_ = os.Remove(file.Name())
		return "", nil, err
	}

	cleanup := func() {
		_ = os.Remove(file.Name())
	}
	return file.Name(), cleanup, nil
}
