# Requirements — Phase 15 Retry Failed Scope Enrichment

## Mục tiêu

- Đánh giá `POST /api/failed-sync-logs/{id}/retry` có cần đổi sang scope-aware input hay không.
- Nếu không cần đổi identity input, vẫn phải enrich contract để operator hiểu đúng source/shadow scope.

## Yêu cầu

1. Phân biệt rõ `retry-by-ID` với `check/heal-by-scope`.
2. Không làm gãy downstream worker đang consume `cdc.cmd.retry-failed`.
3. Cập nhật swagger/comment cho retry endpoint trong cùng phase.
4. FE failed-log view phải ưu tiên metadata nằm ngay trên failed log nếu có.
