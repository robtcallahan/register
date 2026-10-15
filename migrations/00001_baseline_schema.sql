-- +goose Up
-- Baseline schema for register: columns, merchants, transactions.
-- Captures the state after all hand-DDL through 2026-10-13:
--   - columns.letter dropped (2026-10-10)
--   - merchants.match_pattern, match_type, priority added
--   - transactions.key unique index (9.1)

CREATE TABLE IF NOT EXISTS `columns` (
	`id` int NOT NULL AUTO_INCREMENT,
	`name` varchar(100) DEFAULT NULL,
	`color` varchar(45) NOT NULL,
	`column_index` int NOT NULL,
	`is_category` tinyint NOT NULL,
	`created_at` datetime DEFAULT NULL,
	`updated_at` datetime DEFAULT NULL,
	`deleted_at` datetime DEFAULT NULL,
	PRIMARY KEY (`id`),
	UNIQUE KEY `column_index_UNIQUE` (`column_index`),
	UNIQUE KEY `name_UNIQUE` (`name`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `merchants` (
	`id` int NOT NULL AUTO_INCREMENT,
	`name` varchar(100) NOT NULL,
	`bank_name` varchar(150) NOT NULL,
	`column_id` int NOT NULL,
	`created_at` datetime(3) DEFAULT NULL,
	`updated_at` datetime(3) DEFAULT NULL,
	`deleted_at` datetime(3) DEFAULT NULL,
	`tax_deductible` tinyint(1) DEFAULT 0,
	`priority` int NOT NULL DEFAULT 0,
	`match_type` varchar(20) DEFAULT '',
	`match_pattern` varchar(150) DEFAULT '',
	PRIMARY KEY (`id`),
	UNIQUE KEY `bank_name_UNIQUE` (`bank_name`),
	KEY `idx_merchants_deleted_at` (`deleted_at`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `transactions` (
	`id` bigint unsigned NOT NULL AUTO_INCREMENT,
	`created_at` datetime(3) DEFAULT NULL,
	`updated_at` datetime(3) DEFAULT NULL,
	`deleted_at` datetime(3) DEFAULT NULL,
	`key` varchar(255) DEFAULT NULL,
	`source` longtext,
	`date` longtext,
	`name` longtext,
	`bank_name` longtext,
	`note` longtext,
	`amount` double DEFAULT NULL,
	`withdrawal` double DEFAULT NULL,
	`deposit` double DEFAULT NULL,
	`credit_purchase` double DEFAULT NULL,
	`budget` double DEFAULT NULL,
	`credit_card` double DEFAULT NULL,
	`column_index` bigint DEFAULT NULL,
	`color` longtext,
	`is_category` tinyint(1) DEFAULT NULL,
	`tax_deductible` tinyint(1) DEFAULT NULL,
	`is_check` tinyint(1) DEFAULT NULL,
	PRIMARY KEY (`id`),
	UNIQUE KEY `idx_transactions_key` (`key`),
	KEY `idx_transactions_deleted_at` (`deleted_at`)
) ENGINE=InnoDB;

-- +goose Down
DROP TABLE IF EXISTS `transactions`;
DROP TABLE IF EXISTS `merchants`;
DROP TABLE IF EXISTS `columns`;
