# 05 Progress Log — Fix Batch Transform Sensitive Masking

> Append-only. KHÔNG xóa/sửa dòng cũ.

---

## [2026-09-07T16:38] [Brain:Gemini] INIT workspace

**Scope:** Fix batch transform không mask sensitive fields (hmac/aes_gcm) đúng Go native.
**Root Cause xác nhận:** `BatchTransformHandler` thiếu `*governance.MaskingService`, dùng SQL pgcrypto thay vì Go crypto.
**Status:** Planning — chờ user approve plan trước khi Muscle execute.

## [2026-09-07T16:43] [Brain:Gemini] Mid-Session Fix — Vi phạm Mandatory Doc Registry

- **Lỗi:** Tạo workspace chỉ có 2 file (`00_context.md`, `05_progress.md`), vi phạm Rule #4 Hotfix minimum: bắt buộc 3 file (`01_requirements.md`, `05_progress.md`, `08_tasks.md`).
- **Nguyên nhân:** Không tuân thủ chuẩn Hermes — thiếu đọc kỹ quy tắc Task Sizing trước khi tạo workspace.
- **Fix:** Đã bổ sung `01_requirements.md` và `08_tasks.md` theo đúng chuẩn.
- **Lesson ghi nhận:** Mid-Session Fix Rule #5 — ghi nhận vào lessons.md.

## [2026-09-07T16:51] [Brain:Gemini] Adversarial Audit — Plan v1

**Phát hiện 6 sai sót** (3 Critical, 2 Medium, 1 Low) trong demo code của plan v1:
- #3 CRITICAL: Thiếu `HasColumnInSchema` check trước khi append `sensitiveRules`
- #5 CRITICAL: `len(setClauses) == 0` → early exit, không gọi sensitive phase khi toàn bộ force_fields là sensitive
- #6 CRITICAL: `runSensitiveMasking` không được gọi trong unchunked fallback path
- #1 Medium: Scan pattern lệch codebase (struct scan vs `.Rows()`)
- #2 Medium: `whereExpr` naming chưa rõ mapping
- #4 Low: `_updated_at` double update
**Status:** Plan v2 cần REVISE trước khi APPROVE. Xem audit_report_v2.md.

## [2026-09-07T17:28] [Brain:Gemini] Adversarial Audit — Plan v2

**Phát hiện 4 sai sót** (2 Critical, 2 Medium) trong Plan v2:
- #7 CRITICAL: `len(setClauses)==0` trong MODIFY 1e unreachable — line 242 `_updated_at = NOW()` chạy TRƯỚC, làm setClauses luôn ≥ 1. Bug #5 CHƯA được fix đúng.
- #8 CRITICAL: Non-force mixed rules — `runSensitiveMasking` nhận `whereExpr` của non-sensitive fields, sensitive fields bị bỏ qua trên nhiều rows.
- #9 Medium: MODIFY 1g mâu thuẫn `pkCol` vs `pkCol2`.
- #10 Medium: `_updated_at` trong `runSensitiveMasking` — mâu thuẫn nội bộ.
**Fix:** Dùng `hasNonSensitiveRules bool` trước line 242; tính `sensitiveWhere` riêng cho runSensitiveMasking.
**Status:** Plan v3 cần được viết.

## [2026-09-08T09:26] [Brain:Gemini] Session 2 — Yêu cầu mới: Scale 50M–500M records

**Context:** User phát hiện design plan v3.1 (row-by-row UPDATE) hoàn toàn không scale.

**Bottleneck được xác nhận:**
- N+1 round-trips: 50M rows / 2000 QPS = 7-9 giờ → không chấp nhận được
- Open cursor toàn bảng → ghim MVCC `xmin`, chặn AUTOVACUUM → WAL bloat
- `json.Unmarshal → map[string]interface{}` → GC thrashing ở volume lớn
- Double write amplification cho mixed-rules tables
- Bug #1.5: unchunked path hardcode `pkCol=""` → silent skip sensitive masking

**Hướng giải quyết (confirmed trong phiên):**
- Keyset pagination (native PK type, không cast `::text`)
- Bulk UPDATE FROM (VALUES ...) — 1 round-trip/chunk
- gjson zero-alloc JSON extraction
- batchSize clamp: min(2000, 60000/numCols) tránh 65535 param limit
- finishJob SAU CẢ HAI phase (bulk + sensitive)

## [2026-09-08T09:30] [Brain:Gemini] Plan v4 — Draft

