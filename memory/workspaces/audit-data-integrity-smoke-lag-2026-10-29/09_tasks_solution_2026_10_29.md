---
tier: T2
workspace: audit-data-integrity-smoke-lag-2026-10-29
suffix: 2026_10_29
mode: standard
created_at: 2026-10-29T14:17:30+07:00
---

# 09. Technical Solution: Audit & Khắc phục Hiện tượng Lag / Lệch Tạm thời (`1, 2`, `-2`, `+8`) khi Chạy Smoke Recon trên Live Big Data (`/data-integrity`)

## 1. Bối cảnh & Ràng buộc Thực thi (Technical Constraints)
- **Triệu chứng thực tế trên Live Big Data (`/data-integrity`):**
  1. Pipeline `core-trans-proxy-history-service` / `trans_his` (`4,463,652` bản ghi) fan-out ra 3 bảng Master:
     - `public.cdc_transaction_history`: `4,463,652 / 4,463,652 / 4,463,652` -> `ingest: 0`, `transmute: 0` (`Khớp`).
     - `master_core_trans_proxy_history_service.trans_his`: `4,463,652 / 4,463,652 / 4,463,652` -> `ingest: 0`, `transmute: 0` (`Khớp`).
     - `public.cdc_transaction_history_payload`: `4,463,652 / 4,463,652 / 4,463,650` -> `ingest: 0`, `transmute: -2 (thiếu)` (`Lệch`).
  2. Pipeline `payment-bill-service` / `payment_bills` (`6,517,390` bản ghi):
     - `master_payment_bill_service.payment_bills`: `6,517,390 / 6,517,398 / 6,517,398` -> `ingest: +8 (thừa)`, `transmute: 0` (`Lệch`).
  3. **Bằng chứng Trace thực tế từ User trên `bank_requests_bvb` (`15,370,051` bản ghi, phiên chạy lúc `14:24:17` / `07:24:17` UTC):**
     - `14:24:38 | 800302…bbfd | Smoke Check | Source → Shadow | LỆCH | 15.00s : 15,370,045 → 15,370,051 (+6)`
     - `14:24:23 | 800302…bbfd | Smoke Check | Shadow → Master | KHỚP | 6ms : 15,370,051 → 15,370,051 (0)`
     - `14:04:44 | 88879b…9ce7 | Smoke Check | Source → Shadow | LỆCH | 15.00s : 15,369,840 → 15,369,846 (+6)`
     - `14:04:29 | 88879b…9ce7 | Smoke Check | Shadow → Master | LỆCH | 5ms : 15,369,846 → 15,369,845 (-1)`
     - Cây Trace của `pipeline:bank_requests_bvb` (`14:24:17`):
       - `prefetch.master` (`pg.count_rows`, `pg.count_deleted_rows`) + `prefetch.shadow` (`pg.count_rows`, `pg.count_deleted_rows`) chạy từ `14:24:17` đến `14:24:23` (~6s).
       - `source_shadow` chạy từ `14:24:23` đến `14:24:38` (đúng **`15.00s`** = `SmokeQueryTimeout`), có các span: `recon.smoke.mongo_estimate` -> `recon.source.hash_window: bank-requests [05:19:17 -> 07:19:17]` -> `pg.hash_window: shadow_banvietbank_connector_service.bank_requests_bvb [05:19:17 -> 07:19:17]`.
       - `shadow_master` chạy lúc `14:24:23` chỉ mất **`6ms`** (và lúc `14:04:29` mất **`5ms`**), chỉ có đúng 2 span `shadow.max_ts` và `master.max_ts`!
  4. Phản ánh của User: *"mọi thay đổi bắt buộc phải đặt đúng layer theo Design Pattern gốc của bạn và tách file/module theo đúng trách nhiệm đơn, tuyệt đối không nhồi nhét thêm vào các file lớn. a thấy nhận xét bị gap do đang có transmute nó k thuyết phục... anh nhớ là logic của anh là quét lùi time để ko bị gap mà"*
- **Ràng buộc Thực thi (`L003`, `L006`, `L020`, `L029`, `L030`, `L042`):**
  - Chế độ **Propose-Only (`L003`)**: Chỉ đề xuất giải pháp kèm Code Demo, chờ lệnh `APPROVE` trước khi chạm vào source code.
  - **Single-Responsibility Module Split (`L006`)**: Tuyệt đối không nhồi thêm hàm mới vào `recon_smoke.go` (1,285 dòng), `recon_dest_query.go` (708 dòng), hay `recon_hash.go` (517 dòng). Tách bạch đúng layer thành 2 file đơn trách nhiệm:
    1. `internal/service/recon/recon_smoke_guard.go`: Tầng Orchestration Guard của `ReconCore` thực thi quét lùi time (`FreezeMargin Cutoff`) cho Segment A và Segment B.
    2. `internal/service/recon/recon_window_index_probe.go`: Tầng Data-Access Probe của `ReconSourceAgent` & `ReconDestAgent` kiểm tra index cột (`HasColumnIndex`) và đối chiếu cửa sổ lùi time theo B-Tree index (`_id` ObjectID range $\leftrightarrow$ `_source_id` / `_source_ts`) chống COLLSCAN/Seq Scan 15.00s.
  - **Real Freeze-Cutoff Enforcement & Timeout-vs-Drift Separation (`L042`)**: Thực thi quét lùi thời gian thật sự trong câu query DB (chốt số đếm tại mốc lùi `hi = lockTime - WindowFreezeMargin`), và tách bạch lỗi timeout `15.00s` của `HashWindow` khỏi kết luận `drift`.

---

## 2. Kết quả Audit Nguyên nhân Gốc rễ từ Trace Thực tế (`F1` – `F3`, 100% `VERIFIED` kèm `file:line`)

### F1 — Thiết kế gốc ĐÃ CÓ logic "Quét lùi time để không bị gap", nhưng trong `RunTotalOnlyB` (`shadow_master`), mốc lùi `hi = lockTime.Add(-rc.cfg.WindowFreezeMargin)` chỉ được gắn vào Span Attribute trên Trace, còn toàn bộ khối Query trừ lùi time đã bị comment-out từ `commit 47489ff`!
- **Bằng chứng `file:line` & đối chiếu Trace (`VERIFIED`):**
  - Tại `recon_smoke.go:502-512` (`RunTotalOnlyB`):
    ```go
    // Lùi mốc hi lại một khoảng FreezeMargin (vd 5 phút) để đảm bảo In-flight data đã rơi hết xuống đích
    hi := lockTime.Add(-rc.cfg.WindowFreezeMargin)
    lo := hi.Add(-rc.effectiveLookback(ctx))

    // span name = "shadow_master" — parent shadow:{table} đã có table context
    ctx, span := observability.ChildSpan(ctx, "shadow_master",
        attribute.String("segment", "shadow_master"),
        attribute.String("master_table", ref.MasterTable),
        attribute.String("time.from", lo.Format(time.RFC3339)),
        attribute.String("time.to", hi.Format(time.RFC3339)),
    )
    ```
  - **Sự thật vật lý:** Biến `lo` và `hi` (đã lùi `WindowFreezeMargin = 5m`, ví dụ `07:19:17` UTC so với `lockTime = 07:24:17` UTC) **chỉ được truyền vào `observability.ChildSpan` ở dòng `510-511` rồi bị bỏ rơi hoàn toàn!**
  - Toàn bộ khối query thực thi quét lùi time của anh tại `recon_smoke.go:614-666` (`CountInWindow` + `CountRecentDeletedRows` trừ đi các bản ghi mới phát sinh sau mốc lùi time để chốt `shadowActiveClean` và `masterActiveClean` về cùng mốc quá khứ) **đã bị bọc trong `/* ... */` (comment-out) từ `commit 47489ff`**!
  - Đó là lý do vì sao trên cây Trace của anh:
    - Nhánh `5 shadow_master` chỉ có đúng 2 span `2 shadow.max_ts` và `2 master.max_ts` (tốn vỏn vẹn **`5ms` – `6ms`**) và **không có bất kỳ câu query quét lùi time nào**!
    - Trong khi đó, bước `3 prefetch.master` và `3 prefetch.shadow` (`scanExact` tại `recon_smoke.go:131-140`) chạy câu `SELECT COUNT("_gpay_id") FROM relation` **không có điều kiện lùi time** (tốn ~6 giây trên bảng 15.37M dòng từ `14:24:17` đến `14:24:23`). Vì 2 câu `COUNT(*)` trên 2 bảng 15.37M dòng chụp snapshot ở 2 thời điểm lệch nhau vài giây (ở phiên `14:04:29`, `prefetch.master` chụp lúc `15,369,845` còn `prefetch.shadow` chụp lúc vừa lên `15,369,846`) và `RunTotalOnlyB` (`recon_smoke.go:668`) lấy thẳng `diff := shadowActive - masterActive` mà không trừ phần phát sinh sau mốc lùi `hi`, nên nó báo **`LỆCH 5ms : 15,369,846 → 15,369,845 (-1)`**!

