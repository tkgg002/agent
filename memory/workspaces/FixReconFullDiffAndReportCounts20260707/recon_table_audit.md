# Audit Report: Clean Up and Simplify `cdc_reconciliation_report` Table

Bảng `cdc_reconciliation_report` trong cơ sở dữ liệu `cdc_system` hiện tại đang chứa 36 cột. Qua phân tích mã nguồn ở cả hai dự án Backend (`centralized-data-service`, `cdc-cms-service`) và dự án Frontend (`cdc-cms-web`), chúng tôi đã xác định được các trường hợp ghi dữ liệu và các cột không sử dụng (redundant/unused fields) để đưa ra khuyến nghị làm gọn bảng này.

---

## 1. Các trường hợp ghi dữ liệu vào bảng (Write Path Audit)

Dữ liệu được ghi và cập nhật vào bảng này từ 3 luồng xử lý chính trong `centralized-data-service`:

1. **Segment A (Source ↔ Shadow) Checks** (tại `recon_tier_a.go`):
   - **`RunOrphanPrune`**: Ghi log dọn dẹp các bản ghi mồ côi (`check_type = "orphan_prune"`).
   - **`RunSmokeCheck`**: Ghi log khớp tổng số lượng ước lượng (`check_type = "count_total"`, `tier = 1`).
   - **`RunHashWindowCheck`**: Ghi log khớp băm trong khung thời gian (`check_type = "hash_window"`, `tier = 2`).
   - **`RunDeepCheck`**: Ghi log khớp chi tiết thuộc tính (`check_type = "deep_check"`, `tier = 3`).
   - **Error Handling**: Ghi nhận lỗi khi tiến trình kiểm tra thất bại (`error_message` và `error_code`).

2. **Segment B (Shadow ↔ Master) Checks** (tại `recon_tier_b.go`):
   - **`RunSmokeCheckB`**: Ghi log khớp tổng số lượng (`check_type = "smoke"`, `tier = 1`).
   - **`RunHashWindowCheckB`**: Ghi log khớp băm trong khung thời gian (`check_type = "hash_window"`, `tier = 2`).
   - **`RunDeepCheckB`**: Ghi log khớp chi tiết thuộc tính (`check_type = "deep_check"`, `tier = 3`).

3. **Heal Operations** (tại `recon_execute_heal_handler.go`):
   - Khi hoàn thành chữa lành, cập nhật bản ghi đối soát hiện tại thông qua `UpdateByID`:
     - Cập nhật `status` từ `drift` thành `healed`.
     - Điền thời điểm hoàn thành (`healed_at`).
     - Điền số lượng đã sửa (`healed_count`, `healed_mismatched_count`, `healed_missing_dest_count`, `pruned_missing_src_count`).
     - Điền thời gian xử lý (`healed_duration_ms`, `healed_mismatched_duration_ms`, `healed_missing_dest_duration_ms`, `pruned_missing_src_duration_ms`).

---

## 2. Phân tích chi tiết các cột và khuyến nghị (Column-by-Column Audit)

### Nhóm A: Các cột CỐT LÕI (Bắt buộc giữ lại)
Các cột này được đọc/ghi thường xuyên bởi cả backend và frontend để định danh pipeline, hiển thị biểu đồ và chạy logic vận hành:
* `id`, `segment`, `shadow_schema`, `shadow_table`, `master_schema`, `master_table`, `run_id`, `check_type`, `status`, `checked_at`.
* `source_db`, `source_count`, `dest_count`, `diff`, `total_source_count`, `total_dest_count`, `duration_ms`.
* `missing_count`, `missing_ids`, `stale_count`, `stale_ids`, `orphan_count`, `field_diffs`.
* `error_message`, `error_code`.

---

### Nhóm B: Các cột DƯ THỪA / KHÔNG SỬ DỤNG (Khuyến nghị loại bỏ hoặc gộp)

Dưới đây là danh sách các cột thực tế không có bất kỳ logic đọc trực tiếp nào của các phiên check, hoặc rải rác không có thông tin và nên được gộp lại:

