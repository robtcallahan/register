# Code Review — robtcallahan/register

Reviewed at commit `dd20e83` (Phase 11.6, 2026-10-13). ~6,500 lines of non-test Go.
Scope: idioms, bugs, smells, architecture. No fixes applied — analysis only.

Findings are ordered by teaching value within each section. Each has a
file:line reference, what/why, and the direction of the fix.

---

## Critical — bugs and data-loss risks

### C1. Token-expiry guard never refreshes recently-expired tokens
`pkg/sheets_auth/sheets_auth.go:50`
```go
if err != nil || time.Now().Sub(tok.Expiry) >= time.Hour*24*7 {
```
`time.Now().Sub(tok.Expiry)` is the time *since* expiry. A token that died
an hour ago gives `1h >= 7d` → false → the dead token is used. The guard
checks "expired more than 7 days ago," not "expired." Fix:
`time.Now().After(tok.Expiry)`. **Lesson:** read boolean conditions out loud
as English sentences — "now minus expiry is at least 7 days" is not the same
as "the token is expired."

### C2. `CreateMerchant` silently drops `TaxDeductible`
`pkg/repository/mysql/mysql.go:80-95`
It builds a brand-new `models.Merchant{...}` copying only
Name/BankName/ColumnID/Priority/MatchType/Pattern — `TaxDeductible` from the
input is discarded, so a tax-deductible rule is saved as non-deductible.
**Lesson:** when a function takes a struct and re-creates it field-by-field,
every omitted field is a silent bug. Prefer passing the input through
(`m.Pattern = pattern; r.Conn.Create(m)`) and setting only what needs
defaulting.

### C3. `buildTransaction` has no `default` case — unknown banks produce broken rows
`pkg/banking/transactions.go:128`
The `switch bankID` handles WellsFargo/Fidelity/Chase. Any other `bankID`
falls through and returns a transaction with empty `Source`, zero `Amount`,
and **empty `Key`**. With the unique index on `key`, a second such row
upserts over the first — silent data corruption. **Lesson:** a `switch` over
external input needs a `default` that returns an error (which means changing
the signature to return `(*models.Transaction, error)` — the honest shape).

### C4. `GetFormula` can panic on an empty-but-non-nil response
`api/clients/sheets_client/sheets_client.go:33`
```go
if resp == nil { return nil, errors.New("empty response") }
return resp.ValueRanges[0], err
```
If `resp != nil` but `resp.ValueRanges` is empty, `ValueRanges[0]` panics.
Also note: if `err != nil` it still indexes `resp`. **Lesson:** nil-checks
must cover the actual dereference. `len(resp.ValueRanges) == 0` is the check
this code needs.

### C5. `doWellsFargoHeadsExist` panics on files shorter than 6 bytes
`pkg/csv/csv.go:274`
Hand-rolled prefix loop over `[]int8{0..5}` indexes `header[i]` without a
length check. **Lesson:** this is exactly what `bytes.HasPrefix` is for —
one call, no loop, no panic, and it reads as English.

