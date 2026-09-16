# 09_tasks_solution_recon_no_index.md

## Hồ sơ giải pháp kỹ thuật (Technical Solution)

### 1. Phân tích điểm nghẽn
1. `sa.cfg.QueryTimeout` mặc định cố định 30s (`recon_models.go:128`).
2. `sa.cfg.BatchSize` mặc định cố định 1000 (`recon_models.go:125`).
3. Trong `server_setup.go:158`, `recon.NewReconSourceAgent(mongoClientShared, logger)` truyền config rỗng `ReconSourceAgentConfig{}`.
4. Khi MongoDB không có index trên `timestampField`, `coll.Find()` chạy COLLSCAN trên 15 triệu bản ghi tốn 45s-90s, vượt ngưỡng 30s.

### 2. Giải pháp 3 lớp (3-Tier Solution)

#### Lớp 1: Cấu hình Timeout & BatchSize linh hoạt (Minimal Impact)
- `recon_models.go`:
  - Đọc `QueryTimeout` từ env `RECON_QUERY_TIMEOUT` (nếu không set thì fallback 120s cho scan nặng).
  - Đọc `BatchSize` từ env `RECON_BATCH_SIZE` (mặc định nâng lên 5000).
- `server_setup.go`:
  - Truyền timeout và batch size từ `cfg.Worker` vào `ReconSourceAgentConfig` và `ReconDestAgentConfig`.

#### Lớp 2: Safe ObjectId-Derived Seeking Guard (Hiệu năng đột phá khi thỏa điều kiện)
- Trong `recon_hash.go`:
  - Kiểm tra kiểu dữ liệu `_id` của document mẫu: nếu `_id` là `primitive.ObjectID` VÀ dữ liệu là append-only:
    - Ánh xạ `[tLo, tHi)` sang `[loOID, hiOID)`.
    - Dùng B-Tree index `{ _id: 1 }` có sẵn của Mongo.
  - Nếu `_id` là String/UUID hoặc bảng có in-place update:
    - BẮT BUỘC giữ nguyên filter theo `timestampField`, chạy COLLSCAN an toàn với Timeout 120s-300s để bảo toàn tính toàn vẹn tài chính, triệt tiêu nguy cơ False Drift.

#### Lớp 3: Tối ưu hoá truy vấn COLLSCAN
- Loại bỏ `$or` lặp nếu kiểu dữ liệu của timestamp field đã được chuẩn hóa.
- Set projection tối giản `{ _id: 1, tsField: 1 }`.
