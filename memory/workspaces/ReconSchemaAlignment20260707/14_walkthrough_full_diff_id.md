# Báo cáo Kết quả (Walkthrough) - Khắc phục Hiển thị Dữ liệu ID Diff

## Tóm tắt công việc đã thực hiện
Đã khắc phục lỗi không trả về dữ liệu ID bị lệch (`missing_ids`, `stale_ids`, `field_diffs`) trong hàm `GetTableHistory` bằng cách chiếu đầy đủ các trường diff và heal metrics tương ứng trong truy vấn `UNION ALL`.

## Các thay đổi chính
- Cập nhật [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go#L236) để SELECT thêm:
  - `missing_count`, `missing_ids`, `stale_count`, `stale_ids`, `field_diffs`, `orphan_count`, và các trường heal ở bảng `cdc_reconciliation_report`.
  - Các giá trị giả lập kiểu (`0::integer`, `NULL::jsonb`, `NULL::timestamp without time zone`) ở bảng `cdc_recon_smoke_result` để khớp cấu trúc `UNION ALL`.

## Kết quả kiểm thử & xác minh

### 1. Compilation & Unit Tests
- Build code thành công và chạy PASS toàn bộ 100% unit tests trong package queries.

### 2. Xác minh API thực tế
- Gọi API lịch sử đối soát, các bản ghi đối soát sâu (ví dụ ID `34` loại `hash_window`) đã trả về chính xác cấu trúc dữ liệu bị lệch:
  ```json
  "id": 34,
  "check_type": "hash_window",
  "missing_ids": [],
  "stale_ids": {
    "mismatched": [
      "6a448a7cb544c04498b9ba2"
    ],
    "missing_from_src": [],
    "missing_from_dest": []
  }
  ```
