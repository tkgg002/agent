# 00_context.md: Fix Invalid Input Syntax for Type JSON (SQLSTATE 22P02)

## Context & Scope
- **Source Service / Topic**: `vmg-ekyc-connector-service.bank-requests`
- **Destination Table / Schema**: `shadow_vmg_ekyc.bank_requests` / Master table
- **Sample Document ID**: `6a8e3fc31b3d2729eb078a60` (MongoDB ObjectId)
- **Error Category**: `type_error`
- **Exact Exception**: `ERROR: invalid input syntax for type json (SQLSTATE 22P02)`

## Symptom Analysis
Khi CDC Worker (`centralized-data-service`) thực hiện sync/batch upsert các bản ghi từ MongoDB collection `bank-requests` thuộc service `vmg-ekyc-connector-service` (ObjectId `6a8e3fc31b3d2729eb078a60`), PostgreSQL ném lỗi SQLSTATE 22P02 (invalid_text_representation) do dữ liệu truyền vào cột kiểu `json` / `jsonb` (hoặc `_raw_data`) không tuân thủ cú pháp JSON tiêu chuẩn của PostgreSQL.
