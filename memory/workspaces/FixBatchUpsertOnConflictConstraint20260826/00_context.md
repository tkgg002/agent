# 00_context.md: Fix Batch Upsert Chunk Failed - SQLSTATE 42P10

## Context & Scope
- **Error Log**: `flush failed: batch upsert chunk failed: ERROR: there is no unique or exclusion constraint matching the ON CONFLICT specification (SQLSTATE 42P10) (fallback persisted 0 rows)`
- **Service**: `centralized-data-service` (`internal/handler/shadow/batch_buffer.go`)
- **Target Table**: V2 Shadow Tables (e.g. `shadow_vmg_ekyc.bank_requests`)
- **Database**: PostgreSQL

## Problem Description
Khi `BatchBuffer` thực hiện `batchUpsert` các bản ghi CDC/Snapshot vào V2 Shadow Tables trên PostgreSQL, hệ thống văng lỗi SQLSTATE 42P10 (`invalid_column_reference` / `there is no unique or exclusion constraint matching the ON CONFLICT specification`).

V2 Shadow Tables được thiết kế với:
- `_gpay_id BIGINT PRIMARY KEY`
- `_source_id TEXT NOT NULL` mang partial UNIQUE index `ux_<table_name>_source_id_active ON ... (_source_id) WHERE NOT _deleted`.

Tuy nhiên, `batch_buffer.go` hiện tại gán `effectivePK := pk` (hoặc `effectivePK := first.PrimaryKeyField`), giá trị này là `"_id"` (MongoDB source primary key field name).
Do `"_id"` không có UNIQUE constraint hay UNIQUE index trên bảng PostgreSQL, câu lệnh `ON CONFLICT ("_id")` bị PostgreSQL từ chối với lỗi SQLSTATE 42P10.
Khi transaction bị sập, fallback từng dòng đơn (sequential fallback) cũng tiếp tục dùng `effectivePK = "_id"`, dẫn đến toàn bộ fallback cũng bị reject (`fallback persisted 0 rows`).
