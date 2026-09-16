# 01_requirements_recon_no_index.md

## 1. Bối cảnh & Hiện trạng
- Hệ thống CDC Worker chạy tác vụ Full Reconciliation (Segment A: Source MongoDB ↔ Shadow PostgreSQL).
- Table `order_updates_bvb` (MongoDB collection `order-updates`) có quy mô ~15 triệu records.
- Source MongoDB **KHÔNG CÓ INDEX** trên trường thời gian (`timestamp_field`, ví dụ `updated_at`, `created_at`).
- Khi chạy tới bước `recon.source.get_range_hash_and_count` (`ChunkStreamBucketEngine.checkDayChunk`), tiến trình bị hủy runtime với lỗi:
  `context deadline exceeded (30s)`

## 2. Mục tiêu kỹ thuật (Requirements)
- **R1 (Triệt tiêu Timeout):** Loại bỏ hoàn toàn lỗi `context deadline exceeded 30s` khi recon các bảng lớn không có index thời gian.
- **R2 (Bảo toàn tính chính xác - Zero False Drift):** Không tạo ra kết quả so khớp sai lệch (False Positive Drift) giữa Source MongoDB và Shadow PostgreSQL.
- **R3 (Simplicity First & Minimal Impact):** Không phá vỡ kiến trúc generic platform của `centralized-data-service`; không hardcode giải pháp riêng cho 1 collection.
- **R4 (Type Safety & In-Place Update Guard):** Không áp dụng gán ép `_id: ObjectId` khi collection dùng ID dạng String/UUID hoặc khi collection có in-place UPDATE.