### F2 — Tại sao `source_shadow` (`RunTotalOnlyA`) ĐÃ CHẠY `hash_window [05:19:17 -> 07:19:17]` (quét lùi 5 phút) nhưng luôn chạm kịch trần **`15.00s`** và luôn báo **`LỆCH +6`** (`15,370,045 → 15,370,051` lúc `14:24:38` và `15,369,840 → 15,369,846` lúc `14:04:44`)?
- **Bằng chứng `file:line` & đối chiếu Trace (`VERIFIED`):**
  1. **Sai số tĩnh `+6` của `EstimatedDocumentCount()` kích hoạt Guard 1 (`HashWindow [05:19:17 -> 07:19:17]`):**
     - Trong 20 phút từ `14:04` đến `14:24`, cả Shadow và Master đều tăng đúng `205` bản ghi (`15,369,846 -> 15,370,051`), chứng tỏ 100% dữ liệu CDC đã sang đủ. Tuy nhiên `rc.sourceAgent.EstimatedCount` (`recon_smoke.go:279`, `recon_query.go:105`) đọc metadata WiredTiger trên MongoDB Secondary luôn thấp hơn thực tế đúng `6` bản ghi (`diff = -6`, hiển thị `+6` trên UI).
     - Vì `abs(-6) <= threshold` (`500`), code nhảy vào **Guard 1 (`recon_smoke.go:366-407`)** gọi `HashWindow` trên cửa sổ đã lùi 5 phút `[05:19:17 -> 07:19:17]` UTC để xác nhận nếu cửa sổ tĩnh khớp thì xóa sai số `+6` (`diff = 0, statusStr = "ok"`).
  2. **Nhưng Guard 1 (`HashWindow [05:19:17 -> 07:19:17]`) luôn bị treo kịch trần `15.00s` (`SmokeQueryTimeout`) rồi bị quy chụp thành `LỆCH` vì 3 lỗi chí mạng tại `recon_smoke.go:253-258, 373-403`, `recon_hash.go:57-62`, và `recon_dest_hash.go:30-115`:**
     - **Lỗi 2a (COLLSCAN / Seq Scan trên bảng 15.37 triệu dòng chạm cứng `fastCtx` 15.00s):**
       - Tại `recon_smoke.go:253-258`, toàn bộ `RunTotalOnlyA` dùng chung 1 `fastCtx` có timeout **`15 * time.Second` (`15.00s`)**.
       - Trên MongoDB (`recon_hash.go:57-62`), `sourceAgent.HashWindow` chạy `coll.Find` với bộ lọc `$or` 2 kiểu dữ liệu (`{tsField: {$gte: tLo, $lt: tHi}}` hoặc `{tsField: {$gte: tLo.UnixMilli(), $lt: tHi.UnixMilli()}}`). Trên collection `bank-requests` (**15,370,045 docs**), khi `tsField` không có index (hoặc bị `$or` 2 kiểu), MongoDB phải **COLLSCAN 15.37 triệu documents**!
       - Tương tự trên PostgreSQL Shadow (`shadow_banvietbank_connector_service.bank_requests_bvb` — **15,370,051 rows**), bảng Shadow khi tạo tự động (`schema_manager.go:200-220`, `shadow_automator.go:98`) **chỉ có B-tree index trên `_gpay_id`, `_source_id`, và `_source_ts`**, chứ **không có index trên cột thời gian nghiệp vụ `dstTS` (`recon_dest_hash.go:108-112`)**! Quét `WHERE dstTS >= lo AND dstTS < hi` trên bảng 15.37M dòng không có index bắt Postgres phải **Seq Scan 15.37 triệu dòng**!
       - Khi `recon.source.hash_window` (hoặc `pg.hash_window`) chạy chạm đúng mốc **`15.00s`** (từ `14:24:23` đến `14:24:38` và từ `14:04:29` đến `14:04:44`), `fastCtx` bị hủy với lỗi **`context deadline exceeded`**!
       - Đáng chú ý: Ngay cả khi `recon.source.hash_window` đã làm `fastCtx` hết hạn ở giây 15.00, dòng `382` vẫn gọi tiếp `rc.destAgent.HashWindow(fastCtx, ...)`. Hàm `destAgent.HashWindow` (`recon_dest_hash.go:30-40`) **mở span `pg.hash_window` ngay đầu hàm trước khi chạy SQL**, rồi xuống dưới gặp `fastCtx.Err()` nên thoát ra ngay -> vì vậy trên SigNoz trace anh thấy đủ cả 2 span `recon.source.hash_window` và `pg.hash_window` và tổng thời gian luôn dừng ở **đúng `15.00s`**!
     - **Lỗi 2b (Đánh đồng lỗi Timeout `15.00s` của `HashWindow` thành `"real drift confirmed"`):**
       - Tại `recon_smoke.go:383-395`:
         ```go
         if errS == nil && errD == nil && srcHash.Count == dstHash.Count && srcHash.XorHash == dstHash.XorHash {
             diff = 0
             statusStr = "ok"
         } else {
             statusStr = "drift"
         }
         ```
         Khi `errS != nil || errD != nil` (do chạm timeout `15.00s`), code nhảy vào `else { statusStr = "drift" }`, biến lỗi timeout 15.00s của `HashWindow` thành báo cáo **`LỆCH | 15.00s : 15,370,045 → 15,370,051 (+6)`**!
     - **Lỗi 2c (Bypass `resolveSourceAndDestTSFields` và truyền cứng `entry.PrimaryKeyField` `"_id"` thay vì `"_source_id"` tại `recon_smoke.go:373-382`):**
       - Khi `entry.TimestampField != nil`, dòng `374-375` gán cứng `dstTS = *entry.TimestampField` (camelCase trên Mongo) thay vì qua `resolveSourceAndDestTSFields`. Và dòng `382` truyền cứng `entry.PrimaryKeyField` (`"_id"`) vào `destAgent.HashWindow` thay vì phân giải sang `"_source_id"` trên bảng Postgres Shadow.

### F3 — Nhân bản goroutine `RunTotalOnlyA` theo từng `MasterBindingRef` trong `CheckAllUnified`
- **Bằng chứng `file:line` (`VERIFIED`):**
  - Tại `recon_smoke.go:963-997`, vòng lặp `for _, p := range pipelineList` phóng lặp $M$ goroutine `RunTotalOnlyA` cho cùng 1 bảng Shadow khi bảng Shadow đó fan-out ra $M$ bảng Master (như `trans_his` có 3 Master), gây tranh chấp `withTableLock` (`recon_smoke.go:168-172`) và gán `MasterSchema`/`MasterTable` của 1 nhánh con ngẫu nhiên vào dòng `source_shadow` (`recon_smoke.go:462-465`).

---

## 3. Phương án Kỹ thuật Duy nhất & Tối ưu nhất (Tách Module Đơn Trách Nhiệm Đúng Layer, Ánh xạ 1-1 với `F1` – `F3`)

