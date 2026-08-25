# Audit Report — fix-batch-transform-transmute-trigger

**Ngày audit:** 2026-08-24T13:10:55+07:00
**Auditor:** Agent (Self-Adversarial QA)
**Phương pháp:** Đọc file thực tế → so sánh plan → cross-check consumer + peers → adversarial review

---

## PHẦN 1 — KIỂM TRA CODE ĐÃ THÊM

### 1.1 Method `publishTransmuteTrigger` (lines 527-559)

#### ❌ FINDING-01 — THIẾU FIELD `shadow_connection_key` [SEVERITY: MEDIUM — PHẢI FIX]

**Bằng chứng — Consumer HandleTransmuteShadow (line 107):**

```go
if req.ShadowBindingCode != "" || req.ShadowSchema != "" || req.ShadowConnectionKey != "" {
    // nhánh này được trigger vì ta GỬI shadow_schema
    masterTables = ListMasterTablesByShadowIdentity(shadowTable, shadowSchema, shadowConnectionKey, ...)
} else {
    masterTables = ListMasterTablesByShadowTable(ctx, shadowTable)
}
```

Code ta GỬI shadow_schema → consumer vào nhánh ListMasterTablesByShadowIdentity.
Nhưng ta THIẾU shadow_connection_key:"default" → query với key="" → có thể không match binding → silent skip transmute.

**Peer batch_buffer_fanout.go:** gửi "shadow_connection_key": "default" ✅
**Peer sinkworker/worker.go:** gửi "shadow_connection_key": "default" ✅
**Code ta:** THIẾU field này ❌

**Payload cần sửa:**
```go
json.Marshal(map[string]any{
    "shadow_table":          tableName,
    "shadow_schema":         schemaName,
    "shadow_connection_key": "default",
    "triggered_by":          "batch-transform",
    "correlation_id":        fmt.Sprintf("transform-%s-%d", tableName, time.Now().UnixNano()),
})
```

#### ❌ FINDING-02 — THIẾU `correlation_id` [SEVERITY: LOW]

Cả 2 peer đều gửi correlation_id để trace distributed request.
Ta không gửi → mất traceability. Không gây lỗi nhưng vi phạm observability pattern.

#### ✅ FINDING-03 — Không có `_source_ids` là ĐÚNG

HandleTransmute line 206: if len(req.SourceIDs) > 0 { debouncer... return }
Không có source_ids → full-sync transmute (24h timeout goroutine). Đây là behavior đúng.

#### ✅ FINDING-04 — PublishMsg + InjectNATSHeader là ĐÚNG
Pattern đúng theo batch_buffer_fanout.go.

#### ✅ FINDING-05 — Guard h.NatsConn == nil là ĐÚNG
Test dùng natsConn=nil → early return → không panic.

---

### 1.2 Điểm gọi trong runTransformJob

✅ Unchunked branch (line 250): sau finishJob COMPLETED, trước return.
✅ Chunked branch (line 391): sau finishJob COMPLETED, cuối hàm.
✅ KHÔNG gọi khi FAILED (line 336), CANCELLED (line 271), skipped/error (publishAndFinishJob).

---

### 1.3 schemaName và pureTable truyền đúng không

schemaName = metadata.ResolveTargetSchema(h.metadataRegistry, targetTable) — đúng
pureTable = targetTable[idx+1:] — đúng, không có schema prefix

#### ❌ FINDING-06 — schemaName rỗng khi metadataRegistry=nil [SEVERITY: LOW]

Khi metadataRegistry=nil, schemaName có thể = "".
Gửi shadow_schema="" → consumer vào nhánh identity-aware với schema rỗng.
Nhưng query dùng COALESCE(NULLIF('',''), sb.shadow_schema) → match bất kỳ schema → không gây silent skip, nhưng có thể match sai binding nếu nhiều bindings có cùng shadow_table khác schema.
Rủi ro thấp trong thực tế nhưng cần theo dõi.

---

## PHẦN 2 — KIỂM TRA QUÁ TRÌNH IMPLEMENT

### 2.1 Edit Tool gây duplicate code — ĐÃ TỰ PHÁT HIỆN & SỬA ✅

Lần edit đầu tiên TargetContent không khớp chính xác (tool matching lỗi) → insert code sai vị trí → duplicate block.
Agent phát hiện qua view_file ngay sau edit → chạy replace cleanup → file cuối sạch.
KẾT QUẢ CUỐI: Clean, không duplicate. ✅
LESSON: Phải đọc file verify sau mỗi lần edit trước khi chuyển sang edit tiếp theo.

### 2.2 Báo cáo "14 tests PASS" — CHÍNH XÁC về regression, nhưng COVERAGE = 0% cho code mới

14 tests là toàn bộ package shadow, không phải chỉ batch_transform.
NatsConn=nil trong tất cả tests → publishTransmuteTrigger luôn early return.
Không có test case nào verify NATS publish được gọi với payload đúng.
Báo cáo "tests pass" là đúng nhưng KHÔNG nên hiểu là code mới đã được test.

### 2.3 Build pass — XÁC THỰC ✅

go build ./internal/... ./cmd/... ./pkgs/... exit 0, chạy thực tế với BypassSandbox=true.
Lỗi ở docs/ và scratch/ là pre-existing, đã verify không liên quan.

---

## PHẦN 3 — PATTERN COMPLIANCE TABLE

| Pattern | batch_buffer_fanout | Code ta | Status |
|---------|---------------------|---------|--------|
| Subject cdc.cmd.transmute-shadow | OK | OK | PASS |
| PublishMsg có header | OK | OK | PASS |
| InjectNATSHeader | OK | OK | PASS |
| shadow_table | OK | OK | PASS |
| shadow_schema | OK | OK | PASS |
| shadow_connection_key: "default" | OK | THIẾU | FAIL |
| correlation_id | OK | THIẾU | WARN |
| Fire-and-forget | OK | OK | PASS |
| Guard natsConn == nil | OK | OK | PASS |
| Không trigger khi FAILED/CANCELLED | N/A | OK | PASS |

---

## PHẦN 4 — FINDINGS SUMMARY

| ID | Severity | Mô tả | Action |
|----|----------|--------|--------|
| FINDING-01 | MEDIUM | Thiếu shadow_connection_key → consumer silent-skip | PHẢI FIX NGAY |
| FINDING-02 | LOW | Thiếu correlation_id → mất distributed trace | Nên fix |
| FINDING-03 | N/A | Không có _source_ids → đúng, full-sync | OK |
| FINDING-04 | N/A | PublishMsg + header → đúng | OK |
| FINDING-05 | N/A | Guard nil NatsConn → đúng, test-safe | OK |
| FINDING-06 | LOW | schemaName rỗng edge case khi metadataRegistry=nil | Track |
| PROCESS-01 | INFO | Edit gây duplicate, tự sửa | Lesson ghi nhớ |
| PROCESS-02 | INFO | 0% test coverage publishTransmuteTrigger | Cần thêm test |

---

## PHẦN 5 — KẾT LUẬN

**Trạng thái tổng thể:** ⚠️ CẦN FIX FINDING-01 TRƯỚC KHI DEPLOY

Code đúng về: vị trí call, subject NATS, guard nil, PublishMsg+Header pattern, không trigger khi FAILED/CANCELLED.
Code SAI về: thiếu shadow_connection_key:"default" → consumer có thể route sang ListMasterTablesByShadowIdentity với connection_key rỗng → silent-skip transmute.

**Fix nhỏ, impact lớn:** Chỉ cần thêm 2 fields vào map[string]any trong method publishTransmuteTrigger.
Import "time" đã có sẵn (line 16 của file).
