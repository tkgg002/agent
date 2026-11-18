# Review Report — `review_transmute`

- **Repo:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service` | **Range:** `origin/develop..<working tree local chưa commit>` (`0aa70c0..fa4bd8d`, snapshot commit-tree không thuộc nhánh nào)
- **Ngày:** 2026-11-18 | **Agent:** Claude-Sonnet-5.5 | **Mode:** Propose-Only (không sửa source, không push)
- **Rules nạp:** L1 base(4) | L2 packs: `fintech` | L3 profile: `memory/global/project/data-hub/review_profile.md`
- **Scope file:** `scope_review_transmute_wt.json` (R0, 0 token). `scope_review_transmute.json` (337 file) **VÔ HIỆU** — tính trên `origin/develop` cũ `b0ff47c`, sau đó `origin/develop` đã fetch lên `0aa70c0` (đã chứa `recon-heal`).
- **Giới hạn review:** đọc code tĩnh + `go build` + `go vet` + 1 unit test. **Chưa chạy** test tích hợp `transmute_realtime_gate_test.go` (ghi/drop bảng trên Postgres thật `localhost:5435`). **Chưa qua laya gate (R5).**

## 1. Verdict
**`REQUEST_CHANGES`** — S0: `0` | S1: `1` | S2: `2` | S3: `2`

> Quyết định bởi F1: các nhãn `triggered_by` của luồng heal/prune/recon thực tế không nằm trong `switch` mới, nên rơi vào nhãn mặc định `kafka-consumer-hook`. Khi master có schedule `post_ingest` bị tắt, batch bị bỏ qua và ack thành công, nên heal/prune báo xong mà master không đổi. Phần còn lại (cache invalidate, SQL tham số hoá, move `resolveMasterTargets`) ổn. F1 được xác minh bằng đọc code, chưa chạy runtime.

## 2. Scope (từ `scope.py`)
| Chỉ số | Giá trị |
| :--- | :--- |
| Files changed / added / deleted | `10 / 3 / 0` |
| LOC +/- | `+517 / -38` |
| Layers touched | `handler, repository, server` (+ `sinkworker`) |
| `chokepoints_touched` | `CP-4` |
| `god_files` (> 600) | `internal/handler/master/transmute_handler.go (778)` — giảm từ 793 ở base |
| `money_path` files | `1` (chỉ file test, do keyword) |
| `layer_violations` | `0` |

## 3. Findings

### [S1] `B-LOG-03` / `INV-1` (CP-4) — Nhãn heal/prune/recon thực tế rơi vào `kafka-consumer-hook` và bị bỏ qua khi post_ingest tắt (VERIFIED, chưa chạy runtime)
- **REF:** `internal/handler/master/transmute_handler.go:186-196`, `internal/handler/master/transmute_handler.go:623-631`, `internal/handler/master/transmute_realtime_gate.go:86-97`
- **Quote:**
  ```
  hookTrigger := "kafka-consumer-hook"
  switch req.TriggeredBy {
  case "snapshot", "snapshot-hook": ...
  case "bridge-oplog": ...
  case "batch-transform": ...
  case "heal", "recon-heal", "execute-heal":
      hookTrigger = req.TriggeredBy
  }
  ...
  if h.shouldSkipDisabledRealtimeBatch(ctx, masterBindingID, triggeredBy) {
      ... h.reply(...); t.complete(nil) ... return
  }
  ...
  if triggeredBy != "kafka-consumer-hook" && triggeredBy != "snapshot-hook" { return false }
  ```
- **Vì sao:** Các nơi publish `cdc.cmd.transmute-shadow` gửi nhãn khác với 3 nhãn trong `case`:
  - `heal-seg-a` (`internal/handler/recon/recon_execute_heal_handler.go:280`)
  - `prune-seg-a` (`internal/handler/recon/recon_execute_heal_handler.go:311`)
  - `recon-heal-legacy` (`internal/handler/recon/recon_check_heal_handler.go:309`)
  - orphan prune không gửi `triggered_by` (`internal/service/recon/recon_tier_a.go:566-572`)

  Những nhãn này rơi vào default `kafka-consumer-hook`. Nếu master có dòng `transmute_schedule` với `mode='post_ingest'` và `is_enabled=false`, `runDebouncedGroup` trả `nil` (ack) mà không ghi master. Hệ quả: heal và prune orphan có vẻ thành công nhưng master không được cập nhật hay xoá. Chỉ xảy ra khi có dòng schedule tắt tường minh (dòng thiếu thì `EXISTS` trả false và vẫn chạy).
- **Blast-radius:** CP-4 callers (`profile §3`): `recon_execute_heal_segb.go` đi qua `cdc.cmd.transmute` với `execute-heal-b` (không qua `shouldSkip` ở handler này, nhưng debouncer chung); các đường `-shadow` ở trên bị ảnh hưởng. Test `transmute_realtime_gate_test.go:216-219` chỉ phủ `bridge-oplog`, không phủ các nhãn trên.
- **Gợi ý sửa:** Map đúng các nhãn thật vào `case`, hoặc đảo logic thành allow-list các nhãn realtime thay vì mặc định `kafka-consumer-hook`. Thêm test cho `heal-seg-a`, `prune-seg-a`, `recon-heal-legacy` và payload không có `triggered_by`.
- **Gate:** `laya dg3 = chưa chạy` | `EV-` không có (chưa reproduce runtime)

### [S2] `B-LOG-02` — `InvalidateConnectionAdapter` xoá prefix `master:` mà production không tạo (HYPOTHESIS)
- **REF:** `internal/handler/shadow/batch_buffer_lifecycle.go:55-67`, `internal/handler/shadow/batch_buffer_utils.go:66`
- **Quote:**
  ```
  for _, prefix := range []string{"master:", "shadow:"} {
      if v, ok := bb.adapters.LoadAndDelete(prefix + connKey); ok {
  ```
  ```
  cacheKey = strings.TrimSpace(record.ConnectionRole) + ":" + strings.TrimSpace(record.ConnectionKey)
  ```
- **Vì sao:** Key cache được tạo từ `ConnectionRole`. Ngoài test, chỉ thấy `"shadow"` được gán (`internal/handler/shadow/event_handler.go:429`, `internal/handler/source/bridge_handler.go:519`). Nếu lỗi "database is closed" nằm ở master DB thì adapter master không nằm trong cache này, và bản vá có thể chưa chữa đúng chỗ. Test mới tự `Store("master:...")` nên không chứng minh được production.
- **Blast-radius:** `connection.config.changed` subscriber (`internal/server/server_config_reload.go:42-60`) cho cdc/snapshot/heal buffer.
- **Gợi ý sửa:** Xác nhận nơi `ConnectionRole="master"` thật sự được tạo (nếu có) hoặc bỏ prefix thừa. Thêm test dùng đúng đường `resolveSchemaAdapter`.
- **Gate:** `chưa chạy`

### [S2] `B-DES-10` / `B-LOG-11` — Subscribe trong constructor, nuốt lỗi, không có Stop
- **REF:** `internal/sinkworker/worker.go:104-111`
- **Quote:**
  ```
  _, _ = w.natsConn.Subscribe("transmute.schedule.changed", func(_ *nats.Msg) {
      w.FlushPostIngestCache()
  })
  ```
- **Vì sao:** Bỏ qua cả lỗi lẫn subscription, không log. Grep không thấy `Stop`/`Unsubscribe` trong package `sinkworker`. Nếu subscribe lỗi thì chỉ còn TTL 30s (`worker.go:428-433`) mà không có cảnh báo.
- **Blast-radius:** chỉ `cmd/sinkworker/main.go:134`.
- **Gợi ý sửa:** Log lỗi subscribe, lưu subscription để huỷ khi tắt (hoặc đăng ký ở `main` giống `registerConfigReloadSubscribers`).
- **Gate:** `chưa chạy`

### [S3] `B-DES-14` — Comment lệch code
- **REF:** `internal/handler/shadow/batch_buffer_fanout_cache.go:15` ("gia hạn thêm 1 TTL") và `:21` (`stale.at = now.Add(postIngestFailOpenTTL - postIngestCacheTTL)`, tức còn 5s).
- **Gợi ý sửa:** Sửa comment cho khớp.

### [S3] `B-LOG-08` — Thêm 1 query DB (timeout 2s) cho mỗi debounced group, fail-open khác với nhánh filter
- **REF:** `internal/handler/master/transmute_realtime_gate.go:93-96` (lỗi query → vẫn chạy tiếp) so với `internal/handler/master/transmute_handler.go:176-181` (lỗi → trả error, retry).
- **Gợi ý sửa:** Thống nhất hướng xử lý lỗi, hoặc ghi chú vì sao hai nơi khác nhau.

## 4. Blast-radius tổng hợp
| Chokepoint / Contract | Thay đổi trong diff | Callers/Consumers phải kiểm | Trạng thái |
| :--- | :--- | :--- | :--- |
| `CP-4` `switch req.TriggeredBy` | thêm `snapshot-hook`, `batch-transform`, `heal`/`recon-heal`/`execute-heal` | `recon_execute_heal_handler.go:280,311`, `recon_check_heal_handler.go:309`, `recon_tier_a.go:566-572` | **RISK** (F1) |
| NATS `transmute.schedule.changed` | CDS thêm 2 subscriber (`server_config_reload.go:38`, `worker.go:105`) | publisher nằm ở `cdc-cms-service` (4 command handler, **chưa commit**) | OK cho CDS; deploy CDS trước CMS không vỡ vì chỉ subscribe |
| `cdc_system.transmute_schedule` (CMS sở hữu) | CDS **chỉ đọc** (`master_binding_repo.go:178-218`) | — | OK (không vi phạm write-owner) |
| SQL gate `sinkworker` / `batch_buffer_fanout` | mở rộng JOIN `shadow_sources` (đã có ở fanout trên base) | `worker.go:447-460` | OK (khớp bản của `batch_buffer_fanout.go:219-229`) |

## 5. Gate (R5)
- Mode B `laya_gate.py review`: **chưa chạy.**
- Mode A `laya_gate.py run --stage dg3`: **chưa chạy.** Engine VegaML `:8222` đang lên (`/health`=200).
- Dropped: none.
- **Các finding trên chưa được gate kiểm chứng.** Verdict dựa trên đọc code.

## 6. Evidence
| EV | Kind | Lệnh | exit |
| :--- | :--- | :--- | :---: |
| `EV-20261118T105108-4df5` | unit | `go test ./internal/handler/shadow/ -run TestBatchBuffer_FlushPostIngestCacheAndInvalidateConnectionAdapter -count=1` | 0 |
| `EV-20261118T105114-f72a` | unit | `go vet ./internal/handler/master/ ./internal/handler/shadow/ ./internal/server/ ./internal/sinkworker/ ./internal/repository/master/` | 0 |
| (không qua evidence_run) | build | `go build ./internal/...` | 0 |

> Lưu ý: test `TestBatchBuffer_FlushPostIngestCache...` tự `Store("master:...")` nên không chứng minh được hướng S2.

## 7. Không-finding đáng ghi (đã kiểm, OK)
- `B-SEC-05/06`: SQL mới tham số hoá (`master_binding_repo.go:184-193`, `208-216`).
- `B-SEC-01/02`: quét secret/PII trên diff: 0 kết quả.
- `B-LOG-20`: không có `t.Skip` mới trong diff. Lưu ý `setupRealTransmuteTestEnv` skip khi thiếu config (`transmute_debounce_group_test.go:186`), theo `INV-5` không tính là lỗi hệ thống.
- `B-DES-02`: `layer_violations = 0`.
- `B-DES-06`: `transmute_handler.go` 793 → 778 dòng; `resolveMasterTargets` chuyển nguyên văn sang `transmute_realtime_gate.go`, caller duy nhất `transmute_handler.go:165` vẫn đúng.
- `B-LOG-09`: `FlushPostIngestCache` của sinkworker giữ `piCacheMu.Lock` khi thay map (`worker.go:416-420`); của BatchBuffer dùng `sync.Map.Range/Delete` an toàn.
- `B-CON-09`: không có ghi vào `transmute_schedule` (CMS sở hữu).
