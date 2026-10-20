---
tier: T2
workspace: audit-shadow-master-update-drop-2026-10-20
suffix: shadow_master_update_drop
---

# Final Adversarial Audit Report: Phản Biện Toàn Diện Quá Trình & Code Commit `30b1b00` (Chặng Shadow -> Master `F1..F5` & Phát Hiện Mới `AF-01..AF-05`)

## 1. Line-by-Line Diff & Task Critique (Phản biện từng dòng code `git diff` trên 11 file của commit `30b1b00` & từng Task `1..8`)

### 1.1. Phản biện từng File & từng cụm dòng thay đổi trong `git show 30b1b00` (`11 files changed, 1089 insertions(+), 153 deletions(-)`)

| STT | File & Dòng (`git show 30b1b00`) | Thay đổi thực tế | Câu hỏi Phản biện (Cách phá / Edge Case) | Bằng chứng Code & Runtime thật (`VERIFIED`) | Kết luận |
|---|---|---|---|---|---|
| 1 | `internal/handler/master/transmute_handler.go` (`+90 / -29`, `L472-481`, `L690-795`, `L866-899`) | (1) `L472-481`: Khi `isTransientError(err)` trong `runDebouncedGroup`, trả lỗi qua `t.complete(err)` để NATS redeliver. (2) `L690-795`: Thêm `splitSingleTaskPayload` chia đôi `task.Req.Events` hoặc `task.Req.SourceIDs` khi `len(batch) == 1`. (3) `L866-899`: Mở rộng `isTransientError` bắt thêm `connection reset`, `connection closed`, `driver: bad connection`, `unexpected eof`, `sqlstate 40001/57p01/57p02/57p03`. | **Cách phá 1:** Điều gì xảy ra tại `splitSingleTaskPayload` (`L790-795`) khi 1 message NATS có $\ge 2$ phần tử mà 1 nửa thành công (`leftErr == nil`) và 1 nửa thất bại (`rightErr != nil`, không phải transient)?<br>**Cách phá 2:** Tại sao `RunEvents` đã lặp qua từng event độc lập nhưng khi 1 event lỗi lại kích hoạt `binarySearchSplit` chạy lại các event đã thành công? | **Bắt được lỗi thật (`AF-02`):**<br>1. Tại `transmute_handler.go:790-794`: Code viết `if leftErr != nil && rightErr != nil { task.complete(leftErr); return } task.complete(nil)`. Khi chỉ 1 nửa lỗi (`leftErr == nil && rightErr != nil`), hàm gọi `task.complete(nil)` (và không gọi `h.reply(task.Msg, ...)`). Hệ quả: `TransmuteConsumerPool` nhận `err == nil` và gọi `msg.Ack()` xóa luôn message khỏi JetStream ngay lần đầu tiên mà không chờ redeliver (ví dụ khi 1 event có cột mới chờ DDL), đồng thời nếu caller gọi đồng bộ qua `Msg.Reply` sẽ bị timeout!<br>2. Trong `TransmuterModule.RunEvents` (`transmuter.go:700-810`), vòng lặp `for _, ev := range events` đã thực thi DB độc lập cho từng event và chỉ gán `lastEventErr = errUpsert` khi lỗi. Việc trả `lastEventErr` về `runDebouncedGroup` (`transmute_handler.go:652`) làm kích hoạt `binarySearchSplit` $\rightarrow$ `splitSingleTaskPayload`, khiến các event hợp lệ bị upsert lặp lại $\log_2(N)$ lần vào Master DB! | **FAIL (`AF-02`)** |
| 2 | `internal/handler/master/transmute_debounce_group_test.go` (`+20`, `L171-190`) | Thêm unit test `TestIsTransientError_CoversConnectionResetAndBadConnection`. | Trong báo cáo cũ lúc `15:28`, file này bị ghi nhầm tên thành `transmute_handler_test.go`. Ngoài ra đã có unit test cho `splitSingleTaskPayload` khi 1 nửa lỗi chưa? | Thực tế tên file là `internal/handler/master/transmute_debounce_group_test.go:171-190`. Chưa có unit test kiểm tra hành vi `splitSingleTaskPayload` khi `leftErr == nil && rightErr != nil`. | **FAIL (`AF-01`, `AF-02`)** |
| 3 | `internal/service/master/transmuter.go` (`+415 / -65`, `L400-405`, `L542-557`, `L685-810`, `L1185-1208`, `L1425-1684`, `L1710-1745`) | (1) `L685-696` & `L1425-1471`: Thêm `lookupShadowGpayIDs` lấy `_gpay_id` thực từ Shadow trong `RunEvents`. (2) `L702-720` & `L1551-1684`: Thêm `deleteFromMasterBySourceAndGpayIDs` đọc động `conflictTarget` từ `transform_spec.pk` + bọc retry. (3) `L545-547` & `L1475-1546`: Thêm `syncUpdatedShadowRowsBeforeCursor`. (4) `L1711`: Dùng `unionSortedKeys(records)`. (5) `L1740`: Cho phép `ON CONFLICT DO UPDATE` cập nhật `_gpay_id` khi `conflictTarget != "_gpay_id"`. | **Cách phá 1:** Trong `syncUpdatedShadowRowsBeforeCursor` (`L1504-1512`), điều kiện `cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour)` với `_updated_at >= cutoff` quét bao nhiêu dòng trên DB thật?<br>**Cách phá 2:** Tại `L553-557`, điều kiện `if len(onlySourceIDs) == 0` chạy cho cả Incremental Sync (`runType != "full"`) thì chuyện gì xảy ra với `LastCursorJSON`? | **Bắt được 2 lỗi thật (`AF-03`):**<br>1. Truy vấn thực tế trên `shadow_traitestpbs.payment_bills` cho thấy **54,754 / 54,754 dòng** đều có `_updated_at >= NOW() - INTERVAL '7 days'` (trong khi `_updated_at >= NOW() - INTERVAL '1 hour'` chỉ có `22` dòng), vì mọi dòng mới `INSERT` trong 7 ngày qua cũng có `_updated_at = NOW()`. Do không lấy mốc `last_success_at` từ `master_binding_runtime_state` (hoặc lọc `_updated_at > _created_at`), mỗi nhịp Incremental Sync sẽ quét lại toàn bộ 54,754 dòng!<br>2. Tại `transmuter.go:530-533`, vòng lặp vừa lưu `LastCursorJSON = {"last_gpay_id": lastGpayID}`, nhưng ngay sau khi thoát vòng lặp, dòng `553-557` (`if len(onlySourceIDs) == 0`) lại ghi đè `item.LastCursorJSON = []byte("{}")` cho cả Incremental Sync (`runType != "full"`), xóa mất watermark vừa lưu (vi phạm Lesson `L008`)! | **FAIL (`AF-03`)** |
| 4 | `internal/service/master/transmuter_utils.go` (`+30`, `L222-251`) | Thêm `unionSortedKeys(records []map[string]any) []string` lấy tập hợp tất cả các cột của mọi phần tử trong `records` và sắp xếp tăng dần. | Nếu `records` rỗng hoặc có phần tử `map` rỗng thì sao? | Nếu `len(records) == 0` trả về `nil`; nếu chỉ có 1 phần tử gọi thẳng `sortedKeysAny(records[0])`; khi nhiều phần tử gom qua `seen := make(map[string]struct{})` rồi `sort.Strings(keys)`. Hoàn toàn an toàn và đúng đắn. | **PASS** |
| 5 | `internal/service/master/transmuter_orphan_test.go` (`+24 / -1`, `L74-96`) | Thêm `TestUnionSortedKeys_IncludesColumnsFromAllRecords`. | File này có được liệt kê trong báo cáo cũ lúc `15:28` không? | Báo cáo cũ lúc `15:28` đã **bỏ sót hoàn toàn** tên file `internal/service/master/transmuter_orphan_test.go` trong mục danh sách file thay đổi (`11_report`). | **FAIL (`AF-01`)** |
| 6 | `internal/service/master/transmuter_gpayid_test.go` (`+340`, `L204-543`) | Thêm `TestTransmuteShadowToMaster_F1_F2_F3_F5_RealPostgres` (`L204-354`) và `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` (`L356-543`). | **Cách phá 1:** Báo cáo cũ ghi tên file là `transmuter_f1_f5_integration_test.go` — có đúng sự thật không?<br>**Cách phá 2:** Trên DB `cdc_system.master_binding` thật có bao nhiêu binding đang `is_active = true`? Việc hardcode `[]int64{46, 47, 48}` ở dòng `402` có bỏ sót binding nào không? | **Bắt được lỗi thật (`AF-01`):**<br>1. Báo cáo cũ ghi sai tên file thành `transmuter_f1_f5_integration_test.go` (do viết từ tóm tắt nén ngữ cảnh). Tên thật là `transmuter_gpayid_test.go`.<br>2. Truy vấn trực tiếp `SELECT id, shadow_schema, shadow_table, master_connection_key, master_schema, master_table, is_active FROM cdc_system.master_binding WHERE is_active = true` trả về **5 active bindings**: `46, 47, 48, 51, 52`! Do dòng `402` hardcode `[]int64{46, 47, 48}`, 2 binding **`51`** (`export_jobs_2` trên `master_2` port `5437`) và **`52`** (`export_jobs_1` trên `default_master` port `5434`) đã bị bỏ sót: hiện tại `shadow_traitestces.export_jobs` có `540` dòng active (`1` dòng `_deleted = true`), nhưng bảng Master của `Binding 51` và `Binding 52` vẫn đang còn `541` dòng (chưa xóa 1 dòng `_deleted = true`). | **FAIL (`AF-01`)** |
| 7 | `internal/service/recon/recon_engine_segment_b.go` (`+2`, `L27`, `L80`) | Thêm trường `TransformSpec string` vào `MasterBindingRef` và câu `SELECT COALESCE(transform_spec::text, '')` trong `LoadMasterBindingByTarget`. | Có ảnh hưởng tới các caller hiện hữu của `MasterBindingRef` không? | Hoàn toàn tương thích ngược, giúp truyền `TransformSpec` từ `cdc_system.master_binding` sang `ExecuteHealHandler`. | **PASS** |
| 8 | `internal/handler/recon/recon_execute_heal_handler.go` (`+221 / -55`, `L307-381`, `L444-539`, `L595-621`, `L680-755`) | (1) Thêm `publishTransmuteShadowForSegA` (`L339-381`). (2) Cập nhật `executeHealSegB` (`L444-539`) và `resolveSourceIDsForSegmentB` (`L680-755`) dùng `resolvePKColumnAndType` (`L595-621`). | **Cách phá 1:** Báo cáo cũ nói `resolvePKColumnAndType` kiểm tra `information_schema.columns` tại `L314-322` — có thật không?<br>**Cách phá 2:** Trong `publishTransmuteShadowForSegA` (`L361`), `"shadow_connection_key": "default"` có khớp với DB thật không và có chia chunk không?<br>**Cách phá 3:** Trong `resolveSourceIDsForSegmentB` (`L733-751`), khi `len(mapped) < (end - start)`, việc query `gpayIDs[start:end]` trên Master rồi `append(out, masterMapped...)` có gây trùng lặp `sourceIDs` không? | **Bắt được 3 lỗi thật (`AF-01`, `AF-04`, `AF-05`):**<br>1. `AF-01`: Hàm `resolvePKColumnAndType` nằm ở `L595-621` và **không hề truy vấn `information_schema.columns`** như báo cáo cũ viết.<br>2. `AF-04`: Tại `L361`, `publishTransmuteShadowForSegA` hardcode `"shadow_connection_key": "default"`, trong khi `cdc_system.master_binding` thực tế dùng `"default_shadow"`, và bắn toàn bộ `sourceIDs` trong 1 message NATS duy nhất mà không chia chunk theo `healChunkSize = 200`.<br>3. `AF-05`: Tại `L733-751`, khi 1 chunk 500 `gpayIDs` có 499 ID khớp trên Shadow và 1 ID nằm trên Master, nhánh fallback query lại cả 500 `gpayIDs` trên bảng Master rồi `out = append(out, masterMapped...)` mà không khử trùng lặp (`uniqueStrings(out)`), làm nhân đôi 499 `sourceIDs`! | **FAIL (`AF-01`, `AF-04`, `AF-05`)** |
| 9 | `internal/handler/shadow/batch_buffer.go` (`+22 / -5`, `L724-749`) | Gọi `dedupedRecords := dedupByConflictKey(sc.Records)` trước khi chia nhóm `groups[sig]`, và lưu thứ tự vào `sigOrder` để duyệt tuần tự thay vì duyệt Go `map`. | Thứ tự giữa các nhóm `sig` khác nhau đã xác định chưa? | Đã xác định theo thứ tự xuất hiện đầu tiên trong `dedupedRecords` (`sigOrder`), loại bỏ hoàn toàn tính ngẫu nhiên khi lặp qua Go `map`. | **PASS** |
| 10 | `internal/handler/shadow/batch_buffer_utils.go` (`+30 / -2`, `L90-136`) | Gộp `MappedData` của các bản ghi cùng `PrimaryKeyValue` vào `mergedData[pk]` trước khi trả về bản ghi ở vị trí `lastIdx[pk]`. | Điều gì xảy ra nếu trong cùng 1 `sc.Records`, 2 sự kiện của cùng 1 `PrimaryKeyValue` đến không đúng thứ tự thời gian (`records[0].SourceTsMs > records[1].SourceTsMs`) hoặc sự kiện trước là `DELETE` (`_deleted = true`)? | **Bắt được lỗi edge-case (`AF-05`):**<br>Tại `batch_buffer_utils.go:96-122`, vòng lặp luôn lấy chỉ số cuối mảng `lastIdx[r.PrimaryKeyValue] = i` và ghi đè `m[k] = v` theo thứ tự vị trí mảng mà không so sánh `r.SourceTsMs`. Nếu 1 sự kiện cũ (`SourceTsMs` nhỏ hơn) nằm sau sự kiện mới trong cùng slice `sc.Records`, giá trị cột của sự kiện cũ sẽ ghi đè lên sự kiện mới! | **FAIL (`AF-05`)** |
| 11 | `internal/handler/shadow/adaptive_batcher_test.go` (`+48`, `L388-435`) | Thêm `TestDedupByConflictKey_MergesMappedDataAcrossHeterogeneousEvents`. | Báo cáo cũ ghi tên file là `batch_buffer_test.go` — có đúng sự thật không? | Sai tên file trong báo cáo cũ (`AF-01`): thực tế test được viết trong `internal/handler/shadow/adaptive_batcher_test.go:388-435`. | **FAIL (`AF-01`)** |

