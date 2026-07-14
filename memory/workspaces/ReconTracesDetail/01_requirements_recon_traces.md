# Yêu cầu rà soát & bổ sung chi tiết Tracing cho Recon Check

## 1. Bối cảnh
Khi chạy đối soát dữ liệu (Recon Check) qua NATS (`cdc.cmd.recon-check`), trace cha `cdc.recon.check` mất tới 46.79 giây nhưng trong SigNoz không hiển thị bất kỳ span con nào (hoặc các span con không đủ chi tiết). Việc này khiến việc trace bottleneck hiệu năng (như truy vấn DB chậm, xử lý logic chậm) trở nên cực kỳ khó khăn.

## 2. Mục tiêu
- Rà soát lại toàn bộ hệ thống Tracing của Reconciliation (bao gồm Segment A và Segment B).
- Bổ sung chi tiết child spans cho từng tác vụ quan trọng trong Recon:
  - Các truy vấn và logic trong `ReconCore` (`RunDeepCheck`, `RunHashWindowCheck`, `RunDeepCheckB`, `RunHashWindowCheckB`).
  - Các truy vấn trong Source Agent (`ReconSourceAgent`): `CountDocuments`, `HashWindow`, `BucketHash`, `ListIDTsInWindow`.
  - Các truy vấn trong Destination Agent (`ReconDestAgent`): `CountRows`, `CountDeletedRows`, `EstimatedCountRows`, `CountInWindow`, `BucketCounts`, `ListIDTsInWindow`, `MaxWindowTs`, `HashWindow`, `BucketHash`.
- Các child span phải mang tên trực quan, gắn đầy đủ các attributes cần thiết để dễ dàng trace trên SigNoz:
  - `table`: Tên bảng/collection.
  - `db`: Tên database.
  - `query_type`: Loại query (ví dụ: count, hash_window, bucket_counts, list_idts).
  - `duration_ms`: Thời gian thực hiện (nếu cần).
  - `count`/`rows`: Số lượng record quét hoặc trả về.
  - `time_range`: Start/End time của window check.
- Đảm bảo propagation context (truyền Context) thông suốt từ handler xuống tận DB query agent để child spans gắn đúng vào parent trace.

## 3. Khắc phục lock timeout của Transmuter
- **Hiện tượng:** Khi transmute chạy liên tục, việc gọi `EnsureMaster` ở mỗi vòng lặp / lô xử lý sẽ thực thi hàng loạt câu lệnh `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` và `ALTER TABLE ... DROP COLUMN IF EXISTS`. Trong Postgres, việc này sẽ yêu cầu khóa `AccessExclusiveLock` cho dù cột đã tồn tại hay chưa, dẫn đến lock timeout (lỗi 55P03) và block các truy vấn đọc/ghi song song.
- **Giải pháp:** Thiết lập cơ chế cache `ensuredMasters map[string]bool` trong `TransmuterModule`. Chỉ gọi `ddlEnsurer.EnsureMaster` một lần duy nhất cho mỗi bảng master khi bắt đầu chạy hoặc khi rule cache bị invalidate (`InvalidateRuleCache`). Việc này giúp loại bỏ hoàn toàn việc thực thi DDL vô ích trên mỗi lô dữ liệu, giảm thiểu tranh chấp khóa.

## 4. Tối ưu hóa Index để triệt tiêu Slow Query
- **Hiện tượng:** Qua logs hệ thống, các query như `SELECT COUNT(*) FROM shadow WHERE _deleted = true` mất 29s+, và `SELECT MAX(lastUpdatedAt)` hay `BucketCounts` mất 3s–9s+. Đây là nguyên nhân trực tiếp giữ khóa `AccessShareLock` lâu và gây Lock Timeout cho DDL.
- **Giải pháp:**
  - **Partial Index trên `_deleted`:** Tự động tạo `CREATE INDEX IF NOT EXISTS ... ON <table> (_deleted) WHERE _deleted = true` cho cả bảng shadow (trong `EnsureCDCColumnsInSchema`) và bảng master (trong `MasterDDLGenerator`). Việc này chuyển các query đếm tombstone từ Full Table Scan thành Index-Only Scan cực kỳ nhanh.
  - **Index trên cột Timestamp nghiệp vụ (Target Timestamp Field):** Trong `MasterDDLGenerator`, truy vấn bảng cấu hình `cdc_table_registry` để lấy ra cột timestamp (ví dụ `lastUpdatedAt`). Nếu cột này tồn tại và được map, tự động tạo index `CREATE INDEX IF NOT EXISTS ... ON <table> (timestamp_col)` trên master table.


