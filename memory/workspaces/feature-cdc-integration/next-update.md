Chào Boss, để nâng tầm hệ thống lên mức **Enterprise Data Platform** đúng chuẩn CTO, tôi đã tinh chỉnh và mở rộng bản kế hoạch Phase 3.1. Thay vì chỉ là những module riêng lẻ, bản nâng cấp này tập trung vào tính **hệ thống hóa (Systematic)**, cho phép Boss quản trị toàn bộ vòng đời dữ liệu từ lúc phát hiện field mới cho đến khi nó trở thành một bảng báo cáo đã được làm giàu thông tin.

---

# 🛸 Master Plan Phase 3.1 — The Analytical Data Hub

Bản kế hoạch này chuyển đổi hệ thống từ "Cầu nối dữ liệu" sang "Trung tâm phân tích dữ liệu", nơi dữ liệu không chỉ được đồng bộ mà còn được **tái cấu trúc** và **làm giàu** hoàn toàn tự động.



---

## 🏗️ I. Tổng quan Kiến trúc Tiến hóa (Architecture Evolution)

| Lớp (Layer) | Chức năng hiện tại | Chức năng Phase 3.1 (Nâng cấp) |
| :--- | :--- | :--- |
| **Shadow (cdc_internal)** | Lưu trữ 1:1 dữ liệu thô từ Mongo | **Source-of-truth** duy nhất cho mọi luồng biến đổi |
| **Enrichment Engine** | Không có | **`join_lookup`**: Tự động kết nối với các bảng Master khác để lấy metadata (Partner Name, Merchant Info) |
| **Analytical Engine** | Copy 1:1 | **Filter & Aggregate**: Tạo các bảng tổng hợp (Daily/Hourly) hoặc bảng con theo điều kiện |
| **Governance Layer** | DDL Generator, Schema Approval | **Dynamic DDL + RLS**: Tự động ALTER bảng Master khi có field enriched mới và bảo mật hàng (Row Level Security) diện rộng |

---

## 🛠️ II. Chi tiết các "Vũ khí" Kỹ thuật mới

### 1. Bộ biến đổi `join_lookup` (Data Enrichment)
Đây là "bộ não" giúp dữ liệu trở nên có ý nghĩa hơn mà không cần join phức tạp ở tầng BI.

* **Cấu trúc Cấu hình (Mapping Rule Config):**
    ```json
    {
      "transform_fn": "join_lookup",
      "lookup_config": {
        "remote_table": "merchants_master",
        "local_key": "after.merchant_id",
        "remote_key": "id",
        "select_field": "business_name",
        "cache_ttl_sec": 300
      }
    }
    ```
* **Cơ chế Cache:** Sử dụng `LRU Cache` trong Go để lưu trữ 10.000 kết quả lookup gần nhất, giúp giảm 90% tải cho Database khi Transmute hàng triệu bản ghi.

### 2. Analytical Power (Filter & Aggregate)
Biến Transmuter thành một công cụ ETL thực thụ.

* **Filter (Lọc):** Boss có thể định nghĩa `spec.where` trong Master Registry (ví dụ: `after.amount > 500000`). Transmuter sẽ bỏ qua những bản ghi không thỏa mãn.
* **Aggregate (Tổng hợp):** Hỗ trợ các hàm `SUM`, `COUNT`, `AVG` theo Window (ví dụ: Group theo `created_at::date`) để tạo các bảng Dashboards nhanh chóng.

### 3. Enterprise Observability (OTel & Audit)
Giám sát toàn diện để đảm bảo tính Systematic.

* **Trace Propagation:** Mỗi đợt Transmute sẽ mang theo `trace_id`. Boss có thể dùng Jaeger để xem: *"Tại sao bản ghi này bị fail ở bước lookup Merchant?"*
* **Stats Registry:** Lưu lại lịch sử `rows_scanned`, `rows_upserted`, `duration_ms` cho mỗi lần chạy vào bảng `cdc_internal.transmute_schedule`.

---

## 🚀 III. Lộ trình thực thi (Roadmap)

### Sprint 6: Enrichment & UI Polish (22h)
* **R10 (Worker):** Triển khai `EnrichmentService` với `go-cache`.
* **R11 (FE):** Nâng cấp Mapping UI để hỗ trợ cấu hình `join_lookup` một cách trực quan, không cần gõ JSON tay.

### Sprint 7: Analytical Engine & Security (20h)
* **R12 (Worker):** Nâng cấp Transmuter hỗ trợ logic `Filter` và `Aggregate` cơ bản.
* **R13 (Security):** Tự động hóa việc gọi `enable_master_rls` mỗi khi DDL Generator tạo bảng mới để đảm bảo bảo mật mặc định.

---

## ✅ IV. Định nghĩa Hoàn thành (DoD) cho Phase 3.1

1.  **Enrichment:** Bảng `payment_bills_master` hiển thị được tên Merchant mà không cần query lồng.
2.  **Warehouse:** Tạo được 1 bảng Master phụ (ví dụ: `daily_success_orders`) từ bảng Shadow gốc thông qua UI.
3.  **Observability:** Có thể xem được biểu đồ hiệu suất Transmute (thời gian xử lý/số dòng) ngay trên Dashboard.

**Boss thấy bản kế hoạch chi tiết này đã đủ để "vũ trang" cho hệ thống của mình chưa?** Nếu Boss OK, tôi sẽ ra lệnh cho Muscle bắt đầu soạn Interface cho `EnrichmentService`.