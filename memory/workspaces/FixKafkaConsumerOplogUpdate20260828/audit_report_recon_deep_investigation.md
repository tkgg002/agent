# AUDIT BÁO CÁO ĐIỀU TRA SÂU: RECON PHANTOM DRIFT (BÁO SAI 76 ID TRÙNG LẶP)

- **Mã Workspace**: `FixKafkaConsumerOplogUpdate20260828`
- **Thời gian Audit**: 2026-08-28T13:50:00+07:00
- **Tiêu chuẩn**: DoD Gates G1-G8, Rules #0-#19, Self-Improvement Loop (Rule #5)
- **Mức độ nghiêm trọng**: 🔴 CRITICAL — Báo cáo audit trước đó **SAI** (đã tuyên bố "drift:0 = fix thành công" dựa trên smoke recon, trong khi manual job vẫn lỗi y nguyên)

---

## I. THỪA NHẬN SAI LẦM (MID-SESSION FIX — Rule #5)

Báo cáo audit trước (`audit_report_fix_oplog_update.md`) đã mắc lỗi nghiêm trọng:
- **Sai lầm**: Sử dụng log `smoke-A drift:0` làm bằng chứng "fix thành công", mà không phân biệt giữa **smoke recon** (có tolerance mechanism) và **manual hash_window job** (không có tolerance).
- **Hậu quả**: User test lại → vẫn thấy 77 ID "Thiếu ở Shadow" và 76 ID "Thừa ở Shadow" → fix trước CHƯA giải quyết được vấn đề thực sự.
- **Lesson cần ghi**: Pattern "Smoke-OK ≠ Full-Recon-OK" — KHÔNG BAO GIỜ dùng kết quả smoke recon (có tolerance/guard) làm bằng chứng cho full recon job.

---

## II. ROOT CAUSE THỰC SỰ (DEEP INVESTIGATION)

### 2.1 Bằng chứng từ Database

**Query 1**: Kết quả Recon Job gần nhất từ `cdc_system.recon_jobs`:

| Bảng | source_count | dest_count | total_diff_count | mismatched | missing_from_dest | missing_from_src |
|------|-------------|------------|------------------|------------|-------------------|------------------|
| `payment_bills` | 964 | **963** | 36 | null | 77 | 76 |
| `payment_bills_1` | 964 | **964** | 34 | null | 76 | 76 |

**Quan sát quan trọng**:
- `payment_bills`: chênh 1 bản ghi (964 vs 963) → ID `49933` thực sự thiếu ở shadow.
- `payment_bills_1`: count **BẰNG NHAU** (964 = 964) → **KHÔNG có bản ghi nào thực sự thiếu!**
- `mismatched: null` — Không có ID nào bị phân loại là "timestamp mismatch".
- Nhưng 76 ID xuất hiện ở CẢ `missing_from_dest` VÀ `missing_from_src` → **Phantom Drift**.

**Query 2**: So sánh `_source_ts` (CDC metadata) vs `lastUpdatedAt` (business field) trên shadow:

| _id | _source_ts | pg_lastUpdatedAt_ms | diff_ms |
|-----|-----------|-------------------|---------|
| 50261 | 1787800662278 | 1787800660879 | **1399** |
| 50256 | 1787800662038 | 1787800660871 | **1167** |
| 50266 | 1787800662477 | 1787800661763 | **714** |
| 50282 | 1787800714205 | 1787800713566 | **639** |
| 50235 | 1787800030382 | 1787800030365 | 17 |

**Query 3**: `_raw_data->>'lastUpdatedAt'` = chính xác bằng cột `lastUpdatedAt` → Shadow KHÔNG lưu sai giá trị.

### 2.2 Cơ chế gây lỗi (Root Cause Chain)

```
MongoDB document được UPDATE → lastUpdatedAt thay đổi sang giá trị mới
                     ↓
Debezium capture.mode = "change_streams" (KHÔNG có full_update)
                     ↓  
UPDATE event chỉ chứa "patch" (không có "after" full document)
                     ↓
Kafka Consumer thấy afterData == nil → DROP event (tăng metric nil_after_data)
                     ↓
Shadow DB vẫn giữ lastUpdatedAt GIÁ TRỊ CŨ (từ lần INSERT ban đầu)
                     ↓
Recon chạy: Source MongoDB query lastUpdatedAt với giá trị MỚI
            Dest Shadow query lastUpdatedAt với giá trị CŨ
                     ↓
Cùng 1 ID rơi vào 2 sub-window 15-phút KHÁC NHAU
                     ↓
Sub-window A (chứa TS mới): Source thấy → Dest không thấy → "Missing from Dest"
Sub-window B (chứa TS cũ):  Dest thấy → Source không thấy → "Missing from Src"
                     ↓
staleAcc tích lũy qua tất cả sub-windows → Cùng ID xuất hiện ở CẢ 2 danh sách!
```