### 3.1. Giải pháp cho F1: Thực thi thật sự "Quét Lùi Time (`FreezeMargin Cutoff`)" cho Segment B (`RunTotalOnlyB`) trong `recon_smoke_guard.go`
- Trong `RunTotalOnlyB` (`recon_smoke.go:668-674`), khi `shadowActive != masterActive` (ví dụ lệch `-1`, `-2` do `prefetch.master` và `prefetch.shadow` chạy `COUNT(*)` lệch nhau vài giây):
  - Gọi `rc.verifySegmentBAtFreezeCutoff` (đặt tại file đơn trách nhiệm `recon_smoke_guard.go`):
  - **Thực thi chốt chặn lùi time tại đúng mốc `hi = lockTime.Add(-rc.cfg.WindowFreezeMargin)` (ví dụ `07:19:17` UTC, lùi 5 phút):**
    - Vì cả bảng Shadow (`idx_<table>_source_ts`) và bảng Master (`ix_<table>_source_ts`) đều đã có sẵn B-Tree Index trên cột `_source_ts` (tốc độ truy vấn `< 5ms` như `shadow.max_ts` trên trace), ta đếm số bản ghi active phát sinh từ mốc lùi `hi` đến hiện tại (`nowUpper = time.Now().UTC().Add(1 * time.Minute)`):
      - `shAfterHi = CountInWindow(shadowRel, "_source_ts", hi, nowUpper) - CountRecentDeletedRows(shadowRel, "_source_ts", hi, nowUpper)`
      - `msAfterHi = CountInWindow(masterRel, "_source_ts", hi, nowUpper) - CountRecentDeletedRows(masterRel, "_source_ts", hi, nowUpper)`
    - Đưa số đếm của cả Shadow và Master về **cùng mốc chốt chặn quá khứ `hi` (`07:19:17`)**:
      $$\text{shadowAtHi} = \text{shadowActive} - \text{shAfterHi}$$
      $$\text{masterAtHi} = \text{masterActive} - \text{msAfterHi}$$
    - Nếu `shadowAtHi == masterAtHi` (hoặc fallback `HashWindow` trên `[lo, hi]` theo cặp khóa `ResolveSegmentBPKPair` và `_source_ts` khớp nhau), xác nhận 100% tính đến mốc lùi time `hi`, dữ liệu Shadow và Master khớp tuyệt đối -> trả về `diff = 0`, `statusStr = "ok"`, và chuẩn hóa `masterActive = shadowActive`, `masterTotal = shadowTotal`.

### 3.2. Giải pháp cho F2: Chấm dứt Timeout `15.00s` & Sửa Guard 1 Quét Lùi Time `[lo, hi]` cho Segment A (`RunTotalOnlyA`) qua `recon_window_index_probe.go` và `recon_smoke_guard.go`
1. **Tách tầng Data-Access Probe vào file mới `internal/service/recon/recon_window_index_probe.go`:**
   - `(da *ReconDestAgent) HasColumnIndex(ctx, tableName, columnName) bool`: Kiểm tra (và cache thread-safe) xem cột `dstTS` trên bảng PostgreSQL Shadow có B-Tree index trong `pg_indexes` hay không.
   - `(sa *ReconSourceAgent) HashObjectIDWindow(ctx, sourceURL, database, collection, tLo, tHi)`: Khi nguồn là MongoDB và bảng lớn (như `bank-requests` 15.37M dòng) không có index trên `srcTS` (hoặc bảng Shadow không có index trên `dstTS`), thực hiện quét lùi time `[tLo, tHi)` trực tiếp trên **Primary Key Index `_id_` mặc định của MongoDB** bằng `primitive.NewObjectIDFromTimestamp(tLo.UTC())` $\to$ `primitive.NewObjectIDFromTimestamp(tHi.UTC())` (**0% COLLSCAN, `< 10ms`**).
   - `(da *ReconDestAgent) HashSourceIDWindow(ctx, tableName, tLo, tHi)`: Đối ứng 1-1 trên PostgreSQL Shadow, truy vấn cửa sổ `[oidLo.Hex(), oidHi.Hex())` trực tiếp trên **Unique B-Tree Index `ux_<table>_source_id_active` (`_source_id`)** có sẵn trên mọi bảng Shadow (`schema_manager.go:200-220`) (**0% Seq Scan, `< 10ms`**).
2. **Tách tầng Orchestration Guard vào file mới `internal/service/recon/recon_smoke_guard.go` (`verifySegmentAAtFreezeCutoff`):**
   - Phân giải đúng `srcTS, dstTS` qua `rc.resolveSourceAndDestTSFields` và `dstPK` qua `rc.resolveShadowPKField` (`"_source_id"` trên Postgres Shadow).
   - Tạo context riêng cho Guard 1 (không dùng chung `fastCtx` đã hao hụt) và **chạy song song (2 goroutine)** giữa Source và Shadow:
     - **Nhánh A (Khi cột `dstTS` có index trên Shadow):** Chạy song song `sourceAgent.HashWindow` và `destAgent.HashWindow` trên cửa sổ lùi time `[lo, hi]`.
     - **Nhánh B (Khi cột `dstTS` trên Shadow 15.37M dòng KHÔNG có index — như `bank_requests_bvb` — hoặc khi Nhánh A bị timeout trên MongoDB do thiếu index `srcTS`):** Tự động chuyển sang chạy song song **`sourceAgent.HashObjectIDWindow` ($\leftrightarrow$ index `_id_`)** và **`destAgent.HashSourceIDWindow` ($\leftrightarrow$ index `_source_id`)** trên cửa sổ tĩnh đã lùi 5 phút `[lo, hi]` (`[05:19:17 -> 07:19:17]`)!
     - Khi khớp (`Count` & `XorHash` bằng nhau), xóa ngay sai số metadata `+6` / `+8` của `EstimatedDocumentCount()` (`diff = 0, statusStr = "ok"`) chỉ trong **`< 20ms`** thay vì treo `15.00s`!

### 3.3. Giải pháp cho F3: Dedup `RunTotalOnlyA` theo từng bảng Shadow (`entry.QualifiedTarget()`) và làm mới `smokeCountCache` đầu chu kỳ trong `CheckAllUnified`
- Đầu Step 3 của `CheckAllUnified` (`recon_smoke.go:892`), xóa cache cũ của `scanTargets` để mọi bảng trong chu kỳ mới được đếm mới; và tại Step 4 (`recon_smoke.go:963`), dùng `seenSegA[entry.QualifiedTarget()]` để chỉ chạy đúng 1 `RunTotalOnlyA` (với `ref = nil`) cho mỗi bảng Shadow.

---

## 4. Bảng Explicit Runtime Call-Site & Lifecycle Trigger Wiring

| Layer / File | Hàm / Method | Call-Site (Nơi gọi) | Lifecycle Trigger (Khi nào kích hoạt) |
| :--- | :--- | :--- | :--- |
| `internal/service/recon/recon_window_index_probe.go` *(File mới - Data Access Layer)* | `(da *ReconDestAgent) HasColumnIndex(ctx, tableName, col)` | Gọi bởi `verifySegmentAAtFreezeCutoff` trong `recon_smoke_guard.go` | Kiểm tra (kèm cache) cột `dstTS` trên bảng Shadow có B-Tree index trong `pg_indexes` không trước khi gọi `pg.hash_window` để chặn đứng Seq Scan 15.37M dòng. |
| `internal/service/recon/recon_window_index_probe.go` *(File mới - Data Access Layer)* | `(sa *ReconSourceAgent) HashObjectIDWindow(ctx, ...)` & `(da *ReconDestAgent) HashSourceIDWindow(ctx, ...)` | Gọi bởi `verifySegmentAAtFreezeCutoff` trong `recon_smoke_guard.go` | Kích hoạt khi bảng lớn MongoDB/Shadow thiếu index trên cột timestamp nghiệp vụ (hoặc `HashWindow` theo `tsField` chạm timeout), đối chiếu cửa sổ lùi time `[lo, hi)` bằng B-Tree index `_id_` (Mongo) $\leftrightarrow$ `_source_id` (Postgres Shadow) trong `< 20ms`. |
| `internal/service/recon/recon_smoke_guard.go` *(File mới - Service Guard Layer)* | `(rc *ReconCore) verifySegmentAAtFreezeCutoff(ctx, entry, lo, hi, diff)` | Gọi trực tiếp tại `RunTotalOnlyA` (`recon_smoke.go:360`) | Kích hoạt khi `diff = srcEst - dstActive != 0` và `abs(diff) <= threshold` (như `+6` trên `bank_requests_bvb`, `+8` trên `payment_bills`). |
| `internal/service/recon/recon_smoke_guard.go` *(File mới - Service Guard Layer)* | `(rc *ReconCore) verifySegmentBAtFreezeCutoff(ctx, masterAgent, ref, lo, hi, diff, shadowTotal, shadowActive, masterTotal, masterActive)` | Gọi trực tiếp tại `RunTotalOnlyB` (`recon_smoke.go:668`) | Kích hoạt khi `diff = shadowActive - masterActive != 0` và `abs(diff) <= threshold` (như `-1` trên `bank_requests_bvb`, `-2` trên `trans_his`) để trừ cửa sổ sau mốc lùi `hi` trên index `_source_ts`. |
| `internal/service/recon/recon_smoke.go` *(Orchestrator)* | `(rc *ReconCore) CheckAllUnified(ctx)` | Gọi bởi `startUnifiedSmokeLoop` (`recon_engine.go:296`) & `/api/reconciliation/check` (`server_jobs.go:108`) | Làm mới `smokeCountCache` đầu chu kỳ và dedup `seenSegA[entry.QualifiedTarget()]` để chạy đúng 1 `RunTotalOnlyA` cho mỗi bảng Shadow. |

