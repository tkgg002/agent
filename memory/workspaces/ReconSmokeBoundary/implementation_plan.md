# Kế Hoạch Triển Khai: Tối ưu hóa Smoke Check tránh cảnh báo giả khi Pipeline hoạt động

Dưới đây là kế hoạch chi tiết nhằm cải tiến cơ chế đối soát nhanh (Smoke Check) của chặng A (Source-Shadow) và chặng B (Shadow-Master) để không bị ảnh hưởng bởi độ trễ đồng bộ khi các tiến trình ghi/xóa hoạt động tích cực, đồng thời bảo vệ cơ sở dữ liệu MongoDB khỏi quá tải.

## User Review Required

> [!IMPORTANT]
> - **Cơ chế trừ bù cửa sổ thời gian (Boundary Subtraction):** Hệ thống sẽ lấy tổng số lượng `all -> now` trừ đi số lượng bản ghi phát sinh trong cửa sổ gần đây `[from, now]` để có được số lượng "sạch" trước khi so sánh.
> - **Làm tròn mốc thời gian:** Mốc `from` sẽ lùi 120 giây tính từ `now` và được làm tròn lùi về đầu phút (sử dụng `.Truncate(time.Minute)`) để đảm bảo tính ổn định và bao quát giây lẻ.
> - **Bảo vệ MongoDB & Sử dụng EstimatedCount mặc định:**
>   - Ở MongoDB (Chặng A), hệ thống sử dụng `EstimatedCount` (đọc metadata O(1)) thay vì `CountDocuments` để tránh thực hiện quét toàn bộ collection gây quá tải CPU/IO và timeout (tránh đánh sập hệ thống).
> - **Giải quyết sai số EstimatedCount bằng Đối chiếu chéo HashWindow (Phạm vi Tĩnh):**
>   - Dưới ràng buộc Zero-tolerance (lệch 1 row cũng là drift), nếu kết quả trừ bù `srcEstClean` và `dstActiveClean` lệch nhau (`diff != 0`):
>     - Hệ thống không báo drift ngay. Thay vào đó, thực hiện chạy một kiểm tra nhanh **Hash Window** trên cửa sổ thời gian tĩnh đã hoàn tất đồng bộ hoàn toàn.
> - **Chi tiết kiểm tra nhanh HashWindow:**
>   *   **Mốc thời gian (Range):** Để loại bỏ hoàn toàn nhiễu trễ đồng bộ (replication lag), dải quét được xác định như sau:
>       *   `hi` (Mốc trên): Được đặt bằng chính xác `fromTime` (tức `now - 120s` làm tròn phút). Điều này loại bỏ hoàn toàn dải 120s đang ghi/xóa chưa đồng bộ hoàn tất.
>       *   `lo` (Mốc dưới): Được tính bằng `hi.Add(-rc.effectiveLookback(ctx))` (mặc định Hot mode là `hi - 2h`, Cold mode là `hi - 7d`).
>       *   Như vậy, khoảng thời gian đối soát chéo HashWindow là `[lo, hi)` - hoàn toàn nằm ngoài cửa sổ trễ đồng bộ.
>   *   **Truy vấn nguồn (Source - MongoDB):** Gọi `rc.sourceAgent.HashWindow(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, lo, hi)`.
>       *   Hàm này query qua index trên trường timestamp trong khoảng `[lo, hi)` để lấy `_id` và timestamp, băm theo thuật toán `hashIDPlusTsMs`, XOR tích lũy tất cả kết quả băm để tạo ra chữ ký `XorHash` và số lượng `Count`.
>   *   **Truy vấn đích (Destination - Postgres):** Gọi `rc.destAgent.HashWindow(fastCtx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, lo, hi)`.
>       *   Hàm này stream dữ liệu trong khoảng `[lo, hi)`, băm tương tự để sinh ra `XorHash` và `Count`.
>   *   **Đối soát:** So sánh trực tiếp: `srcHash.Count == dstHash.Count` và `srcHash.XorHash == dstHash.XorHash`.
>       *   Nếu khớp hoàn hảo: Có nghĩa là dữ liệu trong khoảng thời gian hoạt động tích cực khớp hoàn hảo, độ lệch tổng ban đầu chỉ là sai số metadata của `EstimatedCount`. Hệ thống gán `diff = 0` và trả về trạng thái `"ok"` (loại bỏ hoàn toàn cảnh báo giả).
>       *   Nếu không khớp: Drift thực sự, trả về trạng thái `"drift"`.
> - **Tính toán số lượng Xóa Mềm trong cửa sổ (Soft Delete Awareness):**
>   - Đếm số bản ghi bị xóa mềm trong cửa sổ thời gian gần đây (`dstRecentDeleted` với cờ `_deleted = true`).
>   - Số lượng active thực tế trong cửa sổ: `dstRecentActive = dstRecentTotal - dstRecentDeleted`.
>   - Số lượng sạch dùng để đối soát: `dstActiveClean = dstActive - dstRecentActive`.

## Proposed Changes

## Proposed Changes

### Component: `recon_dest_agent`

#### [MODIFY] [recon_dest_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go)
- Thêm hàm `CountRecentDeletedRows(ctx, tableName, timestampField string, tLo, tHi time.Time) (int64, error)` để đếm chính xác số lượng bản ghi bị xóa mềm (`_deleted = true`) trong khoảng thời gian xác định.

### Component: `recon_smoke`

#### [MODIFY] [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go)
- **Hàm `RunTotalOnlyA`:**
  - Sử dụng `EstimatedCount` làm mặc định cho MongoDB để bảo vệ database.
  - Áp dụng công thức trừ bù số lượng xóa mềm trong cửa sổ 120s để tính `srcEstClean` và `dstActiveClean`.
  - Nếu `diff != 0`, chạy đối chiếu chéo `HashWindow` trên dải tĩnh `[lo, hi)` (với `hi = fromTime`) giữa MongoDB và Shadow. Nếu khớp, tự động điều chỉnh `diff = 0` và trả về `"ok"`.
- **Hàm `RunTotalOnlyB`:**
  - Áp dụng công thức trừ bù số lượng xóa mềm để tính `shadowActiveClean` và `masterActiveClean`.
  - Thực hiện đối soát không sai lệch.

### Component: `recon_smoke_test.go`

#### [MODIFY] [recon_smoke_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke_test.go)
- Viết bổ sung các ca kiểm thử (Unit Test) cho logic trừ bù cửa sổ kèm đối chiếu chéo HashWindow để chứng minh tính đúng đắn.

---

## Verification Plan

### Automated Tests
- Chạy toàn bộ test suite của recon để đảm bảo không lỗi regression:
  ```bash
  go test -v ./internal/service/recon/...
  ```

### Manual Verification
- Kiểm tra logs hệ thống khi chạy Smoke Check xem có hiển thị các dòng log info đối chiếu chéo HashWindow thành công hay không.
