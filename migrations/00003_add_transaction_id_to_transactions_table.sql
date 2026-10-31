-- +goose Up
ALTER TABLE `transactions`
    ADD COLUMN `transaction_id` VARCHAR(255) NULL AFTER `key`,
    ADD INDEX `idx_transactions_transaction_id` (`transaction_id`);

-- +goose Down
ALTER TABLE `transactions` DROP COLUMN `transaction_id`;

