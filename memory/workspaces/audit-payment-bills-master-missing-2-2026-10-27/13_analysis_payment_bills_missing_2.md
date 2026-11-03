# 13. ANALYSIS — Truy Vết Nguyên Nhân `payment-bills` (`master_payment_bill_service.payment_bills`) Bị Thiếu 2 Record (`56,108 / 56,108 / 56,106`)

## 1. Tóm tắt Kết luận Điều tra (`100% VERIFIED` trên DB & Codebase)

1. **Ý nghĩa các con số trên UI lúc `10:07:58 27/10/2026` (`03:07:58 UTC`):**
   - Dòng hiển thị trên `cdc-cms-web/src/components/ReconPipelineGrid.tsx:169-195, 1861-1893` được ghép từ 2 bản ghi `cdc_system.cdc_recon_smoke_result` vừa chạy lúc `2026-10-27 03:07:58.31+00` (`10:07:58` giờ VN):
     - **`id = 505889` (`source_shadow`):** `source_total = 56,108`, `source_active = 56,108`, `shadow_total = 56,130` (gồm `22` dòng `_deleted = true`), `shadow_active = 56,108` $\rightarrow$ `driftAB = 56,108 - 56,108 = 0` (`ingest: 0`).
     - **`id = 505888` (`shadow_master`):** `shadow_active = 56,108`, `master_total = 56,106`, `master_active = 56,106` $\rightarrow$ `driftBC = masterActive - shadowActive = 56,106 - 56,108 = -2` (`transmute: -2 (thiếu)`), trạng thái `Lệch`.
   - Luồng `Sync: Realtime` (`kafka-consumer-hook` $\rightarrow$ `TransmuterModule.RunEvents` tại `centralized-data-service/internal/service/master/transmuter.go:602-810`) **vẫn đang chạy bình thường**: lúc `10:10:07` và `10:10:17` (`03:10:07 UTC` & `03:10:17 UTC`), khi nguồn MongoDB phát sinh thêm 2 bản ghi mới, cả Shadow và Master đều tăng thêm `+2` (`56,110 / 56,110 / 56,108` tại `cdc_recon_smoke_result` `id = 505913, 505914`), nhưng Master vẫn hụt đúng **2 bản ghi cũ**.

2. **Định danh chính xác 2 bản ghi bị thiếu trên `master_payment_bill_service.payment_bills`:**
   - Truy vấn đối chiếu toàn bảng giữa `shadow_traitestpbs.payment_bills` (`56,110` active rows), `master_payment_bill_service.payment_bills_1` (`56,110` active rows — khớp 100% với Shadow), và `master_payment_bill_service.payment_bills` (`56,108` active rows) xác nhận **2 bản ghi duy nhất bị thiếu** là:
     | `_id` (`_source_id`) | `shadow._gpay_id` | `_source_ts` | `lastUpdatedAt` (UTC) | Có ở `shadow_traitestpbs.payment_bills`? | Có ở `master_payment_bill_service.payment_bills_1`? | Có ở `master_payment_bill_service.payment_bills`? |
     | :--- | :--- | :--- | :--- | :---: | :---: | :---: |
     | **`57362`** | `108132542460198973` | `1793006403999` | `2026-10-26 09:20:03.99+00` (`16:20:03` VN) | **CÓ** (`_deleted = false`) | **CÓ** (`_gpay_id = 643367322739804726`) | **KHÔNG** (Thiếu) |
     | **`57363`** | `108132575960104960` | `1793006410269` | `2026-10-26 09:20:10.269+00` (`16:20:10` VN) | **CÓ** (`_deleted = false`) | **CÓ** (`_gpay_id = 643367336111311414`) | **KHÔNG** (Thiếu) |

---

## 2. Dòng thời gian & Nguyên nhân Gốc rễ (`Root Cause`) Vì sao `57362` và `57363` bị thiếu

