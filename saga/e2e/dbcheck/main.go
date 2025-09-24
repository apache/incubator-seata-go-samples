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
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

type engineConf struct {
	StoreEnabled bool   `yaml:"store_enabled"`
	StoreType    string `yaml:"store_type"`
	StoreDSN     string `yaml:"store_dsn"`
}

func loadEngineConf(path string) (*engineConf, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c engineConf
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if !c.StoreEnabled || c.StoreType == "" || c.StoreDSN == "" {
		return nil, errors.New("engineConf missing store_enabled/store_type/store_dsn")
	}
	return &c, nil
}

func expectSuccess(row smRow) error {
	var errs []string
	if row.Status != "SU" {
		errs = append(errs, fmt.Sprintf("status=%s want=SU", row.Status))
	}
	if row.CompStatus.Valid && row.CompStatus.String != "" {
		errs = append(errs, fmt.Sprintf("compensation_status=%s want=''", row.CompStatus.String))
	}
	if row.IsRunning != 0 {
		errs = append(errs, fmt.Sprintf("is_running=%d want=0", row.IsRunning))
	}
	if !row.GmtEnd.Valid {
		errs = append(errs, "gmt_end is NULL")
	}
	if row.Excep.Valid && len(row.Excep.String) > 0 {
		errs = append(errs, "excep not empty")
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func expectCompensateFail(row smRow) error {
	var errs []string
	if row.Status != "FA" {
		errs = append(errs, fmt.Sprintf("status=%s want=FA", row.Status))
	}
	if !row.CompStatus.Valid || row.CompStatus.String != "SU" {
		want := "<NULL>"
		if row.CompStatus.Valid {
			want = row.CompStatus.String
		}
		errs = append(errs, fmt.Sprintf("compensation_status=%s want=SU", want))
	}
	if row.IsRunning != 0 {
		errs = append(errs, fmt.Sprintf("is_running=%d want=0", row.IsRunning))
	}
	if !row.GmtEnd.Valid {
		errs = append(errs, "gmt_end is NULL")
	}
	// excep may be empty for Fail end state; don't enforce
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

type smRow struct {
	ID         string
	Status     string
	CompStatus sql.NullString
	IsRunning  int
	GmtEnd     sql.NullTime
	Excep      sql.NullString
}

type stRow struct {
	Name    string
	Type    string
	Status  string
	CompFor sql.NullString
}

func fetchStateRows(db *sql.DB, xid string) ([]stRow, error) {
	q := `SELECT name, type, status, state_id_compensated_for 
          FROM seata_state_inst WHERE machine_inst_id=? ORDER BY gmt_started ASC`
	rows, err := db.Query(q, xid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []stRow
	for rows.Next() {
		var r stRow
		if err := rows.Scan(&r.Name, &r.Type, &r.Status, &r.CompFor); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func contains(rows []stRow, pred func(stRow) bool) bool {
	for _, r := range rows {
		if pred(r) {
			return true
		}
	}
	return false
}

func validateStatesSuccess(rows []stRow) error {
	var errs []string
	if !contains(rows, func(r stRow) bool {
		return r.Name == "ReduceInventory" && r.Status == "SU" && !(r.CompFor.Valid && r.CompFor.String != "")
	}) {
		errs = append(errs, "missing SU ReduceInventory forward state")
	}
	if !contains(rows, func(r stRow) bool {
		return r.Name == "ReduceBalance" && r.Status == "SU" && !(r.CompFor.Valid && r.CompFor.String != "")
	}) {
		errs = append(errs, "missing SU ReduceBalance forward state")
	}
	// no compensation states expected
	if contains(rows, func(r stRow) bool { return r.CompFor.Valid && r.CompFor.String != "" }) {
		errs = append(errs, "unexpected compensation state present")
	}
	// Succeed end state is not persisted as state_inst in Go impl; rely on machine_inst status instead
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func validateStatesCompBalance(rows []stRow) error {
	var errs []string
	if !contains(rows, func(r stRow) bool {
		return r.Name == "ReduceInventory" && r.Status == "SU" && !(r.CompFor.Valid && r.CompFor.String != "")
	}) {
		errs = append(errs, "missing SU ReduceInventory forward state")
	}
	if !contains(rows, func(r stRow) bool {
		return r.Name == "ReduceBalance" && r.Status == "FA" && !(r.CompFor.Valid && r.CompFor.String != "")
	}) {
		errs = append(errs, "missing FA ReduceBalance forward state")
	}
	if !contains(rows, func(r stRow) bool {
		return r.Name == "CompensateReduceInventory" && r.Status == "SU" && r.CompFor.Valid && r.CompFor.String != ""
	}) {
		// Allow type-based match in case of name differences
		if !contains(rows, func(r stRow) bool {
			return strings.HasPrefix(r.Name, "Compensate") && r.Status == "SU" && r.CompFor.Valid && r.CompFor.String != ""
		}) {
			errs = append(errs, "missing SU compensation state for ReduceInventory")
		}
	}
	// Fail end state is not persisted as state_inst in Go impl; rely on machine_inst status instead
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func validateStatesCompInventory(rows []stRow) error {
	var errs []string
	if !contains(rows, func(r stRow) bool {
		return r.Name == "ReduceInventory" && r.Status == "FA" && !(r.CompFor.Valid && r.CompFor.String != "")
	}) {
		errs = append(errs, "missing FA ReduceInventory forward state")
	}
	// no compensation states expected
	if contains(rows, func(r stRow) bool { return r.CompFor.Valid && r.CompFor.String != "" }) {
		// If any compensation exists, include a snapshot for debugging
		var names []string
		for _, r := range rows {
			if r.CompFor.Valid && r.CompFor.String != "" {
				names = append(names, r.Name)
			}
		}
		sort.Strings(names)
		errs = append(errs, fmt.Sprintf("unexpected compensation states present: %s", strings.Join(names, ",")))
	}
	// Fail end state row not required; machine status covers it
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func main() {
	var engineConfPath, xid, scenario string
	flag.StringVar(&engineConfPath, "engine", "", "engine config path")
	flag.StringVar(&xid, "xid", "", "XID to validate")
	flag.StringVar(&scenario, "scenario", "success", "success|compensate-balance|compensate-inventory")
	flag.Parse()
	if engineConfPath == "" || xid == "" {
		fmt.Fprintln(os.Stderr, "missing --engine or --xid")
		os.Exit(2)
	}

	cfg, err := loadEngineConf(engineConfPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if cfg.StoreType != "mysql" {
		fmt.Fprintln(os.Stderr, "only mysql is supported in dbcheck")
		os.Exit(2)
	}
	db, err := sql.Open("mysql", cfg.StoreDSN)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// query state_machine_inst row
	q := `SELECT id, status, compensation_status, is_running, gmt_end, excep FROM seata_state_machine_inst WHERE id=?`
	var row smRow
	if err := db.QueryRow(q, xid).Scan(&row.ID, &row.Status, &row.CompStatus, &row.IsRunning, &row.GmtEnd, &row.Excep); err != nil {
		fmt.Fprintf(os.Stderr, "query sm row failed: %v\n", err)
		os.Exit(2)
	}
	states, err := fetchStateRows(db, xid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query state rows failed: %v\n", err)
		os.Exit(2)
	}
	// Debug visibility: show how many rows we saw for this XID
	fmt.Printf("Found %d state rows for XID=%s\n", len(states), xid)
	if len(states) > 0 {
		var names []string
		for _, r := range states {
			names = append(names, r.Name+"/"+r.Status)
		}
		fmt.Printf("States: %s\n", strings.Join(names, ", "))
	}

	switch scenario {
	case "success":
		if err := expectSuccess(row); err != nil {
			fmt.Fprintf(os.Stderr, "validation failed (machine): %v\n", err)
			os.Exit(1)
		}
		if err := validateStatesSuccess(states); err != nil {
			fmt.Fprintf(os.Stderr, "validation failed (states): %v\n", err)
			os.Exit(1)
		}
		fmt.Println("DB validation (success) OK")
		return
	case "compensate-balance":
		if err := expectCompensateFail(row); err != nil {
			fmt.Fprintf(os.Stderr, "validation failed (machine): %v\n", err)
			os.Exit(1)
		}
		if err := validateStatesCompBalance(states); err != nil {
			fmt.Fprintf(os.Stderr, "validation failed (states): %v\n", err)
			os.Exit(1)
		}
		fmt.Println("DB validation (compensate-balance) OK")
		return
	case "compensate-inventory":
		// machine expectation: forward fail without compensation
		var errs []string
		if row.Status != "FA" {
			errs = append(errs, fmt.Sprintf("status=%s want=FA", row.Status))
		}
		if row.CompStatus.Valid && row.CompStatus.String != "" {
			errs = append(errs, fmt.Sprintf("compensation_status=%s want=''", row.CompStatus.String))
		}
		if row.IsRunning != 0 {
			errs = append(errs, fmt.Sprintf("is_running=%d want=0", row.IsRunning))
		}
		if !row.GmtEnd.Valid {
			errs = append(errs, "gmt_end is NULL")
		}
		if len(errs) > 0 {
			fmt.Fprintf(os.Stderr, "validation failed (machine): %s\n", strings.Join(errs, "; "))
			os.Exit(1)
		}
		if err := validateStatesCompInventory(states); err != nil {
			fmt.Fprintf(os.Stderr, "validation failed (states): %v\n", err)
			os.Exit(1)
		}
		fmt.Println("DB validation (compensate-inventory) OK")
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown scenario: %s\n", scenario)
		os.Exit(2)
	}
}
