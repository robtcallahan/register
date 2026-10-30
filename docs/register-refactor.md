# Register Refactor Project

## Phase 1 — Crash fixes

- [X] 1.1 Fix defer f.Close() before error check in import.go (3 spots)
- [X] 1.2 Fix err.Error() on nil err in cmd/server.go (4 handlers)
- [X] 1.3 Fix ae.Reason() nil deref in sheets_provider.go:44
- [X] 1.4 Fix nil context.Context in banking.go:178 → context.Background()
- [X] 1.5 Fix budget off-by-one in budget_service.go:75 (guard vs values[6])
- [X] 1.6 Fix makeNoteRow nil-prefill in register_service.go:252
- [X] 1.7 Add safe accessor for spreadsheet row indexing, use everywhere
- [X] 1.8 Check regex matches before indexing in both readDateValues
- [X] 1.9 Fix createLinkToken where it logs the error if it's "", but then keeps going
- [X] 1.10 change config file and code use so that db_ is an object

## Phase 2 — Errors instead of process kills

- [X] 2.1 checkError (panic) in banking.go → return errors
- [X] 2.2 checkError (os.Exit) in root.go → return errors, exit once in cobra Run
- [X] 2.3 log.Fatalf in sheets_service → return errors
- [X] 2.4 log.Fatalf in sheets_auth → return errors
- [X] 2.5 Check result.Error on every GORM read
- [X] 2.6 Replace panic(result.Error) on writes with returned errors
- [X] 2.7 Sheets-full detection → clean "add rows and re-run" message
- [X] 2.8 Fail fast on empty SESSION_KEY; fix 0644 token file perms

## Phase 3 — De-god update()

- [X] 3.1 Extract fetchTransactions stage (Plaid or CSV)
- [X] 3.2 Extract normalizeMerchants stage (merchant name lookup)
- [X] 3.3 Extract dedupeAgainstSheet stage (fuzzy ±1-day date matching)
- [X] 3.4 Extract persistTransactions stage (DB + interactive prompts)
- [X] 3.5 Extract updateSpreadsheet stage (CopyRows + UpdateRows + balances)
- [X] 3.6 Replace shellout("./register copy") with direct sheetsService.CopyRows call
- [X] 3.7 Rename --no-updates flag to --dry-run (fix double negative)
- [X] 3.8 Move interactive prompts before long silent work
- [X] 3.9 Centralize dedup key construction into one function
- [X] 3.10 Extract WriteRawTransactionsCSV stage (raw fetch dump via encoding/csv)

## Phase 4 — Unify merchant matching in DB

- [X] 4.1 Add priority and match_type columns to merchants table
- [X] 4.2 Prune unused columns (TaxDeductible et al) — keep TaxDeductible (house purchase next year); no prune, table stays
- [X] 4.3 Migrate config.json merchants map into DB, delete from config
- [X] 4.4 Write single MerchantMatcher, first-match-wins by priority
- [X] 4.5 Replace FormatMerchantNames + FormatUniqueTransactionNames with matcher
- [X] 4.6 Reference columns by name, kill magic indexes (10, 42)

## Phase 5 — Paycheck redesign

- [X] 5.1 Define IncomeSource in config (name, match rule, frequency, allocations)
- [X] 5.2 Add weekly paycheck + monthly Social Security as two income sources
- [X] 5.3 Remove hardcoded PayCheckBankName/PayCheckName from banking.go
- [X] 5.4 Replace addSalaryCells hack with income-source allocation logic
- [X] 5.5 Remove ColumnIndex=42 magic number

## Phase 6 — Column management CLI

- [X] 6.1 Make DB columns table the single source of truth (ColumnSet, identity-based cell placement, end column derived from DB — done 2026-10-09)
- [X] finding #5 prompt fix — done
- [X] finding #6 cleanup — done
- [X] 6.2 columns CLI: Add register columns list|add|rename|delete commands (done 2026-10-09)
- [X] 6.3 Add register columns check for sheet-vs-DB drift (done 2026-10-09 — caught the 7 mismatches, reconciled same day)
- [X] 6.4 Retire the is_category flag; category-ness derived from position (done 2026-10-09)
- [X] 6.5 Budget category names from the same source (the Budget-sync CLI work)
- [X] 6.6 Config index cleanup (remove ColumnIndexes/RegisterCategoryEndColumn from config; RegisterIndexes already gone)
- [X] 6.7 Implement register columns move CLI command

## Phase 7 — Constants + readability

- [X] 7.1 Move cell addresses, colors, magic values to constants.go or config
- [X] 7.2 Split register_service.go into focused functions
- [X] 7.3 Split helpers.go into focused functions
- [X] 7.4 Split banking.go GetTransactions
- [X] 7.5 Go-idiom cleanup (boolean returns, *[]string, fmt.Errorf)
- [X] 7.6 Slim SheetsService god-object (RowBuilder type)
- [X] 7.7 Remove handler.Query delegation layer or give it a job

