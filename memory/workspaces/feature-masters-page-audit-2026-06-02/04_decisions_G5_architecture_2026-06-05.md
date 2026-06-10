# 04_decisions_G5_architecture_2026-06-05.md — ADR: 3 gap kiến trúc Shadow→Master (G5)

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-05 | Status: **PROPOSED (chờ User/Brain duyệt — CHƯA code)**
> Phạm vi: chỉ luồng **Shadow→Master** (transmuter/worker). KHÔNG đụng Source→Shadow.
> Nguyên tắc: "không sửa mù" — mỗi đề xuất bám code thật (file:line) + có kế hoạch verify red→green.

---

## Bối cảnh chung
Sau khi đóng B1/B2 + G1–G4, feature đạt ~bug-0 ở mức **chức năng**. Còn 3 gap **kiến trúc/quy mô** (không phải bug chặn test tay, nhưng ảnh hưởng an toàn dữ liệu & hiệu năng khi shadow lớn dần). ADR này phân tích từng gap dựa trên code hiện tại của `centralized-data-service/internal/service/transmuter.go` rồi đề xuất quyết định + thứ tự ưu tiên.

**Evidence nền (đã đọc code thật):**
- Rule cache: `transmuter.go:34-37` (`cache map[string]cachedRules`, `cacheTTL`), `:130` `cacheTTL = 60 * time.Second`, key `:289` `fmt.Sprintf("%d|%s", row.ID, row.MasterTable)`.
- Write: `transmuter.go:427` `upd, err := t.upsertMaster(ctx, binding, record)` nằm **trong vòng `for _, e := range emits`** → ghi **từng record một**. `upsertMaster` `:550` = `INSERT INTO ... VALUES (...) ON CONFLICT ... WHERE _hash IS DISTINCT ... AND _source_ts >= ...` (OCC guard, GAP-02 đã verify).
- Scan: `transmuter.go:173` `var lastGpayID int64` (local trong Run) → `:175` fetch `WHERE (pk)::bigint > cursor`, `:355` `ORDER BY 1 LIMIT batchSize`, `:190` advance cursor; `batchSize=500` `:127`. Cursor **reset 0 mỗi Run** → full re-scan.

---

## ADR-1 — GAP-SAFE-2: Cache rule stale sau DDL Apply

### Vấn đề
Khi user **Approve** một master mapping rule mới → worker chạy `master_ddl_generator` Apply (ALTER TABLE thêm cột). Nhưng `TransmuterModule` cache danh sách rule theo key `id|MasterTable` với TTL **60s** (`:130`, `:289-291`). Nếu user Approve cột mới rồi **bấm Sync ngay** (trong vòng 60s), transmute có thể dùng **rule cache cũ** → cột vừa duyệt **không được map** ở lần sync đó, tới khi cache hết hạn (≤60s) thì tự lành.

### Mức độ
🟡 **THẤP** — tự lành sau ≤60s, **không mất/hỏng dữ liệu** (chỉ trễ map 1 cột trong 1 lần sync). Nhưng đúng kịch bản người dùng hay làm ("duyệt xong sync luôn") → trải nghiệm khó hiểu ("đã duyệt sao chưa có cột?").

### Phương án
- **A. Invalidate có chủ đích (khuyến nghị)**: sau khi DDL Apply thành công cho `master_binding`, gọi `TransmuterModule.InvalidateRuleCache(bindingID, masterTable)` xoá entry `cache[key]`. Cùng tiến trình worker → chỉ cần thêm 1 method + 1 lời gọi sau Apply.
- **B. Giảm TTL** xuống ~5–10s: đơn giản nhưng tăng tần suất query rule (đánh đổi tải DB) và **không triệt để** (vẫn có cửa sổ).
- **C. Bỏ cache rule**: query mỗi batch — đơn giản nhất, đúng nhất, nhưng tăng tải (mỗi Run ≥1 query rule; với cron dày sẽ cộng dồn).

### Quyết định đề xuất: **A**
Minimal-impact, triệt để, không đánh đổi tải. Rủi ro: phải nối `master_ddl_generator`↔`TransmuterModule` (hiện tách struct). Có thể dùng callback `OnDDLApplied func(bindingID int64, table string)` inject lúc khởi tạo worker để **không** tạo phụ thuộc vòng.

### Effort / Risk
- Effort: **S** (~30–50 LOC: 1 method `InvalidateRuleCache` có lock `mu`, 1 callback wiring ở worker bootstrap, 1 lời gọi sau Apply).
- Risk: thấp. Lock `mu` đã có (`:35`). Không đụng Source→Shadow.