#### 1. Các cột thống kê Chữa lành (Heal) cũ và mới
* **Thực trạng**: Các cột liên quan đến chữa lành (gồm 9 cột: `healed_at`, `healed_count`, `healed_duration_ms`, `healed_mismatched_count`, `healed_mismatched_duration_ms`, `healed_missing_dest_count`, `healed_missing_dest_duration_ms`, `pruned_missing_src_count`, `pruned_missing_src_duration_ms`) đều rỗng (0/NULL) ở các bản ghi check thông thường và chỉ có thông tin rải rác khi heal thành công.
* **Khuyến nghị**: **Giữ nguyên trạng thái phân rã cột rõ ràng** (không gộp chung vào JSONB) để phục vụ cho các nghiệp vụ truy vấn tường minh, audit, và đối soát báo cáo chi tiết trong tương lai theo đúng yêu cầu nghiệp vụ.

#### 2. Cột `target_table` (Bare name)
* **Thực trạng**: Cột này lưu tên bảng trần (không kèm schema). Kể từ Migration 085, hệ thống đã chuẩn hóa việc định danh pipeline bằng cặp `(shadow_schema, shadow_table)` và `(master_schema, master_table)` để tránh xung đột tên bảng trùng nhau ở các schema khác nhau.
* **Khuyến nghị**: Có thể loại bỏ cột `target_table` sau khi refactor các câu truy vấn read-side trong `cdc-cms-service` để lấy trực tiếp `shadow_table` hoặc `master_table` tùy theo segment. Hiện tại, nó chỉ đóng vai trò tương thích ngược.

#### 3. Cột `tier` (int)
* **Thực trạng**: Cột này lưu phân tầng đối soát (1, 2, 3). Tuy nhiên, cột `check_type` đã lưu rõ bản chất quét: `smoke`/`count_total` (Tier 1), `hash_window` (Tier 2), và `deep_check` (Tier 3).
* **Khuyến nghị**: Khai tử cột `tier`. Phía frontend có thể suy luận trực tiếp từ `check_type` để gán nhãn Tier (ví dụ: `hash_window` tương đương Tier 2), giảm bớt một cột lưu trữ trùng lặp thông tin.

---

## 3. Đề xuất bổ sung: Cột lưu vết Khoảng thời gian đối soát (Reconciliation Time Range)

* **Vấn đề hiện tại**: Khi chạy đối soát theo khung thời gian (ví dụ: Hot mode 2 giờ, Cold mode 7 ngày, hoặc Custom range tự chọn), bảng `cdc_reconciliation_report` hiện tại **không hề lưu lại thông tin khoảng thời gian bắt đầu và kết thúc của phiên quét đó**. Người vận hành chỉ biết phiên quét chạy vào lúc nào (`checked_at`), nhưng không thể biết phiên đó quét cho khoảng dữ liệu nào.
* **Đề xuất**: Bổ sung thêm 2 cột mới vào bảng `cdc_reconciliation_report`:
  - `recon_start_time` (timestamp without time zone, nullable)
  - `recon_end_time` (timestamp without time zone, nullable)
* **Mục tiêu**: Ghi nhận chính xác `StartTime` và `EndTime` của khung thời gian đối soát (đặc biệt hữu dụng cho `hash_window` check). Giao diện frontend sẽ đọc 2 trường này để hiển thị rõ ràng khung giờ đối soát của từng phiên trong nhật ký đối soát.

---

## 4. Kế hoạch tinh gọn và nâng cấp bảng (Proposed Action Plan)

Để thực hiện tinh gọn và nâng cấp bảng mà không gây gián đoạn hệ thống, chúng ta có thể thực hiện theo lộ trình:

1. **Bước 1: Thiết kế migration cơ sở dữ liệu**:
   - Viết SQL migration:
     - Thêm cột `healed_details JSONB`.
     - Thêm cột `recon_start_time TIMESTAMP` và `recon_end_time TIMESTAMP`.
