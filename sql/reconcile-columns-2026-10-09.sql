-- Column reconciliation, 2026-10-09: mirror the Register sheet in the DB.
-- Sheet is truth (Rob's Oct 8 moves + repurposings). Merchant rules key on
-- column_id, so rows MOVE with their names and rules follow their category.
-- Review first: SELECT id, bank_name, name FROM merchants
--               WHERE column_id IN (17, 30, 61) ORDER BY column_id;

START TRANSACTION;

-- 1. Retire SoberLink (id 61, idx 26): its sheet column became Car Loan.
--    Deletes its 3 merchant rules too.
DELETE FROM merchants WHERE column_id = 61;
DELETE FROM columns WHERE id = 61;

-- 2. Renames in place (columns repurposed in the sheet, same physical seat)
UPDATE columns SET name = 'T-Mobile' WHERE id = 17;        -- was AT&T (idx 17/Q)
UPDATE columns SET name = 'Car Insurance' WHERE id = 30;  -- was Storage Rental (idx 27/AA)

-- 3. Park the moving rows so the column_index UNIQUE constraint
--    never sees a transient collision
UPDATE columns SET column_index = column_index + 1000 WHERE id IN (31, 34, 35, 36);

-- 4. Final seats per the sheet
UPDATE columns SET column_index = 30 WHERE id = 31;  -- Credit Cards -> AD
UPDATE columns SET column_index = 26 WHERE id = 34;  -- Car Loan -> Z
UPDATE columns SET column_index = 32 WHERE id = 35;  -- IRS -> AF
UPDATE columns SET column_index = 29 WHERE id = 36;  -- Smart Tag -> AC

-- 5. AppleCard (sheet AE, idx 31) has no DB row; create it.
--    letter is still NOT NULL in the DDL, so it rides along.
INSERT INTO columns (name, color, column_index, letter, is_category, created_at, updated_at)
VALUES ('AppleCard', 'yellow', 31, 'AE', 1, NOW(), NOW());

COMMIT;

-- Verify with: register columns check   (expect: OK)