### Verify red→green (bắt buộc trước Done)
1. RED: Approve cột mới → trong 60s trigger transmute → kiểm tra dest table **thiếu** giá trị cột mới (log rule_count cũ).
2. Áp A.
3. GREEN: lặp lại → cột mới có giá trị ngay lần sync đầu; log rule_count tăng đúng. Đồng thời check không gọi invalidate dư (chỉ sau Apply thành công).

---

## ADR-2 — GAP-PERF-1: Ghi master theo từng dòng (per-row upsert)

### Vấn đề
`transmuter.go:427` gọi `upsertMaster` cho **mỗi emit** trong batch 500 dòng → **500 lệnh INSERT...ON CONFLICT riêng lẻ** mỗi batch = 500 round-trip DB. Với shadow nhỏ (454 dòng) không sao; shadow lớn (triệu dòng) → nghẽn round-trip, sync chậm tuyến tính.

### Mức độ
🟡 **THẤP→TRUNG** theo quy mô — không sai dữ liệu, chỉ chậm. Trở thành nút thắt khi dữ liệu lớn / cron dày.

### Phương án
- **A. Multi-row VALUES upsert (khuyến nghị)**: gom tối đa K record (vd 100–500) rồi `INSERT INTO ... VALUES (...),(...),... ON CONFLICT (...) DO UPDATE SET ... WHERE _hash IS DISTINCT ... AND _source_ts >=...`. **PHẢI giữ nguyên mệnh đề ON CONFLICT/OCC guard hiện tại** (đã verify GAP-02) — chỉ đổi từ 1 row → N row/statement. Trả về tổng `rows_affected` để vẫn phân biệt inserted/updated (có thể cần `RETURNING (xmax=0)` để đếm chính xác insert vs update).
- **B. COPY vào temp table + MERGE**: nhanh nhất cho khối lớn, nhưng phức tạp (temp table, MERGE/UPSERT 2 bước, transaction lớn) — over-engineer ở quy mô hiện tại.
- **C. Giữ nguyên**: chấp nhận tới khi quy mô thật sự lớn.

### Quyết định đề xuất: **A, nhưng KHÔNG làm ngay** — chờ tín hiệu quy mô
Đề xuất kỹ thuật là A. Tuy nhiên với 454 dòng hiện tại, lợi ích chưa thấy rõ còn rủi ro hồi quy OCC là thật. → **Defer**: ghi nhận thiết kế, triển khai khi (a) shadow vượt ~50k dòng, hoặc (b) đo được sync > vài giây.

### Effort / Risk
- Effort: **M** (~80–150 LOC: gom buffer, build VALUES động, đếm insert/update qua `xmax`, flush cuối batch + cuối Run; cập nhật `batchOutcome`).
- Risk: **TRUNG** — dễ làm hỏng phân biệt inserted/updated và OCC guard nếu cẩu thả. Cần test kỹ.

### Verify red→green
1. Benchmark RED: sync N dòng, đo thời gian + số round-trip (log/pg_stat_statements `calls`).
2. Áp A.
3. GREEN: cùng N dòng, thời gian giảm rõ, `calls` giảm ~K lần; **inserted/updated khớp** baseline; chạy lại lần 2 → inserted=0 (OCC vẫn chặn ghi thừa); out-of-order `_source_ts` vẫn bị bỏ qua (test như GAP-02).

---

## ADR-3 — GAP-COMP-1: Full re-scan mỗi lần sync (thiếu watermark incremental)

### Vấn đề
`lastGpayID` là biến **local trong `Run`** (`:173`), reset `0` mỗi lần trigger → mỗi cron/run-now **quét lại toàn bộ shadow table** từ pk=0. OCC `_hash IS DISTINCT` (`:552`) đảm bảo **không ghi thừa** (dòng không đổi → no-op), nên **kết quả ĐÚNG**, nhưng **chi phí scan = O(toàn bảng)** mỗi lần, bất kể chỉ vài dòng đổi.

### Điểm tinh tế (quan trọng — quyết định thiết kế)
Cursor hiện theo `_gpay_id` (pk tăng dần) chỉ bắt **INSERT** (pk mới lớn hơn). **UPDATE** dòng cũ giữ nguyên pk nhưng đổi `_source_ts` → **nếu** chuyển sang "chỉ quét pk > watermark_cũ" thì sẽ **BỎ SÓT UPDATE** = mất dữ liệu. ⇒ Full-scan hiện tại **an toàn về tính đúng**, chỉ tốn scan. Incremental thật phải key theo **`_source_ts > watermark`** (bắt cả insert lẫn update) hoặc **tiêu thụ Kafka topic shadow** (event-driven).

