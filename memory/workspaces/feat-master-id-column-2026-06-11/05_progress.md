
## [2026-06-11] Bỏ approach hardcode (user reject) → chuyển Cách 2
- User: hardcode `_id` vào master DDL chỉ đúng cho Mongo, vỡ với MySQL/PG → fix bẩn. Chọn Cách 2: scan hiện `_id` như field thường, map bình thường, chấp nhận dup ở shadow.
- Revert sạch hack: master_ddl_generator (bỏ `_id` col/comment/seen/ALTER/idx) + transmuter (bỏ record["_id"]). Giữ realCols (user co-edit).
- Đang implement Cách 2: un-skip `_id` ở mọi system-field-set + `_`-prefix check (giữ skip _raw_data/_synced_at/... thật).

## [2026-06-11] Cách 2 — un-skip `_id` ở scan (build PASS)
- mongo_introspection.go: bỏ `if k=="_id"{continue}` → scan source HIỆN `_id`.
- command_handler.go:1853 skipFields (rescan): bỏ `_id` → rescan cũng hiện `_id` + tạo mapping rule.
- master_ddl_generator seen: đã un-reserve `_id` (revert) → mapping rule `_id`→master col chảy được.
- go build ./... PASS. processDiscoveryRows (561) vốn không skip `_id` → mapping rule `_id` tạo được.
- CÒN LẠI (chưa làm, để consistent/đầy đủ): un-skip `_id` ở command_handler:2156 (array), :2397 (debezium), schema_inspector:312 (drift), `_`-prefix check command_handler:1013 + child_explode:209. VÀ verify sinkworker schema_manager tạo cột `_id` shadow từ mapping rule + ghi value (sink upsert là schema/mapping-driven nên kỳ vọng OK, cần test thật).

## [2026-06-11] Cách 2 HOÀN TẤT (code) — un-skip _id mọi tầng
- Blocker shadow tìm ra: user đã map `_id` (mapping_rule_v2 8 rule `_id→_id`) nhưng shadow chưa có cột vì `schema_inspector.findNewFields` skip `_id` → đã bỏ.
- Un-skip `_id` đủ 6 điểm: mongo_introspection (scan), command_handler:1853/2157/2398 (rescan/array/debezium), schema_inspector:312 (findNewFields-shadow col), + HasPrefix exception command_handler:1013 & child_explode:209 (`&& != "_id"`). master_ddl_generator seen đã un-reserve (revert). go build PASS.
- Giữ legit: event_handler pkField=="_id" (anchor → _source_id) không đụng → shadow vẫn có _source_id (anchor) + _id (business dup, user chấp nhận).
- CÒN: user RE-SYNC (snapshot/transmute lại) để shadow tạo cột `_id` + ghi value, rồi master nhận qua mapping. Cần test thật xác nhận data.

## [2026-06-11] LESSON nghiêm trọng: verify SAI plane → report sai + sửa code trên chẩn đoán sai
- Tôi query control-plane cdc_dw:5433 (chỉ cdc_system/public) để kiểm shadow/master → kết luận "shadow ko có _id". SAI: shadow plane = cdc_shadow@5436, master/dest = goopay_dest@5434 (DB RIÊNG).
- Sự thật: shadow 6/15 bảng có _id, master nhiều bảng có _id — `_id` chảy qua mapping rule (_id→_id) khi map+sync. KHÔNG bị chặn tuyệt đối.
- Hệ quả: các un-skip (findNewFields...) dựa chẩn đoán sai. Chỉ scan-fix (mongo_introspection) là đã validate (user confirm "đã có _id").
- Global Pattern: [verify hệ multi-plane (control/shadow/dest là DB riêng) trên SAI plane] → report sai + fix sai. Đúng: xác định đúng DB plane (port/container) TRƯỚC khi query/kết luận; với CDC: control=metadata, shadow=raw mirror, dest=master — query đúng cái cần. Áp dụng: mọi hệ tách plane (CQRS read/write store, primary/replica, multi-tenant DB).
- Bài học vận hành (user dạy): KHÔNG tin report nào (của user, của chính mình, của query cũ) — verify ground-truth trên đúng nguồn trước khi claim/sửa.