**Plan v4 viết:** keyset + bulk VALUES + worker pool (errgroup) + gjson.
**Lỗi phát hiện bởi user:**
1. `pk::text` phá B-Tree index + sort lexicographic sai
2. `errgroup` import thừa → compile error (không dùng)
3. `finishJob(COMPLETED)` gọi TRƯỚC sensitive masking → vi phạm lifecycle
4. Unchunked: hardcode `""` thay vì `pkCol` từ line 273
5. batchSize×numCols có thể vượt 65535 PostgreSQL params

## [2026-09-08T09:36] [Brain:Gemini] Plan v5 — Fix 5 critical bugs

**Fix áp dụng:**
- Xóa tất cả `::text` cast trên PK (WHERE, SELECT, VALUES JOIN)
- Xóa `errgroup` import; xử lý HMAC tuần tự (2000 × 1.5µs = 3ms, goroutine overhead > 3ms)
- `finishJob` chuyển xuống SAU `runSensitiveMasking` trong cả unchunked và chunked path
- Unchunked: truyền `pkCol` (biến từ line 273), không hardcode `""`
- `getSensitiveBatchSize(numCols)`: clamp = min(base, 60000/numCols)

## [2026-09-08T09:43] [Brain:Gemini] Plan v5 Revised — Standalone (theo yêu cầu)

**Lỗi quy trình:** Plan v5 lần đầu viết theo dạng diff/patch, reference plan cũ → vi phạm quy tắc "plan phải là tổng hợp từ đầu".
**Fix:** Viết lại plan hoàn chỉnh standalone, bao gồm:
- Toàn bộ code final của từng thành phần (không dùng diff syntax)
- 10 phần code rõ ràng: imports, type, struct, setters, for loop, post-loop, unchunked fix, chunked fix, runSensitiveMasking, bulkUpdateMasked
- Execution flow diagram đầy đủ
- Self-audit checklist 16 điểm

**Status:** Chờ APPROVE từ user để Muscle execute.
**Plan lưu tại:** `12_implementation_plan_v5.md`

## [2026-09-08T09:59] [Brain:Gemini] User Audit Plan v5 → 6 điểm sửa

**Audit từ user:**
- Bug 1 (CRITICAL): Phần 8 chunked error path gọi `publishTransmuteTrigger` → transmute worker đồng bộ data chưa mask
- Bug 2 (CRITICAL): Sensitive-only path dùng `finishJob` trực tiếp, thiếu Activity Logger
- Bug 3 (CRITICAL): VALUES type inference: nil → "could not determine data type"; UUID PK → "operator does not exist: uuid = text"
- Design 1: Dirty state — bulk SQL chạy xong, sensitive fail → data ghi dở dang
- Design 2: Double-count — 50M bulk + 50M sensitive = 100M báo về (200%)
- Design 3: Unchunked path có sensitive masking block unreachable sau fail-fast

**Fix áp dụng trong Plan v6:**
- Xóa publishTransmuteTrigger khỏi chunked error block
- Sensitive-only: publishAndFinishJob + Activity Logger + fail-fast PK check riêng
- bulkUpdateMasked: first-row ?::text + v.pk::pkType trong WHERE
- Thêm detectPrimaryKeyType helper
- Fail-fast TRƯỚC bulk SQL khi sensitiveRules>0 && pkCol==""
- finalAffected = max(totalRows, sensitiveUpdated)
- Unchunked simplified: no sensitiveRules block

**Status:** Plan v6 hoàn chỉnh. Chờ APPROVE từ user để Muscle execute.
**Plan lưu tại:** `12_implementation_plan_v6.md` (666 lines)

## [2026-09-08T10:32] [Brain:Gemini] User Audit Plan v6 → 3 điểm sửa cuối

**Audit từ user:**
- Risk 1 (CRITICAL): Keyset SELECT chunk 2+: lastPK bind as text → `uuid > text` → sập
- Risk 2 (DATA CORRUPTION): nil guard `nil → ""` gây semantic corruption (NULL→"", phá CHECK constraint, sai incremental IS NULL)
- Risk 3: ph[0] (PK) ở row 0 VALUES không cast → type inference bất định cho v.pk

**Fix trong Plan v7:**
- Risk 1: `pkCastPH = "?::pkType"` cho lastPK trong keyset SELECT WHERE
- Risk 2: Xóa hoàn toàn nil guard — giữ nil là SQL NULL; `?::text` ở row 0 VALUES đã đủ
- Risk 3: Row 0: `ph[0] = "?::pkType"` khi pkType != "" (đúng type cho v.pk ngay từ đầu)

**Status:** Plan v7 FINAL — đủ điều kiện APPROVE.
**Plan lưu tại:** `12_implementation_plan_v7.md` (683 lines)
