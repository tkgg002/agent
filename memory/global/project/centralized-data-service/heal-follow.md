recon-heal-trigger
    ↓
RunTier2 (7d window) → noop
    ↓
FullIDDiffMissingFromShadow → 1 missing
    ↓
FetchAndWriteByIDs
    → MongoDB.Find({_id: {$in: [id]}})
    → buildHealEnvelope (giống snapshot runner)
    → EventHandler.HandleRaw → BatchBuffer → shadow upsert
    → FlushBatchBuffer
    ↓
transmute hook (auto)
    → master table sync
    ↓
report updated: missing_count=1, healed_count=1, status=healed

---

# HƯỚNG DẪN VÀ TÀI LIỆU VẬN HÀNH HEAL A (SAFETY NET & DIRECT FETCH-WRITE)

## 1. Bối cảnh & Vấn đề
- **Cơ chế cũ**: `RunTier2` sử dụng time-windowed hash scan (`cold_lookback` = 7 ngày) để so sánh chéo và phát hiện dữ liệu lệch (drift). Cơ chế này tối ưu cho việc quét liên tục nhưng **hoàn toàn vô hình** trước các record bị xóa khỏi shadow Postgres có timestamp cũ hơn 7 ngày (ví dụ: các record cũ bị xóa tay để kiểm thử, lỗi đồng bộ lâu ngày).
- Ngoài ra, cơ chế phục hồi cũ dựa trên **Debezium Incremental Snapshot Signal**. Tuy nhiên, trong nhiều môi trường thực tế (không có cấu hình signal collection, DB nguồn không readonly nên không khởi tạo snapshot được), Debezium chỉ làm nhiệm vụ sink upstream, và signal này trở thành no-op.

## 2. Giải pháp Kiến trúc Mới (Mô tả chi tiết)

### A. Cơ chế Phát hiện Lệch Toàn diện (Full ID Diff Safety Net)
Khi trigger `recon-heal-a`, nếu `RunTier2` trả về kết quả clean (`drifted_windows = 0`), hệ thống sẽ kích hoạt một lớp phòng vệ thứ hai: **Full ID Diff Safety Net**.
- **Cách thức hoạt động**:
  1. Stream toàn bộ `_source_id` không bị soft-delete từ shadow Postgres lên một `shadowSet` trong RAM.
  2. Stream toàn bộ `_id` từ MongoDB source thông qua `StreamAllIDs` của `ReconSourceAgent`.
  3. So sánh chéo trực tiếp để lọc ra danh sách các `_id` tồn tại ở source MongoDB nhưng hoàn toàn vắng bóng hoặc đã bị đánh dấu soft-delete (`_deleted = true`) ở shadow Postgres.
  4. Trả về danh sách missing IDs mà không phụ thuộc vào bất kỳ timestamp hay time-window nào.
- **Biện pháp bảo vệ (Circuit Breaker)**: Nếu source trả về 0 IDs nhưng shadow Postgres có dữ liệu, hệ thống tự động từ chối vận hành (nghi ngờ lỗi kết nối MongoDB) để tránh làm hỏng dữ liệu.

### B. Cơ chế Phục hồi Trực tiếp (Direct Fetch & Upsert - Bypass Debezium)
Khi phát hiện ra các record bị lệch:
- Thay vì gửi command signal qua Kafka/Debezium (có thể bị chặn hoặc no-op), hệ thống sẽ sử dụng trực tiếp **EventHandler** (tương tự như custom snapshot runner).
- **Quy trình thực thi**:
  1. Tải trực tiếp document từ MongoDB bằng truy vấn `{_id: {$in: [...]}}`.
  2. Chuyển đổi và đóng gói document thành định dạng event CDC chuẩn (`CDCEvent` envelope) bằng hàm `buildHealEnvelope` (giống như định dạng sinh ra từ snapshot runner).
  3. Đẩy trực tiếp envelope này vào `EventHandler.HandleRaw` để đi qua toàn bộ pipeline chuẩn (masking, mapping, shadow upsert) và ghi trực tiếp vào shadow Postgres.
  4. Gọi `FlushBatchBuffer` để hoàn tất ghi xuống.
  5. Sau khi ghi vào shadow Postgres, transmute hook sẽ tự động được kích hoạt để đồng bộ record lên master table ngay lập tức.

## 3. Các File Code Thay Đổi & Nhiệm vụ
1. **`internal/service/recon/recon_tier_a.go`**:
   - Thêm `FullIDDiffMissingFromShadow`: Thực hiện quét toàn bộ ID giữa source và shadow để tìm record bị miss.
   - Thêm `GetClient` public method trong `ReconSourceAgent` để expose MongoDB client kết nối bằng pool.
2. **`internal/handler/recon/recon_heal_fetch.go` [NEW]**:
   - Thêm `FetchAndWriteByIDs`: Đóng vai trò cầu nối, trực tiếp fetch dữ liệu từ MongoDB và đẩy vào `eventHandler.HandleRaw`.
3. **`internal/handler/recon/recon_heal_v4.go`**:
   - Sửa đổi `healSegmentA` để liên kết: Nếu `RunTier2` tìm ra 0 drift → chạy `FullIDDiffMissingFromShadow` → Nếu phát hiện lệch → thực thi `FetchAndWriteByIDs` (hoặc fallback về debezium signal nếu không wire eventHandler).
   - Patch lại report của reconciliation (`TotalSourceCount`, `TotalDestCount`, `missing_count`, `healed_count`) để phản ánh đúng lên frontend.
4. **`internal/server/server_setup.go`**:
   - Wire `eventHandler` từ shadow vào `reconHandler` thông qua method `WithEventHandler`.

## 4. Hướng dẫn Vận hành & Kiểm tra Logs
Khi trigger `recon-heal` cho một bảng có record bị lệch ngoài window 7 ngày, hãy quan sát các dấu hiệu logs sau:

1. **Chạy Full ID Diff**:
   `[heal-a] RunTier2 found no drift — running full ID diff safety net (catches records outside time window)`
2. **Kết quả quét ID**:
   `[full_id_diff] complete ... src_count=455 shadow_count=454 missing_from_shadow=1`
3. **Thực thi ghi trực tiếp**:
   `[heal-a] dispatching via FetchAndWriteByIDs (direct MongoDB→shadow)`
4. **Ghi và Upsert thành công**:
   `batch upsert ok ... persisted=1`
   `[heal-fetch] complete ... ids_requested=1 docs_written=1`
5. **Transmute tự động đẩy lên master**:
   `transmute complete ... scanned=1 inserted=1 updated=0`