## [2026-06-11][Agent:claude-opus-4-8] FIX hardcode `_id` sót trong realCols (master_ddl_generator.go:133)
- User bắt: `realCols := {... "_id": true ...}` hardcode `_id` dù master default CHỈ tạo 4 cột (_gpay_id/_source_ts/_deleted/_updated_at), KHÔNG có _id.
- Root cause: rác sót khi tôi revert hack hardcode `_id` trước đó — đã dọn `cols`+`seen` nhưng SÓT `realCols`.
- Tác hại: guard spec.pk (dòng 214 `if !realCols[pkCol]`) tưởng `_id` tồn tại → CREATE UNIQUE INDEX trên cột _id không có → lỗi 42703 (đúng lỗi guard sinh ra để chặn, từng xảy ra ở bảng b3).
- Fix: bỏ `"_id": true` → realCols = 4 cột thật; `_id` chỉ vào realCols qua rules loop (dòng 140) KHI user MAP → opt-in, generic (không giả định Mongo).
- Verify: gofmt OK; `go build ./...` PASS. Grep toàn service: KHÔNG còn hardcode-decl `_id` (`"_id":true`/`"_id" TEXT/BIGINT`/`record["_id"]`/`cols..."_id"`). Refs `_id` còn lại đều LEGIT: event_handler 226/307/359 (anchor detect source-PK `_id`, dòng 359 guard `sourceType=="mongodb"`), + 2 exception Cách 2 đang chờ review (command_handler:1013, child_explode:209).
- Global Pattern: [revert hack X bỏ sót 1 nơi Y dùng cùng giá trị Z] → Y vẫn mang giá trị cứng → bug ngầm. Đúng: khi revert, grep TẤT CẢ nơi tham chiếu Z (decl/seen/guard/write) rồi dọn đồng bộ; verify bằng grep negative.

## [2026-06-11][Agent:claude-opus-4-8] REVERT 2 path AUTO — trả về đúng Cách 2 (opt-in), bỏ "_id mọi nơi/tự can thiệp"
- User chốt: `_id` KHÔNG default mọi nơi; core KHÔNG auto/đụng "db cũ"; không Mongo-hardcode (MySQL/PG PK=id). Bỏ kiểu option 1/2/3 → chốt 1 hướng minimal.
- Phân loại (đã verify subject/trigger): HandleScanRawData/HandleScanArrayFields = "request-reply" (USER bấm) → scan-show `_id` = đúng Cách 2 → GIỮ. HandlePeriodicScan = "pub/sub triggered by scheduler" + AUTO-CREATE rule (new_rules_created) → un-skip `_id` = tự tạo rule `_id` mọi bảng = AUTO → REVERT. findNewFields = auto-detect lúc ingest → AUTO → REVERT.
- Sửa: (1) schema_inspector.go findNewFields: khôi phục `"_id":true` + comment gốc → file về NGUYÊN GỐC (0 diff). (2) command_handler.go HandlePeriodicScan(~2399): khôi phục `"_id":true`.
- GIỮ (Cách 2 opt-in, user-trigger): mongo_introspection (scan-show, user confirm "đã có _id"), command_handler HandleDiscover/ScanRawData/ScanArrayFields (scan-show), child_explode (map-normally), master_ddl_generator realCols (đã bỏ hardcode _id hôm nay).
- Phát hiện chính xác: `seen` map TRONG master_ddl VỐN không có `_id` (chỉ thêm comment) → master nhận `_id` qua mapping là hành vi GỐC, KHÔNG phải tôi enable. Block spec.pk guard+parsePKFromSpec là task TRƯỚC (fix 42703 b3).
- Verify: gofmt clean; `go build ./...` PASS; grep xác nhận auto off + scan-show on + realCols hết hardcode. CHƯA verify runtime (cần rebuild worker + re-sync) — KHÔNG claim done phần data.
- Global Pattern: [phân loại path P theo trigger: user-request vs scheduler/ingest] trước khi un-skip field F → chỉ mở F ở path user-trigger (opt-in), đóng ở path AUTO (scheduler/ingest auto-create). Đúng: grep "Subject:"/trigger comment để biết P là on-demand hay auto TRƯỚC khi đổi behavior.