### 1.2. Phản biện việc thực hiện từng Task (`Task 1..8`) trong `08_tasks_shadow_master_update_drop.md`

- **Task 1, Task 2, Task 3 (Intake, Investigation, Solution Design):** Đã xác định đúng 5 nguyên nhân gốc rễ `F1..F5`. Tuy nhiên ở bước Investigation ban đầu chỉ kiểm tra 3 bindings `46, 47, 48` mà chưa kiểm tra 2 active bindings `51, 52` mới được tạo thêm.
- **Task 4 (`F1`), Task 5 (`F2`), Task 6 (`F3`), Task 7 (`F4`), Task 8 (`F5`):**
  - Đã giải quyết được lỗi chính giúp đồng bộ thành công `54,722/54,722` bản ghi trên `Binding 46, 47` và `540/540` bản ghi trên `Binding 48`.
  - **Tuy nhiên còn thiếu sót phạm vi cam kết ở `Task 6.2` (`AF-04`):** Trong `09_tasks_solution_shadow_master_update_drop.md` (`L213-216`), giải pháp cam kết cập nhật cả `internal/service/recon/recon_heal.go` (`HealMissingIDs` / `HealWindow`), nhưng thực tế commit `30b1b00` chỉ sửa `ExecuteHealHandler` (`recon_execute_heal_handler.go`) mà chưa sửa `recon_heal.go` hay `executeLegacyHeal` trong `internal/handler/recon/recon_check_heal_handler.go:302-360`.

