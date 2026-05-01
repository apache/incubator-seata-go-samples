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
	"fmt"
	"os"
)

const (
	DefaultMySQLHost     = "127.0.0.1"
	DefaultMySQLPort     = "3306"
	DefaultMySQLUser     = "root"
	DefaultMySQLPassword = "secret"
	DefaultMySQLDB       = "seata_saga"

	DefaultIdentityPort   = "18081"
	DefaultAssessmentPort = "18082"
	DefaultFundsPort      = "18083"
	DefaultSurveyorPort   = "18084"
	DefaultTransferPort   = "18085"
)

type Settings struct {
	MySQLHost string
	MySQLPort string
	MySQLUser string
	MySQLPass string
	MySQLDB   string

	IdentityPort   string
	AssessmentPort string
	FundsPort      string
	SurveyorPort   string
	TransferPort   string
}

func LoadSettings() Settings {
	return Settings{
		MySQLHost:      envOrDefault("MYSQL_HOST", DefaultMySQLHost),
		MySQLPort:      envOrDefault("MYSQL_PORT", DefaultMySQLPort),
		MySQLUser:      envOrDefault("MYSQL_USER", DefaultMySQLUser),
		MySQLPass:      envOrDefault("MYSQL_PWD", DefaultMySQLPassword),
		MySQLDB:        envOrDefault("MYSQL_DB", DefaultMySQLDB),
		IdentityPort:   envOrDefault("IDENTITY_SERVICE_PORT", DefaultIdentityPort),
		AssessmentPort: envOrDefault("ASSESSMENT_SERVICE_PORT", DefaultAssessmentPort),
		FundsPort:      envOrDefault("FUNDS_SERVICE_PORT", DefaultFundsPort),
		SurveyorPort:   envOrDefault("SURVEYOR_SERVICE_PORT", DefaultSurveyorPort),
		TransferPort:   envOrDefault("TRANSFER_SERVICE_PORT", DefaultTransferPort),
	}
}

func (s Settings) MySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		s.MySQLUser, s.MySQLPass, s.MySQLHost, s.MySQLPort, s.MySQLDB)
}

func (s Settings) IdentityBaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%s/", s.IdentityPort)
}

func (s Settings) AssessmentBaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%s/", s.AssessmentPort)
}

func (s Settings) FundsBaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%s/", s.FundsPort)
}

func (s Settings) SurveyorBaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%s/", s.SurveyorPort)
}

func (s Settings) TransferBaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%s/", s.TransferPort)
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
