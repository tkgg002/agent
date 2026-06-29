# Context - bug-schema-shadow-cls-testing-not-exist-2026-06-23

## Problem
During the execution of `cmd-create-default-columns` on target schema `shadow_cls_testing.export_jobs_1`, the operation fails because the database schema `"shadow_cls_testing"` does not exist in the destination database.

Error logs:
```
create table failed: ERROR: schema "shadow_cls_testing" does not exist (SQLSTATE 3F000)
Operation: cmd-create-default-columns
Scope: export_jobs_1
Status: error
```

## Solution Outline
In the `SchemaAdapter` component of `centralized-data-service`, ensure that whenever `CreateEmptyTable` is called, we issue a `CREATE SCHEMA IF NOT EXISTS` statement first. Since the unit test suite runs on SQLite, which doesn't support schema namespaces natively without attached databases and empty columns in CREATE TABLE statements, dialect-specific guard clauses must be added to ensure the test suite passes seamlessly.