---

## 5. Code Demo (Propose-Only — Chờ `APPROVE`)

### 5.1. File mới 1 (Tầng Data-Access Probe): `centralized-data-service/internal/service/recon/recon_window_index_probe.go`
```go
package recon

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"centralized-data-service/pkgs/observability"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel/attribute"
)

var (
	pgColIndexCacheMu sync.RWMutex
	pgColIndexCache   = make(map[string]bool)
)

// objectIDBoundaryFromTime constructs a deterministic zero-padded ObjectID boundary (TTTT00000000000000000000)
// for the given timestamp so MongoDB _id and PostgreSQL _source_id range queries share identical boundaries.
func objectIDBoundaryFromTime(t time.Time) primitive.ObjectID {
	var oid primitive.ObjectID
	binary.BigEndian.PutUint32(oid[0:4], uint32(t.UTC().Truncate(time.Second).Unix()))
	return oid
}

// HasColumnIndex checks whether a PostgreSQL table has a B-Tree index covering columnName.
// Results are cached thread-safely to prevent repeated catalog queries.
func (da *ReconDestAgent) HasColumnIndex(ctx context.Context, tableName, columnName string) bool {
	if da == nil || da.IsMongo() {
		return false
	}
	col := strings.TrimSpace(columnName)
	if col == "" {
		return false
	}
	if col == "_gpay_id" || col == "_source_id" || col == "_source_ts" {
		return true
	}
	if err := validateIdent(tableName); err != nil {
		return false
	}
	cacheKey := strings.ToLower(tableName + "." + col)
	pgColIndexCacheMu.RLock()
	if indexed, ok := pgColIndexCache[cacheKey]; ok {
		pgColIndexCacheMu.RUnlock()
		return indexed
	}
	pgColIndexCacheMu.RUnlock()

	schema, table := splitSchemaTable(tableName)
	qCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	tx := da.readOnlyDB(qCtx)
	defer tx.Rollback()

	var count int64
	sql := `SELECT COUNT(*) FROM pg_indexes WHERE LOWER(schemaname) = LOWER(?) AND LOWER(tablename) = LOWER(?) AND LOWER(indexdef) LIKE ?`
	pattern := "%(" + strings.ToLower(col) + "%"
	if err := tx.Raw(sql, schema, table, pattern).Scan(&count).Error; err != nil {
		return false
	}
	indexed := count > 0
	pgColIndexCacheMu.Lock()
	pgColIndexCache[cacheKey] = indexed
	pgColIndexCacheMu.Unlock()
	return indexed
}

// HashObjectIDWindow computes (count, xorHash) in [tLo, tHi) using the mandatory MongoDB _id_ B-Tree index.
// Avoids 15s COLLSCAN when large MongoDB collections (e.g. 15.37M docs) lack an index on the business timestamp field.
func (sa *ReconSourceAgent) HashObjectIDWindow(ctx context.Context, sourceURL, database, collection string, tLo, tHi time.Time) (*WindowResult, error) {
	ctx, span := observability.ChildSpan(ctx, fmt.Sprintf("recon.source.oid_window: %s [%s -> %s]",
		collection, tLo.UTC().Format("15:04:05"), tHi.UTC().Format("15:04:05")),
		attribute.String("db.database", database),
		attribute.String("db.collection", collection),
	)
	defer span.End()

	qCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := sa.getClient(qCtx, sourceURL)
	if err != nil {
		return nil, err
	}
	coll := sa.secondaryColl(client, database, collection)

	oidLo := objectIDBoundaryFromTime(tLo)
	oidHi := objectIDBoundaryFromTime(tHi)
	filter := bson.M{"_id": bson.M{"$gte": oidLo, "$lt": oidHi}}
	opts := sa.selectOpts(bson.M{"_id": 1})

	cursor, err := coll.Find(qCtx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(qCtx)

	var (
		xorAcc uint64
		count  int64
	)
	for cursor.Next(qCtx) {
		var raw bson.Raw
		if err := cursor.Decode(&raw); err != nil {
			return nil, err
		}
		v := raw.Lookup("_id")
		oid, ok := v.ObjectIDOK()
		if !ok {
			continue
		}
		idHex := oid.Hex()
		xorAcc ^= hashIDPlusTsMs(idHex, oid.Timestamp().UnixMilli())
		count++
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return &WindowResult{Count: count, XorHash: xorAcc}, nil
}

// HashSourceIDWindow computes the matching (count, xorHash) on PostgreSQL Shadow using the indexed _source_id column.
// Lexicographical order of 24-char hex ObjectIDs matches binary ObjectID timestamp order 1-to-1.
func (da *ReconDestAgent) HashSourceIDWindow(ctx context.Context, tableName string, tLo, tHi time.Time) (*WindowResult, error) {
	ctx, span := observability.ChildSpan(ctx, fmt.Sprintf("pg.source_id_window: %s [%s -> %s]",
		tableName, tLo.UTC().Format("15:04:05"), tHi.UTC().Format("15:04:05")),
		attribute.String("db.table", tableName),
	)
	defer span.End()

	if err := validateIdent(tableName); err != nil {
		return nil, err
	}
	qCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	oidLoHex := objectIDBoundaryFromTime(tLo).Hex()
	oidHiHex := objectIDBoundaryFromTime(tHi).Hex()

	sql := fmt.Sprintf(`
		SELECT "_source_id"
		FROM %s
		WHERE "_source_id" >= ? AND "_source_id" < ?
		  AND LENGTH("_source_id") = 24
		  AND NOT "_deleted"`,
		quoteRelation(tableName),
	)

	tx := da.readOnlyDB(qCtx)
	defer tx.Rollback()

	rows, err := tx.Raw(sql, oidLoHex, oidHiHex).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		xorAcc uint64
		count  int64
	)
	for rows.Next() {
		var idHex string
		if err := rows.Scan(&idHex); err != nil {
			return nil, err
		}
		oid, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			continue
		}
		xorAcc ^= hashIDPlusTsMs(idHex, oid.Timestamp().UnixMilli())
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &WindowResult{Count: count, XorHash: xorAcc}, nil
}
```