---

## 2. Anti-Smoke-Speculation & Real Runtime Trace Verification (Chống Mockup, Smoke Test & Hallucination)

- **Kiểm tra Production Mock Pollution:**
  - Quét toàn bộ 7 file production trong commit `30b1b00`: **0 closure giả (`*Fn`), 0 `sqlmock`, 0 `fakemodel` trong code production.**
- **Kiểm tra Thực trạng Tiến trình Runtime & Docker Containers Thực tế (Không suy diễn):**
  1. **Cấu hình Port Docker PostgreSQL thực tế (`docker ps` & `config/config-local.yml`):**
     - `127.0.0.1:5433` $\rightarrow$ `gpay-postgres-cdc` (DB: `cdc_dw`, chứa `cdc_system.master_binding`)
     - `127.0.0.1:5436` $\rightarrow$ `gpay-postgres-shadow` (DB: `cdc_shadow`, chứa `shadow_traitestpbs.payment_bills` & `shadow_traitestces.export_jobs`)
     - `127.0.0.1:5434` $\rightarrow$ `gpay-postgres-dest` (DB: `goopay_dest`, chứa `Binding 46, 47, 48, 52`)
     - `127.0.0.1:5437` $\rightarrow$ `gpay-postgres-master-2` (DB: `goopay_master_2`, chứa `Binding 51`)
     - *(Lưu ý: Báo cáo cũ lúc `15:28` đã ghi nhầm port `5435` và `5433`).*
  2. **Thực trạng Dữ liệu trên cả 5 Active Bindings trong `cdc_system.master_binding`:**
     - **Binding 46 (`goopay_dest.master_payment_bill_service.payment_bills`):** Shadow active = `54,722`, Master = `54,722`, `droppedDelete = 0`, `droppedUpdate = 0`, `gpayMismatch = 0` (**ĐÃ KHỚP 100%**).
     - **Binding 47 (`goopay_dest.master_payment_bill_service.payment_bills_1`):** Shadow active = `54,722`, Master = `54,722`, `droppedDelete = 0`, `droppedUpdate = 0`, `gpayMismatch = 0` (**ĐÃ KHỚP 100%**).
     - **Binding 48 (`goopay_dest.master_centrallized_export_service.export_jobs`):** Shadow active = `540`, Master = `540`, `droppedDelete = 0`, `droppedUpdate = 0`, `gpayMismatch = 0` (**ĐÃ KHỚP 100%**).
     - **Binding 51 (`goopay_master_2.master_centrallized_export_service.export_jobs_2` trên port `5437`):** Shadow active = `540` (`1` dòng `_deleted = true`), nhưng Master đang có **`541` dòng** (**CHƯA ĐỒNG BỘ XÓA 1 DÒNG `_deleted = true`** do `TestLiveBindings` hardcode `{46, 47, 48}`).
     - **Binding 52 (`goopay_dest.master_centrallized_export_service.export_jobs_1` trên port `5434`):** Shadow active = `540` (`1` dòng `_deleted = true`), nhưng Master đang có **`541` dòng** (**CHƯA ĐỒNG BỘ XÓA 1 DÒNG `_deleted = true`** do `TestLiveBindings` hardcode `{46, 47, 48}`).
  3. **Thực trạng 3 Tiến trình Daemon đang chạy trên máy (`ps` & `lsof`):**
     - `PID 34453` (`:8081` — API/Control Plane): khởi chạy lúc `08:55:21` ngày `20/10/2026`.
     - `PID 50156` (`:8083` — Worker): khởi chạy lúc `10:30:43` ngày `20/10/2026`.
     - `PID 50249` (`:8082` — Worker): khởi chạy lúc `10:31:16` ngày `20/10/2026`.
     - **Sự thật:** Cả 3 tiến trình daemon đang lắng nghe trên `:8081/:8082/:8083` đều được khởi chạy **trước** thời điểm commit `30b1b00` (`15:19:20`), tức là tiến trình daemon nền vẫn đang chạy binary cũ! Kết quả đồng bộ `Binding 46, 47, 48` lúc `15:18` là do chạy trực tiếp `TransmuterModule.Run` từ tiến trình `go test` kết nối vào DB thật, chứ chưa phải do daemon `:8081/:8082/:8083` đã được restart với code mới.