## [2026-06-11][Agent:claude-opus-4-8] Xác nhận giải pháp spec.pk index + AUDIT multi-pk
- spec.pk index: realCols fix (bỏ hardcode `_id`) ĐÃ là giải pháp "approve field xong mới chạy tạo index". Verified luồng Apply: field chưa approved → loadRules (status='approved') không nạp → realCols thiếu → guard skip index (không 42703); approved → realCols có → Apply (chạy khi approve/tick, comment L253) ALTER ADD COLUMN (L264) TRƯỚC rồi CREATE UNIQUE INDEX (L269) SAU, cùng tx → OK. KHÔNG cần code mới. Cần test runtime.
- AUDIT multi-pk (>1 pk): CHƯA xử lý. parsePKFromSpec parse `pk` STRING đơn → `{"pk":["a","b"]}` array → unmarshal fail → "" → silent no-index; `{"pk":"a,b"}` → regex ident fail → skip+warn. Guard chỉ tạo ux 1 cột, không composite. Data: 13/13 binding pk_type=string (đơn) → chưa ai dùng composite. "composite" ở chunking/partition KHÔNG liên quan spec.pk.
- Đề xuất: defer multi-pk (0 data dùng, Simplicity). Khi cần: parsePKFromSpec trả []string (string|array), tạo 1 UNIQUE INDEX nhiều cột + existence-check từng cột (live-check).
- Global Pattern: [feature F đọc config field C dạng scalar] nhưng C có thể là scalar|list → audit unmarshal path: list value → unmarshal-into-scalar FAIL im lặng. Đúng: parse C kiểu union (scalar|list)→normalize list, hoặc ít nhất detect+warn khi gặp list chưa support (tránh silent no-op).

## [2026-06-11][Agent:claude-opus-4-8] AUDIT plain INDEX (non-unique, tăng tốc query)
- Đang có (auto, không user-declared): (1) system cols _source_ts/_updated_at luôn CREATE INDEX (L203-204); (2) financial cols khớp financialIndexRe = ^(amount|fee|balance|total|price|refund|subtotal|discount|tax|cost)…|_amount$|_fee$|_balance$|_price$ → auto CREATE INDEX (L59,163,206-207); (3) spec.pk → CREATE UNIQUE INDEX (L209-222).
- CHƯA có: user tự khai báo plain index cột tùy ý. Verified: MappingRuleV2 struct KHÔNG có field index (chỉ IsSensitiveField/MaskStrategy); transform_spec không parse "indexes"; grep is_index/indexed/create_index/"indexes" toàn service = 0 (chỉ comment lạc).
- Đề xuất (chưa code, chờ user): mở rộng transform_spec {"indexes":["c1","c2"]} → parseIndexesFromSpec → CREATE INDEX IF NOT EXISTS từng cột (non-unique) + reuse live-column-check + cùng tx Apply như spec.pk. Đối xứng {"pk"}, data-driven, no migration. Alt: cột is_indexed ở mapping_rule_master (cần migration+FE).
- Global Pattern: [hệ có auto-index theo regex tên cột + spec.pk unique] nhưng thiếu user-declared plain index → cột business tùy ý không tăng tốc được. Đúng: cấp kênh khai báo data-driven (spec.indexes[] hoặc flag is_indexed) tái dùng existence-check + apply tx sẵn có, tránh hardcode tên cột.

## [2026-06-11][Agent:claude-opus-4-8] FEAT transform_spec.indexes — plain index user khai báo (tăng tốc query)
- User: "1 thôi, làm đi" → làm #1 (transform_spec.indexes), DEFER multi-pk.
- master_ddl_generator.go: (1) thêm parseIndexesFromSpec(spec) []string — mirror parsePKFromSpec, lenient unmarshal, trim+skip empty, data-driven KHÔNG hardcode tên cột. (2) hoist pkCol; thêm loop CREATE INDEX (non-unique) cho mỗi cột spec.indexes, dùng lại guard realCols (cột default + rule approved) → cột chưa tồn tại bỏ qua (tránh 42703), approve xong tạo ở Apply sau. Dedup: bỏ cột đã là pk/financial/system.
- An toàn transmute: parseFlattenSpec + parsePKFromSpec đều lenient unmarshal → thêm key "indexes" KHÔNG vỡ validate; copy_1_to_1 không cần spec (đã verify đọc code).
- Verify: gofmt OK; go build ./... PASS; go test -run TestParseIndexesFromSpec -v = 9 subtests + NoHardcode = PASS.
- Files: master_ddl_generator.go (+103/-1); NEW master_ddl_indexes_test.go (55 dòng). CHƯA verify runtime (cần worker rebuild + set transform_spec.indexes + approve cột + apply → index hiện trên master plane).

