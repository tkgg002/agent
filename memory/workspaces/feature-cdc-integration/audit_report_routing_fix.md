# Audit Report: Routing Fix & Cross-Contamination Resolution

**Date:** 2026-09-25
**Workspace:** feature-cdc-integration
**Status:** QC Passed

## 1. Mục Tiêu (Objective)
Giải quyết dứt điểm vấn đề nhiễm chéo (cross-contamination) khi nhiều connector cùng chia sẻ một bảng/database, đồng thời **bảo toàn 100% tính tương thích ngược** cho các connector cũ (legacy 4-part topic prefix) dựa trên nguyên tắc "Lọc Kết Quả" thay vì "Xoá Khóa Tìm Kiếm".

## 2. Các Thay Đổi Mã Nguồn (Code Changes)

### 2.1. Phục hồi Tương Thích Ngược (Backward Compatibility)
- **File:** `internal/service/source/metadata_registry_utils.go`
- **Hành động:** 
  - Khôi phục `buildSourceLookupKeys` để luôn sinh ra Generic Cache Key (`db|table`) cho tất cả các kết nối, kể cả khi `connCode` có giá trị.
  - Khôi phục `IsGenericConnectionCode` về trạng thái sử dụng `switch case` cho các tiền tố tĩnh (`goopay`, `mongodb`, v.v.), kết hợp với tiền tố động từ config.
- **Lý do:** Đây là hai điểm tựa sống còn của các connector 4-part. Nếu không có Generic Key trong cache và không có logic cắt bỏ `connCode`, các topic 4-part cổ điển (ví dụ `cdc.goopaylocal`) sẽ không thể tìm thấy route đích.

### 2.2. Lọc Chống Nhiễm Chéo (Cross-Contamination Filtering)
- **File:** `internal/service/source/metadata_registry_service.go`
- **Hành động:** 
  - Bổ sung reverse map `connCodeToTopicPrefix map[string]string` để tra cứu nhanh `topic_prefix` của từng connector (từ `cdc_system.sources`).
  - Trong `ResolveSourceRoutes`, nếu `sourceConn == ""` (đồng nghĩa với message 4-part đang dùng Generic Key để tìm), duyệt qua các kết quả trả về (`masterRoutes`).
  - Nếu kết quả nào thuộc về connector có cấu hình `topic_prefix` 5-part (`StripGenericPrefix(tp) != ""`), chủ động lọc bỏ kết quả đó khỏi danh sách trả về.
- **Lý do:** `traitestctphs` (4-part) và `traitestmongodevct` (5-part) dùng chung bảng `trans_his`. Bằng cách lọc ở đầu ra, message 4-part chỉ kích hoạt route của `traitestctphs`, trong khi message 5-part vốn đã có `sourceConn` cụ thể nên không bị ảnh hưởng.

## 3. Laya Critical Review (QC & Phản Biện)

Laya script (`laya_review_routing.py`) đã thực hiện kiểm định logic:

- **Backward Compatibility (PASS):** Khôi phục Generic Key `db|table` giúp 4-part hoạt động y như 6 tháng qua.
- **Cross-Contamination Resolution (PASS):** Phân lập rõ ràng giữa 4-part và 5-part mà không xóa nguồn sinh dữ liệu ở đầu vào.
- **Architectural Alignment (PASS):** Sử dụng hàm `GetTopicPrefixToConnectionMap` truy vấn từ `cdc_system.sources` đúng chuẩn, không hardcode hay phá vỡ kiến trúc cache.
- **Edge Cases (PASS):** 
  - Nếu connector 4-part không khai báo `topic_prefix`, bộ lọc `tp != ""` không kích hoạt, route vẫn được giữ lại (đúng chuẩn tương thích ngược).
  - Nếu 2 connector 5-part trùng bảng, chúng đều được query qua `connCode:db|table`, không qua Generic Key, nên không bị filter nhầm.

## 4. Bài Học Rút Ra (Self-Improvement Loop)
- Đã ghi nhận bài học mới vào `lessons.md`: `#backward-compatibility-annihilation #misunderstood-generic-lookup`.
- Không được phép sửa/xóa các cơ chế nền tảng (Generic Lookup) để phục vụ cho một tính năng mới mà quên mất hệ quả lên hệ thống cũ. Thay vào đó, phải dùng cơ chế Lọc Kết Quả (Result Filtering) để xử lý.

## 5. Kết Luận
- Cấu trúc hệ thống được đảm bảo.
- Compile lỗi `0` ở Core Service.
- Đáp ứng 100% tài liệu plan và phản hồi từ User.
