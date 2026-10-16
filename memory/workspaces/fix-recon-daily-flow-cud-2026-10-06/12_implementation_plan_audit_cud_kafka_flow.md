# Kế hoạch Triển khai (Implementation Plan): Audit c,u,d & Chuẩn Hóa CDC Daily Flow

## Phase 1: Điều tra & Đối soát Bằng chứng Vật lý DB [ĐÃ HOÀN THÀNH]
1. Query bảng `cdc_system.cdc_activity_log` trên `gpay-postgres-cdc` lọc ngày `2026-09-30`.
2. Trích xuất chính xác 6 đợt Kafka consumer log (519 + 519 + 520 + 4 stream = 1,562 events).
3. Trích xuất 8 đợt Transmute log cho target table `master_centrallized_export_service.export_jobs_1` (3 + 3 + 519 + 519 + 519 + 1 + 20 + 500 = 2,084 records).
4. Đếm số dòng vật lý trên cả Shadow DB và Master DB (chính xác 536 records).

## Phase 2: Chỉnh sửa Frontend UI (`ReconPipelineGrid.tsx`) [ĐÃ HOÀN THÀNH]
1. Xóa cột `Read(r)` khỏi Modal Phân tích đối soát theo ngày:
   - Mục `1. KAFKA CDC BẮN VỀ` chỉ còn: Tổng, Insert(c), Update(u), Delete(d).
2. Xóa nhãn `r:` ở Card 1 Box tổng hợp ngoài:
   - Hiển thị gọn gàng: `c: ... • u: ... • d: ...`.
3. Xóa `kafkaR` khỏi `totals` useMemo và `Table.Summary`. Căn chỉnh 18 cột chính xác từ index 0 đến 17.
4. Xóa fallback tự ép dồn `batchSize` vào `c` để không làm sai lệch ý nghĩa các ngày chưa có số liệu bóc tách.
5. Chạy kiểm tra biên dịch `npx tsc --noEmit` đạt 0 lỗi.

## Phase 3: Triển khai Backend Go CDS (`centralized-data-service`) [ĐÃ HOÀN THÀNH]
1. Thêm `Op string` vào struct `UpsertRecord` (`cdc_event.go`).
2. Gán `event.Data.Op` tại `event_handler.go` và `bridge_handler.go`.
3. Trong `batch_buffer.go:batchUpsert`:
   - Phân định `isSnapshotBuffer` để gán `triggeredBy = "snapshot"`, `bufferType = "snapshot"` vs `"cdc-stream"`.
   - Đếm `cntC`, `cntU`, `cntD`.
   - Ghi `c, u, d, buffer_type, is_snapshot` vào `details` của `cdc_activity_log` khi flush batch.
4. Bổ sung test case `TestBatchBuffer_OpFieldHandling` trong `snapshot_cdc_isolation_test.go`.
5. Chạy `go test ./internal/handler/shadow/...` và `./internal/handler/source/...` PASS 100%.

## Phase 4: Kế hoạch Khắc phục 2 Issues Phát Hiện Từ Audit Phản Biện [LẬP KẾ HOẠCH & CHỜ DUYỆT]
1. **Issue 1: Phân tách luồng Snapshot khỏi KAFKA CDC BẮN VỀ tại Frontend**:
   - Trong `ReconPipelineGrid.tsx:410-440`: Nhận diện `isSnapshot = details.buffer_type === 'snapshot' || details.is_snapshot === true || k.triggered_by === 'snapshot'`.
   - Nếu `isSnapshot == true`:
     - **KHÔNG cộng** vào `row.kafka.total` và `row.kafka.c, u, d` (vì đây là do Snapshot Runner nạp trực tiếp, không phải Kafka bắn về).
     - **CỘNG** vào `row.shadow.total` và `row.shadow.c` (vì dữ liệu đã được nạp thành công vào Shadow DB dưới dạng Insert).
   - Nếu `isSnapshot == false`:
     - Cộng đầy đủ vào cả `row.kafka` và `row.shadow` theo đúng luồng CDC Streaming.
2. **Issue 2: Xử lý hiển thị thông minh cho các bản ghi lịch sử cũ**:
   - Nhận diện các batch snapshot cũ (qua `triggered_by = 'snapshot'` hoặc tương quan với job snapshot).
   - Đảm bảo trạm 1 (KAFKA CDC BẮN VỀ) ngày 2026-09-30 trở về đúng giá trị **4 events stream** thực tế thay vì bị dồn 1,558 records snapshot.
   - Trạm 2 (GHI VÀO DB SHADOW) ngày 2026-09-30 phản ánh đúng **1,562 records** ($1,558$ snapshot + $4$ stream).
3. **Kiểm thử**: Chạy `npx tsc --noEmit` và verify rendering UI.
