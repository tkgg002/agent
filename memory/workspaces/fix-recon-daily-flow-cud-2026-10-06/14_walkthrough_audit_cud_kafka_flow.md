# Hướng Dẫn Kiểm Tra & Vận Hành (Walkthrough): Chuẩn Hóa c,u,d & 2 Batch Buffers

## 1. Giới thiệu tổng quan
Tài liệu này hướng dẫn người vận hành và kiểm thử viên kiểm chứng toàn diện các cải tiến:
1. Giao diện đối soát theo ngày không còn cột `Read(r)` ở mục 1 (KAFKA CDC BẮN VỀ).
2. Dữ liệu $c, u, d$ được bóc tách chính xác từ backend `cdc_activity_log`.
3. Hai Batch Buffer (`cdcBatchBuffer` vs `snapshotBatchBuffer`) được phân định rõ ràng qua `triggered_by` và `buffer_type`.

---

## 2. Kiểm tra trên Giao diện (Frontend CMS Web)

### 2.1. Kiểm tra Box Ngoài (Thẻ Tổng Hợp 3 Trạm)
- Truy cập CMS Web: Menu **Đối soát dữ liệu** -> Chọn Pipeline bất kỳ (ví dụ: `shadow_traitestpbs.payment_bills`).
- Mở Drawer chi tiết Pipeline:
  - Ngay dưới sơ đồ luồng dữ liệu, quan sát **Box Tổng hợp số liệu**:
    - **Card 1: 1. KAFKA CDC BẮN VỀ**:
      - Chỉ còn hiển thị: `c: {kafkaC} • u: {kafkaU} • d: {kafkaD}`.
      - Tuyệt đối **không còn** nhãn `r:` hay bất kỳ số liệu read nào.
    - **Card 2: 2. GHI VÀO DB SHADOW**:
      - Hiển thị: `c: {shadowC} • u: {shadowU} • d: {shadowD}`.
    - **Card 3: 3. MASTER TRANSMUTE**:
      - Hiển thị: `Insert: {ins} • Update: {upd} • Skip: {skip}`.
    - **Card 4: ĐỐI SOÁT & DLQ**:
      - Hiển thị: `DLQ: {dlq} • Thiếu: {missing} • Stale: {stale} • Đã heal: {healed}`.

### 2.2. Kiểm tra Modal Chi Tiết Đối Soát Theo Ngày
- Nhấp nút **"Phân tích theo ngày"** trên Box tổng hợp:
  - Bảng Modal mở ra gồm đúng 18 cột:
    - Cột Ngày
    - Nhóm 1: KAFKA CDC BẮN VỀ (Tổng, c, u, d) — **4 cột, không có r**
    - Nhóm 2: GHI VÀO DB SHADOW (Tổng, c, u, d) — **4 cột**
    - Nhóm 3: MASTER TRANSMUTE (Tổng, Insert, Update, Skip) — **4 cột**
    - Nhóm 4: ĐỐI SOÁT & DLQ (DLQ, Thiếu, Stale, Đã heal) — **4 cột**
    - Đánh giá ngày — **1 cột**
  - Dòng Footer **"Tổng cộng"**: Các ô căn chỉnh chính xác 100% dưới từng cột tương ứng, không bị lệch ô.

---

## 3. Kiểm tra trên Cơ sở Dữ liệu & Backend (Backend Go)

### 3.1. Truy vấn kiểm tra log mới trong `cdc_system.cdc_activity_log`
```sql
SELECT 
    id, 
    operation, 
    target_table, 
    triggered_by, 
    rows_affected, 
    details->>'buffer_type' AS buffer_type,
    details->>'is_snapshot' AS is_snapshot,
    details->>'c' AS c,
    details->>'u' AS u,
    details->>'d' AS d,
    started_at
FROM cdc_system.cdc_activity_log
WHERE operation = 'kafka-consumer'
ORDER BY id DESC
LIMIT 10;
```
**Kỳ vọng**:
- Với batch từ Kafka streaming: `triggered_by = 'kafka-consumer'`, `buffer_type = 'cdc-stream'`, `is_snapshot = false`, `c, u, d` phản ánh đúng các events trong batch.
- Với batch từ Snapshot Runner: `triggered_by = 'snapshot'`, `buffer_type = 'snapshot'`, `is_snapshot = true`, `c = batch_size, u = 0, d = 0`.

---

## 4. Kiểm tra Lệnh (CLI Verification)

### Backend Tests:
```bash
cd centralized-data-service
go test -v ./internal/handler/shadow/... -run "TestSnapshotBuffer|TestCDCBuffer|TestBatchBuffer_OpFieldHandling"
```
Kết quả kỳ vọng: **PASS 100%**.

### Frontend Type Check:
```bash
cd cdc-cms-web
npx tsc --noEmit
```
Kết quả kỳ vọng: **0 errors**.