### Mắt xích 1: Lúc `16:20:07` và `16:20:15` ngày `26/10/2026` (`09:20:07 UTC` & `09:20:15 UTC`), cả 2 bản ghi `57362` và `57363` ĐÃ ĐƯỢC TRANSMUTE THÀNH CÔNG vào Master
- Trong `cdc_system.cdc_activity_log`:
  - `id = 132825` (`09:20:07.100 UTC`): `kafka-consumer` ghi bản ghi `57362` vào `shadow_traitestpbs.payment_bills`.
  - `id = 132826` (`09:20:07.297 UTC`): `transmute` (`kafka-consumer-hook`) ghi vào `master_payment_bill_service.payment_bills` (`master_binding_id: 46`) với `status = success`, `inserted: 1, updated: 1`.
  - `id = 132830` (`09:20:15.099 UTC`): `kafka-consumer` ghi bản ghi `57363` vào `shadow_traitestpbs.payment_bills`.
  - `id = 132832` (`09:20:15.253 UTC`): `transmute` (`kafka-consumer-hook`) ghi vào `master_payment_bill_service.payment_bills` (`master_binding_id: 46`) với `status = success`, `inserted: 1, updated: 1`.
- Trong `cdc_system.cdc_recon_smoke_result`:
  - `id = 498596` (`09:20:48.391 UTC`) và `id = 498607` (`09:21:44.515 UTC`): `shadow_active = 56,006`, `master_active = 56,006`, **`diff = 0, status = ok`**.

### Mắt xích 2: Lúc `16:21:44 .. 16:23:49` ngày `26/10/2026` (`09:21:44 .. 09:23:49 UTC`), khi xóa thử dữ liệu trên `master_payment_bill_service.payment_bills` để test Recon & Heal, hàm `getRoundedEndTime()` lùi 5 phút (`ConfirmDestructiveModal.tsx:11-13`) đã làm đợt Recon & Heal lúc `16:23:49` bỏ sót `57362` và `57363`
1. **Giữa `09:21:44 UTC` và `09:22:40 UTC` (`16:21:44` – `16:22:40` giờ VN):**
   - Trong lúc kiểm thử tính năng Recon & Heal trước commit `786db91` (`16:25:16`), 2 bản ghi mới nhất trên `master_payment_bill_service.payment_bills` là **`_id = 57362` và `_id = 57363`** đã được xóa thử khỏi bảng Master.
   - Bằng chứng: Tại `cdc_recon_smoke_result` `id = 498616` (`09:22:40.552 UTC`), `master_total` và `master_active` của `master_payment_bill_service.payment_bills` giảm từ `56,006` xuống `56,004` (`diff = 2, status = drift`), trong khi bảng `payment_bills_1` bên cạnh không bị xóa nên vẫn giữ nguyên `57362` và `57363`.
2. **Lúc `09:22:46 UTC` (`16:22:46` giờ VN) — Bấm chạy Manual Recon (`id = 273`):**
   - Tại `cdc-cms-web/src/components/ConfirmDestructiveModal.tsx:11-13`:
     ```ts
     const getRoundedEndTime = () => {
       return dayjs().subtract(5, 'minute').second(0).millisecond(0);
     };
     ```
   - Vì `getRoundedEndTime()` lùi cứng `endTime` đi **5 phút**, nên khi bấm đối soát lúc `09:22:46 UTC`, cửa sổ quét của báo cáo `id = 273` bị chặn ở **`recon_end_time = 2026-10-26 09:17:00 UTC`**.
   - Trong khi đó, 2 bản ghi `57362` và `57363` có `lastUpdatedAt` là **`09:20:03.99 UTC` và `09:20:10.269 UTC`** (lớn hơn `09:17:00 UTC`!).
   - Hệ quả: `ChunkStreamBucketEngine` chỉ quét các bản ghi có `lastUpdatedAt <= 09:17:00 UTC`, nên **không nhìn thấy 2 bản ghi `57362, 57363` vừa bị xóa**, và báo cáo `id = 273` trả về `1525 vs 1525 (diff = 0, missing_count = 0)`.