### C6. `UpdateTransactionTables`: N+1 writes with no transaction
`pkg/repository/mysql/mysql.go:39-52`
One `Create` round-trip per transaction, no wrapping DB transaction. A
mid-batch failure leaves half the batch written with no way to retry cleanly
(the upsert makes re-running *mostly* safe, but the batch isn't atomic).
**Lesson:** `CreateInBatches` or `r.Conn.Transaction(...)` around the loop —
"all or nothing" is a correctness property, not a performance tweak.

### C7. `Merchant`/`Column` embed `gorm.Model` *and* declare their own `ID int`
`pkg/models/models.go:30-40`, `:43-48`
`gorm.Model` already contains `ID uint`. The outer `ID int` shadows it, so
`m.ID` (int) and `m.Model.ID` (uint, dead) both exist. It works today because
GORM picks up the shadowing field, but it's a landmine for anyone reading or
refactoring. **Lesson:** embedding is composition — don't redeclare what you
embedded. Either use `gorm.Model` alone or declare the timestamp fields
explicitly.

---

## Best practices — idiomatic Go

### B1. `%v` instead of `%w` severs error chains
`cmd/update.go:182,196,201`, `api/services/sheets_service/register_service.go:173,365`
```go
return nil, fmt.Errorf("error creating file %s: %v", ..., err)  // should be %w
```
`%v` prints the error but destroys matchability — no caller can ever use
`errors.Is`/`errors.As` on the cause. The rest of the codebase uses `%w`
correctly; these five are the exceptions. **Lesson:** `%w` is free and keeps
both human-readable text and machine-matchable causes.

### B2. Nobody uses `errors.Is` — so the wrapping is decorative, and one spot string-matches errors
`api/services/sheets_service/register_service.go:169`
```go
if strings.Contains(err.Error(), "grid limits") {
```
Meanwhile `errSheetFull` is carefully wrapped with `%w` (`:171,189`) but no
caller ever checks it with `errors.Is`. **Lesson:** wrapping and matching are
a pair — wrap with `%w` *and* match with `errors.Is`/`errors.As` (the Sheets
SDK error supports `errors.As` into `apierror.APIError`; see the existing
example in `sheets_provider.go:45`). String-matching error text breaks the
day Google rewords a message.

### B3. Bare `return err` with zero context
`pkg/plaid_auth/plaid_auth.go:29-33,40-46`, `register_service.go:195`,
`register_write.go:283-284`, `mysql.go` (`GetNameMapToColumn`)
When a Plaid call fails, the caller sees only the raw SDK error — nothing
says *which operation* failed. **Lesson:** each layer adds a breadcrumb:
`fmt.Errorf("create Plaid link token: %w", err)`. Think "what was I trying
to do" at every return.

### B4. `errors.New`, not `fmt.Errorf`, for static strings
`pkg/banking/banking.go:176`, `cmd/migrate.go:59`, `cmd/columns.go:171,174,232`
```go
return nil, fmt.Errorf("available balance is not set")  // no verbs → errors.New
```
**Lesson:** `fmt.Errorf` without verbs is `errors.New` wearing a costume.
Linters (`perfsprint`) flag it; readers expect formatting.

### B5. `context.Context` must be the first parameter
`pkg/plaid_auth/plaid_auth.go:38`, `pkg/banking/banking.go:109,157`
```go
func ExchangePublicToken(c *banking.Client, publicToken string, ctx context.Context)
```
**Lesson:** this is in Go's official CodeReviewComments — readers scan
signatures expecting `ctx` first, and linters flag violations.

### B6. Replace `golang.org/x/net/context` with stdlib `context`
`sheets_auth.go:26`, `plaid_auth.go:8`, `transactions.go:28`, `banking.go:24`,
`cmd/balances.go:8`, `sheets_provider.go:12`
`x/net/context` is a pre-Go-1.7 compatibility shim (2016). The stdlib
`context` is identical. **Lesson:** old tutorials never die — when you copy
an import, check whether the stdlib has absorbed it since.

### B7. Initialisms are all-caps, even mid-word
`pkg/csv/csv.go` (`bankId` ×~17) → `bankID`; `sheets_client.go:22,27`
(`spreadsheetId`) → `spreadsheetID`; `banking.go:52,56`
(`DocumentationUrl`, `RequestId`) → `URL`, `ID`.
**Lesson:** the one naming rule Go is strict about: `userID`, `parseJSON`,
`requestURL`. `bankId` reads as "bankid."

### B8. `any` instead of `interface{}` in new code
`pkg/csv/csv.go:89,107`, `api/services/sheets_service/helpers.go:50`
Since Go 1.18 they're aliases; `any` is the preferred spelling. (The
`[][]interface{}` in `values.go` is forced by the Google SDK's types —
leave those.) **Lesson:** small, but it's the kind of thing that signals
"written after 2022" vs "copied from a 2016 tutorial."

### B9. Delete the `else` after `return`
`cmd/root.go:91-95`
```go
if cfg, err := cfg.ReadConfig(ConfigFile); err != nil {
    fmt.Println(err.Error())
    return
} else {
    config = cfg
}
```
**Lesson:** once the `if` branch returns, the `else` is dead weight.
Dedent and delete — Go style is "errors return early, happy path goes left."

