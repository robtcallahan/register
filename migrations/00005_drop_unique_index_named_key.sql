-- The live table carries a second unique index on `key`, named `key` itself
-- (not created by any goose migration, so 00004 missed it). It still enforces
-- Key uniqueness and drops legitimate duplicate-Key rows on insert.
-- +goose Up
ALTER TABLE `transactions` DROP INDEX `key`;

-- +goose Down
ALTER TABLE `transactions` ADD UNIQUE INDEX `key` (`key`);