2. **Bước 2: Cập nhật Models & Logic xử lý ở Backend**:
   - Cập nhật struct `ReconciliationReport` ở cả hai dự án (`centralized-data-service` và `cdc-cms-service`).
   - Sửa logic `finalizeReport` trong `recon_execute_heal_handler.go` để đóng gói dữ liệu chi tiết vào `healed_details`.
   - Cập nhật các hàm băm đối soát (`RunHashWindowCheck` / `RunHashWindowCheckB`) để gán giá trị `recon_start_time` và `recon_end_time` lấy từ Context.
3. **Bước 3: Loại bỏ các cột dư thừa cũ**:
   - Chạy lệnh `ALTER TABLE ... DROP COLUMN ...` loại bỏ các cột heal chi tiết cũ, cột `tier`, và cột `target_table`.
   - Dọn dẹp triệt để các câu truy vấn SQL trong CMS và frontend.

---

## 5. Kiểm tra chi tiết từng Field (Field-by-Field Detailed Audit)

Dưới đây là kết quả rà soát chi tiết về nguồn ghi (Write Path), khả năng rỗng (Nullability), và mục đích sử dụng của từng trường trong danh sách anh yêu cầu:

| Tên Field | Kiểu DB | Có thể Rỗng (NULL / 0) | Nguồn ghi & Cách hoạt động | Phân tích thực tế |
| :--- | :--- | :--- | :--- | :--- |
| `id` | BIGSERIAL | **Không** | Tự động sinh bởi DB PostgreSQL. | Khóa chính bắt buộc của bảng. |
| `segment` | VARCHAR | **Không** | Do `stampA` (`source_shadow`) hoặc `stampB` (`shadow_master`) gán cứng. | Dùng phân tách 2 chặng đối soát. Luôn có giá trị. |
| `shadow_schema` | VARCHAR | **Không** | Do `stampA` / `stampB` gán từ Table Registry / Master Binding. | Định danh duy nhất pipeline cùng với `shadow_table`. |
| `shadow_table` | VARCHAR | **Không** | Do `stampA` / `stampB` gán từ Table Registry / Master Binding. | Định danh duy nhất pipeline cùng với `shadow_schema`. |
| `master_schema` | VARCHAR | **Có (ở chặng A)** | Chỉ được điền ở chặng B (`stampB`), chặng A để NULL. | Xác định schema đích của Master trong Segment B. |
| `master_table` | VARCHAR | **Có (ở chặng A)** | Chỉ được điền ở chặng B (`stampB`), chặng A để NULL. | Xác định table đích của Master trong Segment B. |
| `run_id` | VARCHAR | **Không** | Sinh ngẫu nhiên (UUID) ở đầu mỗi phiên Check. | Dùng để gom nhóm các pipeline chạy trong cùng 1 đợt. |
| `check_type` | VARCHAR | **Không** | Gán cứng theo loại quét (`smoke`, `hash_window`, `deep_check`). | Giúp phân biệt hình thức quét. |
| `status` | VARCHAR | **Không** | Trạng thái đối soát (`ok`, `drift`, `error`). | Kết quả chung của phiên quét. |
| `checked_at` | TIMESTAMP | **Không** | `time.Now().UTC()` tại thời điểm lưu report. | Thời gian chạy đối soát. |
| `source_db` | VARCHAR | **Không** | Tên DB nguồn (ví dụ: `mongo_db` cho chặng A, hoặc shadow DB cho chặng B). | Xác định DB nguồn quét. |
| `source_count` | BIGINT | **Có (khi lỗi)** | Số lượng bản ghi đếm được ở Nguồn trong window. | Sẽ bị **NULL** khi DB Nguồn bị ngắt kết nối hoặc truy vấn lỗi. Tránh ghi `0` giả làm sai lệch báo cáo. |
| `dest_count` | BIGINT | **Không** | Số lượng bản ghi đếm được ở Đích trong window. | Luôn ghi nhận số lượng quét được (mặc định 0). |
| `diff` | BIGINT | **Không** | Công thức: `source_count - dest_count`. | Độ chênh lệch trong window. |
| `total_source_count` | BIGINT | **Có (khi lỗi)** | Tổng số lượng thực tế của Collection nguồn. | Sẽ bị **NULL** khi truy vấn lỗi hoặc ở các report cũ. |
| `total_dest_count` | BIGINT | **Có (khi lỗi)** | Tổng số lượng thực tế của Table đích. | Sẽ bị **NULL** khi truy vấn lỗi hoặc ở các report cũ. |
| `duration_ms` | INTEGER | **Không** | Thời gian chạy của riêng phiên check đó (ms). | Latency của truy vấn đối soát. |
| `missing_count` | INTEGER | **Có (khi khớp/smoke)**| Số lượng ID ở nguồn không tồn tại ở đích. | Bằng `0` khi khớp hoặc khi chạy `smoke` check (chỉ đếm COUNT, không quét ID). |
| `missing_ids` | JSONB | **Có (khi khớp/smoke)**| Danh sách JSON array chứa các ID bị thiếu. | Bằng **NULL** khi khớp hoặc khi chạy `smoke` check. |
| `stale_count` | INTEGER | **Có (khi khớp/smoke)**| Số lượng bản ghi bị mismatch hoặc mồ côi. | Bằng `0` khi khớp hoặc khi chạy `smoke` check. |
| `stale_ids` | JSONB | **Có (khi khớp/smoke)**| Chi tiết các ID bị mismatch hoặc mồ côi. | Bằng **NULL** khi khớp hoặc khi chạy `smoke` check. |
| `orphan_count` | INTEGER | **Có (khi khớp/smoke)**| Số lượng ID ở đích không tồn tại ở nguồn. | Bằng `0` khi khớp hoặc khi chạy `smoke` check. |
| `field_diffs` | JSONB | **Có (khi khớp/hash)** | Chi tiết các field bị lệch giá trị. | Bằng **NULL** đối với 99% các phiên chạy `smoke` và `hash_window`. Chỉ có thông tin khi chạy `deep_check`. |
| `error_message` | TEXT | **Có (khi thành công)**| Thông báo lỗi chi tiết khi check bị crash/timeout. | Bằng **NULL** đối với 99% các phiên chạy thành công. |
| `error_code` | VARCHAR | **Có (khi thành công)**| Mã lỗi phân loại (ví dụ: `SRC_TIMEOUT`). | Bằng **NULL** đối với 99% các phiên chạy thành công. |
| `healed_at` | TIMESTAMP | **Có (khi check)** | Thời điểm hoàn thành Chữa lành (Heal). | Bằng **NULL** đối với 100% các phiên Check. Chỉ ghi nhận khi chạy Heal thành công. |
| `healed_count` | INTEGER | **Có (khi check)** | Số lượng bản ghi đã được sửa lỗi thành công. | Bằng `0` đối với 100% các phiên Check. Chỉ ghi nhận khi chạy Heal thành công. |

