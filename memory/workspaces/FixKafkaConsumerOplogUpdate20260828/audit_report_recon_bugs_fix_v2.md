# Audit Report V2 — Deep QC Lần 2 (2026-09-03)

**Agent**: Gemini  
**Timestamp**: 2026-09-03T09:00:00+07:00  
**Workspace**: FixKafkaConsumerOplogUpdate20260828  
**Mode**: QC Audit V2 — Adversarial Review (Second Pass)  
**Mục tiêu**: Re-verify toàn bộ, KHÔNG tin audit lần 1. Tìm bugs còn sót.

---

## 1. TÓM TẮT TOÀN BỘ FILE ĐÃ THAY ĐỔI (6 files, 3 services)

| # | File | Service | Thay đổi | Risk |
|---|---|---|---|---|
| 1 | bridge_handler.go L340-349 | centralized-data-service | Override pgPKField → _source_id cho V2 shadow | MEDIUM |
| 2 | recon_stream_bucket_engine.go L44-78, L184-186, L726-730 | centralized-data-service | deduplicatePhantomDrift() + 2 call sites (Execute + executeSegmentB) | HIGH |
| 3 | reconciliation_handler_commands.go L15, L85-96 | cdc-cms-service | Import oteltrace + return OTel trace_id in TriggerCheck | LOW |
| 4 | reconciliation_handler_heal.go L14, L66-72 | cdc-cms-service | Import oteltrace + return OTel trace_id in TriggerHeal | LOW |
| 5 | useReconStatus.ts L191, L249, L305, L339 | cdc-cms-web | Return trace_id từ cả 2 mutations (check + heal) | LOW |
| 6 | DataIntegrity.tsx L312, L332 | cdc-cms-web | Dùng API trace_id (OTel) với fallback FE-generated | LOW |

---

## 2. AUDIT TỪNG THAY ĐỔI — TƯ DUY ADVERSARIAL

### 2.1 deduplicatePhantomDrift() — LOGIC CORRECTNESS

**Attack vector 1: Race condition between sub-windows**
- `staleAcc` (StaleIDsPayload) accumulates across sub-windows trong `drillSubWindows()`
- `drillSubWindows` chạy SEQUENTIAL (not concurrent) — L389-445 dùng nested for loop
- → KHÔNG có race condition ✅

**Attack vector 2: Plan vs Implementation mismatch**
- Plan dòng 70: `delete(srcSet, id)` — mutate srcSet
- Implementation dòng 63: `phantomSet[id] = true` — separate tracking set
- **Kết quả**: Semantically equivalent. Implementation KHÔNG mutate srcSet → less side-effect → BETTER design ✅
- **NHƯNG**: Audit lần 1 KHÔNG chỉ ra sự khác biệt này → **Thiếu sót audit lần 1**

**Attack vector 3: Gọi dedup TRƯỚC hay SAU counting?**
- Execute() L184-186: dedup TRƯỚC `DriftWindowCount` (L188) → ĐÚNG ✅
- executeSegmentB() L728-730: dedup TRƯỚC `DriftWindowCount` (L732) → ĐÚNG ✅
- ExecuteSegment("both") L462-486: Merge staleIDs từ A + B SAU khi mỗi bên đã dedup riêng → OK vì A và B là domains khác nhau (Source→Shadow vs Shadow→Master) → KHÔNG có cross-domain phantom ✅

**Attack vector 4: Duplicate IDs within same list**
- Mỗi record có duy nhất 1 timestamp → 1 sub-window → KHÔNG duplicate per list ✅
- Ngoại lệ lý thuyết: nếu record bị update GIỮA LÚC recon đang chạy (hot update) → timestamp thay đổi → có thể xuất hiện ở 2 windows? → Recon dùng SNAPSHOT read (SQL `WHERE ts BETWEEN ? AND ?`) → mỗi query is point-in-time → KHÔNG duplicate ✅

### 2.2 bridge_handler.go — V2 _source_id OVERRIDE

