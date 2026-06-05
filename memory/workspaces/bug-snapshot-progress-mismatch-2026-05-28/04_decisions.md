# 04_decisions — ADR Bug Snapshot Progress Mismatch

## ADR-001 — Cursor exhaustion: KHÔNG dùng `len(batch) < BatchSize`

**Context**: `coll.Find` với `SetLimit(BatchSize)` + `SetReadPreference(SecondaryPreferred)` có thể trả partial khi secondary replication lag, dù collection vẫn còn data.

**Decision**: Chỉ dựa vào `len(batch) == 0` (đã có ở line 383-385) để quyết định exhausted. Bỏ block line 553-555.

**Consequence**:
- (+) Hết bug early-exit.
- (+) Vòng lặp tự nhiên sẽ exit khi cursor thực sự cạn (batch tiếp theo `[]`).
- (-) Thêm 1 round-trip Find thừa khi batch cuối thật sự là tail (vd: 2981 docs cho lần cuối). Cost: ~150ms, negligible.

**Alternative rejected**:
- Đổi sang `cursor.Next()` loop: overkill, refactor lớn, không cần thiết.
- Đổi `SetReadPreference(Primary)`: tăng tải primary, risk khác. Defer.

---

## ADR-002 — Completeness threshold = 0.99 (configurable)

**Context**: `EstimatedDocumentCount` không exact (Mongo `count` collection stats có sai số ~1% trên collection lớn + write-heavy). Strict equality `rowsTotal == totalRows` sẽ false-positive trip guard trên collection bình thường.

**Decision**: Default `snapshotCompletenessThreshold = 0.99` (hardcoded constant, có thể đổi sang `os.Getenv` sau).

**Consequence**:
- (+) Absorb estimate skew + concurrent insert race.
- (-) Vẫn cho phép miss ~1% data. Trên 177,980 docs → tolerance 1,780 docs. Trade-off acceptable cho fix urgency.

**Alternative rejected**:
- Strict 1.00: nhiều false trip.
- Threshold 0.95: quá lỏng, không catch được bug 23% như hôm nay (23.23% << 95%).

---

## ADR-003 — Pause = terminal cho snapshot run, KHÔNG resume trong cùng goroutine

**Context**: Hiện tại `break` thoát loop nhưng vẫn chạy final flush + markProgressDone. Hành vi đúng: pause là terminal cho run hiện tại; resume sẽ tạo run mới từ `last_seen_id`.

**Decision**: `break` → `return nil` ngay sau khi UPDATE status=paused. Loại bỏ "wait-for-resume in-process" pattern.

**Consequence**:
- (+) Resume idempotent: snapshot runner restart đọc `last_seen_id`, chạy tiếp.
- (+) Không cần atomic.Bool poll wait.
- (-) Resume cần CMS publish lại `cdc.cmd.snapshot.v2` (đã có flow này).

**Alternative rejected**:
- Spin loop `for isPaused.Load()` chờ resume: tốn goroutine + có thể leak nếu user không resume.

---

## ADR-004 — Metric path: tăng `snapshot_partial_done_total` chỉ khi GUARD trip

**Context**: Có 3 reason guard trip: `cursor_short`, `pause_fallthrough`, `persist_mismatch`. Mỗi reason có trigger khác nhau.

**Decision**: Chỉ tăng metric khi `markProgressDone` BỊ guard reject. Pause thường không phải lỗi → không tăng metric.

**Consequence**:
- (+) Metric chỉ alert khi thực sự có bug, không noise.
- Alert rule: `rate(cdc_snapshot_partial_done_total[5m]) > 0` → page on-call.

**Alternative rejected**:
- Tăng metric mỗi pause: ồn, false alert khi user chỉ pause để bảo trì.

---

## ADR-005 — Test strategy: mock Mongo cursor batches, không integration full

**Context**: Synthetic test 177k docs + secondary lag rất phức tạp.

**Decision**: Mock Mongo cursor trả `[][]int` predefined batches để simulate "partial mid-stream". Test guard logic + pause path bằng unit test với SQLite in-memory.

**Consequence**:
- (+) Fast test suite (~1s).
- (+) Deterministic, không phụ thuộc Mongo replica set.
- (-) Không catch được edge case replication lag thực — bù lại bằng metric + alert ở production.

---

## ADR-006 — Lesson cross-reference: ENFORCE invariant ở 1 NƠI DUY NHẤT

**Context**: Bug 2026-05-27 fix layer Flush; bug 2026-05-28 lại lộ ra layer cursor + pause + markProgressDone. Whack-a-mole.

**Decision**: Invariant "status=done IFF rows_processed >= total_rows * 0.99" enforce ở **`markProgressDone`** — bottleneck duy nhất transition status sang `done`. Mọi caller phải đi qua đây.

**Consequence**:
- (+) Không phụ thuộc cursor / pause / flush behaviour — guard ở cuối line là defense-in-depth.
- (+) Tương lai có thêm root cause D, E, F vẫn bị guard catch.
- Lesson global: "Bug fixed ở layer N có thể trồi sang layer M+1; enforce invariant ở edge (terminal transition), không phải intermediate" → ghi vào `lessons.md` global pattern.

---

## ADR-007 — KHÔNG đổi `SetReadPreference(SecondaryPreferred)`

**Context**: Reading from secondary là design quyết định (giảm tải primary trong write-heavy CDC). Đổi sang Primary sẽ tăng tải primary, risk khác.

**Decision**: Giữ nguyên `SecondaryPreferred`. Fix bug ở consumer side (cursor exhaustion logic), không ở source.

**Consequence**:
- Snapshot vẫn benefit từ secondary throughput.
- Replication lag được absorb bởi `len(batch) == 0` check.
