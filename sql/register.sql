DROP TABLE IF EXISTS `columns`;
CREATE TABLE `columns` (
	`id` int NOT NULL AUTO_INCREMENT,
	`name` varchar(100) DEFAULT NULL,
	`color` varchar(45) NOT NULL,
	`column_index` int NOT NULL,
	`letter` varchar(45) NOT NULL,
	`is_category` tinyint NOT NULL,
	`created_at` datetime DEFAULT NULL,
	`updated_at` datetime DEFAULT NULL,
	`deleted_at` datetime DEFAULT NULL,
	PRIMARY KEY (`id`),
	INDEX `id_UNIQUE` (`id`),
	INDEX `column_index_UNIQUE` (`column_index`),
	INDEX `name_UNIQUE` (`name`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `merchants`;
CREATE TABLE `merchants` (
	`id` int NOT NULL AUTO_INCREMENT,
	`name` varchar(100) NOT NULL,
	`bank_name` varchar(150) NOT NULL,
	`column_id` int NOT NULL,
	`created_at` datetime(3) DEFAULT NULL,
	`updated_at` datetime(3) DEFAULT NULL,
	`deleted_at` datetime(3) DEFAULT NULL,
	`tax_deductible` tinyint(1) DEFAULT 0,
	PRIMARY KEY (`id`),
	INDEX `id_UNIQUE` (`id`),
	INDEX `bank_name_UNIQUE` (`bank_name`),
	INDEX `idx_merchants_deleted_at` (`deleted_at`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `transactions`;
CREATE TABLE `transactions` (
	`id` bigint unsigned NOT NULL AUTO_INCREMENT,
	`created_at` datetime(3) DEFAULT NULL,
	`updated_at` datetime(3) DEFAULT NULL,
	`deleted_at` datetime(3) DEFAULT NULL,
	`key` longtext DEFAULT NULL,
	`source` longtext DEFAULT NULL,
	`date` longtext DEFAULT NULL,
	`name` longtext DEFAULT NULL,
	`bank_name` longtext DEFAULT NULL,
	`merchant_name` longtext DEFAULT NULL,
	`amount` double DEFAULT NULL,
	`withdrawal` double DEFAULT NULL,
	`deposit` double DEFAULT NULL,
	`credit_card` double DEFAULT NULL,
	`column_index` bigint DEFAULT NULL,
	`color` longtext DEFAULT NULL,
	`is_category` tinyint(1) DEFAULT NULL,
	`tax_deductible` tinyint(1) DEFAULT NULL,
	`is_check` tinyint(1) DEFAULT NULL,
	`note` longtext DEFAULT NULL,
	`credit_purchase` double DEFAULT NULL,
	`budget` double DEFAULT NULL,
	PRIMARY KEY (`id`),
	INDEX `idx_transactions_deleted_at` (`deleted_at`)
) ENGINE=InnoDB;

