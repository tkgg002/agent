# Implementation Plan: Remove Hardcoded "_id" -> "id" Overrides

## Overview
Xóa bỏ hoàn toàn các câu lệnh ép đè cưỡng chế `_id` thành `"id"` tại `event_handler.go` và `bridge_handler.go` để bảo toàn giá trị `PrimaryKeyField = "_id"` cho các bảng MongoDB.

## User Review Required

> [!IMPORTANT]
> - **Phát hiện Nguyên nhân Gốc rễ Chính xác:** Trong `event_handler.go` (dòng 353 & 384) và `bridge_handler.go` (dòng 281), có các câu lệnh `if pkField == "_id" { pgPKField = "id" }` tự ý ép đè `_id` thành `"id"`. Mặc dù `PrimaryKeyField` trong Registry và TableConfig đã đúng là `_id`, chính các dòng code này đã làm `record.PrimaryKeyField` bị đổi thành `"id"`, dẫn đến SQL `INSERT INTO relation ("id", ...)` văng lỗi `SQLSTATE 42703`.

## Proposed Changes

### `centralized-data-service`

#### [MODIFY] [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)
- Xóa dòng 353-355: `if pgPKField == "_id" { pgPKField = "id" }`.
- Xóa dòng 384-386: `if !mappedPK && pkField == "_id" { pgPKField = "id" }`.

#### [MODIFY] [bridge_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/bridge_handler.go)
- Xóa dòng 281-283: `if resolved.pgPKField == "" || resolved.pgPKField == "_id" { resolved.pgPKField = "id" }`.

## Verification Plan

### Automated Tests
- Chạy `go test ./internal/handler/shadow/...` và `go test ./internal/handler/source/...` trong `centralized-data-service`.
- Chạy linter `python3 agent/tooling/verify_governance.py`.