### B10. `regexp.MustCompile` inside hot loops
`monthly_service.go:90` (inside the per-register-row loop in `Aggregate`),
`pkg/csv/csv.go:262,343,383,403`, `transactions.go:266`
Every row pays a full regex compile. **Lesson:** package-level
`var checkRe = regexp.MustCompile(...)` compiles once. The codebase already
does this right elsewhere (`creditCardPaymentPatterns`, `cellRefRegex`) —
same rule, applied inconsistently.

### B11. `for err = fmt.Errorf(""); err != nil;` — just write `for {}`
`cmd/update.go:549`
Constructing an empty error solely to enter a loop is bewildering. The
idiom is an infinite loop with a `break` on success. **Lesson:** when a
construct makes the reader stop and re-read, there's almost always a simpler
shape — here, `for { columnID, err = readInt(...); if err == nil { break } }`.

### B12. `log.Fatalf` in tests kills the whole test binary
`api/services/sheets_service/sheets_service_test.go:85,90`
`log.Fatal` calls `os.Exit(1)` — deferred cleanups never run and the entire
suite dies instead of one test failing. **Lesson:** `t.Fatalf` fails just
that test and lets the suite continue. (`log.Fatal` in `cmd/` entry points
is correct — that's the one place exiting is the job.)

---

## Architecture — structural improvements

### A1. `SheetsService` is a god object (52 methods) with hidden ordering rules
`api/services/sheets_service/` — 9 files, one struct: register read/write,
budget read/write, monthly aggregates, column add/delete/move, copy-rows.
Worse than the method count is **temporal coupling**: `UpdateRows`
(`register_write.go:17-22`) dereferences `ss.BudgetSheet.CategoriesMap` —
nil-pointer panic unless `NewBudgetSheet`+`ReadBudgetSheet` ran first.
`New()` sets only `Provider`; everything else is a public mutable field
assigned from cmd (`sheetsService.IncomeSources = ...`, `cmd/update.go:82`).
**Lesson:** split into `RegisterService`, `BudgetService`, `MonthlyService`,
each constructed with exactly what it needs. That converts "remember to call
X first" from tribal knowledge into compile-time structure.

### A2. Google SDK types leak through every layer — nothing is translated
The provider interface speaks `sheets.ValueRange`,
`sheets.BatchUpdateSpreadsheetRequest`; services use `sheets.CellData` (92
references), `sheets.CopyPasteRequest`, etc. directly; `RegisterSheet` embeds
`sheets.Spreadsheet` by value (`register_service.go:44`); `WriteCell`
returns `*sheets.UpdateValuesResponse` (`:228`). Four layers deep, everyone
speaks Google. **Lesson:** a layer earns its keep by *translating* — SDK
types in, domain types out. Today swapping the backend (or faking it without
the SDK) means rewriting the services package. The provider should return
domain types; `sheets.*` should appear only inside the provider.

### A3. `sheets_client` is ceremony — and `ClearValues` proves it
`api/clients/sheets_client/sheets_client.go` (56 lines): five one-line SDK
pass-throughs, called through a mutable package global
(`sheets_client.ClientStruct`, used at `sheets_provider.go:59-95`), so the
`ClientInterface` can never actually be swapped. And `ClearValues`
(`sheets_provider.go:100-103`) bypasses the client entirely and calls the
SDK directly. **Lesson:** if the layer above demonstrably works without a
layer, that layer is forwarding, not abstracting. Delete it (Rob already
said "sounds good") — and prefer constructor injection over globals, which
`SheetsService.New(provider)` already does right.

### A4. The update pipeline lives in `cmd` — 634 lines of untestable domain logic
`cmd/update.go`: `update()` composes fetch → normalize → dedupe → persist →
spreadsheet-write, and implements domain steps as cmd functions
(`normalizeMerchants` :208, `fuzzyKeySet` :248, `persistTransactions` :277,
`updateSpreadsheet` :305). No test exercises `update()`; the HTTP server
(`cmd/server.go`) couldn't reuse any of it without dragging cobra along.
`readFromUser` (:540-616, 77 lines) mixes stdin prompting, DB writes, and
merchant-rule business logic, then `return true, trans, nil` from deep inside
a loop after the *first* prompt. **Lesson:** cmd is the composition root —
parse flags, build deps, call one service method, print. Interactive
prompting genuinely belongs in cmd; the pure pipeline (normalize → dedupe →
persist) wants a service home where tests can reach it.

### A5. `QueryRepo` is 17 methods wide; 5 are dead and 2 are debug helpers
`pkg/repository/repository.go:12-30`. Zero non-test callers: `CreateDB`,
`SaveTransaction` (superseded by `UpdateTransactionTables`),
`GetMerchants` (superseded by `GetMerchantsByColumn`), `PrintData`,
`PrintTable`. One implementation exists; nothing fakes the interface in
tests. **Lesson:** discover interfaces from consumers ("I need to fake this
boundary"), don't declare them as wish lists. Debug printers on a
*repository interface* show how kitchen-sink interfaces accrete — trim to
what's used.

### A6. `banking.Client` mixes a Plaid HTTP client with pure functions
`SortTransactions` (:177), `FilterRecordedTransactions` (:193),
`FilterCreditCardPayments` (:218), `merchants.go:51 FormatMerchantNames` —
pure transforms hung as methods on the Plaid client, most never touching the
receiver. Reusing them requires `NewClient`, which builds a real
`plaid.APIClient` with credentials. **Lesson:** a method that ignores its
receiver is a package function in costume. Free functions are directly
testable with no secrets; the client keeps only what actually calls Plaid.

### A7. Three `readDateValue`s, same name/signature, different behavior
`pkg/csv/csv.go:402` (M/D/YYYY), `pkg/banking/transactions.go:265`
(YYYY-MM-DD), `api/services/sheets_service/values.go:50` (M/D/YYYY again,
near-identical to the csv one). Feed the banking one a slash date and it
returns `""` silently — no error path. **Lesson:** shared logic in three
packages means a caller can import the wrong one and get wrong dates with
zero signal. Hoist to one `pkg`-level helper with distinct names
(`ParseSlashDate`, `ParseISODate`) and let errors surface.

### A8. N+1 remote calls: `moveColumn` and per-transaction scans
`cmd/columns.go:380-407` calls `ss.BudgetRowFor(c.Name)` per same-color
column — each a separate Sheets `GetValues` round trip. Read column A once
into a `map[string]int` and scan in memory. Same family:
`isIncomeName`/`incomeFrequency` (`row_builder.go:110-123`) linearly scan
`IncomeSources` and are called **twice per transaction** (`:44,53`).
**Lesson:** the N+1 query pattern isn't just for databases — it's any loop
that does I/O per item. Build the map once per run.

---

## Minor — nits and style

### M1. Dead struct fields send readers hunting
`SheetsService.service` (`sheets_service.go:44`) is never read (pre-provider
residue); `SheetsService.SpreadsheetID` (`:46`) and
`BudgetSheet.SpreadsheetID` (`budget_service.go:27`) are never assigned;
`RegisterSheet.Spreadsheet`/`BudgetSheet.Spreadsheet` are never read (and
copy the whole metadata blob by value). **Lesson:** dead fields on shared
structs are worse than dead functions — every future reader wastes time
finding where the value gets set.

### M2. Magic strings where constants already exist
`"CHECK"` literals (`csv.go:269-270`, `transactions.go:133-134`,
`update.go:502`) while `banking.CheckTransactionName` exists;
`"WellsFargo"` literals (`csv.go:246`, `transactions.go:136`,
`update.go:420`) while `sheets_service.CheckingAccountSourceName` exists
(and note the lowercase-`bankID` vs titlecase-`Source` split —
`banking.WellsFargoID` vs the literals — exactly what shared constants
prevent); raw `"white"`/`"grey"` in `monthly_service.go` and
`row_builder.go:66,68` while `models.ColorWhite`/`ColorGrey` exist.
**Lesson:** constants are the compiler-checked spelling of a domain value.

### M3. `string(rune('A'+j))` instead of `models.ColumnLetter` — and it breaks past Z
`budget_service.go:273`. A correct helper exists (`column_set.go:116`,
handles AA/AB/…); the hand-rolled version silently produces non-letters past
column 26. **Lesson:** when a helper exists, the hand-rolled duplicate is
where the next bug lives.

### M4. Long functions with extractable jobs
`InsertBudgetCategory` (`budget_service.go:214`, 70 lines — the totals-formula
extension loop `:246-270` is a self-contained `extendTotalsFormulas` helper);
`addSummaryRows` (`monthly_service.go:196-203`) and `addSummarySalaryRow`
(`:247-254`) build identical SUM/AVERAGE cells verbatim, and
`populateYearlySummary` (`:321`) repeats the totals-row construction a third
time — one `addTotalsCells` serves all three; `moveColumn`
(`cmd/columns.go:347`, 103 lines — budget-neighbor scan `:380-407`
extracts cleanly); `sortAggregateMapKeys` (`helpers.go:37`) hand-rolls what
`slices.Sorted(maps.Keys(...))` does two functions away in the same package.
**Lesson:** "extract till you drop" — a 70-line function doing 4 jobs is 4
functions wearing a trench coat.

### M5. Dead and commented-out code
Commented-out `getPlaidTransactions` (~30 lines, `transactions.go:236`);
commented `weeksAgo` lines (`update.go:410-411`, `server.go:205-207`);
`//printExpirationDifference` (`sheets_auth.go:48`); `//"bytes"` in csv
imports. Uncalled: `printRegister` (`update.go:436`), `WriteCSV`
(`transactions.go:245`, hardcoded `"csv/"` dir), `PrintTransactionHead`
(`:187`), `printPlaidTransaction`/`printTransaction` (`:61-67`). Plus a
stale re-check: `update.go:82-84` re-tests the `sheets_provider.New` error
three lines after it was already handled — classic copy-paste error handling.
**Lesson:** version control remembers; commented code doesn't need to. Delete
with confidence — `git log` is the archive.

### M6. Naming nits
Snake-case packages stutter: `sheets_service.SheetsService`,
`sheets_client.ClientStruct` (Go: single lowercase word, no underscores, and
the type shouldn't repeat the package). `range_` params
(`sheets_client.go:22,27`, `sheets_provider.go:58,67,103`) → `rng` (the
codebase already uses `readRange` elsewhere). Capitalized local
`Day := time.Hour * 24` (`sheets_auth.go:63`). Typo "covert" for "convert"
(`transactions.go:144`).

---

## Suggested curriculum order (for the teaching walkthrough)

1. **A1 + A2 + A3** — what a layer is *for* (translation, not forwarding).
   The single biggest architectural lesson in the codebase.
2. **C1** — read boolean conditions as English sentences.
3. **B1 + B2** — `%w`/`errors.Is` as a pair; never string-match errors.
4. **A4** — keep `cmd` thin; pure pipeline vs. interactive I/O.
5. **A7 + M2** — one home for shared logic; constants as compiler-checked spelling.
6. **B10 + A8** — cost of repeated work (regex compiles, N+1 calls).
7. **C2 + C7** — struct field shadowing and silent field-dropping.
8. **M4 + M5** — extraction and deletion: the two cheapest refactors.

## Verified healthy (no action)

- Dependency direction is clean: no import cycles, no upward/sideways deps;
  `cmd` is a proper composition root. This is the hard thing to fix later —
  it's already right.
- `SheetsProviderInterface` earns its keep: the test suite's `providerMock`
  implements it, so 2,100 lines of tests run without Google credentials.
  (Caveat: the mock is driven by package-level mutable func vars —
  `getValuesProviderFunc` etc. — order-sensitive and parallel-unsafe; a
  struct-held fake is the next step up.)
- No `panic` calls in non-test code; `log.Fatal`/`os.Exit` only in `cmd/`
  entry points; error strings consistently lowercase; `%w` is the norm
  outside the B1 exceptions.