---

## 6. Phân tích Trùng lặp Logic Smoke Check (`RunSmokeCheck` và `cdc_recon_smoke_result`)

Qua rà soát mã nguồn Go của dự án `centralized-data-service`, em phát hiện sự tương quan và trùng lặp logic của Smoke Check như sau:

### 1. Phân biệt đích ghi của 2 luồng Smoke Check:
* **Luồng chạy tự động theo lịch (Background Scheduled Job):**
  - Hàm điều phối chính là **`CheckAllUnified`** (trong `recon_smoke.go`, được gọi bởi `server_jobs.go`).
  - Hàm thực thi chính: **`RunTotalOnlyA`** (Segment A) và **`RunTotalOnlyB`** (Segment B).
  - **Đích ghi:** Lưu dữ liệu vào bảng riêng biệt **`cdc_system.cdc_recon_smoke_result`**.
* **Luồng chạy thủ công khi người dùng bấm trên giao diện (Manual Check Trigger):**
  - Hàm thực thi chính: **`RunSmokeCheck`** (trong `recon_tier_a.go`) và **`RunSmokeCheckB`** (trong `recon_tier_b.go`).
  - **Đích ghi:** Lưu dữ liệu vào bảng nhật ký chính **`cdc_system.cdc_reconciliation_report`** với `check_type = "count_total"`.
  - **Phân tích khả năng kích hoạt thực tế từ UI:**
    1. **Nút "Bắt đầu đối soát" (Check All - Quét toàn bộ bảng):** Bị comment ẩn hoàn toàn trên giao diện `DataIntegrity.tsx` (dòng 865 - 900). Người dùng không thể click kích hoạt luồng này.
    2. **Nút "Đối soát thủ công" từng bảng:** Khi click sẽ mở modal `ConfirmDestructiveModal` với prop `isManualRecon = true`.
    3. Giao diện modal này **chỉ hiển thị 4 tùy chọn**: `2h` (Hot mode), `7d` (Cold mode), `custom` (Custom range), và `deep` (Deep check). **Hoàn toàn không có nút bấm hay tùy chọn nào cho phép chạy "Smoke Check" (kiểm tra ước lượng tổng số lượng)**.
    4. Do đó, tham số `type_recon` truyền lên API từ Drawer detail của Frontend không bao giờ là `"smoke"` cho table check đơn lẻ.
  - **Kết luận:** Hàm `RunSmokeCheck` và `RunSmokeCheckB` ở Backend hiện tại là **mã nguồn dư thừa (dead code)**, hoàn toàn không có bất kỳ luồng bấm trực tiếp nào từ UI chạm tới chúng nữa.

