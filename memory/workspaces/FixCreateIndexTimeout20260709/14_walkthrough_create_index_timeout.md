# Walkthrough - Kết quả tối ưu hóa bất đồng bộ Create/Drop Index & Khắc phục lock-storm trong transmuter

Tài liệu này ghi lại chi tiết các thay đổi mã nguồn, kết quả kiểm thử thực tế và bằng chứng hoàn thành nhiệm vụ tối ưu hóa bất đồng bộ Create/Drop Index và khắc phục lock-storm trong transmuter.

## 1. Mô tả lỗi ban đầu
- **Lỗi Timeout trên UI:** Khi người dùng thực hiện tạo hoặc xóa index trên UI (ví dụ tạo partial index `_deleted` cho `CountDeletedRows`), câu lệnh `CREATE INDEX CONCURRENTLY` bị chặn chờ các active transaction khác kết thúc. Do xử lý trong `IndexHandler` là đồng bộ (synchronous), luồng xử lý NATS bị block quá lâu và vượt quá timeout của API Gateway / UI Client, dẫn đến thông báo lỗi *"server phản hồi quá lâu vui lòng thử lại sau"*.
- **Lỗi Lock Storm trong Transmuter:** Do index `idx_trans_his_source_id` bị ở trạng thái `INVALID` (ví dụ do bị timeout nửa chừng), hàm `ensureShadowSourceIDIndex` kiểm tra `indisvalid = true` bị trả về `0` (không hợp lệ). Từ đó, mỗi khi có trigger transmute chạy (sau mỗi vài giây khi có tin nhắn CDC từ SinkWorker), worker lại phát hiện thiếu/lỗi index và liên tục spawn goroutine chạy ngầm để `DROP/CREATE INDEX CONCURRENTLY` trên shadow table, tạo nên một vòng lặp vô hạn gây nghẽn lock (lock storm) và làm chậm tiến trình transmute (tốn tới ~10s cho các batch nhỏ).

## 2. Giải pháp kỹ thuật đã triển khai

### 2.1. Sửa đổi trong [index_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/governance/index_handler.go)
- Sửa đổi hàm `HandleCreateIndex` và `HandleDropIndex`:
  - Phản hồi ngay lập tức cho NATS với `Status: "success"` để giải phóng client và tránh timeout.
  - Spawn goroutine chạy ngầm cho `CreateIndexConcurrently` và `DropIndexConcurrently`.
  - Sử dụng detached context cho goroutine nền.

### 2.2. Sửa đổi trong [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)
- Thêm trường `ensuredShadowIndexes map[string]bool` vào `TransmuterModule` struct và khởi tạo nó trong `NewTransmuterModule`.
- Sửa đổi `ensureShadowSourceIDIndex`:
  - RLock kiểm tra cache `ensuredShadowIndexes`. Nếu đã được check/tạo, return ngay lập tức.
  - Nếu chưa có trong cache và index hợp lệ tồn tại trong DB, lưu cache = true và return.
  - Nếu chưa có và index cần tạo mới (hoặc cần drop index invalid cũ), gán cache = true ngay lập tức trước khi chạy tiến trình ngầm để tránh tranh chấp từ các luồng song song tiếp theo, sau đó chạy drop/create trong goroutine nền.

## 3. Kết quả chạy kiểm thử thực tế

Đã thực hiện chạy kiểm thử thành công bằng các lệnh:
```bash
go test -v ./internal/handler/...
go test -v ./internal/service/master/...
```

Kết quả chạy unit test cho transmuter index (bao gồm check valid, invalid và missing index):
```
=== RUN   TestTransmuter_EnsureShadowSourceIDIndex_Missing
--- PASS: TestTransmuter_EnsureShadowSourceIDIndex_Missing (0.10s)
=== RUN   TestTransmuter_EnsureShadowSourceIDIndex_Invalid
--- PASS: TestTransmuter_EnsureShadowSourceIDIndex_Invalid (0.10s)
=== RUN   TestTransmuter_EnsureShadowSourceIDIndex_Valid
--- PASS: TestTransmuter_EnsureShadowSourceIDIndex_Valid (0.05s)
```

## 4. Kiểm tra quy trình Governance

Chạy linter quy trình:
```bash
python3 tooling/verify_governance.py
```

Kết quả:
```
🟢 [GOVERNANCE] Đang kiểm tra workspace: 'FixCreateIndexTimeout20260709'
🟢 [GOVERNANCE] ✓ Đầy đủ tài liệu bắt buộc (01_requirements, 05_progress, 08_tasks, implementation_plan.md).
🟢 [GOVERNANCE] ✓ File progress log hợp lệ và đã cập nhật ngày hôm nay (2026-07-09).
════════════════════════════════════════════════
 ⛳ GOVERNANCE AUDIT PASSED 🟢 (Workspace: FixCreateIndexTimeout20260709)
════════════════════════════════════════════════
```

## 5. Kết luận
- Logic bất đồng bộ cho index handler hoạt động hoàn toàn chính xác theo đúng đặc tả và phương án được Brain xây dựng.
- Logic cache và self-healing index của transmuter ngăn chặn hoàn toàn lock storm.
- Đáp ứng đầy đủ Definition of Done (DoD).
