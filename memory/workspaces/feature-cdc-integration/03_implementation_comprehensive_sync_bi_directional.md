# Implementation Plan: Comprehensive Bi-directional Sync (Airbyte & CMS)

Tài liệu này xác định các kịch bản đồng bộ hai chiều để đảm bảo CMS và Airbyte luôn nhất quán.

## 1. Môi trường & Bối cảnh
- **Workspace**: feature-cdc-integration
- **Tình trạng**: Hệ thống đang ở chế độ CMS-centric, thiếu cơ chế cập nhật ngược từ Airbyte.

## 2. Các kịch bản chi tiết (Sync Matrix)

### CMS-Driven (CMS ra lệnh)
- **Register Table**: Thêm stream vào catalog hiện có.
- **Toggle Active**: Cập nhật `Selected` của stream trong Airbyte.
- **Delete Registry**: Prune (loại bỏ) stream khỏi Airbyte connection.

### Airbyte-Driven (CMS lắng nghe)
- **Reconciliation Loop**: Worker chạy ngầm hằng phút (Poller) để fetch catalog thực tế từ Airbyte. Nếu User tắt trên UI Airbyte, CMS sẽ tự cập nhật DB `is_active = false`.
- **Smart Import**: Phát hiện Connection đã tồn tại trên Airbyte nhưng chưa có trong Registry -> Hiển thị danh sách để Admin Import nhanh.
- **Monitoring & Alerts**: Nếu Job Airbyte Fail > 3 lần, CMS tự động thông báo và Pause stream đó.

## 3. Các bước thực hiện
1. [ ] Bổ sung API `v1/connections/list` và logic so sánh catalog vào `AirbyteClient`.
2. [ ] Xây dựng background worker (Reconciliation Service) trong CDC Worker.
3. [ ] Cập nhật UI CMS để hiển thị trạng thái "Mismatched" nếu có sự lệch pha giữa CMS và Airbyte.

## 4. Root Cause Analysis (Governance Violation)
- **Lỗi**: Brain (Agent) đã tạo kế hoạch nhưng chỉ lưu ở thư mục Artifact tạm thời, không lưu vào Workspace Memory của dự án.
- **Nguyên nhân**: Quên bước "Maintenance" sau khi nhận phản hồi từ User.
- **Biện pháp khắc phục**: Luôn kiểm tra danh sách file trong `agent/memory/workspace` trước khi kết thúc session.
