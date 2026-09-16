# Audit Report V3 — Delta QC (2026-09-03)

**Agent**: Gemini  
**Timestamp**: 2026-09-03T09:05:00+07:00  
**Workspace**: FixKafkaConsumerOplogUpdate20260828  
**Mode**: QC V3 Delta — Focus: Coverage Gaps, Runtime Verification, Tier B Parity  

---

## 1. MỤC TIÊU

QC V1+V2 đã cover:
- Code correctness từng dòng ✅
- Call site parity (Execute vs executeSegmentB) ✅
- Trace ID propagation chain ✅
- Plan vs Implementation diff ✅

**V3 delta focus:**
- Unit test coverage cho deduplicatePhantomDrift
- Tất cả callers của diffIDTs ngoài engine V4
- Runtime verification — services đang chạy có compile fix chưa?

---

## 2. FINDINGS

### QC3-1: 🔴 HIGH — RUNTIME BINARY CŨ — SERVICES CHƯA CÓ QC1 FIXES

**Evidence:**
- Processes started: `Fri Aug 28 15:51:21 +07 2026` (ps aux + date -r 1787907081)
- QC1 fixes applied: `2026-08-28T15:53-15:55 ICT` (08:53-08:55 UTC from tool timestamps)
- Services restart **2 phút TRƯỚC** QC1 fixes → binary CHƯA compile QC1 changes

**Impact:**
- `executeSegmentB()` CHƯA gọi `deduplicatePhantomDrift()` → Segment B phantom drift vẫn bị double-count
- `TriggerHeal` handler CHƯA return `trace_id` → heal toast hiển thị FE random hex
- `useHealMutation` return `trace_id` nhưng API chưa gửi → luôn undefined → fallback FE trace (graceful but wrong)

**Fix:** Anh cần restart cả 2 services:
```bash
# Terminal 1 (centralized-data-service)
Ctrl+C → make run

# Terminal 2 (cdc-cms-service)  
Ctrl+C → make run
```

### QC3-2: 🟡 MEDIUM — NO UNIT TESTS cho deduplicatePhantomDrift()

**Evidence:**
```bash
grep -r "deduplicatePhantomDrift\|PhantomDrift\|phantomDrift" *_test.go
# → 0 results
```

**Impact:** Không có regression test. Nếu ai đó refactor StaleIDsPayload hoặc sửa accumulation logic → phantom drift bug quay lại âm thầm.

**Recommendation:** Cần viết ít nhất 3 test cases:
1. No phantom (disjoint sets) → nothing moves
2. Full phantom (identical sets) → all become Mismatched
3. Partial phantom (subset overlap) → split correctly

### QC3-3: 🟡 MEDIUM — Tier B legacy engine (recon_tier_b.go) CŨNG THIẾU dedup

**Evidence:**
- `recon_tier_b.go` L233 và L431: gọi diffIDTs → accumulate missingFromMaster/missingFromShadow
- L257: `handle.mismatches = len(missingFromMaster) + len(missingFromShadow) + len(mismatchedIDs)`
- KHÔNG có deduplicatePhantomDrift call

**Impact:**
- Tier B chạy background (reconcile cycle mỗi 1 phút) cho Shadow→Master
- Phantom drift sẽ double-count trong smoke reports
- User ÍT thấy (CMS hiển thị V4 engine results chủ yếu) nhưng smoke report metrics sẽ sai

**Verdict:** MEDIUM — không ảnh hưởng user flow trực tiếp, nhưng metrics bị nhiễu. Fix khi có capacity.

### QC3-4: 🟢 INFO — FE `npm run dev` vẫn OK vì Vite hot-reload

FE chạy `npm run dev` (Vite) → hot-reload → mọi thay đổi TS/TSX đã tự động apply.
→ FE fixes (useReconStatus.ts, DataIntegrity.tsx) ĐÃ ACTIVE ✅

---

## 3. ĐỐI CHIẾU VỚI V1+V2

| V1/V2 Claim | V3 Verify | Status |
|---|---|---|
| "Build PASS cho cả 2 Go services" | ✅ Build pass, nhưng services CHƯA restart | 🔴 RUNTIME GAP |
| "Tất cả HIGH bugs đã fix" | Code fix ĐÃ có, nhưng chưa deploy (local restart) | 🔴 DEPLOY GAP |
| "0 suy diễn" | Confirmed ✅ | ✅ |
| "Tier B legacy không affected" | **SAI** — Tier B CÓ cùng bug nhưng lower visibility | 🟡 UNDERESTIMATED |

---

## 4. SELF-IMPROVEMENT

### Lesson mới
```
### [2026-09-03] Build OK ≠ Deployed — phải verify runtime binary timestamp
- **Global Pattern:** Khi fix code và chạy `go build`, PHẢI verify services đang chạy 
  đã restart với binary mới. `go run` compile on-demand — nếu process started TRƯỚC fix, 
  binary CŨ vẫn chạy.
- **Bối cảnh (Trigger):** Services restart 15:51, fixes apply 15:53 → 2-minute gap → 
  runtime CHƯA CÓ fixes suốt 6 ngày.
- **Root Cause:** Audit V1+V2 verify `go build` nhưng không verify running process 
  binary timestamp.
- **Fix/Correct Flow:** Sau fix, LUÔN nhắc user restart services. Verify bằng 
  `ps aux | grep service` → so sánh start time vs fix time.
- **Tags:** #runtime-verification #build-vs-deploy #binary-staleness
```

---

## 5. TỔNG KẾT

| Priority | Finding | Action |
|---|---|---|
| 🔴 HIGH | Services chạy binary CŨ (trước QC1 fixes) | **Restart cả 2 services NGAY** |
| 🟡 MEDIUM | Không có unit test cho deduplicatePhantomDrift | Viết 3 test cases |
| 🟡 MEDIUM | Tier B legacy engine thiếu dedup | Fix khi có capacity |
| 🟢 INFO | FE đã hot-reload OK | No action |
