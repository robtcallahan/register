# Migrations

This directory holds goose database migrations.

## Workflow

Create a new migration:
```bash
goose -dir migrations mysql "user:pass@tcp(localhost:3306)/register?parseTime=true" create add_new_column sql
```

Apply pending migrations:
```bash
register migrate up
```

Roll back the last migration:
```bash
register migrate down
```

Check status:
```bash
register migrate status
```