### 2. Logic có bị trùng lặp (Duplicate) hay không?
* **Về mặt Logic nghiệp vụ: CÓ BỊ TRÙNG LẶP.**
  Cả `RunSmokeCheck` (Tier 1) và `RunTotalOnlyA` đều thực hiện các bước giống hệt nhau:
  1. Lock bảng đối soát để tránh chạy đè.
  2. Đo `ingestLagMs` và kích hoạt Circuit Breaker nếu lag quá cao.
  3. Lấy số lượng ước lượng nguồn (`EstimatedCount`/`CountDocuments`).
  4. Lấy số lượng đích (`EstimatedCountRows`/`CountRows`) và trừ đi soft-deleted rows.
  5. So sánh chênh lệch (`diff`) và kiểm tra xem có vượt ngưỡng Tolerance hay không để kết luận trạng thái (`ok` hoặc `drift`).
* **Về mặt thiết kế:**
  - Sở dĩ có sự phân tách làm 2 bảng này là do bảng `cdc_recon_smoke_result` được thiết kế dẹt (flattened) và tối ưu để lưu trữ dữ liệu thống kê quy mô lớn của các phiên quét tự động định kỳ (chứa đầy đủ thông tin Host, DB, Table, Total, Active của cả 3 trạm Source, Shadow, Master trên cùng một dòng).
  - Còn bảng `cdc_reconciliation_report` là sổ cái nhật ký vận hành dùng chung cho tất cả các loại quét (Smoke, Hash Window, Deep Check) khi người dùng bấm nút trực tiếp. Giao diện Frontend hiển thị lịch sử đối soát thông qua câu lệnh `UNION ALL` gộp dữ liệu từ cả 2 bảng này để người dùng thấy đầy đủ cả phiên chạy tự động lẫn phiên bấm tay.

### 3. Khuyến nghị:
* Do giao diện Frontend đã đóng hoàn toàn nút bấm trigger Smoke check thủ công và "Check All":
  - **Hàm `RunSmokeCheck` và `RunSmokeCheckB` có thể cân nhắc khai tử (delete)** ở các task dọn dẹp tiếp theo để giảm độ phức tạp của codebase.
  - Bảng `cdc_reconciliation_report` sẽ không cần chứa các dòng kết quả của `check_type = count_total` (chỉ cần nhận dữ liệu từ các phiên check time-bounded `hash_window` và `deep_check`).
