-- Key is source:date:amount, which is not unique: distinct transactions can
-- share one (e.g. two $2.00 transactions on the same day), and the unique
-- index drops the second one on insert. A plain index keeps the lookup.
-- +goose Up
ALTER TABLE `transactions` DROP INDEX `idx_transactions_key`;
ALTER TABLE `transactions` ADD INDEX `idx_transactions_key` (`key`);

-- +goose Down
ALTER TABLE `transactions` DROP INDEX `idx_transactions_key`;
ALTER TABLE `transactions` ADD UNIQUE INDEX `idx_transactions_key` (`key`);
