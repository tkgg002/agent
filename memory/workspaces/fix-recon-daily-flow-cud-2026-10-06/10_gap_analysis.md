# Phân tích Lỗ hổng Kiến trúc (Gap Analysis V2): Kế hoạch Clear Activity Log & Grid

## 1. Lỗ hổng 1: Thiếu Partition Pruning trên PostgreSQL Partitioned Table
- **Hiện trạng Schema DB:** Bảng `cdc_system.cdc_activity_log` được partition theo `RANGE (created_at)` (xác minh từ file migration `010_partitioning.sql:99`).
- **Lỗ hổng:** Nếu chỉ lọc `WHERE started_at < before_date`, PostgreSQL KHÔNG THỂ kích hoạt cơ chế Partition Pruning. Lệnh DELETE sẽ phải scan tuần tự và lock toàn bộ 33+ child partitions, gây nguy cơ lock timeout trên Production.
- **Khắc phục chuẩn:** Bắt buộc filter đồng thời: `WHERE created_at < ? AND started_at < ?`. Laya Router đã thẩm định và đồng thuận: **AGREE 58.9%**.

---

## 2. Lỗ hổng 2: Lỗi GORM Missing Where Clause & Nguy cơ Xóa Sạch Toàn bộ Hệ thống
- **Lỗ hổng:** Nếu API nhận tham số rỗng mà gọi `.Delete()`, GORM sẽ ném lỗi `gorm: missing WHERE conditions` (500) hoặc xóa sạch toàn bộ log của mọi pipeline.
- **Khắc phục chuẩn:** Cài đặt **Guard Clause** nghiêm ngặt ở cả Persistence và HTTP Handler: bắt buộc phải có ít nhất 1 điều kiện lọc (`TargetTable`, `MasterTable`, `BeforeDate`, `FromDate`). Laya Router đã thẩm định và đồng thuận cao: **AGREE 83.13%** (Confidence 0.3453).

---

## 3. Lỗ hổng 3: Cross-Tenant Blast Radius khi dùng `LIKE %.table`
- **Lỗ hổng:** Ban đầu đề xuất dùng `target_table LIKE %.table` để xóa cả schema prefix. Tuy nhiên, trong hệ thống microservices/multi-tenant, nhiều service có thể có bảng trùng tên (ví dụ: `payment_bills`, `orders`). Câu lệnh `LIKE %.table` sẽ xóa nhầm log của các service khác! Laya Router đã phản đối: **DISAGREE 68.89%**.
- **Khắc phục chuẩn:** Frontend gửi chính xác danh sách FQN (`shadowTable` và `masterTable`), Backend dùng mệnh đề so khớp chính xác: `WHERE target_table IN (?)`. Triệt tiêu hoàn toàn rủi ro xóa lan (Zero Blast Radius).

---

## 4. Lỗ hổng 4: Over-Engineering Vi phạm Nguyên tắc Simplicity First & Minimal Impact
- **Lỗ hổng:** Đề xuất ban đầu vẽ thêm `ActivityLogAdminRepo`, tạo Command bus `ClearActivityLogsHandler`, và thay đổi DI wiring trong `server.go` (tạo thêm 3-4 files mới).
- **Khắc phục chuẩn:** Tuân thủ nguyên tắc cốt lõi của hệ thống: Simplicity First, Minimal Impact. Bổ sung trực tiếp `ClearActivity` vào struct `ActivityLogReadRepo` hiện tại (vốn nắm giữ DB session của `cdc_activity_log`), gọi trực tiếp từ `ActivityLogHandler.Clear`. Tiết kiệm hàng trăm dòng code thừa và giữ cho codebase cực kỳ tinh gọn. Laya Router đã đồng thuận: **AGREE 59.64%**.

---

## 5. Lỗ hổng 5: Semantic Ambiguity giữa Batch Snapshot và Continuous CDC Stream
- **Lỗ hổng:** Log của Snapshot nạp mẻ (`buffer_type: 'snapshot'`, `operation: 'snapshot.v2'`) nếu bị cộng lẫn vào cột DB Shadow hoặc Master sẽ làm sai lệch đối soát luồng Kafka CDC. Ngày 09/10 của `export_jobs` Kafka CDC = 0 nhưng Shadow lại hiện 539, Master hiện 539, tạo ra báo cáo sai sự thật.
- **Khắc phục chuẩn:** Tách riêng Khối 1: `1. SNAPSHOT / RE-SYNC` (Shadow Nạp, Master Sync) và Khối 2-4: `KAFKA CDC BẮN VỀ`, `GHI VÀO DB SHADOW (Từ CDC Stream)`, `GHI VÀO DB MASTER (Từ CDC Stream)`. Luồng Snapshot được ghi nhận trung thực, luồng CDC stream được đối soát chính xác 100%.

---

## 6. Lỗ hổng 6: Hardcode Mốc Ngày Cố định Thay vì DatePicker Động
- **Lỗ hổng:** Hardcode mốc ngày "07/10/2026" là tư duy cục bộ, cứng nhắc; trên Production hay Staging, thời điểm deploy và chu kỳ dọn dẹp là hoàn toàn khác nhau.
- **Khắc phục chuẩn:** Sử dụng Ant Design `DatePicker` động, cho phép người vận hành chọn ngày bất kỳ (gợi ý mặc định 7 ngày trước) hoặc chọn chế độ xóa toàn bộ nhật ký của bảng. Laya Router đã thẩm định và đồng thuận cao: **AGREE 68.31%**.


---
## [2026-10-12] Audit V4 — Gap bổ sung (bằng chứng: `13_analysis_audit_v4_clear_and_grid.md`)
- GAP-V4-1 (CRITICAL): Writer activity log Shadow không ghi định danh pipeline → cách ly multi-connector ở CMS vô hiệu.
- GAP-V4-2 (CRITICAL): `DELETE /activity-log` nằm ngoài `destructiveChain`.
- GAP-V4-3 (CRITICAL): Không có dấu snapshot trên chuỗi trigger transmute → Khối Snapshot phía Master luôn 0.
- GAP-V4-4 (HIGH): Debouncer key = master table nhưng dùng binding/trigger của `batch[0]` (ngoài phạm vi log, ảnh hưởng tính đúng của transmute khi 2 binding chung master).
- GAP-V4-5 (HIGH): Bảng ngày tính client-side trên ≤200 log, bucket UTC, bỏ `transmute_sub`.
- GAP-V4-6 (LOW/ghi nhận): Khối "KAFKA" suy ra từ log Shadow, FSM `kafka==shadow` là tautology — chưa có nguồn đo Kafka độc lập.
