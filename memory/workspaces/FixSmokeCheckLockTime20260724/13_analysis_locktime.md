# Self-Reflection & Root Cause Analysis — Smoke Check Audit (2026-07-27)

## 1. Kết Quả Audit Thực Tế Từ User
```text
15:54:00 27/7/2026 — Smoke Check Shadow → Master KHỚP 15ms : 2,788,460 → 2,788,460 (0)
15:54:00 27/7/2026 — Smoke Check Source → Shadow LỆCH 45ms : 2,788,465 → 2,788,460 (-5)
```

---

## 2. Vòng Lặp Phản Tỉnh & Nguyên Nhân Gốc Rễ (Root Cause Analysis)

### ❌ Lỗi 1: `RunTotalOnlyA` Vẫn Đang Dùng `EstimatedCount` Thay Vì Exact Count Trên Mongo Source
- **Vấn đề**: Trong [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go#L250), code vẫn gọi `rc.sourceAgent.EstimatedCount(...)` khi source là MongoDB.
- **Hậu quả**: `EstimatedCount` đọc từ metadata collection của Mongo, không phản ánh chính xác số lượng document thực tế at runtime (`2,788,465`). Trong khi Shadow đếm exact `COUNT(*)` trả về `2,788,460`. Sự chênh lệch 5 bản ghi này là do Metadata ước lượng sai, dẫn tới báo LỆCH -5 giả!

### ❌ Lỗi 2: Mốc Thời Gian Phiên (`CheckedAt`) Vẫn Bị Làm Tròn Phút (`:00`)
- **Vấn đề**: Trong `CheckAllUnified`, `lockTime` lấy từ `start` lúc cron job chạy tại tích tắc đầu phút (`15:54:00`).
- **Hậu quả**: Khi đã loại bỏ cửa sổ 120s làm tròn phút, Smoke Check là phép đếm snapshot realtime. Mốc thời gian phiên `CheckedAt` không được phép bị lùi/tròn về `:00`, mà phải là exact runtime timestamp (`time.Now().UTC()`).

---

## 3. Kế Hoạch Khắc Phục Triệt Để

1. **Khắc phục Lỗi 1 (Source Count)**:
   - Chuyển `RunTotalOnlyA` gọi `rc.sourceAgent.CountDocuments` (Exact Count) cho cả MongoDB và PostgreSQL, hoặc dùng Shadow-Guided PK Probe để đếm exact 100% Mongo Source mà không COLLSCAN.
   - Loại bỏ hoàn toàn `EstimatedCount` khỏi luồng Smoke Check.

2. **Khắc phục Lỗi 2 (CheckedAt Timestamp)**:
   - Gán `CheckedAt: time.Now().UTC()` exact tại thời điểm tạo `SmokeResult`, không dùng mốc `start` lùi/tròn phút `:00`.
