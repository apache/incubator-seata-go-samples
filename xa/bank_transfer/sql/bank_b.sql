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

CREATE DATABASE IF NOT EXISTS db_bank_b DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE db_bank_b;

CREATE TABLE IF NOT EXISTS account_tbl (
  id BIGINT NOT NULL AUTO_INCREMENT,
  account_no VARCHAR(64) NOT NULL,
  balance BIGINT NOT NULL DEFAULT 0,
  frozen TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uk_account_no (account_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO account_tbl (account_no, balance, frozen)
VALUES
  ('B-2001', 500, 0),
  ('B-FROZEN', 500, 1)
ON DUPLICATE KEY UPDATE balance = VALUES(balance), frozen = VALUES(frozen);