**Attack vector: V1 table có _source_id?**
- V1 tables (tạo bằng sinkworker v1 hoặc manual) KHÔNG có _source_id column
- V2 tables (sinkworker v2) LUÔN có _source_id + partial unique index
- **Proof**: sinkworker sql_builder.go CREATE TABLE pattern → V2 thêm _source_id, V1 không
- → Override AN TOÀN ✅

**Attack vector: Order of operations**
- L330-338: Auto-detect nếu config field không tồn tại → có thể resolve _id
- L345-348: Override thành _source_id nếu column exists
- Nếu config = "_id" VÀ _id EXISTS → skip auto-detect → nhưng VẪN override thành _source_id
- → ĐÚNG intent: V2 tables LUÔN dùng _source_id bất kể config ✅

### 2.3 TriggerCheck trace_id — CTX ANALYSIS

**Attack vector: ctx span ownership sau Dispatch**
- L70: `ctx := messaging.WithMetadata(c.UserContext(), ...)`
- L72: `h.bus.Dispatch(ctx, cmd)` — bên trong tạo child span, nhưng trả về (newCtx, span) LOCAL
- L87: `SpanFromContext(ctx)` — lấy span từ ctx BAN ĐẦU (parent HTTP span)
- Go context immutable → Dispatch KHÔNG modify caller's ctx → traceID = HTTP request TraceID ✅

**Attack vector: HttpTracer middleware absence**
- Verified: server.go L372 `app.Use(middleware.HttpTracer(logger))`
- HttpTracer L44-56: tạo root span + SetUserContext(ctx)
- → SpanFromContext(ctx) LUÔN có valid span ✅

**Attack vector: TraceID consistency CMS↔centralized-data-service**
- CMS Dispatch (nats_command_bus.go) injects traceparent vào NATS header
- centralized-data-service ReconJobWorker extracts traceparent từ NATS header
- OTel child spans SHARE TraceID với parent → **CONSISTENT** ✅
- FE toast hiển thị CÙNG TraceID mà SigNoz tracing dùng ✅

### 2.4 FE useReconStatus.ts + DataIntegrity.tsx

**Attack vector: `data?.trace_id` khi API error**
- Nếu API trả 500 → `mutateAsync` throws → toast KHÔNG gọi → OK ✅
- Nếu API trả 202 nhưng trace_id rỗng → `""` falsy → fallback FE trace → SAFE ✅

**Attack vector: TypeScript type safety**
- Return type `{ job_id?: string; trace_id?: string }` — optional, compatible với cả old và new API ✅

---

## 3. MỚI PHÁT HIỆN TRONG QC LẦN 2

### QC2-1: TriggerCheckAll THIẾU trace_id (🟡 LOW)
- `reconciliation_handler_commands.go` L140-145 và L166-171: TriggerCheckAll response KHÔNG có trace_id
- **Impact**: FE check-all flow dùng `message.success()` (KHÔNG dùng `showActionToast`) → KHÔNG hiển thị trace_id
- **Verdict**: LOW priority — FE không consume → không ảnh hưởng user
- **Recommendation**: Fix sau nếu FE thêm trace toast cho check-all

### QC2-2: TriggerPrune THIẾU trace_id (🟡 LOW)
- `reconciliation_handler_commands.go` L210-214: TriggerPrune response KHÔNG có trace_id
- **Impact**: Tương tự QC2-1 — FE dùng message.success, không hiển thị trace_id
- **Verdict**: LOW priority

### QC2-3: Audit lần 1 KHÔNG phát hiện Plan vs Implementation diff (🟢 INFO)
- Plan dùng `delete(srcSet, id)`, implementation dùng `phantomSet[id] = true`
- Semantic equivalent nhưng audit lần 1 không mention → thiếu depth
- **Verdict**: Không ảnh hưởng correctness, chỉ là audit coverage gap

---

## 4. ĐỐI CHIẾU VỚI IMPLEMENTATION PLAN

