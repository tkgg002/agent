# HỒ SƠ GIẢI PHÁP KỸ THUẬT: CHUẨN HÓA VĂN PHONG BÁO CÁO CONFLUENCE DÀNH CHO NGƯỜI ĐỌC (READER-FRIENDLY REPORT FORMAT & COMMENTED TECH VERSION)

- **Ngày lập:** 2026-10-15
- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`
- **File mục tiêu:**
  1. [`centralized-data-service/internal/service/confluence/confluence_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go)
  2. [`centralized-data-service/internal/service/confluence/confluence_runner_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner_test.go)

---

## 1. Bối cảnh & Yêu cầu từ User
User đã trực tiếp chỉnh sửa [`confluence_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go) và chỉ rõ định hướng sản phẩm:
> *"em đọc lại file `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go` a có edit text cho chuẩn, em phải hiểu đây là báo cáo cho người khác đọc, không phải cho dev đọc, họ sẽ ko quan tâm shadow và cả `[traitestces]` hay `centrallized-export-service`. em commnet lại cái ver trước (kiểu tech) + làm thêm 1 ver văn phong ok hơn giúp anh."*

### Phân tích các điểm cần chuẩn hóa trong `confluence_runner.go`:
1. **Tách bạch Tầng Truy Vết Kỹ Thuật (Internal Lineage) và Tầng Trình Bày (Reader Presentation):**
   - **Ngầm bên dưới (Backend Query):** Vẫn giữ nguyên 100% logic `resolveBranchLineage` và `lookupLayerDailyCount` theo đúng bộ khóa 5 thành phần `(report_date, layer_type, connection_code, container_namespace, object_name)` để lấy chính xác số đếm của đúng nhánh `Source ➔ Shadow ➔ Master` đã chọn.
   - **Hiển thị trên Báo cáo Confluence (Reader View):**
     - **Ẩn hoàn toàn tầng trung gian `Shadow`** và lưu lượng pipeline nội bộ (`Kafka / Shadow / Master`).
     - **Ẩn hoàn toàn mã kết nối `[traitestces]`, `[master_2]`, `[default_master]` và tiền tố database/schema `centrallized-export-service.`, `master_centrallized_export_service.`:** Chỉ hiển thị tên bảng/đối tượng gọn gàng (`SourceCleanName = "export-jobs"`, `MasterCleanName = "export_jobs_2"`).
     - **Đối chiếu trực tiếp `Source ↔ Master` (`hasSrc && hasMst`, `srcCount == mstCount`):** Loại bỏ việc kiểm tra `hasShd` và `shdCount` ra khỏi điều kiện đánh giá `Status` của bản Reader (tránh lỗi: dù không hiển thị cột Shadow nhưng nếu Shadow chưa đếm lại báo `"Thiếu: Shadow"`).
2. **Lưu trữ đầy đủ các phiên bản ngay trong `confluence_runner.go`:**
   - **Comment lại Ver Tech (Kiểu Tech 3 trạm `Source ➔ Shadow ➔ Master` dành cho Dev):** Giữ nguyên khối code đánh giá 3 trạm và hàm `buildConfluenceXHTMLTech` trong comment để khi cần debug kỹ thuật có thể bật lại ngay.
   - **Comment giữ lại Ver Rút gọn anh vừa chỉnh sửa** để dễ đối chiếu.
   - **Kích hoạt Ver Văn Phong Báo Cáo Cho Người Đọc (Reader-Friendly Version):**
     - Ngôn ngữ tự nhiên, lịch sự, rõ ràng cho người đọc nghiệp vụ/quản lý.
     - Định dạng số có dấu phẩy phân cách hàng nghìn (`formatNumberWithCommas`, ví dụ `1,500`) giúp dễ đọc với dữ liệu lớn.

---

## 2. Thiết kế So sánh 2 Phiên bản (Commented Tech Version vs Active Reader-Friendly Version)

### 2.1. Phiên bản Văn phong cho Người đọc (Active trong `confluence_runner.go`)
- **Tiêu đề & Mở đầu:**
  - `<h2>Báo Cáo Đối Soát Dữ Liệu Hằng Ngày (%s)</h2>`
  - `<p>Báo cáo tự động tổng hợp và đối chiếu số lượng bản ghi phát sinh trong ngày giữa <strong>Dữ liệu Nguồn (Source)</strong> và <strong>Dữ liệu Đích (Master)</strong>.</p>`
- **Cấu trúc bảng 5 cột (Không có Shadow, không có `[connection_code]` hay `schema.`):**

| Bảng dữ liệu Nguồn (Source) | Bảng dữ liệu Đích (Master) | Số lượng phát sinh tại Nguồn | Số lượng đã đồng bộ sang Đích | Kết quả đối soát |
| :--- | :--- | ---: | ---: | :--- |
| **export-jobs** | **export_jobs_2** | 0 | 0 | 🟢 Đồng bộ đầy đủ (Khớp 100%) |

- **Văn phong cột `Kết quả đối soát` (`row.Status`):**
  - Khi Nguồn và Đích khớp 100% (`srcCount == mstCount`):
    `🟢 Đồng bộ đầy đủ (Khớp 100%)`
  - Khi Nguồn > Đích (`srcCount > mstCount`):
    `🔴 Chênh lệch số liệu (Đích thiếu X bản ghi so với Nguồn — Nguồn: A | Đích: B)`
  - Khi Đích > Nguồn (`mstCount > srcCount`):
    `🔴 Chênh lệch số liệu (Đích nhiều hơn Nguồn X bản ghi — Nguồn: A | Đích: B)`
  - Khi cả 2 bên chưa chạy kiểm đếm (`!hasSrc && !hasMst`):
    `⚪ Chưa có số liệu kiểm đếm trong ngày`
  - Khi thiếu 1 bên (`!hasSrc` hoặc `!hasMst`):
    `🟡 Chờ cập nhật số liệu từ bảng Nguồn` / `🟡 Chờ cập nhật số liệu từ bảng Đích`
- **Giá trị ô số đếm khi chưa có dữ liệu:**
  - Hiển thị `"Chưa có số liệu"` (thay vì `"Chưa đếm"`).

### 2.2. Phiên bản Kiểu Tech (Được comment lại nguyên vẹn trong `confluence_runner.go`)
- Giữ nguyên logic đánh giá 3 trạm (`Source = Shadow = Master`), định danh `[connection_code] namespace.table`, và bảng 8 cột (`STT`, `Trạm Nguồn (Source)`, `Trạm Trung Gian (Shadow)`, `Trạm Đích (Master)`, `Số Đếm Ngày Tạo (Source | Shadow | Master)`, `Lưu Lượng CDC (Kafka / Shadow / Master)`, `Trạng Thái`, `Ghi Chú Kỹ Thuật`) ở dạng block comment rõ ràng.
