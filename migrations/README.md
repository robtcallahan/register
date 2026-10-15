# Database Migrations

This directory holds goose database migrations for the register app.

## Why migrations?

Before Phase 11, schema changes were done by hand (SQL files in `sql/`) or via
GORM `AutoMigrate`. That meant no versioning, no rollback, and no way to know
what state a database was in. Migrations fix that: each change is a versioned
file, applied in order, with a down direction for rollback.

## Workflow

### Check status

```bash
register migrate status
```

Shows which migrations are applied vs pending.

### Apply pending migrations

```bash
register migrate up
```

### Roll back the last migration

```bash
register migrate down
```

### Roll back to a specific version

```bash
register migrate down-to 1
```

### Redo (down then up) — useful when testing a new migration

```bash
register migrate redo
```

## Creating a new migration

Use goose to scaffold:

```bash
goose -dir migrations mysql "user:pass@tcp(localhost:3306)/register?parseTime=true" create add_column_name sql
```

This creates `migrations/00002_add_column_name.sql`. Edit it:

```sql
-- +goose Up
ALTER TABLE merchants ADD COLUMN foo varchar(50);

-- +goose Down
ALTER TABLE merchants DROP COLUMN foo;
```

Then apply with `register migrate up`.

## Rules

1. **Never edit an applied migration.** If it's already run on any database,
   write a new migration instead. Editing history breaks the version chain.

2. **Always write the Down.** Even if you think you'll never roll back, future
   you will thank present you.

3. **Test both directions.** After writing a migration, run:
   ```bash
   register migrate up      # apply it
   register migrate down    # roll it back
   register migrate up      # apply it again
   ```
   All three should succeed without errors.

4. **The baseline (00001) is special.** It's the foundation — `down` drops all
   tables. Never run `down` on the baseline against a database with real data.
   (You have SQL dumps as backup, but don't test that.)

## Migration history

| Version | Name            | Description                              |
|---------|-----------------|------------------------------------------|
| 00001   | baseline_schema | Initial schema: columns, merchants, transactions |