### 5.2. File mới 2 (Tầng Service Guard): `centralized-data-service/internal/service/recon/recon_smoke_guard.go`
```go
package recon

import (
	"context"
	"fmt"
	"sync"
	"time"

	"centralized-data-service/internal/model/source"
	"go.uber.org/zap"
)

// resolveShadowPKField resolves the actual primary key column name on the PostgreSQL shadow table.
// For MongoDB sources where PrimaryKeyField is "_id", it probes the shadow table and falls back to "_source_id".
func (rc *ReconCore) resolveShadowPKField(ctx context.Context, entry source.TableRegistry) string {
	primary := entry.PrimaryKeyField
	if primary == "" {
		primary = "_source_id"
	}
	if isPostgres(entry.SourceURL) {
		return primary
	}
	if rc.destAgent != nil {
		if realCol, err := rc.destAgent.GetRealColumnName(ctx, entry.QualifiedTarget(), primary); err == nil && realCol != "" {
			return realCol
		}
		if realCol, err := rc.destAgent.GetRealColumnName(ctx, entry.QualifiedTarget(), "_source_id"); err == nil && realCol != "" {
			return realCol
		}
	}
	return "_source_id"
}

// verifySegmentAAtFreezeCutoff enforces the backward freeze-cutoff [lo, hi] for Segment A (Source <-> Shadow).
// Runs source and shadow window checks concurrently on dedicated contexts, and falls back to indexed _id <-> _source_id
// window verification when large tables (e.g. 15.37M rows) lack an index on the business timestamp column.
func (rc *ReconCore) verifySegmentAAtFreezeCutoff(
	ctx context.Context,
	entry source.TableRegistry,
	lo, hi time.Time,
	diff int64,
) (resolvedDiff int64, statusStr string) {
	if diff == 0 {
		return 0, "ok"
	}
	threshold := rc.cfg.CountDriftThreshold
	if threshold <= 0 {
		threshold = 100
	}
	if abs(diff) > threshold {
		return diff, "drift"
	}

	timeout := rc.cfg.SmokeQueryTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	metaCtx, metaCancel := context.WithTimeout(ctx, 3*time.Second)
	srcTS, dstTS, errTS := rc.resolveSourceAndDestTSFields(metaCtx, entry, "smoke")
	dstPK := rc.resolveShadowPKField(metaCtx, entry)
	dstIndexed := errTS == nil && dstTS != "" && rc.destAgent.HasColumnIndex(metaCtx, entry.QualifiedTarget(), dstTS)
	metaCancel()

	// Path 1: Concurrent HashWindow on [lo, hi] when dstTS is indexed and shares the same time axis as srcTS
	if dstIndexed && !(dstTS == "_source_ts" && !isPostgres(entry.SourceURL)) {
		hashCtx, hashCancel := context.WithTimeout(ctx, 6*time.Second)
		var (
			srcHash, dstHash *WindowResult
			errS, errD       error
			wg               sync.WaitGroup
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			srcHash, errS = rc.sourceAgent.HashWindow(hashCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, lo, hi)
		}()
		go func() {
			defer wg.Done()
			dstHash, errD = rc.destAgent.HashWindow(hashCtx, entry.QualifiedTarget(), dstPK, dstTS, lo, hi)
		}()
		wg.Wait()
		hashCancel()

		if errS == nil && errD == nil && srcHash != nil && dstHash != nil {
			if srcHash.Count == dstHash.Count && srcHash.XorHash == dstHash.XorHash {
				rc.logger.Info("[smoke-A] Discrepancy resolved via indexed HashWindow match on freeze range",
					zap.String("table", entry.TargetTable),
					zap.Int64("estimatedDiff", diff),
					zap.String("srcTS", srcTS),
					zap.String("dstTS", dstTS),
					zap.String("dstPK", dstPK),
					zap.Time("lo", lo),
					zap.Time("hi", hi),
					zap.Int64("windowCount", srcHash.Count),
				)
				return 0, "ok"
			}
			rc.logger.Warn("[smoke-A] HashWindow mismatch — real drift confirmed on freeze range",
				zap.String("table", entry.TargetTable),
				zap.Int64("estimatedDiff", diff),
				zap.Int64("srcWindowCount", srcHash.Count),
				zap.Int64("dstWindowCount", dstHash.Count),
			)
			return diff, "drift"
		}
	}

	// Path 2: For MongoDB sources where business tsField lacks an index (causing 15s COLLSCAN/SeqScan timeout),
	// verify the backward freeze window [lo, hi] via mandatory B-Tree indexes: MongoDB _id_ <-> Postgres _source_id.
	if !isPostgres(entry.SourceURL) {
		oidCtx, oidCancel := context.WithTimeout(ctx, 5*time.Second)
		defer oidCancel()

		var (
			srcOid, dstOid *WindowResult
			errS, errD     error
			wg             sync.WaitGroup
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			srcOid, errS = rc.sourceAgent.HashObjectIDWindow(oidCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, lo, hi)
		}()
		go func() {
			defer wg.Done()
			dstOid, errD = rc.destAgent.HashSourceIDWindow(oidCtx, entry.QualifiedTarget(), lo, hi)
		}()
		wg.Wait()

		if errS == nil && errD == nil && srcOid != nil && dstOid != nil &&
			srcOid.Count == dstOid.Count && srcOid.XorHash == dstOid.XorHash {
			rc.logger.Info("[smoke-A] Discrepancy resolved via indexed ObjectID/_source_id freeze window match",
				zap.String("table", entry.TargetTable),
				zap.Int64("estimatedDiff", diff),
				zap.Time("lo", lo),
				zap.Time("hi", hi),
				zap.Int64("windowCount", srcOid.Count),
			)
			return 0, "ok"
		}
		rc.logger.Warn("[smoke-A] Freeze window verification did not match",
			zap.String("table", entry.TargetTable),
			zap.Int64("estimatedDiff", diff),
			zap.String("errS", fmt.Sprintf("%v", errS)),
			zap.String("errD", fmt.Sprintf("%v", errD)),
		)
	}

	return diff, "drift"
}

// verifySegmentBAtFreezeCutoff enforces the backward freeze-cutoff (hi = lockTime - WindowFreezeMargin) for Segment B.
// Subtracts records arriving after `hi` on indexed `_source_ts` so both Shadow and Master counts are anchored at `hi`.
func (rc *ReconCore) verifySegmentBAtFreezeCutoff(
	ctx context.Context,
	masterAgent *ReconDestAgent,
	ref MasterBindingRef,
	lo, hi time.Time,
	diff, shadowTotal, shadowActive, masterTotal, masterActive int64,
) (resolvedDiff, outMasterTotal, outMasterActive int64, statusStr string) {
	outMasterTotal = masterTotal
	outMasterActive = masterActive
	if diff == 0 {
		return 0, outMasterTotal, outMasterActive, "ok"
	}

	threshold := rc.cfg.CountDriftThreshold
	if threshold <= 0 {
		threshold = 100
	}
	if abs(diff) > threshold {
		return diff, outMasterTotal, outMasterActive, "drift"
	}

	verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	shadowRel, masterRel := ref.ShadowRel(), ref.MasterRel()
	nowUpper := time.Now().UTC().Add(1 * time.Minute)

	// Step 1: Subtract post-cutoff [hi, nowUpper] records using indexed _source_ts on Shadow and Master
	var (
		shAfterTotal, shAfterDel int64
		msAfterTotal, msAfterDel int64
		errSh1, errSh2           error
		errMs1, errMs2           error
		wg                       sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		shAfterTotal, errSh1 = rc.destAgent.CountInWindow(verifyCtx, shadowRel, "_source_ts", hi, nowUpper)
		shAfterDel, errSh2 = rc.destAgent.CountRecentDeletedRows(verifyCtx, shadowRel, "_source_ts", hi, nowUpper)
	}()
	go func() {
		defer wg.Done()
		msAfterTotal, errMs1 = masterAgent.CountInWindow(verifyCtx, masterRel, "_source_ts", hi, nowUpper)
		msAfterDel, errMs2 = masterAgent.CountRecentDeletedRows(verifyCtx, masterRel, "_source_ts", hi, nowUpper)
	}()
	wg.Wait()

	if errSh1 == nil && errSh2 == nil && errMs1 == nil && errMs2 == nil {
		shAfterActive := shAfterTotal - shAfterDel
		msAfterActive := msAfterTotal - msAfterDel
		shadowAtHi := shadowActive - shAfterActive
		masterAtHi := masterActive - msAfterActive
		if shadowAtHi == masterAtHi {
			rc.logger.Info("[smoke-B] Transient gap resolved via backward freeze-cutoff subtraction at hi",
				zap.String("shadow", shadowRel),
				zap.String("master", masterRel),
				zap.Int64("rawDiff", diff),
				zap.Int64("shAfterHi", shAfterActive),
				zap.Int64("msAfterHi", msAfterActive),
				zap.Int64("countAtHi", shadowAtHi),
				zap.Time("hi", hi),
			)
			return 0, shadowTotal, shadowActive, "ok"
		}
	}

	// Step 2: Fallback HashWindow on frozen range [lo, hi] using resolved Segment B PK pair and indexed _source_ts
	shadowPKCol, masterPKCol, _ := ResolveSegmentBPKPair(verifyCtx, ref, masterAgent)
	var (
		shHash, msHash *WindowResult
		errH1, errH2   error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		shHash, errH1 = rc.destAgent.HashWindow(verifyCtx, shadowRel, shadowPKCol, "_source_ts", lo, hi)
	}()
	go func() {
		defer wg.Done()
		msHash, errH2 = masterAgent.HashWindow(verifyCtx, masterRel, masterPKCol, "_source_ts", lo, hi)
	}()
	wg.Wait()

	if errH1 == nil && errH2 == nil && shHash != nil && msHash != nil &&
		shHash.Count == msHash.Count && shHash.XorHash == msHash.XorHash {
		rc.logger.Info("[smoke-B] Discrepancy resolved via HashWindow match on frozen range [lo, hi]",
			zap.String("shadow", shadowRel),
			zap.String("master", masterRel),
			zap.Int64("rawDiff", diff),
			zap.Time("lo", lo),
			zap.Time("hi", hi),
			zap.Int64("windowCount", shHash.Count),
		)
		return 0, shadowTotal, shadowActive, "ok"
	}

	return diff, outMasterTotal, outMasterActive, "drift"
}
```

