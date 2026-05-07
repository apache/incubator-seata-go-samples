-- Licensed to the Apache Software Foundation (ASF) under one or more
-- contributor license agreements.  See the NOTICE file distributed with
-- this work for additional information regarding copyright ownership.
-- The ASF licenses this file to You under the Apache License, Version 2.0
-- (the "License"); you may not use this file except in compliance with
-- the License.  You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

CREATE TABLE IF NOT EXISTS `seata_state_machine_def` (
  `id` varchar(128) NOT NULL,
  `tenant_id` varchar(32) DEFAULT NULL,
  `app_name` varchar(64) DEFAULT NULL,
  `name` varchar(128) NOT NULL,
  `status` varchar(16) DEFAULT NULL,
  `gmt_create` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
  `ver` varchar(16) DEFAULT NULL,
  `type` varchar(32) DEFAULT NULL,
  `content` mediumtext,
  `recover_strategy` varchar(32) DEFAULT NULL,
  `comment_` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_smdef_name_tenant` (`name`,`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `seata_state_machine_inst` (
  `id` varchar(128) NOT NULL,
  `machine_id` varchar(128) NOT NULL,
  `tenant_id` varchar(32) DEFAULT NULL,
  `parent_id` varchar(256) DEFAULT NULL,
  `gmt_started` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
  `gmt_end` datetime(6) DEFAULT NULL,
  `status` varchar(16) DEFAULT NULL,
  `compensation_status` varchar(16) DEFAULT NULL,
  `is_running` tinyint(1) DEFAULT 0,
  `gmt_updated` datetime(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `business_key` varchar(128) DEFAULT NULL,
  `start_params` mediumtext,
  `end_params` mediumtext,
  `excep` blob,
  PRIMARY KEY (`id`),
  KEY `idx_sminst_machine` (`machine_id`),
  KEY `idx_sminst_parent` (`parent_id`),
  KEY `idx_sminst_bizkey_tenant` (`business_key`,`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `seata_state_inst` (
  `id` varchar(128) NOT NULL,
  `machine_inst_id` varchar(128) NOT NULL,
  `name` varchar(128) NOT NULL,
  `type` varchar(32) NOT NULL,
  `gmt_started` datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
  `service_name` varchar(255) DEFAULT NULL,
  `service_method` varchar(255) DEFAULT NULL,
  `service_type` varchar(32) DEFAULT NULL,
  `is_for_update` tinyint(1) DEFAULT 0,
  `input_params` mediumtext,
  `status` varchar(16) DEFAULT NULL,
  `business_key` varchar(128) DEFAULT NULL,
  `state_id_compensated_for` varchar(128) DEFAULT NULL,
  `state_id_retried_for` varchar(128) DEFAULT NULL,
  `output_params` mediumtext,
  `excep` blob,
  `gmt_end` datetime(6) DEFAULT NULL,
  `gmt_updated` datetime(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `idx_stinst_machine` (`machine_inst_id`),
  KEY `idx_stinst_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `claim_identity` (
  `claim_id` varchar(64) NOT NULL,
  `business_key` varchar(64) NOT NULL,
  `claimant_id` varchar(64) NOT NULL,
  `verified` tinyint(1) NOT NULL DEFAULT 0,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`claim_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `claim_assessment` (
  `claim_id` varchar(64) NOT NULL,
  `business_key` varchar(64) NOT NULL,
  `assessment_id` varchar(64) NOT NULL,
  `status` varchar(32) NOT NULL,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`claim_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `claim_fund_reservation` (
  `claim_id` varchar(64) NOT NULL,
  `business_key` varchar(64) NOT NULL,
  `amount` int NOT NULL,
  `status` varchar(32) NOT NULL,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`claim_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `claim_surveyor_notice` (
  `claim_id` varchar(64) NOT NULL,
  `business_key` varchar(64) NOT NULL,
  `surveyor_id` varchar(64) NOT NULL,
  `status` varchar(32) NOT NULL,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`claim_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `claim_transfer` (
  `claim_id` varchar(64) NOT NULL,
  `business_key` varchar(64) NOT NULL,
  `bank_account` varchar(64) NOT NULL,
  `amount` int NOT NULL,
  `status` varchar(32) NOT NULL,
  `last_error` varchar(255) NOT NULL DEFAULT '',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`claim_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `claim_step_log` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `business_key` varchar(64) NOT NULL,
  `claim_id` varchar(64) NOT NULL,
  `step_name` varchar(64) NOT NULL,
  `action_name` varchar(64) NOT NULL,
  `note` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_claim_step_log_claim` (`claim_id`),
  KEY `idx_claim_step_log_biz` (`business_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