| Plan Item | Implemented? | Đúng spec? | Deviation |
|---|---|---|---|
| Bug #1: deduplicatePhantomDrift() | ✅ | ✅ | Implementation dùng phantomSet thay vì delete(srcSet) — better |
| Bug #1: Call site in Execute() | ✅ | ✅ | Exact match |
| Bug #1: Call site in executeSegmentB() | ✅ | N/A (plan không mention) | QC1 đã bổ sung |
| Bug #2: CMS return trace_id | ✅ TriggerCheck | ✅ | |
| Bug #2: CMS return trace_id TriggerHeal | ✅ | N/A (plan không mention) | QC1 đã bổ sung |
| Bug #2: FE dùng API trace_id | ✅ check + heal | ✅ | |
| Bug #3: Audit only (no code change) | ✅ | ✅ | Plan kết luận "lý thuyết OK", verified ✅ |

---

## 5. KIỂM TRA SUY DIỄN / BÁO CÁO LÁO — LẦN 2

| # | Claim | Evidence | Verdict |
|---|---|---|---|
| 1 | "dedup logic ĐÚNG" | Line-by-line code walk + edge case analysis | ✅ Có evidence |
| 2 | "ctx giữ span sau Dispatch" | Go context immutable spec + WithMetadata chỉ WithValue | ✅ Có evidence |
| 3 | "HttpTracer tạo span" | server.go L372 + http_tracer.go L44-56 | ✅ Có evidence |
| 4 | "V1 tables không có _source_id" | sinkworker sql_builder.go pattern | ✅ Có evidence |
| 5 | "TraceID consistent CMS↔centralized" | OTel child span share TraceID + NATS injection | ✅ Có evidence |
| 6 | "Audit lần 1: 0 suy diễn" | Plan vs Implementation diff không mention | 🟡 MINOR omission |

**KẾT LUẬN**: KHÔNG có báo cáo láo. 1 thiếu sót nhỏ (plan vs impl diff) đã bổ sung ở QC2-3.

---

## 6. VÒNG LẶP PHẢN TỈNH — SELF-IMPROVEMENT

### 6.1 Từ QC lần 1 đã học
- ✅ "Sister function scan" — đã áp dụng: scan TriggerCheckAll, TriggerPrune

### 6.2 Bài học mới từ QC lần 2
```
### [2026-09-03] Audit phải so sánh Plan vs Implementation chi tiết — không chỉ "đúng ý"
- **Global Pattern:** Khi audit, PHẢI đối chiếu TỪNG DÒNG plan vs implementation. 
  Semantic equivalent ≠ identical. Phải ghi rõ deviation cho traceability.
- **Bối cảnh (Trigger):** Plan dùng delete(srcSet), impl dùng phantomSet — cả 2 đúng 
  nhưng audit lần 1 không mention.
- **Root Cause:** Audit lần 1 focus "kết quả đúng" thay vì "implementation có đúng plan".
- **Fix/Correct Flow:** Mở implementation_plan.md song song với code khi audit. 
  Mỗi section trong plan PHẢI có 1 row trong bảng đối chiếu.
- **Tags:** #audit-depth #plan-vs-impl #traceability
```

---

## 7. TỔNG KẾT FINAL

| Category | QC1 Findings | QC2 New Findings | Total |
|---|---|---|---|
| 🔴 HIGH — đã fix | 4 (segmentB dedup, heal handler, heal mutation, heal toast) | 0 | 4 |
| 🟡 LOW — noted | 1 (duplicate IDs edge) | 2 (CheckAll, Prune thiếu trace_id) | 3 |
| 🟢 INFO | 0 | 1 (plan vs impl diff) | 1 |
| Suy diễn/Láo | 0 | 0 | 0 |

**BUILD STATUS**: ✅ cả centralized-data-service + cdc-cms-service build PASS.

**VERDICT**: Code changes CORRECT. Không còn bug HIGH. 3 LOW items không ảnh hưởng user (FE không consume trace_id cho những flows đó). Audit lần 1 đã fix tất cả vấn đề thực sự, lần 2 chỉ bổ sung depth.
