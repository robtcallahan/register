#  starHouse-Affordability Planning — design brief

**Idea:** a small read-only command, `register affordability`, that answers one question from Rob's real numbers: *can I afford this house, and what does it leave me?* No new data entry, no writes to the sheet or DB, no monthly/yearly summaries, no pie chart.

## What it computes

1. **Proposed monthly housing cost** — the total of the categories on Rob's separate housing budget sheet: mortgage, mortgage insurance, home insurance, property tax, home-expenses savings, plus any others he listed. The home-expenses savings counts as a real monthly cost, not an optional extra.
2. **Versus rent** — the proposed total against his current \~$1,400/mo rent, as a plain delta: how much more (or less) per month the house costs.
3. **Actual monthly surplus** — average monthly income minus average monthly spending, computed from real transactions in the app over a trailing window (SS \~$2,981/mo + Taylor House bi-weekly pay, both already in the app). Uses observed deposits, so the Social Security earnings-test withholding is already reflected in the numbers.
4. **Surplus after the move** — today's surplus minus the rent-to-house delta. This is the headline: if it's negative, the house as priced doesn't fit on current income.
5. **Maximum supportable housing cost** — today's surplus + current rent, i.e. the monthly housing total that would leave surplus at zero. Optionally with a minimum-surplus floor (see open questions).
6. **Down-payment picture** — for a given home price, what 10% and 20% down look like drawn from the IRA (\~$130k) first, per Rob's call — his thinking is the stocks (\~$50k) grow faster, so spend the IRA and let the stocks keep growing. Show BOTH options side by side (IRA-first vs stocks-first) so the trade-off is visible: what remains in each account under each option, and the tax note below. Note in the output that IRA withdrawals are taxable income — the app shows the numbers, it doesn't give tax advice.

## Inputs and where they come from

| Input | Source |
| --- | --- |
| Housing categories + amounts | Rob's separate housing budget sheet, read-only, reusing the existing BudgetSheet read path (needs the tab name / spreadsheet — see open question 1) |
| Income and spending actuals | Existing transactions (sheet is the truth; aggregates via the existing `Aggregate` pattern in monthly\_service) |
| Current rent | Actual Rent transactions, averaged over the same window |
| IRA / stocks balances | Not in the app — supplied as flags or a small config entry (e.g. `--ira 130000 --stocks 50000`) |
| Home price, mortgage rate | Flags, only needed for the down-payment scenarios and any price-implied-by-payment conversion |

## Example output (all numbers ILLUSTRATIVE, not Rob's real figures)

```
Housing (from your housing sheet)      $2,100/mo
  mortgage $1,450 · PMI $95 · insurance $180 · tax $325 · home savings $100
Current rent                            $1,400/mo
Housing costs $700/mo more than rent

Trailing 6-mo income                    $5,200/mo
Trailing 6-mo spending (incl. rent)     $4,700/mo
Surplus today                             $500/mo
Surplus after the move                   -$200/mo   <- doesn't fit as priced

Max housing cost at zero surplus        $1,900/mo

Down payment on a $250,000 home:
  20% ($50,000)  from IRA — $80,000 IRA left, stocks untouched
  10% ($25,000)  from IRA — $105,000 IRA left, stocks untouched
  (stocks-first shown alongside for comparison)
```

## Decisions (Rob, 2026-11-06)

1. **Housing sheet location** — a tab in the same Finances 2026 Google sheet, named **"House Budget"**. Rob is updating it to match the Budget format exactly, with more rows. Read it via the existing BudgetSheet read path, same as the main Budget tab. The command reads whatever rows are in the tab — it does not hardcode the categories, so a category added later shows up with no code change.
2. **Trailing window and floor** — the window is a parameter (flag, e.g. `--months 6`), not a fixed number. Minimum surplus floor is **$200/mo**.
3. **Down payment** — 100% IRA first (Rob's call: stocks grow faster, so let them keep growing). Show both options (IRA-first vs stocks-first) so the trade-off is visible.

Note: Rob is checking his Social Security withholding (SSA site) and his Taylor House W4 withholding, possibly tomorrow. The function reads actual deposits, so whatever withholding is in effect is already reflected — the numbers follow automatically once those change.