- **Điểm cần phối hợp trực tiếp với User (Anh Trai) để Trace Runtime Value thật:**
  1. Sau khi anh `APPROVE` bản vá khắc phục 5 lỗi `AF-01..AF-05`, em sẽ cập nhật code, chạy test đồng bộ cho toàn bộ **5 active bindings (`46, 47, 48, 51, 52`)**, và cần **phối hợp với anh để rebuild & restart 3 tiến trình `main` trên `:8081`, `:8082`, `:8083`** (vì 3 tiến trình này đang chạy từ các terminal `s000/s001/s002` của anh).
  2. Sau khi restart daemon mới, phối hợp cùng anh bắn 1 sự kiện `UPDATE` và `DELETE` thực tế từ MongoDB (hoặc qua NATS `cdc.cmd.transmute-shadow`) để trace log của daemon `:8082/:8083` và kiểm chứng `master_binding_runtime_state.last_cursor_json` trên runtime thật.

---

## 3. Concurrency, Timeout/Heartbeat & Data Continuity Audit (Kiểm tra Đa luồng, Timeout & Liên tục dữ liệu)

- **Concurrency / Multi-worker / Poison Pill Isolation (`AF-02`):**
  - Phát hiện lỗi nghiêm trọng trong `splitSingleTaskPayload` (`transmute_handler.go:790-795`): Khi chia đôi 1 NATS message thành `leftReq` và `rightReq`, nếu 1 nửa thành công và 1 nửa thất bại (`leftErr == nil && rightErr != nil`), hàm gọi `task.complete(nil)` làm `TransmuteConsumerPool` gọi `msg.Ack()` ngay lập tức, mất cơ hội redeliver của nửa bị lỗi và không trả NATS reply cho caller.
  - Ngoài ra, `TransmuterModule.RunEvents` (`transmuter.go:700-810`) vốn đã lặp qua từng event (`for _, ev := range events`) và thực thi SQL kèm retry `maxRetries = 3` độc lập cho từng event. Nếu `RunEvents` tự cô lập event hỏng (ghi log `transmute_poison_pill` cho event hỏng sau khi hết retry và chỉ trả lỗi khi gặp transient DB error hoặc toàn bộ events hỏng), ta không cần đẩy ngược ra `binarySearchSplit` làm chạy lại `bulkUpsertMaster` $\log_2(N)$ lần cho các event đã thành công!
