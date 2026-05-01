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

package app

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Snapshot struct {
	IdentityVerified   bool
	AssessmentStatus   string
	FundsStatus        string
	FundsAmount        int
	SurveyorStatus     string
	TransferStatus     string
	TransferLastError  string
	OrderedActionTrail []string
}

func OpenDB() (*sql.DB, error) {
	settings := LoadSettings()
	db, err := sql.Open("mysql", settings.MySQLDSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func EnsureBusinessSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS claim_identity (
			claim_id VARCHAR(64) PRIMARY KEY,
			business_key VARCHAR(64) NOT NULL,
			claimant_id VARCHAR(64) NOT NULL,
			verified TINYINT(1) NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS claim_assessment (
			claim_id VARCHAR(64) PRIMARY KEY,
			business_key VARCHAR(64) NOT NULL,
			assessment_id VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS claim_fund_reservation (
			claim_id VARCHAR(64) PRIMARY KEY,
			business_key VARCHAR(64) NOT NULL,
			amount INT NOT NULL,
			status VARCHAR(32) NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS claim_surveyor_notice (
			claim_id VARCHAR(64) PRIMARY KEY,
			business_key VARCHAR(64) NOT NULL,
			surveyor_id VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS claim_transfer (
			claim_id VARCHAR(64) PRIMARY KEY,
			business_key VARCHAR(64) NOT NULL,
			bank_account VARCHAR(64) NOT NULL,
			amount INT NOT NULL,
			status VARCHAR(32) NOT NULL,
			last_error VARCHAR(255) NOT NULL DEFAULT '',
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS claim_step_log (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			business_key VARCHAR(64) NOT NULL,
			claim_id VARCHAR(64) NOT NULL,
			step_name VARCHAR(64) NOT NULL,
			action_name VARCHAR(64) NOT NULL,
			note VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func ResetClaimData(db *sql.DB, businessKey string, claimID string) error {
	statements := []string{
		"DELETE FROM claim_step_log WHERE business_key = ? OR claim_id = ?",
		"DELETE FROM claim_transfer WHERE claim_id = ?",
		"DELETE FROM claim_surveyor_notice WHERE claim_id = ?",
		"DELETE FROM claim_fund_reservation WHERE claim_id = ?",
		"DELETE FROM claim_assessment WHERE claim_id = ?",
		"DELETE FROM claim_identity WHERE claim_id = ?",
	}
	for index, statement := range statements {
		var err error
		if index == 0 {
			_, err = db.Exec(statement, businessKey, claimID)
		} else {
			_, err = db.Exec(statement, claimID)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func RecordIdentityVerified(db *sql.DB, businessKey string, claimID string, claimantID string) error {
	query := `INSERT INTO claim_identity(claim_id, business_key, claimant_id, verified)
		VALUES(?, ?, ?, 1)
		ON DUPLICATE KEY UPDATE business_key = VALUES(business_key), claimant_id = VALUES(claimant_id), verified = 1`
	if _, err := db.Exec(query, claimID, businessKey, claimantID); err != nil {
		return err
	}
	return appendStepLog(db, businessKey, claimID, "identity", "verify", "claimant verified")
}

func UnverifyIdentity(db *sql.DB, businessKey string, claimID string) error {
	if _, err := db.Exec(`UPDATE claim_identity SET verified = 0 WHERE claim_id = ?`, claimID); err != nil {
		return err
	}
	return appendStepLog(db, businessKey, claimID, "identity", "compensate", "claimant verification rolled back")
}

func CreateAssessment(db *sql.DB, businessKey string, claimID string, assessmentID string) error {
	query := `INSERT INTO claim_assessment(claim_id, business_key, assessment_id, status)
		VALUES(?, ?, ?, 'CREATED')
		ON DUPLICATE KEY UPDATE business_key = VALUES(business_key), assessment_id = VALUES(assessment_id), status = 'CREATED'`
	if _, err := db.Exec(query, claimID, businessKey, assessmentID); err != nil {
		return err
	}
	return appendStepLog(db, businessKey, claimID, "assessment", "create", "damage assessment created")
}

func DeleteAssessment(db *sql.DB, businessKey string, claimID string) error {
	if _, err := db.Exec(`DELETE FROM claim_assessment WHERE claim_id = ?`, claimID); err != nil {
		return err
	}
	return appendStepLog(db, businessKey, claimID, "assessment", "compensate", "damage assessment deleted")
}

func ReserveFunds(db *sql.DB, businessKey string, claimID string, amount int) error {
	query := `INSERT INTO claim_fund_reservation(claim_id, business_key, amount, status)
		VALUES(?, ?, ?, 'RESERVED')
		ON DUPLICATE KEY UPDATE business_key = VALUES(business_key), amount = VALUES(amount), status = 'RESERVED'`
	if _, err := db.Exec(query, claimID, businessKey, amount); err != nil {
		return err
	}
	return appendStepLog(db, businessKey, claimID, "funds", "reserve", fmt.Sprintf("reserved payout amount=%d", amount))
}

func ReleaseFunds(db *sql.DB, businessKey string, claimID string) error {
	if _, err := db.Exec(`UPDATE claim_fund_reservation SET status = 'RELEASED' WHERE claim_id = ?`, claimID); err != nil {
		return err
	}
	return appendStepLog(db, businessKey, claimID, "funds", "compensate", "reserved payout released")
}

func NotifySurveyor(db *sql.DB, businessKey string, claimID string, surveyorID string) error {
	query := `INSERT INTO claim_surveyor_notice(claim_id, business_key, surveyor_id, status)
		VALUES(?, ?, ?, 'NOTIFIED')
		ON DUPLICATE KEY UPDATE business_key = VALUES(business_key), surveyor_id = VALUES(surveyor_id), status = 'NOTIFIED'`
	if _, err := db.Exec(query, claimID, businessKey, surveyorID); err != nil {
		return err
	}
	return appendStepLog(db, businessKey, claimID, "surveyor", "notify", "assigned surveyor notified")
}

func CancelSurveyorNotification(db *sql.DB, businessKey string, claimID string) error {
	if _, err := db.Exec(`UPDATE claim_surveyor_notice SET status = 'CANCELLED' WHERE claim_id = ?`, claimID); err != nil {
		return err
	}
	return appendStepLog(db, businessKey, claimID, "surveyor", "compensate", "surveyor notification cancelled")
}

func ExecuteTransfer(db *sql.DB, businessKey string, claimID string, bankAccount string, amount int, failTransfer bool) error {
	status := "SUCCESS"
	lastError := ""
	if failTransfer {
		status = "FAILED"
		lastError = "BANK_TRANSFER_FAILED"
	}
	query := `INSERT INTO claim_transfer(claim_id, business_key, bank_account, amount, status, last_error)
		VALUES(?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE business_key = VALUES(business_key), bank_account = VALUES(bank_account), amount = VALUES(amount), status = VALUES(status), last_error = VALUES(last_error)`
	if _, err := db.Exec(query, claimID, businessKey, bankAccount, amount, status, lastError); err != nil {
		return err
	}
	note := fmt.Sprintf("bank transfer amount=%d status=%s", amount, status)
	if err := appendStepLog(db, businessKey, claimID, "transfer", "execute", note); err != nil {
		return err
	}
	if failTransfer {
		return fmt.Errorf("BANK_TRANSFER_FAILED")
	}
	return nil
}

func LoadSnapshot(db *sql.DB, claimID string) (Snapshot, error) {
	snapshot := Snapshot{
		AssessmentStatus:  "MISSING",
		FundsStatus:       "MISSING",
		SurveyorStatus:    "MISSING",
		TransferStatus:    "MISSING",
		TransferLastError: "",
	}

	if err := db.QueryRow(`SELECT verified FROM claim_identity WHERE claim_id = ?`, claimID).Scan(&snapshot.IdentityVerified); err != nil && err != sql.ErrNoRows {
		return snapshot, err
	}
	if err := db.QueryRow(`SELECT status FROM claim_assessment WHERE claim_id = ?`, claimID).Scan(&snapshot.AssessmentStatus); err != nil && err != sql.ErrNoRows {
		return snapshot, err
	}
	if err := db.QueryRow(`SELECT status, amount FROM claim_fund_reservation WHERE claim_id = ?`, claimID).Scan(&snapshot.FundsStatus, &snapshot.FundsAmount); err != nil && err != sql.ErrNoRows {
		return snapshot, err
	}
	if err := db.QueryRow(`SELECT status FROM claim_surveyor_notice WHERE claim_id = ?`, claimID).Scan(&snapshot.SurveyorStatus); err != nil && err != sql.ErrNoRows {
		return snapshot, err
	}
	if err := db.QueryRow(`SELECT status, last_error FROM claim_transfer WHERE claim_id = ?`, claimID).Scan(&snapshot.TransferStatus, &snapshot.TransferLastError); err != nil && err != sql.ErrNoRows {
		return snapshot, err
	}

	rows, err := db.Query(`SELECT step_name, action_name, note FROM claim_step_log WHERE claim_id = ? ORDER BY id`, claimID)
	if err != nil {
		return snapshot, err
	}
	defer rows.Close()

	for rows.Next() {
		var stepName string
		var actionName string
		var note string
		if err := rows.Scan(&stepName, &actionName, &note); err != nil {
			return snapshot, err
		}
		snapshot.OrderedActionTrail = append(snapshot.OrderedActionTrail, fmt.Sprintf("%s:%s(%s)", stepName, actionName, note))
	}
	if err := rows.Err(); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func FormatSnapshot(snapshot Snapshot) string {
	lines := []string{
		fmt.Sprintf("identity.verified=%t", snapshot.IdentityVerified),
		fmt.Sprintf("assessment.status=%s", snapshot.AssessmentStatus),
		fmt.Sprintf("funds.status=%s amount=%d", snapshot.FundsStatus, snapshot.FundsAmount),
		fmt.Sprintf("surveyor.status=%s", snapshot.SurveyorStatus),
		fmt.Sprintf("transfer.status=%s lastError=%s", snapshot.TransferStatus, snapshot.TransferLastError),
	}
	if len(snapshot.OrderedActionTrail) == 0 {
		lines = append(lines, "actionTrail=<empty>")
	} else {
		lines = append(lines, "actionTrail="+strings.Join(snapshot.OrderedActionTrail, " -> "))
	}
	return strings.Join(lines, "\n")
}

func appendStepLog(db *sql.DB, businessKey string, claimID string, stepName string, actionName string, note string) error {
	_, err := db.Exec(`INSERT INTO claim_step_log(business_key, claim_id, step_name, action_name, note) VALUES(?, ?, ?, ?, ?)`,
		businessKey, claimID, stepName, actionName, note)
	return err
}
