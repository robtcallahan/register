-- +goose Up
ALTER TABLE `columns` DROP COLUMN `is_category`;

-- +goose Down
ALTER TABLE `columns` ADD `is_category` tinyint NOT NULL,