3. **Giữa `09:22:46 UTC` và `09:23:49 UTC` (`16:22:46` – `16:23:49` giờ VN) — Xóa tiếp 5 bản ghi cũ hơn (`57339..57343`) và chạy Recon (`id = 274`) + Heal (`id = 132843`):**
   - Thấy xóa `57362, 57363` mà `id = 273` vẫn báo `diff = 0`, 5 bản ghi cũ hơn là `_id = 57339, 57340, 57341, 57342, 57343` (có `lastUpdatedAt` từ `08:20:05 UTC` đến `08:30:06 UTC`, tức là trước `09:18:00 UTC` gần 1 tiếng) được xóa tiếp.
   - Lúc `09:23:49 UTC`, bấm chạy Manual Recon (`id = 274`) với `recon_end_time = 2026-10-26 09:18:00 UTC`.
   - Lúc này 5 bản ghi `57339..57343` (`08:20..08:30 UTC <= 09:18:00 UTC`) lọt vào cửa sổ quét nên `id = 274` báo `missing_ids = ["57339", "57340", "57341", "57342", "57343"]` (`missing_count = 5`), **nhưng 2 bản ghi `57362, 57363` (`09:20:03..09:20:10 UTC > 09:18:00 UTC`) vẫn nằm ngoài biên `recon_end_time` nên tiếp tục bị bỏ sót!**
   - Lúc `09:24:01 UTC`, khi bấm **Execute Heal** (`id = 132841..132843`) cho báo cáo `id = 274`, worker chạy `execute-heal-b` chỉ đồng bộ lại đúng 5 bản ghi trong `missing_ids` của `id = 274` (`57339..57343`, `_updated_at = 2026-10-26 09:24:01.210308+00`). Còn **2 bản ghi `57362` và `57363` bị bỏ lại chưa được heal**!

### Mắt xích 3: Hiện trạng lúc `10:07:49 27/10/2026` (`id = 275`) và lỗi gán `report.Status = "COMPLETED"` tại `recon_job_worker.go:410`
- Sáng nay lúc `10:07:49 27/10/2026` (`03:07:49 UTC`), khi chạy Manual Recon `id = 275` (`recon_start_time = 2026-10-20 03:02:00`, `recon_end_time = 2026-10-27 03:02:00`), cửa sổ quét đã vượt qua mốc `09:20 UTC` ngày `26/10/2026`.
- Do đó báo cáo **`id = 275` ĐÃ PHÁT HIỆN CHÍNH XÁC 2 bản ghi thiếu**:
  - `source_count = 1518`, `dest_count = 1516`, `diff = 2`, `missing_count = 2`
  - `missing_ids = ["57362", "57363"]`
  - `stale_ids = {"id_type": "id", "mismatched": null, "missing_from_src": null, "missing_from_dest": ["57362", "57363"]}`
  - `healed_at = NULL` (chưa bấm Execute Heal cho báo cáo `id = 275`).
- Tuy nhiên, trong `centralized-data-service/internal/service/recon/recon_job_worker.go:281, 410`, `ReconJobWorker.ProcessJob` lại gán trực tiếp `report.Status = status` (`"COMPLETED"` — trạng thái của job trong bảng `recon_jobs`) vào cột `status` của `cdc_system.cdc_reconciliation_report`, thay vì gán `"drift"` khi `diffVal != 0 || missingCount > 0 || staleCount > 0 || orphanCount > 0` và `"ok"` khi khớp.
inaddition, hàm `getRoundedEndTime()` trong `cdc-cms-web/src/components/ConfirmDestructiveModal.tsx:11-13` lùi cứng 5 phút khiến mọi thao tác đối soát thủ công tức thì đều bỏ sót biến động trong 5 phút gần nhất.
