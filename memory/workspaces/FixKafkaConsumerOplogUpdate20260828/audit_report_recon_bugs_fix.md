# Audit Report — Session 2026-08-28 (Bug #1 Phantom Drift + Bug #2 Trace ID)

**Agent**: Gemini  
**Timestamp**: 2026-08-28T15:55:00+07:00  
**Workspace**: FixKafkaConsumerOplogUpdate20260828  
**Mode**: QC Audit — Critical Review  

---

## 1. KIỂM KÊ TOÀN BỘ FILE ĐÃ THAY ĐỔI

| # | File | Service | Thay đổi |
|---|---|---|---|
| 1 | bridge_handler.go | centralized-data-service | Override pgPKField → _source_id cho V2 shadow |
| 2 | recon_stream_bucket_engine.go | centralized-data-service | Thêm deduplicatePhantomDrift() + call sites |
| 3 | reconciliation_handler_commands.go | cdc-cms-service | Return OTel trace_id trong TriggerCheck response |
| 4 | reconciliation_handler_heal.go | cdc-cms-service | Return OTel trace_id trong TriggerHeal response |
| 5 | useReconStatus.ts | cdc-cms-web | Return trace_id từ cả 2 mutations |
| 6 | DataIntegrity.tsx | cdc-cms-web | Dùng API trace_id cho cả check + heal toast |

---

## 2. AUDIT TỪNG THAY ĐỔI — TƯ DUY PHẢN BIỆN

### 2.1 bridge_handler.go — V2 _source_id override

**Dòng 345-348:**
```go
if resolved.pgPKField != "_source_id" {
    if _, hasSourceID := schema.Columns["_source_id"]; hasSourceID {
        resolved.pgPKField = "_source_id"
    }
}
```

**✅ Đánh giá: ĐÚNG**
- Logic: Nếu shadow table CÓ cột _source_id → luôn ưu tiên dùng nó làm conflict target. Vì V2 schema chỉ có partial unique index trên (_source_id) WHERE NOT _deleted.
- Rủi ro: Nếu table KHÔNG phải V2 nhưng vẫn có cột _source_id (legacy) → override sai? KIỂM TRA: Tất cả tables có _source_id đều là V2 (do sinkworker tạo). Các legacy table KHÔNG có cột này. → AN TOÀN.

---

### 2.2 recon_stream_bucket_engine.go — deduplicatePhantomDrift()

**PHÂN TÍCH TỪNG DÒNG:**

| Dòng | Logic | Đúng? | Lý do |
|---|---|---|---|
| L49 | Early return nếu 1 list rỗng | ✅ | Không thể có phantom nếu thiếu 1 chiều |
| L52 | Build set từ MissingFromSrc | ✅ | O(n) lookup |
| L58-66 | Scan MissingFromDest, nếu ID cũng ở MissingFromSrc → Mismatched | ✅ | Phantom = tồn tại cả 2 phía nhưng timestamp khác |
| L69-73 | Loại bỏ phantom IDs khỏi MissingFromSrc | ✅ | Dùng phantomSet đã build ở trên |
| L76-77 | Assign lại slices | ✅ | |

**Edge case kiểm tra:**
1. Duplicate IDs trong cùng list: Lý thuyết có thể gây double-add vào Mismatched. NHƯNG mỗi record chỉ có 1 timestamp → nằm trong đúng 1 sub-window → không duplicate. AN TOÀN.
2. Nil slices: len(nil) = 0 → early return ✅

---

### 2.3 Call sites cho deduplicatePhantomDrift()

**🔴 PHÁT HIỆN QC (ĐÃ FIX):**

| Call site | Có gọi? | Trạng thái |
|---|---|---|
| Execute() (Segment A) L184-186 | ✅ | Đã có từ fix ban đầu |
| executeSegmentB() L725-731 | ✅ | **PHÁT HIỆN THIẾU → ĐÃ BỔ SUNG** |
| ExecuteSegment("both") L462-486 | N/A | Gọi Execute() + executeSegmentB(), dedup đã xử lý bên trong |

---

### 2.4 CMS TriggerCheck — trace_id

```go
var traceID string
if sc := oteltrace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
    traceID = sc.TraceID().String()
}
```

**Verification:**
- ctx giữ nguyên span context (WithMetadata chỉ dùng context.WithValue) ✅
- SpanFromContext lấy span từ HTTP middleware → TraceID = HTTP request trace ✅
- Dispatch tạo child span nhưng KHÔNG modify caller ctx (Go context immutable) ✅
- Graceful degradation: nếu no span → traceID="" → FE fallback ✅

---

### 2.5 TriggerHeal — trace_id