- **Data Continuity / Watermark / Cursor & Scan Efficiency (`AF-03`):**
  - Phát hiện lỗi ghi đè watermark tại `transmuter.go:553-557`: Điều kiện `if len(onlySourceIDs) == 0` chạy cho cả Incremental Sync (`runType != "full"`), ghi đè `item.LastCursorJSON = []byte("{}")` và làm mất `last_gpay_id` vừa lưu ở dòng `530-533`. Chỉ được phép reset `LastCursorJSON = "{}"` khi `runType == "full"` và không có lỗi (`lastBatchErr == nil`).
  - Phát hiện lỗi hiệu năng tại `syncUpdatedShadowRowsBeforeCursor` (`transmuter.go:1504-1513`): Dùng `cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour)` làm quét lại toàn bộ `54,754` dòng mỗi nhịp Incremental Sync. Cần lấy mốc `cutoff` từ `runtimeState.LastSuccessAt` (trừ lùi biên an toàn 10 phút, hoặc fallback `24h` nếu chưa có `LastSuccessAt`) để chỉ quét đúng vài chục dòng vừa được cập nhật kể từ lần sync thành công trước đó.
- **Event Ordering & Deduplication (`AF-05`):**
  - Trong `dedupByConflictKey` (`batch_buffer_utils.go:96-135`), cần so sánh `r.SourceTsMs` để đảm bảo khi gộp `MappedData` và chọn bản ghi đại diện cho cùng 1 `PrimaryKeyValue`, sự kiện có `SourceTsMs` lớn hơn (hoặc đứng sau khi `SourceTsMs` bằng nhau) luôn thắng, không bị sự kiện cũ hơn ghi đè nếu thứ tự trong slice bị xáo trộn.
  - Trong `resolveSourceIDsForSegmentB` (`recon_execute_heal_handler.go:733-751`), cần chỉ truy vấn các `missingGpayIDs` chưa tìm thấy trên Shadow và gọi `uniqueStrings(out)` trước khi trả về.