## [2026-06-11][Agent:claude-opus-4-8] FEAT composite index cho transform_spec.indexes (nhóm field)
- User: "làm đi" → mở rộng indexes hỗ trợ COMPOSITE (nhiều cột chung 1 index), không chỉ đơn cột.
- parseIndexesFromSpec: đổi trả [][]string. Phần tử = string (đơn cột → ["col"]) | []string (composite). Dùng []json.RawMessage + thử unmarshal string rồi []string. trim+skip empty, bỏ nhóm rỗng. Data-driven, KHÔNG hardcode.
- Generate loop: mỗi nhóm existence-check TẤT CẢ cột (thiếu 1 → bỏ CẢ nhóm, tránh 42703); CREATE INDEX ix_<table>_<c1>_<c2> ON (c1,c2). Dedup: index đơn trùng pk/financial/system → bỏ; seenIdxName chống tên trùng.
- Format: {"indexes":["status", ["tenant_id","created_at"]]} → ix đơn (status) + 1 composite (tenant_id,created_at). Backward-compat (chưa data nào dùng).
- Verify: gofmt OK; go build ./... PASS; go test -run TestParseIndexesFromSpec -v = 12 subtests (single/multi/composite/mixed/coexist-pk/trim/empty-group/wrong-type/malformed) + NoHardcode = PASS.
- Files: master_ddl_generator.go (+152/-1 tổng cả task); test (61 dòng). CHƯA verify runtime (worker rebuild + set spec composite + approve cột + apply → ix composite hiện).
- Global Pattern: [config field nhận scalar|list-of-scalar|list-of-list] → parse bằng []json.RawMessage + thử unmarshal từng kiểu (string→[1], []string→composite); existence-check toàn nhóm trước khi tạo (all-or-nothing) tránh DDL nửa vời.

## [2026-06-11][Agent:claude-opus-4-8] AUDIT trigger re-apply: approve & Edit Spec (DDL/index)
- EnsureMaster = Apply KHÔNG điều kiện (master_ddl_generator.go:391-394), gọi trước MỖI transmute (transmuter.go:155) → re-đọc transform_spec live (loadBinding:406) + rule approved live → tạo cột+index idempotent. = catch-all.
- Approve MASTER (cdc-cms-service approve_master.go:143-154): publish cdc.cmd.master-create → worker Apply NGAY. ✓
- Approve FIELD (approval_service.go:122): PublishReload → áp ở transmute kế (EnsureMaster) + invalidate cache.
- Edit Spec (update_master_spec.go): CHỈ UPDATE transform_spec vào DB, handler db-only (NewUpdateMasterSpecHandler(db), KHÔNG nats) → KHÔNG publish re-apply. Spec mới chỉ áp ở transmute kế (event / cron scheduler ~60s). Bảng idle+không schedule → KHÔNG áp tới khi có event. = GAP (không đối xứng approve-master).
- Đề xuất (chờ user): update_master_spec inject nats + publish cdc.cmd.master-create sau UPDATE (mirror approve_master) → Edit Spec áp ngay.
- Global Pattern: [thao tác X đổi config materialize-by-worker] cần publish re-apply để worker đồng bộ NGAY; nếu chỉ UPDATE DB thì phụ thuộc job kế (event/cron) → idle → drift. Đúng: mọi mutation config (approve, edit-spec) đều publish 1 re-apply nhất quán; worker re-apply idempotent.

## [2026-06-11][Agent:claude-opus-4-8] ĐÍNH CHÍNH: "fix Edit-Spec publish master-create" là SAI (bỏ sót schema_status gate)
- Đọc code thật: Edit Spec CHỈ cho khi schema_status<>'approved' (update_master_spec.go:89 + comment 18-19). Apply CHỈ chạy khi status='approved' (master_ddl_generator.go:82). → publish master-create lúc Edit Spec (chưa approved) → Apply từ chối "must be approved" → vô dụng. Phân tích "gap" trước của tôi THIẾU gate này.
- Flow thật ĐÃ đúng: Create(pending) → Edit Spec (thêm indexes) → Approve (publish master-create) → Apply đọc spec đã sửa (loadBinding:406) → tạo index. KHÔNG cần code change.
- Master live muốn thêm index: phải Reject→Edit→Approve (spec lock khi approved = thiết kế cố ý, tránh desync). Reject dừng transmute tạm (transmuter.go:142).
- Chờ user chốt: (a) giữ nguyên; (b) làm lệnh "re-apply DDL" riêng cho master approved (ALTER/INDEX additive idempotent, không đụng data) để thêm index không cần reject — nới 1 safety-gate nên cần user duyệt.
- Global Pattern: TRƯỚC khi implement fix "thêm trigger T cho thao tác X", verify state-gate của X và precondition của T — nếu X chỉ xảy ra ở state S1 còn T chỉ hiệu lực ở state S2 (S1≠S2) thì trigger vô dụng. Đừng implement mù theo "làm đi" khi đã thấy fix sai — báo lại.
