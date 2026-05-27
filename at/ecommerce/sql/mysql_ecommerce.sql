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

CREATE DATABASE IF NOT EXISTS seata_ecommerce_order DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS seata_ecommerce_inventory DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS seata_ecommerce_account DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE seata_ecommerce_order;

CREATE TABLE IF NOT EXISTS order_tbl (
  id INT NOT NULL AUTO_INCREMENT,
  user_id VARCHAR(64) NOT NULL,
  commodity_code VARCHAR(64) NOT NULL,
  count INT NOT NULL,
  money INT NOT NULL,
  status VARCHAR(32) NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS undo_log (
  id BIGINT NOT NULL AUTO_INCREMENT,
  branch_id BIGINT NOT NULL,
  xid VARCHAR(100) NOT NULL,
  context VARCHAR(128) NOT NULL,
  rollback_info LONGBLOB NOT NULL,
  log_status INT NOT NULL,
  log_created DATETIME NOT NULL,
  log_modified DATETIME NOT NULL,
  ext VARCHAR(100) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_unionkey (xid, branch_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

USE seata_ecommerce_inventory;

CREATE TABLE IF NOT EXISTS inventory_tbl (
  id INT NOT NULL AUTO_INCREMENT,
  commodity_code VARCHAR(64) NOT NULL,
  stock INT NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_commodity_code (commodity_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO inventory_tbl (commodity_code, stock) VALUES ('C100001', 100)
ON DUPLICATE KEY UPDATE stock = VALUES(stock);

CREATE TABLE IF NOT EXISTS undo_log (
  id BIGINT NOT NULL AUTO_INCREMENT,
  branch_id BIGINT NOT NULL,
  xid VARCHAR(100) NOT NULL,
  context VARCHAR(128) NOT NULL,
  rollback_info LONGBLOB NOT NULL,
  log_status INT NOT NULL,
  log_created DATETIME NOT NULL,
  log_modified DATETIME NOT NULL,
  ext VARCHAR(100) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_unionkey (xid, branch_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

USE seata_ecommerce_account;

CREATE TABLE IF NOT EXISTS account_tbl (
  id INT NOT NULL AUTO_INCREMENT,
  user_id VARCHAR(64) NOT NULL,
  balance INT NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO account_tbl (user_id, balance) VALUES ('U100001', 50)
ON DUPLICATE KEY UPDATE balance = VALUES(balance);

CREATE TABLE IF NOT EXISTS undo_log (
  id BIGINT NOT NULL AUTO_INCREMENT,
  branch_id BIGINT NOT NULL,
  xid VARCHAR(100) NOT NULL,
  context VARCHAR(128) NOT NULL,
  rollback_info LONGBLOB NOT NULL,
  log_status INT NOT NULL,
  log_created DATETIME NOT NULL,
  log_modified DATETIME NOT NULL,
  ext VARCHAR(100) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_unionkey (xid, branch_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