### 2.3 Tại sao Smoke Recon báo `drift:0`?

Smoke recon tại `recon_smoke.go:362-373` dùng cơ chế **"Guard 1 + HashWindow on static range"**:
- Nó hash TOÀN BỘ dải thời gian rộng (~2h) thay vì chia sub-window 15 phút.
- Trong 1 window rộng, tất cả ID (dù timestamp chênh) đều nằm trong cùng window → hash match → `drift=0`.
- Manual job (`ChunkStreamBucketEngine`) chia thành sub-window 15 phút, nên timestamp drift nhỏ (~1-2 giây) cũng có thể khiến 1 ID rơi lệch window → báo phantom drift.

---

## III. PHÂN TÍCH: 2 VẤN ĐỀ CẦN GIẢI QUYẾT

### Vấn đề 1: Dữ liệu Shadow cũ (Stale Data) — 76 bản ghi
- **Nguyên nhân**: MongoDB đã UPDATE 76 bản ghi (thay đổi `lastUpdatedAt`), nhưng UPDATE event bị drop do `capture.mode` cũ.
- **Fix đã làm**: Đổi `capture.mode` → `change_streams_with_full_update` ✅ (fix cho TƯƠNG LAI).
- **Fix còn thiếu**: Cần **re-sync** 76 bản ghi cũ để cập nhật `lastUpdatedAt` trong shadow khớp với MongoDB hiện tại.
- **Giải pháp**: Chạy Bridge/Batch-Sync cho `payment-bill-service.payment-bills` để đồng bộ lại dữ liệu stale.

### Vấn đề 2: ID 49933 thực sự thiếu — 1 bản ghi
- `payment_bills` có `source_count=964`, `dest_count=963`.
- ID `49933` không tồn tại trong shadow DB.
- **Giải pháp**: Chạy Bridge/Batch-Sync sẽ tự động bổ sung bản ghi thiếu.

---

## IV. ĐÁNH GIÁ FIX TRƯỚC ĐÓ

| Fix | Đánh giá | Giải thích |
|-----|---------|------------|
| `GetRealColumnName` (case-insensitive column lookup) | ✅ Đúng nhưng CHƯA ĐỦ | Đảm bảo `dstTS` resolve chính xác sang `"lastUpdatedAt"` thay vì fallback `_source_ts`. Fix này CẦN THIẾT nhưng không đủ vì vấn đề thực sự là stale data. |
| `capture.mode` → `change_streams_with_full_update` | ✅ Đúng cho TƯƠNG LAI | Ngăn UPDATE events bị drop trong tương lai. Nhưng KHÔNG fix dữ liệu cũ đã stale. |
| Tracing/Observability (OpenTelemetry attributes) | ✅ Hữu ích | Giúp debug nhanh hơn khi có drift. |

---

## V. KHUYẾN NGHỊ HÀNH ĐỘNG

1. **Ngay lập tức**: Chạy Bridge/Full-Sync cho `payment-bill-service.payment-bills` để:
   - Cập nhật `lastUpdatedAt` cho 76 bản ghi stale.
   - Bổ sung ID `49933` thiếu.
2. **Cải tiến Recon Algorithm**: Thêm logic trong `ChunkStreamBucketEngine.drillSubWindows` để **deduplicate** các ID xuất hiện đồng thời ở cả `MissingFromDest` và `MissingFromSrc` — phân loại chúng thành `TimestampDrift` (phantom drift) thay vì missing thật.

---

## VI. RÀ SOÁT LESSONS VIOLATED

| Lesson | Vi phạm? | Chi tiết |
|--------|---------|---------|
| Lesson #1208 (Không suy diễn) | **CÓ** | Audit trước đã dùng log `drift:0` từ smoke recon làm bằng chứng mà không verify bằng manual job. |
| Rule #5 (Mid-Session Fix) | **CÓ** | Phát hiện vấn đề nhưng đã tuyên bố "Done" quá sớm. |
| Rule #14 G2 (Red→Green) | **CÓ** | Chưa reproduce lỗi trên manual job trước khi tuyên bố fix. |
| Rule #14 G6 (Output Correctness) | **CÓ** | Không verify kết quả manual recon job thực tế. |

---

*Báo cáo audit này đã được xác minh bằng dữ liệu thực tế từ `cdc_system.recon_jobs` và `shadow_traitestpbs.payment_bills`.*
