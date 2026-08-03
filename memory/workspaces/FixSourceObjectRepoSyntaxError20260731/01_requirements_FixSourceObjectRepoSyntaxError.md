# Yêu cầu Fix Lỗi Cú Pháp SQL trong Source Object Repo (FixSourceObjectRepoSyntaxError)

## 1. Bối cảnh & Hiện trạng
Hệ thống ghi nhận lỗi HTTP 500 khi gọi API `/api/v1/source-objects`:
- File: `internal/infra/persistence/source/source_object_read_repo_gorm.go:127`
- Log lỗi: `ERROR: syntax error at or near "LEFT" (SQLSTATE 42601)`
- Nguyên nhân: Hằng số `listBaseFromWhere` tại subquery `LEFT JOIN LATERAL` của bảng `cdc_reconciliation_report rr` bị thiếu câu lệnh đóng subquery (`LIMIT 1 \n ) rr ON TRUE`), khiến câu SQL bị thiếu ngoặc đóng trước khi sang `LEFT JOIN LATERAL tj`.

## 2. Mục tiêu (Definition of Done)
- [ ] Khôi phục ngoặc đóng `LIMIT 1 \n ) rr ON TRUE` cho subquery `rr` trong `listBaseFromWhere`.
- [ ] Bổ sung `WHERE rr.checked_at >= NOW() - INTERVAL '7 days'` cho subquery `rr` để tối ưu performance.
- [ ] Chạy `go build ./cmd/server` biên dịch THÀNH CÔNG 100%.
- [ ] Sửa dứt điểm lỗi HTTP 500 trên API `/api/v1/source-objects`.
