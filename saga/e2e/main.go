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
	"database/sql"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"

	"seata.apache.org/seata-go/pkg/client"
	engcfg "seata.apache.org/seata-go/pkg/saga/statemachine/engine/config"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/core"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/invoker"
)

// InventoryAction (DB-backed) implements reduce/compensate with explicit params
type InventoryAction struct{ db *sql.DB }

func NewInventoryAction(db *sql.DB) *InventoryAction { return &InventoryAction{db: db} }

// Reduce(businessKey, productId, count) -> (bool, error)
func (a *InventoryAction) Reduce(businessKey string, productId string, count int) (bool, error) {
	if count <= 0 {
		count = 1
	}
	res, err := a.db.Exec("UPDATE e2e_inventory SET stock = stock - ? WHERE product_id = ? AND stock >= ?", count, productId, count)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, fmt.Errorf("INVENTORY_NOT_ENOUGH")
	}
	fmt.Printf("InventoryAction.Reduce: biz=%s, product=%s, count=%d\n", businessKey, productId, count)
	return true, nil
}

func (a *InventoryAction) CompensateReduce(businessKey string, productId string, count int) (bool, error) {
	if count <= 0 {
		count = 1
	}
	if _, err := a.db.Exec("UPDATE e2e_inventory SET stock = stock + ? WHERE product_id = ?", count, productId); err != nil {
		return false, err
	}
	fmt.Printf("InventoryAction.CompensateReduce: biz=%s, product=%s, count=%d\n", businessKey, productId, count)
	return true, nil
}

// BalanceAction (DB-backed) implements reduce/compensate with explicit params
type BalanceAction struct{ db *sql.DB }

func NewBalanceAction(db *sql.DB) *BalanceAction { return &BalanceAction{db: db} }

// Reduce(businessKey, userId, amount) -> (bool, error)
func (b *BalanceAction) Reduce(businessKey string, userId string, amount int) (bool, error) {
	if amount <= 0 {
		amount = 1
	}
	res, err := b.db.Exec("UPDATE e2e_balance SET amount = amount - ? WHERE user_id = ? AND amount >= ?", amount, userId, amount)
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, fmt.Errorf("BALANCE_NOT_ENOUGH")
	}
	fmt.Printf("BalanceAction.Reduce: biz=%s, user=%s, amount=%d\n", businessKey, userId, amount)
	return true, nil
}

func (b *BalanceAction) CompensateReduce(businessKey string, userId string, amount int) (bool, error) {
	if amount <= 0 {
		amount = 1
	}
	if _, err := b.db.Exec("UPDATE e2e_balance SET amount = amount + ? WHERE user_id = ?", amount, userId); err != nil {
		return false, err
	}
	fmt.Printf("BalanceAction.CompensateReduce: biz=%s, user=%s, amount=%d\n", businessKey, userId, amount)
	return true, nil
}

func main() {
	var seataConf string
	var engineConf string
	flag.StringVar(&seataConf, "seataConf", "saga/e2e/seatago.yaml", "path to seata-go client yaml")
	flag.StringVar(&engineConf, "engineConf", "saga/e2e/config.yaml", "path to saga engine config")
	flag.Parse()

	client.InitPath(seataConf)
	if err := checkSeataConnectivity(seataConf); err != nil {
		fmt.Fprintf(os.Stderr, "Seata server connectivity check failed: %v\n", err)
		os.Exit(1)
	}

	eng, err := core.NewProcessCtrlStateMachineEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "create state machine engine failed: %v\n", err)
		os.Exit(1)
	}
	cfgIface := eng.GetStateMachineConfig()
	if cfg, ok := cfgIface.(*engcfg.DefaultStateMachineConfig); ok {
		if err := cfg.LoadConfig(engineConf); err != nil {
			fmt.Fprintf(os.Stderr, "load engine config failed: %v\n", err)
			os.Exit(1)
		}
		if wd, err := os.Getwd(); err == nil {
			absPattern := filepath.Join(wd, "saga/e2e/statelang/*.json")
			_ = cfg.RegisterStateMachineDef([]string{absPattern})
		}
		if err := cfg.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "init engine config failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "unexpected state machine config type: %T\n", cfgIface)
		os.Exit(1)
	}

	// Open business DB and seed data
	bizDB, err := openBusinessDB(engineConf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open business db failed: %v\n", err)
		os.Exit(1)
	}
	if err := seedBusinessData(bizDB); err != nil {
		fmt.Fprintf(os.Stderr, "seed business data failed: %v\n", err)
		os.Exit(1)
	}

	// Register local services (DB-backed)
	if lv := cfgIface.ServiceInvokerManager().ServiceInvoker("local"); lv != nil {
		if lsi, ok := lv.(*invoker.LocalServiceInvoker); ok {
			lsi.RegisterService("inventoryAction", NewInventoryAction(bizDB))
			lsi.RegisterService("balanceAction", NewBalanceAction(bizDB))
		}
	}

	// Run three scenarios sequentially, driven by parameters (no orders table)
	scenarios := []struct {
		name   string
		params map[string]any
	}{
		{"success", successParams()},
		{"compensate-balance", compensateBalanceParams()},
		{"compensate-inventory", compensateInventoryParams()},
	}
	for _, sc := range scenarios {
		inst, err := eng.Start(context.Background(), "ReduceInventoryAndBalance", "", sc.params)
		if err != nil {
			fmt.Fprintf(os.Stderr, "start saga failed (%s): %v\n", sc.name, err)
			os.Exit(1)
		}
		fmt.Println("======================================================")
		fmt.Printf("SCENARIO %s XID=%s status=%s compStatus=%s\n", sc.name, inst.ID(), inst.Status(), inst.CompensationStatus())
		fmt.Println("======================================================")

		fmt.Println("")
		fmt.Println("")
		fmt.Println("")
		time.Sleep(time.Second * 2)
	}
}