**🔴 PHÁT HIỆN QC (ĐÃ FIX):**
Cùng pattern với TriggerCheck nhưng ban đầu KHÔNG được fix. Audit phát hiện → đã bổ sung.

---

### 2.6 FE: useReconStatus.ts + DataIntegrity.tsx

**Check mutation return type:** { job_id?: string; trace_id?: string } — cả 2 mutations ✅
**Check toast usage:** res?.trace_id || trace.traceId — cả check + heal ✅ (heal là QC fix)

---

## 3. KIỂM TRA SUY DIỄN / BÁO CÁO LÁO

### 3.1 Claim: "Bug #1 root cause = phantom drift deduplication"
- Có evidence code chứng minh? ✅ — diffIDTs() gọi per sub-window, staleAcc accumulates
- Có reproduce scenario? ✅ — User xác nhận edit records
- Counting logic verified: 2 phantom × 2 = 4 ✅
- **KẾT LUẬN: ĐÚNG, không suy diễn**

### 3.2 Claim: "Bug #2 root cause = FE generates random trace ID"
- FE code evidence: createActionTrace dòng 8 = random hex ✅
- CMS Service evidence: OTel span tạo TraceID riêng ✅
- Response evidence: Response cũ chỉ có job_id ✅
- **KẾT LUẬN: ĐÚNG, không suy diễn**

### 3.3 Claim: "Bug #3 — Code PHẢI phát hiện drift cho mọi scan range"
- Phân tích tĩnh (static analysis), chưa test thực tế
- **ĐÁNH GIÁ: Hợp lý nhưng chưa confirm 100%. KHÔNG BÁO LÁO vì đã ghi rõ "lý thuyết"**

---

## 4. QC FINDINGS — BUGS THIẾU TỪ FIX BAN ĐẦU

| # | Severity | Finding | Status |
|---|---|---|---|
| QC-1 | 🔴 HIGH | executeSegmentB thiếu deduplicatePhantomDrift() — cùng counting pattern | **ĐÃ FIX** |
| QC-2 | 🔴 HIGH | TriggerHeal handler thiếu trace_id trong response | **ĐÃ FIX** |
| QC-3 | 🔴 HIGH | FE useHealMutation không return trace_id | **ĐÃ FIX** |
| QC-4 | 🔴 HIGH | FE heal toast dùng FE trace thay vì API trace_id | **ĐÃ FIX** |
| QC-5 | 🟡 LOW | deduplicatePhantomDrift chưa handle duplicate IDs trong cùng list | **ACCEPTED** |

---

## 5. VÒNG LẶP PHẢN TỈNH (Self-Improvement Loop)

### 5.1 Lỗi quy trình mắc phải

**Pattern: Khi fix 1 bug ở hàm A, KHÔNG kiểm tra hàm B có cùng pattern/bug tương tự hay không.**

- Lần 1: Fix deduplicatePhantomDrift() trong Execute() nhưng QUÊN executeSegmentB()
- Lần 2: Fix trace_id trong TriggerCheck nhưng QUÊN TriggerHeal

**Root Cause quy trình**: Không chạy "sister function scan" — khi fix 1 hàm, BẮT BUỘC grep toàn bộ repo cho các hàm sibling có cùng pattern.

### 5.2 Bài học mới

```
### [2026-08-28] Fix hàm A nhưng quên hàm B có cùng pattern → phantom regression
- **Global Pattern:** Khi fix bug ở hàm [A], PHẢI grep toàn repo cho pattern tương tự ở các hàm [B], [C]...
- **Bối cảnh (Trigger):** Fix Execute() nhưng miss executeSegmentB(). Fix TriggerCheck nhưng miss TriggerHeal.
- **Root Cause:** Không thực hiện "sibling function grep" sau khi fix.
- **Fix/Correct Flow:** Sau khi fix hàm X, chạy grep -rn 'cùng_pattern' ./module/ để tìm các hàm sibling.
- **Tags:** #sister-function-scan #pattern-parity #regression-prevention
```

---

## 6. BUILD VERIFICATION

| Service | Command | Result |
|---|---|---|
| centralized-data-service | go build ./cmd/... ./internal/... | ✅ PASS |
| cdc-cms-service | go build ./cmd/... ./internal/... | ✅ PASS |

---

## 7. CHECKLIST CUỐI PHIÊN

- [x] Mọi thay đổi code đã audit từng dòng
- [x] Kiểm tra suy diễn/báo cáo láo → 0 trường hợp
- [x] Phát hiện 4 bugs thiếu sót → tất cả đã fix
- [x] Build verification pass cho cả 2 Go services
- [x] Bài học mới đã draft
- [x] Audit report đã lưu file vật lý