### 5.3. Kết nối gọn nhẹ trong `centralized-data-service/internal/service/recon/recon_smoke.go` (Giảm dòng cho `recon_smoke.go`)
```go
// 1. Trong RunTotalOnlyA (thay thế toàn bộ khối Two-Guard 58 dòng tại recon_smoke.go:351-408 bằng 1 dòng):
	diff, statusStr := rc.verifySegmentAAtFreezeCutoff(ctx, entry, lo, hi, srcEst-dstActive)

// 2. Trong RunTotalOnlyB (thay thế khối tính diff thô tại recon_smoke.go:668-674 bằng lời gọi thực thi mốc lùi `hi`):
	var diffTimeJSON []byte
	diff, masterTotal, masterActive, statusStr := rc.verifySegmentBAtFreezeCutoff(
		ctx, masterAgent, ref, lo, hi,
		shadowActive-masterActive, shadowTotal, shadowActive, masterTotal, masterActive,
	)

// 3. Trong CheckAllUnified (tại recon_smoke.go:892-905 & 962-998):
	// Xóa snapshot cache cũ trước khi chạy batch COUNT(*) của chu kỳ mới để tránh lệch pha TTL 45s
	for i := range scanTargets {
		rc.smokeCountCache.Delete(scanTargets[i].Relation)
	}
	...
	// Dedup Segment A theo shadow relation (entry.QualifiedTarget()) — chỉ chạy 1 RunTotalOnlyA cho mỗi bảng Shadow
	seenSegA := make(map[string]bool)
	for _, p := range pipelineList {
		entry := *p.Entry
		ref := *p.Ref

		shadowKey := entry.QualifiedTarget()
		if !seenSegA[shadowKey] {
			seenSegA[shadowKey] = true
			idxA, okA := scanIdx[shadowKey]
			if !okA {
				rc.logger.Warn("smoke: scanIdx miss for SegA", zap.String("key", shadowKey))
			} else {
				stA := scanTargets[idxA]
				tableCtxA := tableCtxs[entry.TargetTable]
				csA := connSem(entry.SourceURL)

				executeGo(func() {
					select {
					case csA <- struct{}{}:
					case <-ctx.Done():
						return
					}
					defer func() { <-csA }()

					if stA.Err != nil {
						mu.Lock()
						reports = append(reports, rc.smokeErrorReportA(tableCtxA, entry, fmt.Errorf("shadow count failed: %w", stA.Err), 0))
						mu.Unlock()
						return
					}
					lockTime := start
					if rA := rc.RunTotalOnlyA(tableCtxA, entry, nil, stA.Total, stA.Active, lockTime, traceID); rA != nil {
						mu.Lock()
						reports = append(reports, rA)
						mu.Unlock()
					}
				})
			}
		}
```

---

## 6. Round 2 Fixes (APPROVE 2026-10-29T16:16) — ánh xạ 1-1 với `audit_report_2026_10_29.md` §R2

### 6.1. Lệch spec đã thực thi ở S4 — công bố & chấp nhận có điều kiện
- **R2-F3 (nguồn Postgres):** `verifySegmentAAtFreezeCutoff` với nguồn Postgres giữ nguyên hành vi cũ (`entry.TimestampField` dùng chung tên cột cho source và shadow; PK là `entry.PrimaryKeyField`) và **không gọi `HasColumnIndex`**. Lý do: phát hiện F2c chỉ đúng cho nguồn Mongo (camelCase, `_id` vs `_source_id`). **Rủi ro còn lại:** bảng Postgres-source lớn thiếu index trên `dstTS` vẫn có thể Seq Scan (nay giới hạn ở `hashCtx` 6s thay vì 15s) và không có Path 2. Rủi ro này được ghi nhận, chưa xử lý trong phạm vi này.
- **R2-F5 (Step 1 Segment B tuần tự):** 4 câu `CountInWindow`/`CountRecentDeletedRows` chạy tuần tự thay vì 2 goroutine. Lý do: đều là range scan trên index `_source_ts`, và `masterAgent` có thể dùng chung pool với `destAgent`.

### 6.2. Fix R2-F1 + R2-F2 — cache theo agent (đúng pattern `colTypes`) + chỉ khớp cột đầu tiên của index
`recon_dest_agent.go` (struct, chỉ thêm field):
```go
	idxColMu      sync.RWMutex
	idxCol        map[string]bool
```
`recon_window_index_probe.go` (`HasColumnIndex`):
```go
	cacheKey := strings.ToLower(tableName + "." + col)
	da.idxColMu.RLock()
	indexed, ok := da.idxCol[cacheKey]
	da.idxColMu.RUnlock()
	if ok {
		return indexed
	}
	...
	sql := `SELECT COUNT(*) FROM pg_indexes WHERE LOWER(schemaname) = LOWER(?) AND LOWER(tablename) = LOWER(?) AND indexdef ~* ?`
	pattern := leadingIndexColumnPattern(col)
	...
	da.idxColMu.Lock()
	if da.idxCol == nil {
		da.idxCol = make(map[string]bool)
	}
	da.idxCol[cacheKey] = indexed
	da.idxColMu.Unlock()

// leadingIndexColumnPattern matches col only as the FIRST key column right after "USING <method> (",
// so partial-index predicates like "WHERE (status = ...)" and non-leading composite columns never match.
func leadingIndexColumnPattern(col string) string {
	return `using\s+\w+\s+\(\s*"?` + regexp.QuoteMeta(strings.ToLower(col)) + `"?\s*[,) ]`
}
```

### 6.3. Fix R2-F4 — telemetry dùng số đo thật của Master
`recon_smoke.go` (`RunTotalOnlyB`):
```go
	rawMasterTotal, rawMasterActive := masterTotal, masterActive
	var diffTimeJSON []byte
	diff, masterTotal, masterActive, statusStr := rc.verifySegmentBAtFreezeCutoff(
		ctx, masterAgent, ref, lo, hi,
		shadowActive-masterActive, shadowTotal, shadowActive, masterTotal, masterActive,
	)
	span.SetAttributes(
		...
		attribute.Int64("db.row_count.master_active", rawMasterActive),
	)
	metrics.MasterTableRowCount.WithLabelValues(masterFQN).Set(float64(rawMasterTotal))
	metrics.MasterActiveRowCount.WithLabelValues(masterFQN).Set(float64(rawMasterActive))
```
Dòng kết quả UI vẫn giữ hành vi chuẩn hóa (`masterActive = shadowActive` khi resolve) như đã APPROVE ở §3.1.

### 6.4. Nghiệm thu thật (thay sqlmock) — file test mới `recon_window_index_probe_live_test.go`
- Theo pattern `discover_handler_live_pg_test.go`: đọc env `TEST_SYSTEM_DB_DSN` (PostgreSQL) và `TEST_MONGO_URL` (MongoDB), tự `t.Skip` khi thiếu; **không bao giờ log DSN**.
- Tạo schema tạm `recon_it_<nanos>` (DDL chỉ trong test, `DROP SCHEMA ... CASCADE` khi cleanup) và collection tạm.
- Kiểm chứng trên DB thật: (a) `HasColumnIndex` đúng với cột đầu index, `created` không khớp nhầm `created_at`, cột ở vị trí thứ 2 trả false, cache tách biệt giữa 2 agent; (b) `HashObjectIDWindow` (Mongo) == `HashSourceIDWindow` (PG) cả Count lẫn XorHash, biên `lo` được tính, biên `hi` bị loại, dòng `_deleted` bị loại; (c) `verifySegmentBAtFreezeCutoff` trên 2 bảng PG thật: chênh lệch do in-flight sau `hi` → `ok`; thiếu bản ghi trước `hi` → `drift`.
- Chạy trên Docker local dev (`gpay-postgres-shadow`, `gpay-mongo`), **không chạy trên Live**. Trace trên Live vẫn cần phối hợp với User (audit_report §R2-7).

---

## 7. Round 4 Solution (S3 Propose-Only) — Chốt Chặn Lùi Thời Gian (`cutoffHi`) Trực Tiếp Tại Bước `prefetch.{kind}` (`pg.count_rows` & `pg.count_deleted_rows`) và `source_shadow`