## Phase 8 — Monthly summary correction

- [X] 8.1 Fix Aggregate's category read: correct the 1-based ColumnIndex into the 0-based row slice (currently reads one column right); replace magic j := 10 and hardcoded cols[j].Name == "Credit Cards" with ColumnSet-driven iteration (FirstCategoryIndex for the start, color for skipping payoff columns)
- [X] 8.2 Fix populateMonthlyPayees: build the payee list from the inner map keys (currently month keys → month labels and zeros); stop addSummaryRows from skipping 10 entries for payees
- [X] 8.3 Derive the summary formula range from the actual month count (hardcoded B:L = 11 months today) in addSummaryRows/addSummarySalaryRow
- [X] 8.4 Return errors from UpdateMonthlyCategories/UpdateMonthlyPayees in monthly() instead of swallowing them (monthly() always returns nil today)
- [X] 8.5 Remove dead if err != nil after sheets_service.New() in monthly() (New returns no error; it re-checks the provider's)
- [X] 8.6 Remove commented-out JSON debug block + dead jsonDir const in monthly.go
- [X] 8.7 Create a pie chart for monthly summary on a separate worksheet to the right of the register tab
- [ ] 8.8 Pie chart of yearly category totals on YearlySummary (first Sheets chart API work)
- [X] fix-fold-driver-into-repository-v3.patch

## Phase 9 — Miscellaneous fixes

- [X] 9.1 Add unique index on transactions.key so UpdateTransactionTables upserts instead of duplicating (empty the table once first; OnConflict UpdateAll already in place)
- [X] 9.2 Amazon ±1-day date drift can still mint a second key — handled by Phase 3 fuzzy matching (cross-reference, not a separate fix)
- [X] 9.3 Replace positional values[index] sheet/CSV reads with header-named row decoding (Budget first, then Register row-4 headers, then CSV header row); missing expected header errors loudly instead of reading the wrong column

## Phase 10 — Functional tests

- [X] 10.1 Make `go test ./...` build and run: fix the three stale monthly-test signatures, resolve Test_getTransactionKey1 and the csv ReadCSVFiles/Banks references, delete or fill the 16 csv skeleton tests, remove the machine-absolute fixture paths (fixtures under testdata/, no log.Fatalf in init)
- [X] 10.2 Direct unit tests for models.TransactionKey (canonical format: lowercase source, %.2f amount, date passthrough, 100.5 vs 100.50 equivalence)
- [X] 10.3 banking.buildTransaction matrix per bank: Wells Fargo sign conventions + check detection, Chase/Fidelity credit-purchase fields, Key stamping on each path
- [X] 10.4 banking filters: FilterCreditCardPayments, FilterRecordedTransactions (exact-key), FindByBankPattern (CHECK resolution)
- [X] 10.5 Income allocation: allocationFor table test (weekly / monthly / other-frequency visible zeros) + addSalaryCells/addCategoryCells with a GAP in the column set (pins 6.1 identity placement beyond the gapless goldens)
- [X] 10.6 NewRegisterSheet end-column derivation from DB columns (fake provider), including the config fallback used by register copy
- [X] 10.7 CSV conversions: real cases for the process*Data functions + processCheck through the FakeReadFiler seam
- [X] 10.8 Small cmd units: filterNonCategoryColumns (6.4 threshold rule) and WriteRawTransactionsCSV (3.10 quoting regression)
- [X] 10.9 Columns CLI safety: AddColumn/DeleteColumn index shifting + fixed-block/--after/delete guards (DB test strategy first)
- [X] 10.10 Golden fixture audit against post-6.x behavior (budget fixture must encode the post-column-D layout); regenerate or retire stale goldens, dedupe the two addCategoryCells goldens
- [X] 10.11 UpdateTransactionTables upsert/idempotence DB test (lands with or just after 9.1)
- [X] 10.12 (low) SESSION_KEY server guard test; monthly Aggregate/populateMonthlyPayees tests rewritten only after Phase 8 settles behavior
- [X] 10.13 Add 18 mysql DB tests (will use register_test DB)

## Phase 11 — Database migration with goose

- [X] 11.1 Add goose dependency, create migrations/ directory
- [X] 11.2 Write baseline migration from current schema (columns, merchants, transactions)
- [X] 11.3 Add `register migrate up|down|status|redo` CLI commands
- [X] 11.4 Remove GORM AutoMigrate from UpdateTransactionTables (migrations own schema)
- [X] 11.5 Remove hand-DDL from mysql package (CreateTable calls, etc.)
- [X] 11.6 Document migration workflow in repo (how to create/apply a new migration)

## Phase 12 — Critical bug fixes

- [X] 12.1 Fix CreateMerchant silently dropping TaxDeductible (mysql.go:80-95) — pass input struct through instead of rebuilding field-by-field
- [X] 12.2 Add default case to buildTransaction switch (transactions.go:128) — return error for unknown banks, change signature to (*models.Transaction, error)
- [X] 12.3 Fix GetFormula panic on empty ValueRanges (sheets_client.go:33) — check len before indexing
- [X] 12.4 Fix doWellsFargoHeadsExist panic on short files (csv.go:274) — use bytes.HasPrefix
- [X] 12.5 Wrap UpdateTransactionTables writes in a DB transaction (mysql.go:39-52) — all-or-nothing batch
- [X] 12.6 Fix Merchant/Column ID shadowing gorm.Model (models.go:30-48) — remove redeclared ID field (dropped as too many casts of uint to int. Will be fixed in 12.7)

## Phase 13 — Error handling done right

- [X] 13.1 Replace %v with %w in 5 spots (update.go:182,196,201; register_service.go:173,365)
- [ ] 13.2 Replace strings.Contains error matching with errors.Is/errors.As (register_service.go:169) Deferred
- [X] 13.3 Add context to bare return err (plaid_auth.go, register_service.go:195, register_write.go:283-284, mysql.go)
- [X] 13.4 Use errors.New instead of fmt.Errorf for static strings (banking.go:176, migrate.go:59, columns.go:171,174,232)

## Phase 14 — Go idioms

- [X] 14.1 Move ctx to first parameter (plaid_auth.go:38, banking.go:109,157)
- [X] 14.2 Replace golang.org/x/net/context with stdlib context (6 files)
- [X] 14.3 Fix initialisms to all-caps: bankId→bankID, spreadsheetId→spreadsheetID, etc.
- [X] 14.4 Use any instead of interface{} in new code (csv.go:89,107; helpers.go:50)
- [X] 14.5 Delete else after return (root.go:91-95)
- [X] 14.6 Hoist regexp.MustCompile out of hot loops (monthly_service.go:90, csv.go, transactions.go:266)
- [X] 14.7 Replace for-err-loop with for{}+break (update.go:549)
- [X] 14.8 Replace log.Fatalf with t.Fatalf in tests (sheets_service_test.go:85,90)

## Phase 15 — Layering: what a layer is for

- [X] 15.1 Split SheetsService god object into RegisterService, BudgetService, MonthlyService (52 methods, temporal coupling)
- [X] 15.2 Translate Google SDK types at provider boundary — no sheets.* outside provider package
- [X] 15.3 Delete sheets_client package (56 lines of pass-through) — fold into provider (Rob approved)

## Phase 16 — Structural improvements

- [X] 16.1 Thin out cmd/update.go (634 lines) — move pure pipeline to service, keep cmd as composition root
- [X] 16.2 Trim QueryRepo to used methods only — remove CreateDB, SaveTransaction, PrintData, PrintTable
- [X] 16.3 Convert banking.Client pure methods to package functions (SortTransactions, Filter*, FormatMerchantNames)
- [X] 16.4 Consolidate three readDateValues into one pkg-level helper with distinct names
- [X] 16.5 Fix N+1 Sheets calls in moveColumn — read column A once into map
- [X] 16.6 Make fuzzy date matching merchant-specific — currently fuzzyKeySet applies ±1 day to EVERY transaction; scope it to Amazon merchants (or make the window configurable per merchant/source)

## Phase 17 — Cleanup

- [X] 17.1 Remove dead struct fields (SheetsService.service, SpreadsheetID fields)
- [X] 17.2 Replace magic strings with existing constants (CHECK, WellsFargo, white/grey)
- [X] 17.3 Replace string(rune('A'+j)) with models.ColumnLetter
- [X] 17.4 Extract long functions (InsertBudgetCategory, moveColumn, update, recategorize, readFromUser, deleteColumn, addColumn, PlanCategoryMoves, & UpdateSpreadsheet)
- [X] 17.5 Delete commented-out and dead code (WriteCSV, etc.)
- [X] 17.6 Split sheets_service_test.go by service (register/budget/monthly + shared helpers in helpers_test.go)
- [X] 17.7 Replace gorm.Model with explicit fields (deferred 12.6)
- [X] 17.8 Drop dead is_category DB column via goose migration
- [X] moveColumn(name, moveAfter string) signature fix
- [X] Shared bufio.Reader for readString/readInt (one reader, created once)
- [X] Printing policy design item (progress callback)
- [X] IncomeSource.Match → MatchPattern rename (deferred)
- [ ] Remove external JSON files from test that use them
  - [X] Test_WriteCell — includes the mock's read of the same file, so that goes too
  - [X] Test_addAmountCell
  - [X] Test_monthlyService_addSummarySalaryRow
  - [ ] Test_monthlyService_addSummaryTopRow

## Phase 18 — Test coverage

*Scope: instructive subset by Rob, bulk by Muse. Specific cases to be chosen at kickoff from a coverage report.*
