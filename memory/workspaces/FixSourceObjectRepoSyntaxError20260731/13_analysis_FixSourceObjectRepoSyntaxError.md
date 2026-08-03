# Phân Tích Sâu Nguyên Nhân & Giải Pháp Lỗi 500 SQLSTATE 42601

## I. Phân Tích Nguyên Nhân Lỗi
- Log: `2026/07/31 17:35:54 internal/infra/persistence/source/source_object_read_repo_gorm.go:127 ERROR: syntax error at or near "LEFT" (SQLSTATE 42601)`
- API: `GET /api/v1/source-objects` (HTTP 500)
- Nguyên nhân: Hằng số `listBaseFromWhere` trong `source_object_read_repo_gorm.go` bị mất dòng kết thúc subquery LATERAL `rr` (`LIMIT 1 \n ) rr ON TRUE`).
- Đoạn SQL lỗi hiện tại:
  ```sql
  ORDER BY rr.checked_at DESC
  LEFT JOIN LATERAL (
          SELECT status, rows_affected, job_id, error_message
          FROM cdc_system.transform_jobs tj
          ...
  ) tj ON TRUE
  ```
  PostgreSQL gặp từ khóa `LEFT` ngay sau `ORDER BY rr.checked_at DESC` mà không có `LIMIT 1` và ngoặc đóng subquery `)` nên báo lỗi cú pháp 42601.

## II. Giải Pháp
1. Bổ sung `LIMIT 1 \n ) rr ON TRUE` cho subquery `rr`.
2. Đồng thời bổ sung `AND rr.checked_at >= NOW() - INTERVAL '7 days'` để tối ưu tốc độ scan.
