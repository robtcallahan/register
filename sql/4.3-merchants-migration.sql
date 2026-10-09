-- Register refactor 4.3: migrate the config.json "merchants" map into the merchants table.
-- Each rule is inserted at priority 100 (legacy rows sit at 0) with match_type 'substring',
-- preserving how the config overrides behaved. The column FK is resolved from the config's
-- column_index at run time. Run once; the NOT EXISTS guard makes re-running harmless.

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Transfer to/from Ally Bank', 'ALLY BANK P2P', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 37
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'ALLY BANK P2P' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Amazon Marketplace', 'AMAZON MKTPL', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 35
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'AMAZON MKTPL' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'American Strategies Rental Insurance', 'AMERICAN STRATEG', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 25
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'AMERICAN STRATEG' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Amazon Marketplace', 'AMZN MKTP', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 35
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'AMZN MKTP' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'AppleCard Payment', 'APPLECARD GSBANK PAYMENT', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 29
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'APPLECARD GSBANK PAYMENT' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'ATM Cash Withdrawal', 'ATM WITHDRAWAL AUTHORIZED', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 11
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'ATM WITHDRAWAL AUTHORIZED' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Amazon', 'Amazon.com', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 35
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'Amazon.com' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'App Folio, Inc', 'AppFolio, Inc', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 24
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'AppFolio, Inc' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Transfer from Betterment IRA', 'BETTERMENT SEC   TRANSFER', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 37
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'BETTERMENT SEC   TRANSFER' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Fidelity Visa Payment', 'BILL PAY Fidelity Visa', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 29
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'BILL PAY Fidelity Visa' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Bank of America Mastercard Payment', 'BILL PAY MasterCard', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 29
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'BILL PAY MasterCard' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'City of Winchester Bill Pay', 'CITY OF WINCHEST BILLPAY', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 22
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'CITY OF WINCHEST BILLPAY' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Cash Withdrawal', 'Cash eWithdrawal in Branch', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 11
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'Cash eWithdrawal in Branch' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Transfer from E*Trade', 'E*TRADE BANK TRANSFER', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 37
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'E*TRADE BANK TRANSFER' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'GloFiber', 'GLO FIBER        BILLPA', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 19
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'GLO FIBER        BILLPA' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Homesite Renter''s Insurance', 'HOMESITE INS PREM', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 25
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'HOMESITE INS PREM' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'HP Instant Ink', 'HP *INSTANT INK', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 18
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'HP *INSTANT INK' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Non-Wells Fargo ATM Cash Withdrawal', 'NON-WF ATM WITHDRAWAL', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 11
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'NON-WF ATM WITHDRAWAL' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT '50/50 Taphouse Paycheck', 'NOVA BEER LLC PAYROLL', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 33
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'NOVA BEER LLC PAYROLL' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Olive Tree Property Management', 'Olive Tree Prop', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 24
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'Olive Tree Prop' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'PlexPass', 'PLEXINC', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 18
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'PLEXINC' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Shenandoah Valley Electric', 'SHENANDOAH VALLE UTILITY', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 20
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'SHENANDOAH VALLE UTILITY' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Spotify', 'SPOTIFY', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 18
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'SPOTIFY' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Star Asian Spa', 'STAR ASIAN SPA', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 23
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'STAR ASIAN SPA' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Uber Trip', 'UBER   *TRIP', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 13
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'UBER   *TRIP' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'Venmo', 'VENMO', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 36
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'VENMO' AND m.priority = 100);

INSERT INTO merchants (name, bank_name, column_id, tax_deductible, priority, match_type, created_at, updated_at, deleted_at)
SELECT 'eBay', 'eBay O', c.id, 0, 100, 'substring', NOW(), NOW(), NULL
FROM columns c
WHERE c.column_index = 36
  AND NOT EXISTS (SELECT 1 FROM merchants m WHERE m.bank_name = 'eBay O' AND m.priority = 100);

-- Verify (expect 27):
SELECT COUNT(*) AS migrated_rules FROM merchants WHERE priority = 100;
SELECT bank_name, name, column_id FROM merchants WHERE priority = 100 ORDER BY bank_name;