### 7.1. Nguyên nhân gốc rễ vì sao trên SigNoz `prefetch.shadow -> pg.count_rows` (`shadow_traitestpbs.payment_bills`) chưa có `time` (`R4-F1..R4-F3`)
1. **`R4-F1` (`scanExact` tại `recon_smoke.go:118-147` và `CountRows`/`CountDeletedRows` tại `recon_dest_query.go:16-94` chưa nhận `cutoffHi`):**
   - Tiến trình `cdc-worker` trên máy (`PID 16034`, chạy từ `16:28` qua `go run cmd/worker/main.go`) đã nạp code của Round 3, nhưng **trong Round 1–3 em hoàn toàn chưa chạm vào hàm `scanExact` (`prefetch.shadow` / `prefetch.master`) và `CountRows` (`pg.count_rows`)**.
   - Tại `recon_smoke.go:863`, `scanExact(tCtx, t.Agent, t.Relation, t.Kind)` vẫn gọi thẳng `agent.CountRows(traceCtx, relation, "_gpay_id")` (`SELECT COUNT("_gpay_id") FROM "shadow_traitestpbs"."payment_bills"`) và `agent.CountDeletedRows(traceCtx, relation)` **đếm trần tới `NOW()` mà không truyền mốc lùi thời gian `cutoffHi = start.Add(-rc.cfg.WindowFreezeMargin)`**.
   - Do đó, trên SigNoz (`trace_id = 05c8901f9c1ef749707cb58eaf5f875a`), span `prefetch.shadow -> pg.count_rows` chỉ có 2 attribute `db.pk_column = _gpay_id` và `db.table = shadow_traitestpbs.payment_bills`, hoàn toàn không có tham số thời gian (`time.to` / `recon.cutoff_ms`).
2. **`R4-F2` (Đặt phép trừ cửa sổ lùi ở nhánh hậu kiểm `if diff != 0` tại `recon_smoke_guard.go:43, 177` thay vì tại bước đếm gốc `prefetch`):**
   - Trong Round 1–3, `verifySegmentAAtFreezeCutoff` và `verifySegmentBAtFreezeCutoff` đều mở đầu bằng `if diff == 0 { return 0, ... }`.
   - Khi bảng `shadow_traitestpbs.payment_bills` trên môi trường `development` có `diff == 0` tại `NOW()`, cả 2 hàm hậu kiểm thoát ngay ở dòng đầu tiên, không chạy bất kỳ truy vấn cửa sổ thời gian nào, và số đếm lưu vào DB/UI (`56,854 -> 56,856 -> 56,857 -> 56,859`) vẫn là số đếm trôi theo `NOW()` thay vì chốt tại `hi = start - WindowFreezeMargin`.
3. **`R4-F3` (Bẫy hiệu năng `Seq Scan` nếu viết trực tiếp `WHERE "_source_ts" < hi` vs Phép bù 2 `Index Only Scan` trong 1 transaction, và đồng bộ `Source` tại `hi`):**
   - Kiểm chứng thực tế bằng `EXPLAIN (ANALYZE, BUFFERS)` trên `gpay-postgres-shadow` (`shadow_traitestpbs.payment_bills`, `56,881` dòng):
     - Nếu viết trực tiếp `SELECT COUNT("_gpay_id") FROM shadow_traitestpbs.payment_bills WHERE "_source_ts" < hi`: Do `_source_ts < hi` chọn `99.99%` bảng, PostgreSQL Planner từ chối Index-Only Scan và chuyển sang **`Seq Scan on payment_bills`** (đọc `14,366` heap buffers chứa `_raw_data` JSONB), tốn **`2,506.31 ms`** trên 56k dòng (chậm gấp 70 lần) và sẽ timeout `> 60s` trên bảng 15.37M dòng ở Live.
     - Ngược lại, dùng đồng nhất thức tập hợp $A_{< hi} = A \setminus A_{\ge hi}$ ngay trong **1 câu SQL duy nhất trên cùng 1 read-only transaction** (`CountRowsBeforeCutoff`):
       ```sql
       SELECT
         (SELECT COUNT("_gpay_id") FROM "shadow_traitestpbs"."payment_bills") AS raw_total,
         (SELECT COUNT(*) FROM "shadow_traitestpbs"."payment_bills" WHERE "_source_ts" >= ?) AS post_cutoff
       ```
       PostgreSQL thực thi bằng **2 `Index Only Scan` InitPlans** (`payment_bills_pkey` + `idx_payment_bills_source_ts`), chỉ đọc `4` buffer ở nhánh `_source_ts >= hi` (`0.055 ms`), tổng thời gian **`37.40 ms`**, trả về chính xác `countAtHi = rawTotal - postCutoff`!
     - Tương tự, `CountDeletedRowsBeforeCutoff` chạy `SELECT COUNT(*) FROM ... WHERE "_deleted" = true AND "_source_ts" < ?` dùng `Index Scan using idx_payment_bills_deleted_partial` chỉ tốn **`0.89 ms`** (`shared hit=18`).
   - Đồng thời, khi `prefetch.shadow` đã chốt `dstActive` tại `hi`, bước đếm `Source` trong `RunTotalOnlyA` (`recon.smoke.mongo_estimate` / `recon.smoke.postgres_count`) cũng bắt buộc gắn `time.to = hi` và gọi `rc.adjustSourceCountAtFreezeCutoff(fastCtx, entry, srcEst, hi)` để trừ số bản ghi phát sinh trong `[hi, now]` trên Source (tránh lệch pha giữa `srcEst` tại `NOW()` và `dstActive` tại `hi`).

### 7.2. Bảng Ánh xạ Điểm Gọi Thực thi & Tách File Đơn Trách Nhiệm (Explicit Runtime Call-Site Wiring & Line-Budget Guard)
| Thứ tự | Tầng Kiến trúc (Layer) | File & Hàm mới / cập nhật | Được gọi tại (`caller file:line`) | Thời điểm kích hoạt (`Lifecycle Stage`) | Ngân sách dòng (`wc -l`) |
| :---: | :--- | :--- | :--- | :--- | :--- |
| 1 | **Data-Access Layer** (`ReconDestAgent`) | `recon_window_index_probe.go`: thêm `CountRowsBeforeCutoff` & `CountDeletedRowsBeforeCutoff` | `recon_smoke.go:132, 137` (`scanExact`) | Bước 3 (`prefetch.shadow` & `prefetch.master`) của mỗi chu kỳ `CheckAllUnified` | `188 -> ~285` dòng (File chuyên biệt; **không chạm `recon_dest_query.go` 708 dòng**) |
| 2 | **Service Guard Layer** (`ReconCore`) | `recon_smoke_guard.go`: thêm `adjustSourceCountAtFreezeCutoff` | `recon_smoke.go:286` (`RunTotalOnlyA`) | Ngay sau khi lấy `srcEst` trong `RunTotalOnlyA` (`source_shadow`) để neo `srcEst` về cùng mốc `hi` với `prefetch.shadow` | `239 -> ~280` dòng (File chuyên biệt) |
| 3 | **Service Orchestrator Layer** (`ReconCore`) | `recon_smoke.go`: truyền `cutoffHi := start.Add(-rc.cfg.WindowFreezeMargin)` vào `scanExact` và gắn `time.to` lên `prefetch.{kind}` + `mongoSpan` | `recon_smoke.go:118-147, 269-288, 841-866` (`CheckAllUnified`, `scanExact`, `RunTotalOnlyA`) | Toàn trình chu kỳ `CheckAllUnified` | `1,240 -> ~1,245` dòng (`<= 1,285` dòng của `HEAD`) |

### 7.3. Code Demo Đầy Đủ (Trước / Sau kèm `file:line`)