type runtimeStoreConf struct {
	StoreEnabled bool   `yaml:"store_enabled"`
	StoreType    string `yaml:"store_type"`
	StoreDSN     string `yaml:"store_dsn"`
	TCEnabled    bool   `yaml:"tc_enabled"`
}

// checkSeataConnectivity parses grouplist from seatago.yaml and dials the first address
func checkSeataConnectivity(seataConf string) error {
	raw, err := os.ReadFile(seataConf)
	if err != nil {
		return err
	}
	type seataYaml struct {
		Seata struct {
			Service struct {
				GroupList map[string]string `yaml:"grouplist"`
			} `yaml:"service"`
		} `yaml:"seata"`
	}
	var cfg seataYaml
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	for _, addr := range cfg.Seata.Service.GroupList {
		target := addr
		// allow multiple via ","
		if strings.Contains(addr, ",") {
			parts := strings.Split(addr, ",")
			target = strings.TrimSpace(parts[0])
		}
		if target == "" {
			continue
		}
		conn, err := net.DialTimeout("tcp", target, 2_000_000_000)
		if err != nil {
			return fmt.Errorf("dial %s failed: %w", target, err)
		}
		_ = conn.Close()
		return nil
	}
	return fmt.Errorf("no seata service.grouplist address found in %s", seataConf)
}

// openBusinessDB opens a DB connection using engine store_dsn from YAML.
func openBusinessDB(engineConf string) (*sql.DB, error) {
	raw, err := os.ReadFile(engineConf)
	if err != nil {
		return nil, err
	}
	var r runtimeStoreConf
	if err := yaml.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	if !r.StoreEnabled || r.StoreType == "" || r.StoreDSN == "" {
		return nil, fmt.Errorf("engine store not configured")
	}
	driver := r.StoreType
	if driver == "sqlite" || driver == "sqlite3" {
		driver = "sqlite3"
	}
	db, err := sql.Open(driver, r.StoreDSN)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// seedBusinessData creates business tables and deterministic rows to drive three scenarios
func seedBusinessData(db *sql.DB) error {
	// tables
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS e2e_inventory (
  product_id VARCHAR(64) PRIMARY KEY,
  stock INT NOT NULL
)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS e2e_balance (
  user_id VARCHAR(64) PRIMARY KEY,
  amount INT NOT NULL
)`)
	if err != nil {
		return err
	}
	// upserts (MySQL syntax)
	// inventories
	if _, err := db.Exec(`INSERT INTO e2e_inventory(product_id, stock) VALUES
        ('p_s', 100), ('p_b', 100), ('p_i', 100)
      ON DUPLICATE KEY UPDATE stock=VALUES(stock)`); err != nil {
		return err
	}
	// balances
	if _, err := db.Exec(`INSERT INTO e2e_balance(user_id, amount) VALUES
        ('u_s', 1000), ('u_b', 50), ('u_i', 1000)
      ON DUPLICATE KEY UPDATE amount=VALUES(amount)`); err != nil {
		return err
	}
	return nil
}

// no time helpers needed
