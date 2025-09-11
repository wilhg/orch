# orch

[![CI](https://github.com/wilhg/orch/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/wilhg/orch/actions/workflows/ci.yml)

## Database

Configure the database with `DATABASE_URL`. Both PostgreSQL and SQLite are supported.

- PostgreSQL example:

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/orch?sslmode=disable"
```

- SQLite example (file-based):

```bash
export DATABASE_URL="sqlite:///./orch.sqlite?cache=shared&_pragma=busy_timeout(5000)"
```

The ent-backed store provides schema migration via code. Call `entstore.Open(ctx, os.Getenv("DATABASE_URL"))` and `store.Migrate(ctx)` during initialization. A CLI helper will be added in later milestones.

