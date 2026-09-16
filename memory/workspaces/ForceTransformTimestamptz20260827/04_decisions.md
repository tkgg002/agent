# 04 Architectural Decision Records (ADRs)

## ADR-01: Không dùng `UPDATE col = NULL` trên toàn bộ bảng lớn
- **Bối cảnh:** Dữ liệu sau khi ALTER TYPE từ `timestamp` sang `timestamptz` bị sai giá trị. Cần trích xuất lại từ `_raw_data`.
- **Quyết định:** Tuyệt đối không chạy `UPDATE table SET col = NULL` trước khi transform vì sẽ gây AccessExclusive / RowExclusive lock nặng, phát sinh WAL khổng lồ trên bảng 100M+ bản ghi. Thay vào đó, thiết kế cơ chế **Force Transform** trực tiếp ghi đè dữ liệu mới trích xuất từ `_raw_data` theo từng chunk 1000 rows.
- **Hệ quả:** Tiến trình chạy êm ái dưới nền, không khóa bảng, cập nhật tiến độ realtime theo từng chunk.

## ADR-02: Bắt buộc `force_fields` không rỗng khi `force=true`
- **Bối cảnh:** Nếu người dùng bật force mode mà không chỉ định danh sách field, toàn bộ các cột có thể bị ghi đè không cần thiết.
- **Quyết định:** Validate nghiêm ngặt `force=true` phải đi kèm mảng `force_fields` có ít nhất 1 phần tử. Nếu rỗng, worker lập tức reject job.
- **Hệ quả:** Đảm bảo tính cô lập và an toàn, chỉ ghi đè đúng những cột cần khắc phục.

## ADR-03: Tách riêng nhánh CAST `timestamptz` và `timestamp`
- **Bối cảnh:** Trước đây 2 kiểu dữ liệu này dùng chung biểu thức CASE trong `BuildCastExpr` với fallback `::TIMESTAMP`.
- **Quyết định:** Tách 2 case độc lập. Nhánh `timestamptz` dùng `::TIMESTAMPTZ` ở fallback; nhánh `timestamp` giữ `::TIMESTAMP`.
- **Hệ quả:** Bảo toàn thông tin timezone offset đối với chuỗi ISO 8601 / RFC 3339 khi parse từ JSON.