### Mức độ
🟠 **TRUNG (scale)** — không sai dữ liệu, nhưng không scale: shadow càng lớn, mỗi cron càng nặng; lãng phí khi cron dày.

### Phương án
- **A. Watermark theo `_source_ts` (khuyến nghị khi cần scale)**: lưu `last_source_ts` per `master_binding` (cột mới ở `master_binding`/`transmute_runtime`). Mỗi Run: `WHERE _source_ts > last_source_ts` (giữ keyset pk làm pagination phụ trong Run). Bắt cả insert & update. **Lưu ý**: cần `_source_ts` đơn điệu & chỉ commit watermark sau khi batch ghi thành công (tránh mất dòng nếu lỗi giữa chừng) → watermark = min(_source_ts chưa chắc chắn) hoặc commit sau Run thành công.
- **B. Event-driven qua Kafka shadow topic**: worker nghe topic shadow (giống cách Source→Shadow nghe Debezium) → chỉ xử lý dòng thật sự đổi, realtime. Đúng "CDC" nhất nhưng là thay đổi kiến trúc lớn (thêm consumer, offset, ordering, dedup) — **đụng nhiều**, cần thiết kế riêng.
- **C. Giữ full-scan + OCC**: đúng & đơn giản, chấp nhận chi phí scan tới khi quy mô buộc đổi.

### Quyết định đề xuất: **C bây giờ, A khi vượt ngưỡng** — KHÔNG làm ngay
Full-scan + OCC hiện **đúng** và rẻ ở 454 dòng. A là bước nâng cấp tự nhiên khi scale; B chỉ khi cần realtime thật sự. → **Defer + đặt ngưỡng**: chuyển sang A khi shadow > ~100k dòng hoặc thời gian 1 lần sync > ~10s. **Tuyệt đối không** đổi cursor sang "pk > watermark" (sẽ sót UPDATE).

### Effort / Risk
- A: Effort **M-L** (cột watermark + migration + logic commit-after-success + xử lý `_source_ts` không đơn điệu). Risk **TRUNG-CAO** (sai watermark = **mất dữ liệu** → cần test rất kỹ, đặc biệt crash giữa Run).
- B: Effort **L** (consumer mới). Risk **CAO** (ordering/dedup/offset).

### Verify red→green (nếu làm A)
1. RED: đo dòng scanned mỗi Run = toàn bảng dù chỉ đổi 1 dòng.
2. Áp A.
3. GREEN: đổi 1 dòng → Run kế chỉ scan ~1 dòng; **chèn + sửa dòng cũ đều được bắt** (test cả 2); kill worker giữa Run → restart **không mất dòng** (watermark chỉ tiến sau khi ghi chắc); so tổng dòng master = baseline full-scan.

---

## Tổng kết quyết định & thứ tự ưu tiên

| Gap | Mức độ | Quyết định | Làm ngay? | Effort | Risk | Ngưỡng kích hoạt |
|-----|--------|-----------|-----------|--------|------|------------------|
| **SAFE-2** cache stale | 🟡 thấp (tự lành 60s) | PA-A invalidate sau Apply | **Có thể làm ngay** (nhỏ, an toàn, cải thiện UX rõ) | S | Thấp | — |
| **PERF-1** per-row write | 🟡 thấp→trung (scale) | PA-A multi-row upsert | **Defer** | M | Trung (OCC) | shadow >50k hoặc sync >vài s |
| **COMP-1** full re-scan | 🟠 trung (scale) | C now, A sau | **Defer** | M-L | Trung-cao (mất dữ liệu) | shadow >100k hoặc sync >10s |

### Khuyến nghị cho User/Brain
1. **SAFE-2**: đề xuất **duyệt làm ngay** trong phiên tới (S, risk thấp, sửa đúng kịch bản "duyệt xong sync luôn"). Có verify red→green rõ.
2. **PERF-1 & COMP-1**: **defer có chủ đích, đã đặt ngưỡng số đo** — không over-engineer ở 454 dòng. Khi chạm ngưỡng → mở lại ADR này, làm theo PA-A, verify theo mục red→green đã ghi.
3. **Tuyệt đối không**: đổi cursor sang "pk > watermark" (sót UPDATE = mất dữ liệu) — đây là cái bẫy chính của COMP-1.

> Không có mục nào là **bug chặn test tay**. Feature sẵn sàng cho User test tay (xem `06_test_readiness_2026-06-05.md`). ADR này để quyết định nâng cấp kiến trúc, không phải điều kiện để test.