#### (A) `centralized-data-service/internal/service/recon/recon_window_index_probe.go` (Thêm `CountRowsBeforeCutoff` & `CountDeletedRowsBeforeCutoff`)
```go
// CountRowsBeforeCutoff counts rows as of cutoffHi (_source_ts < cutoffHi) using two Index-Only Scans
// in a single read-only transaction: COUNT(pkColumn) - COUNT(*) WHERE _source_ts >= cutoffHi.
// Avoids the Full-Table Seq Scan caused by direct WHERE _source_ts < cutoffHi (99.99% selectivity).
func (da *ReconDestAgent) CountRowsBeforeCutoff(ctx context.Context, tableName, pkColumn string, cutoffHi time.Time) (int64, error) {
	if cutoffHi.IsZero() {
		return da.CountRows(ctx, tableName, pkColumn)
	}
	cutoffMs := cutoffHi.UnixMilli()
	if da.IsMongo() {
		total, err := da.mongoCountRows(ctx, tableName)
		if err != nil {
			return 0, err
		}
		post, errW := da.mongoCountInWindow(ctx, tableName, "_source_ts", cutoffHi, time.Now().UTC().Add(time.Minute))
		if errW == nil && total >= post {
			total -= post
		}
		return total, nil
	}
	ctx, span := observability.ChildSpan(ctx, "pg.count_rows",
		attribute.String("db.table", tableName),
		attribute.String("db.pk_column", pkColumn),
		attribute.String("time.to", cutoffHi.UTC().Format(time.RFC3339)),
		attribute.Int64("recon.cutoff_ms", cutoffMs),
	)
	defer span.End()

	if err := validateIdent(tableName); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	type cutoffCountRow struct {
		RawTotal   int64 `gorm:"column:raw_total"`
		PostCutoff int64 `gorm:"column:post_cutoff"`
	}

	result, err := da.breaker.Execute(func() (interface{}, error) {
		var row cutoffCountRow
		rel := quoteRelation(tableName)
		if pkColumn != "" && validateIdent(pkColumn) == nil {
			tx := da.readOnlyDB(ctx)
			sql := fmt.Sprintf(
				`SELECT (SELECT COUNT(%s) FROM %s) AS raw_total, (SELECT COUNT(*) FROM %s WHERE "_source_ts" >= ?) AS post_cutoff`,
				quoteIdent(pkColumn), rel, rel,
			)
			err := tx.Raw(sql, cutoffMs).Scan(&row).Error
			tx.Rollback()
			if err == nil {
				return row, nil
			}
		}

		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		sql := fmt.Sprintf(
			`SELECT (SELECT COUNT(*) FROM %s) AS raw_total, (SELECT COUNT(*) FROM %s WHERE "_source_ts" >= ?) AS post_cutoff`,
			rel, rel,
		)
		if err := tx.Raw(sql, cutoffMs).Scan(&row).Error; err != nil {
			return nil, err
		}
		return row, nil
	})
	if err != nil {
		return 0, err
	}
	row := result.(cutoffCountRow)
	count := row.RawTotal - row.PostCutoff
	if count < 0 {
		count = 0
	}
	span.SetAttributes(
		attribute.Int64("db.row_count.raw", row.RawTotal),
		attribute.Int64("db.row_count.post_cutoff", row.PostCutoff),
		attribute.Int64("db.row_count", count),
	)
	return count, nil
}

// CountDeletedRowsBeforeCutoff counts soft-deleted rows (_deleted = true) with _source_ts < cutoffHi,
// leveraging the partial B-Tree index idx_<table>_deleted_partial (WHERE _deleted = true).
func (da *ReconDestAgent) CountDeletedRowsBeforeCutoff(ctx context.Context, tableName string, cutoffHi time.Time) (int64, error) {
	if cutoffHi.IsZero() {
		return da.CountDeletedRows(ctx, tableName)
	}
	cutoffMs := cutoffHi.UnixMilli()
	if da.IsMongo() {
		delTotal, err := da.mongoCountDeletedRows(ctx, tableName)
		if err != nil {
			return 0, err
		}
		delPost, errW := da.mongoCountRecentDeletedRows(ctx, tableName, "_source_ts", cutoffHi, time.Now().UTC().Add(time.Minute))
		if errW == nil && delTotal >= delPost {
			delTotal -= delPost
		}
		return delTotal, nil
	}
	ctx, span := observability.ChildSpan(ctx, "pg.count_deleted_rows",
		attribute.String("db.table", tableName),
		attribute.String("time.to", cutoffHi.UTC().Format(time.RFC3339)),
		attribute.Int64("recon.cutoff_ms", cutoffMs),
	)
	defer span.End()

	if err := validateIdent(tableName); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	tx := da.readOnlyDB(ctx)
	defer tx.Rollback()
	var count int64
	sql := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE "_deleted" = true AND "_source_ts" < ?`, quoteRelation(tableName))
	if err := tx.Raw(sql, cutoffMs).Scan(&count).Error; err != nil {
		return 0, err
	}
	span.SetAttributes(attribute.Int64("db.row_count", count))
	return count, nil
}
```

#### (B) `centralized-data-service/internal/service/recon/recon_smoke_guard.go` (Thêm `adjustSourceCountAtFreezeCutoff`)
```go
// adjustSourceCountAtFreezeCutoff subtracts post-cutoff [hi, nowUpper] records from rawSrcCount
// so that Source count in RunTotalOnlyA is anchored at the exact same freeze cutoff `hi` as prefetch.shadow.
func (rc *ReconCore) adjustSourceCountAtFreezeCutoff(
	ctx context.Context,
	entry source.TableRegistry,
	rawSrcCount int64,
	hi time.Time,
) int64 {
	if hi.IsZero() || rc.sourceAgent == nil {
		return rawSrcCount
	}
	nowUpper := time.Now().UTC().Add(time.Minute)
	winCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var srcTS string
	if entry.TimestampField != nil && *entry.TimestampField != "" {
		srcTS = *entry.TimestampField
	} else {
		srcTS, _, _ = rc.resolveSourceAndDestTSFields(winCtx, entry, "smoke")
	}

	if srcTS != "" {
		if postHi, err := rc.sourceAgent.CountInWindow(winCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, hi, nowUpper); err == nil {
			if rawSrcCount >= postHi {
				return rawSrcCount - postHi
			}
			return 0
		}
	}

	if !isPostgres(entry.SourceURL) {
		oidCtx, oidCancel := context.WithTimeout(ctx, 2*time.Second)
		defer oidCancel()
		if oidWin, err := rc.sourceAgent.HashObjectIDWindow(oidCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable, hi, nowUpper); err == nil && oidWin != nil {
			if rawSrcCount >= oidWin.Count {
				return rawSrcCount - oidWin.Count
			}
			return 0
		}
	}
	return rawSrcCount
}
```

#### (C) `centralized-data-service/internal/service/recon/recon_smoke.go` (Truyền `cutoffHi` vào `scanExact` và `RunTotalOnlyA`)
```go
// 1. Tại scanExact (recon_smoke.go:118-147):
func (rc *ReconCore) scanExact(ctx context.Context, agent *ReconDestAgent, relation, kind string, cutoffHi time.Time) (total, active int64, err error) {
	if snap, ok := rc.loadCountSnapshot(relation); ok {
		return snap.total, snap.active, nil
	}
	traceCtx, span := observability.ChildSpan(ctx, "prefetch."+kind,
		attribute.String("db.system", "postgresql"),
		attribute.String("db.sql.table", relation),
		attribute.String("table.role", kind),
		attribute.String("query.total.method", "count_star_exact"),
		attribute.String("time.to", cutoffHi.UTC().Format(time.RFC3339)),
	)
	defer func() { observability.EndSpan(span, &err) }()

	total, err = agent.CountRowsBeforeCutoff(traceCtx, relation, "_gpay_id", cutoffHi)
	if err != nil {
		return 0, 0, err
	}
	active = total
	if del, errD := agent.CountDeletedRowsBeforeCutoff(traceCtx, relation, cutoffHi); errD == nil {
		if a := total - del; a >= 0 {
			active = a
		}
	}
	span.SetAttributes(
		attribute.Int64("db.row_count.total", total),
		attribute.Int64("db.row_count.active", active),
	)
	rc.storeCountSnapshot(relation, total, active)
	return total, active, nil
}

// 2. Tại RunTotalOnlyA (recon_smoke.go:269-286):
	mongoCtx, mongoSpan := observability.ChildSpan(fastCtx, spanName,
		attribute.String("db.system", dbSystem),
		attribute.String("db.name", entry.SourceDB),
		attribute.String("db.collection", entry.SourceTable),
		attribute.String("time.to", hi.UTC().Format(time.RFC3339)),
	)
	...
	if mongoErr == nil {
		mongoSpan.SetAttributes(attribute.Int64("db.row_count.raw", srcEst))
		srcEst = rc.adjustSourceCountAtFreezeCutoff(fastCtx, entry, srcEst, hi)
		mongoSpan.SetAttributes(attribute.Int64("db.row_count", srcEst))
	}

// 3. Tại CheckAllUnified Step 3 (recon_smoke.go:841-866):
	cutoffHi := start.Add(-rc.cfg.WindowFreezeMargin)
	...
	t.Total, t.Active, t.Err = rc.scanExact(tCtx, t.Agent, t.Relation, t.Kind, cutoffHi)
```