---

## 4. Anti-Speculation & Honesty Check (Kiểm tra Suy diễn & Báo cáo láo)

Qua rà soát phản biện từng chữ trong báo cáo lúc `15:28` đối chiếu với `git show 30b1b00` và DB runtime thật, tôi xin **nhận lỗi thẳng thắn và minh bạch 100%** về 4 sai lệch (`AF-01`, `AF-04`) đã xảy ra trong báo cáo trước:
1. **Viết tên file và số dòng từ trí nhớ tóm tắt (`Context Summary`) sau khi phiên bị nén ngữ cảnh mà không chạy lại `git show --stat 30b1b00`:**
   - Báo cáo cũ ghi tên file là `transmute_handler_test.go`, `batch_buffer_test.go`, `transmuter_f1_f5_integration_test.go` và thiếu `transmuter_orphan_test.go`. Tên file thực tế trong commit `30b1b00` là `transmute_debounce_group_test.go`, `adaptive_batcher_test.go`, `transmuter_gpayid_test.go`, `transmuter_orphan_test.go`.
2. **Viết sai chi tiết cài đặt của `resolvePKColumnAndType` trong báo cáo cũ:**
   - Báo cáo cũ (mục 1, dòng 16) viết rằng `resolvePKColumnAndType` kiểm tra sự tồn tại của cột trong `information_schema.columns` (`L314-322`). Thực tế kiểm tra lại `recon_execute_heal_handler.go:595-621` cho thấy hàm này chỉ đọc `ref.TransformSpec`, fallback sang `resolveTargetTableConfig` rồi fallback `"_id"`, hoàn toàn không truy vấn `information_schema.columns`. Đây là lỗi suy diễn/nhớ nhầm từ nháp thiết kế sang báo cáo kết quả.
3. **Ghi nhầm số cổng Docker trong báo cáo cũ:**
   - Báo cáo cũ ghi `127.0.0.1:5435` là `gpay-postgres-cdc` và `127.0.0.1:5433` là `gpay-postgres-shadow`. Thực tế `5433` là `gpay-postgres-cdc` và `5436` là `gpay-postgres-shadow`.
4. **Bỏ sót `Binding 51, 52` và `Task 6.2` (`recon_heal.go`):**
   - Số liệu đối soát `54,722/54,722` trên `Binding 46, 47` và `540/540` trên `Binding 48` là **sự thật 100% trên PostgreSQL**, nhưng báo cáo cũ chưa kiểm tra `Binding 51` và `Binding 52` (đang còn `541` dòng vs `540` active shadow), đồng thời chưa sửa `recon_heal.go` như đã ghi trong `Task 6.2`.

---

## 5. Self-Improvement Loop (Vòng lặp Phản tỉnh & Khắc phục — Lessons Audit & Closed-Loop Re-Plan `S5 -> S3`)

