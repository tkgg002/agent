# 09_tasks_solution — Rationale + Alternative Rejected

## Solution overview

Patch **3 root cause** trong 1 PR HOLISTIC, với invariant guard ở **edge** (`markProgressDone`) — không patch lẻ tẻ từng layer.

---

## S1 — Bỏ early-exit `len(batch) < BatchSize`

### Lý do chọn
Cursor exhaustion đã có check `len(batch) == 0` ở line 383-385 (đầu mỗi iteration). Block ở line 553-555 là tối ưu hóa sai — giả định Mongo Find trả full BatchSize khi còn data, nhưng `SecondaryPreferred` + replication lag phá vỡ giả định này.

### Alternative rejected
| Alternative | Lý do reject |
|---|---|
| Đổi sang `cursor.Next()` loop | Refactor lớn (~50 LOC). Bug chỉ là 1 dòng sai → fix 1 dòng. |
| Track `rowsFetched` so với `totalRows` thay vì `len(batch)` | Phụ thuộc estimate → vẫn có sai số. Check `len == 0` đã đủ và deterministic. |
| Đổi `SetReadPreference(Primary)` | Tăng tải primary; risk khác. Defer (ADR-007). |

### Risk + Mitigation
- **Risk**: Vòng lặp Find thừa 1 round-trip ở batch cuối (~150ms). **Mitigation**: Negligible cost so với fix 23% data miss bug.

---

## S2 — Pause `break` → `return nil`

### Lý do chọn
`break` chỉ thoát vòng `for`, không thoát hàm. Code tiếp tục chạy line 561 (final flush) → line 569 (markProgressDone). Kết quả: status bị ghi đè paused → done. Sửa thành `return nil` thoát hàm ngay.

### Alternative rejected
| Alternative | Lý do reject |
|---|---|
| Thêm flag `wasPaused bool` + check trước `markProgressDone` | Thừa state. `return nil` clean hơn. |
| Spin loop `for isPaused.Load()` chờ resume | Tốn goroutine + leak risk. Pause = terminal, resume = new run. |
| Lift `break` → call `return r.handlePause(ctx, progressID)` helper | Over-engineer cho 1 dòng. |

### Risk + Mitigation
- **Risk**: Resume cần CMS publish lại `cdc.cmd.snapshot.v2`. **Mitigation**: Flow này đã có sẵn; `last_seen_id` checkpointed; ADR-003 ghi rõ semantics "pause = terminal for current run".

---

## S3 — `markProgressDone` guard completeness

### Lý do chọn (key decision — defense in depth)
Bug 2026-05-27 fix layer Flush; bug 2026-05-28 lại lộ ra ở cursor + pause + markProgressDone. **Whack-a-mole pattern**. Giải pháp: enforce invariant `status=done IFF rows_processed >= total_rows * 0.99` ở **terminal transition point** (`markProgressDone`) — bottleneck duy nhất.

Caller path nào (cursor early-exit, pause fall-through, flush error...) đến đây đều bị guard catch.

### Alternative rejected
| Alternative | Lý do reject |
|---|---|
| Guard ở mỗi caller (snapshot_runner sau loop) | Duplicate check; nếu thêm caller mới sẽ quên. Edge enforcement tốt hơn. |
| Hard threshold = 1.00 | False trip do EstimatedDocumentCount sai số ±1%. |
| Threshold = 0.95 | Không catch được bug 23% (vẫn vượt 5% threshold nếu data lệch nhỏ). 0.99 chặt vừa. |
| Đổi `EstimatedDocumentCount` thành `CountDocuments` exact | `CountDocuments` chạy COLLSCAN trên 177k+ docs → 30s+ overhead. Tradeoff không đáng. |

### Risk + Mitigation
- **Risk**: Threshold 0.99 chặt cho dataset write-heavy đang grow nhanh → false trip. **Mitigation**: Constant `snapshotCompletenessThreshold` dễ tune sau monitoring; có thể thay bằng env var.
- **Risk**: `totalRows == 0` (collection empty hoặc estimate fail) → guard skip. **Mitigation**: Logic `if totalRows > 0 && ...` — đúng hành vi mong muốn.

---

## O1 — Prometheus metric

### Lý do chọn
Bug 2026-05-28 không có metric → không alert được. Khi guard trip → increment counter có label `reason`. Alert rule đơn giản: `rate(cdc_snapshot_partial_done_total[5m]) > 0` → page on-call.

### Alternative rejected
| Alternative | Lý do reject |
|---|---|
| Chỉ log error | Log không trigger alert auto. Phụ thuộc human grep. |
| Push event sang NATS | Over-engineer; Prometheus đã có infra. |
| Single counter không label | Mất chi tiết reason → khó triage. |

---

## T1-T3 — Test strategy

### Lý do chọn
Unit test với SQLite in-memory + Mongo cursor mock — fast (~1s) + deterministic. Integration test full Mongo replica set có lag → flaky + chậm.

### Alternative rejected
| Alternative | Lý do reject |
|---|---|
| Testcontainers Mongo replica set + simulate lag | Heavy + flaky; tăng CI time 5-10min. Defer ADR. |
| Property-based fuzzing | Overkill cho bug deterministic này. |
| Skip test, chỉ smoke runtime | Vi phạm §3 Verify Before Done + DoD-1,2,3. |

---

## Holistic patch over lẻ tẻ

### Lý do chọn 1-PR multi-patch
3 root cause đều liên quan cùng 1 file (`snapshot_runner_handler.go`) + cùng 1 invariant (status=done IFF rows_processed >= threshold). Split 3 PR sẽ:
- Tăng review overhead.
- Risk regression giữa các PR (PR 2 áp dụng trong khi PR 1 chưa merge).
- Whack-a-mole tiếp tục nếu user thấy fix lẻ.

### Alternative rejected
- Split 3 PR sequential merge: chậm + không catch root cause đầy đủ.
- Patch chỉ 1 trong 3 (chỉ A, hoặc chỉ C): lesson `whack-a-mole` chính là bug hôm nay.