1. **Cập nhật Bài học mới vào `memory/global/lessons.md` (Rule #5 & Rule #6):**
   - Đã ghi bài học mới ngày `[2026-10-20]` (*"Viết báo cáo Audit từ tóm tắt nén ngữ cảnh gây sai lệch tên file/hàm, nuốt lỗi 1 nửa trong splitSingleTaskPayload làm JetStream Ack sớm, quét tràn 7 ngày trong syncUpdatedShadowRowsBeforeCursor, và bỏ sót Active Bindings 51, 52 cùng tiến trình Daemon cũ"*) vào đầu nhóm trong `/Users/trainguyen/Documents/work/agent/memory/global/lessons.md` (`wc -l` tăng từ `1972` $\rightarrow$ `1979` dòng) và chạy `build_lessons_index.py` thành công (`254/254/254` entries).
2. **Kích hoạt Vòng lặp Khép kín `S5 → S3 Re-Plan` (Rule #9.7d & Lesson `L020`):**
   - Tuân thủ tuyệt đối kỷ luật **CẤM tự ý sửa source code `.go` trong chặng Audit S5 khi chưa có `APPROVE` mới của User**.
   - Đã cập nhật phương án khắc phục triệt để 5 vấn đề `AF-01..AF-05` vào `09_tasks_solution_shadow_master_update_drop.md` và thêm `Task 9..Task 13` (trạng thái `[ ]`) vào `08_tasks_shadow_master_update_drop.md`, chạy lại `DG1`/`DG2` để trình User `APPROVE`.

---

## 6. Decision Gate `/laya_router` DG3 Final Verdict

- **Mode A — Calibrated Claim Verification (`laya_gate.py run --stage dg3`):**
  - **Summary file:** `06_laya_summary_dg3_20261020T160810.json` | **Trusted Rounds:** `["r3"]` | **Gate:** `PASS`
  - **Chi tiết 5 claims phản biện (`AF-01..AF-05`):**
    1. `AF-01-HARDCODED-BINDINGS-AND-PK-RESOLVER`: **`ACCEPT`** (`r1: CORRECT (0.5140)`, `r2: CORRECT (0.1991)`, `r3: YES (0.7129)`).
    2. `AF-02-SPLIT-SINGLE-TASK-PARTIAL-ERROR-SWALLOW`: **`FLAG -> RESOLVED`** (`r1: CORRECT (0.7675)`, `r2: CORRECT (0.7089)`, `r3: NO (0.0342)`, resolved kèm `REF: transmute_handler.go:790-795`).
    3. `AF-03-SYNC-BEFORE-CURSOR-7DAYS-AND-CURSOR-RESET`: **`ACCEPT`** (`r1: CORRECT (0.4427)`, `r2: CORRECT (0.3910)`, `r3: YES (0.6767)`).
    4. `AF-04-SEG-A-HARDCODED-CONN-KEY-AND-UNCHUNKED-PUBLISH`: **`ACCEPT`** (`r1: CORRECT (0.5599)`, `r2: CORRECT (0.5172)`, `r3: YES (0.5910)`).
    5. `AF-05-RESOLVE-SOURCE-IDS-DUPLICATES-AND-DEDUP-ORDER`: **`ACCEPT`** (`r1: CORRECT (0.5485)`, `r2: CORRECT (0.2870)`, `r3: YES (0.6613)`).
- **Mode B — Agentic Debate Diff Review (`laya_gate.py review` trên toàn bộ diff commit `30b1b00`):**
  - **Summary file:** `06_laya_summary_dg3review_20261020T155849.json` | **Gate:** `BLOCK` (`REJECT_LOGIC` trên `internal/handler/recon/recon_execute_heal_handler.go`, `control_passed: true`)
  - **Nhận xét của `/laya_router`:** Hội đồng `/laya_router` Mode B đã độc lập phát hiện vấn đề logic trong `internal/handler/recon/recon_execute_heal_handler.go` (trùng khớp với `AF-04` và `AF-05`), khẳng định quyết định chưa dừng ở commit `30b1b00` mà mở vòng lặp `S5 → S3 Re-Plan` để vá triệt để `AF-01..AF-05` là hoàn toàn chính xác.
- **Re-Plan `DG1` & `DG2` (cho Giải pháp & Code Demo khắc phục `AF-01..AF-05` trong `09_tasks_solution_shadow_master_update_drop.md`):**
  - `DG1`: `06_laya_summary_dg1_20261020T160744.json` (`trusted_rounds: ["r3"]`, `gate: "PASS"`).
  - `DG2`: `06_laya_summary_dg2_20261020T160746.json` (`trusted_rounds: ["r3"]`, `gate: "PASS"`).

---

## 7. Kết quả Thực thi Khắc phục `AF-01..AF-05` (Round 2 — Commit `08c05a8`) & Kiểm định DG3 Cuối Phiên

Sau khi nhận lệnh `APPROVE` của User (`2026-10-20T16:23:58+07:00`), toàn bộ 5 phát hiện phản biện `AF-01..AF-05` (`Task 9..Task 13`) đã được triển khai, kiểm thử thực tế trên PostgreSQL và kiểm định qua `/laya_router` **DG3**:
1. **`AF-01` (`Task 9` — `internal/service/master/transmuter_gpayid_test.go:403-574`):**
   - Đã nâng cấp `TestLiveBindings_46_47_48_TransmuteAndVerifyZeroDrift` tự động truy vấn toàn bộ `activeBindingIDs` từ `cdc_system.master_binding WHERE is_active = true ORDER BY id` (`[46, 47, 48, 51, 52]`) và lấy đúng `ShadowConnectionKey`/`MasterConnectionKey` cho từng binding (`default_master` `:5434` và `master_2` `:5437`).
   - Kết quả chạy thực tế trên PostgreSQL:
     - `Binding 46` (`default_master`, `payment_bills`): `activeShadow = 54,755`, `masterCount = 54,755`, `droppedDelete = 0`, `droppedUpdate = 0`, `gpayMismatch = 0`.
     - `Binding 47` (`default_master`, `payment_bills_1`): `activeShadow = 54,755`, `masterCount = 54,755`, `droppedDelete = 0`, `droppedUpdate = 0`, `gpayMismatch = 0`.
     - `Binding 48` (`default_master`, `export_jobs`): `activeShadow = 540`, `masterCount = 540`, `droppedDelete = 0`, `droppedUpdate = 0`, `gpayMismatch = 0`.
     - `Binding 51` (`master_2` `:5437`, `export_jobs_2`): Đã heal `523` dòng (`1` dòng xóa mềm `_deleted = true` và `521` dòng lệch `_gpay_id` SHA-256), đạt `activeShadow = 540`, `masterCount = 540`, `droppedDelete = 0`, `droppedUpdate = 0`, `gpayMismatch = 0`.
     - `Binding 52` (`default_master` `:5434`, `export_jobs_1`): Đã heal `4` dòng (`1` dòng xóa mềm `_deleted = true` và `2` dòng lệch `_gpay_id` SHA-256), đạt `activeShadow = 540`, `masterCount = 540`, `droppedDelete = 0`, `droppedUpdate = 0`, `gpayMismatch = 0`.
2. **`AF-02` (`Task 10` — `internal/handler/master/transmute_handler.go:782-805`):**
   - Đã sửa `splitSingleTaskPayload` dùng `if leftErr != nil || rightErr != nil` kèm `h.replyErr(...)` và `task.complete(firstErr)` để không bao giờ `Ack()` sớm khi 1 nửa payload bị lỗi; bổ sung unit test `TestSplitSingleTaskPayload_ReturnsErrorWhenSubTaskFails` (`PASS`).
3. **`AF-03` (`Task 11` — `internal/service/master/transmuter.go:357-365, 558-563, 1510-1518`):**
   - Đã lưu checkpoint `last_gpay_id` khi `inferLastGpayIDFromMaster` khôi phục cursor, chỉ reset `LastCursorJSON = "{}"` khi `runType == "full" && len(onlySourceIDs) == 0 && lastBatchErr == nil`, và giới hạn `cutoff` trong `syncUpdatedShadowRowsBeforeCursor` theo `LastSuccessAt - 10m` (fallback `24h`). Kiểm chứng thực tế trên cả 5 bindings: `last_cursor_json` luôn được bảo toàn sau mỗi nhịp Incremental Sync.
4. **`AF-04` (`Task 12` — `internal/handler/recon/recon_execute_heal_handler.go:339-392` & `recon_check_heal_handler.go:308-311`):**
   - Đã khử trùng lặp `uniqueStrings(sourceIDs)`, chia chunk theo `healChunkSize = 200`, truyền `"shadow_connection_key": "default_shadow"`, và bổ sung gọi `publishTransmuteShadowForSegA` trong `executeLegacyHeal`.
5. **`AF-05` (`Task 13` — `internal/handler/shadow/batch_buffer_utils.go:90-153` & `recon_execute_heal_handler.go:732-780`):**
   - Đã nâng cấp `dedupByConflictKey` so sánh `r.SourceTsMs` (unit test `TestDedupByConflictKey_OrdersBySourceTsMsWhenOutOfOrder` `PASS`), đồng thời chỉ truy vấn `missingGpayIDs` chưa tìm thấy trên Shadow và khử trùng lặp `uniqueStrings(out)` trong `resolveSourceIDsForSegmentB`.
6. **Decision Gate `/laya_router` DG3 Final Verdict (Round 2 — sau commit `08c05a8`):**
   - **Summary file:** `06_laya_summary_dg3_20261020T164916.json` | **Trusted Rounds:** `["r2", "r3"]` | **Gate:** **`PASS`** (`DG3-R2-AF01`: `ACCEPT`, `DG3-R2-AF02`: `ACCEPT`, `DG3-R2-AF03`: `ACCEPT`, `DG3-R2-AF04`: `FLAG -> RESOLVED`, `DG3-R2-AF05`: `FLAG -> RESOLVED`).
