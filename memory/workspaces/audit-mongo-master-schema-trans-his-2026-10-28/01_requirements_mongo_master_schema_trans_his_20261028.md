---
tier: T2
project: data-hub
workspace: audit-mongo-master-schema-trans-his-2026-10-28
suffix: mongo_master_schema_trans_his_20261028
created_at: 2026-10-28T16:50:30+07:00
---

# 01. Requirements — Kiểm tra cấu trúc Document MongoDB Đích (Flattened vs Nested Schema) cho `trans_his`

## 1. Yêu cầu của User (Masked PII theo Rule #20)
User phản ánh ở MongoDB đích hiện tại khi tool chạy ra document bị làm phẳng (`snake_case` kèm metadata nội bộ `_gpay_id`, `_source_id`, `_source_ts`, `_created_at`, `_updated_at`, `_deleted`):
```javascript
{
  _id: ObjectId('6ae1c490e065f1404434b277'),
  id: '67ceaf0380d60cea6a262cc3',
  _created_at: ISODate('2026-10-28T09:46:56.766Z'),
  _deleted: false,
  _gpay_id: NumberLong('643659933107292726'),
  _source_id: '67ceaf0380d60cea6a262cc3',
  _source_ts: NumberLong('1793180634572'),
  _updated_at: ISODate('2026-10-28T09:46:56.766Z'),
  bill_id: null,
  channel_id: null,
  created_at: ISODate('2025-03-10T09:21:07.515Z'),
  fee: NumberDecimal('0'),
  is_delete: false,
  master_merchant_code: null,
  merchant_info_email: null,
  merchant_info_id: null,
  merchant_info_name: null,
  noted: '',
  order_id: null,
  original_trans_his_id: null,
  partner_code: null,
  receiver_credit: null,
  receiver_customer_id: '672c8bcfc271eae782c59515',
  receiver_email: '***@goopay.vn',
  receiver_full_name: '<MASKED_NAME>',
  receiver_merchant_code: null,
  receiver_merchant_id: null,
  receiver_phone: '<MASKED_PHONE>',
  request_id: null,
  sender_customer_id: '672b40157a76482529ad9608',
  sender_full_name: '<MASKED_NAME>',
  sender_phone: '<MASKED_PHONE>',
  service_code: null,
  status: 'SUCCESS',
  sys_trans: false,
  trans_id: '250310AWKLBD',
  trans_type: 'ADJUST',
  updated_at: ISODate('2025-03-10T09:21:07.635Z'),
  value: NumberDecimal('10000')
}
```

Trong khi cấu trúc mong muốn ở MongoDB đích là cấu trúc nested gốc (`camelCase`: `transId`, `sysTrans`, `sender: {...}`, `receiver: {...}`, `info: {...}`, `createdAt`, `updatedAt`, `__v`, `_id: ObjectId(...)`):
```javascript
{
  _id: ObjectId('6ad8355d439d4502af13c48f'),
  transId: '261021GJMVZG',
  sysTrans: false,
  originalTransHisId: null,
  sender: {
    fullName: '<MASKED_NAME>',
    email: '***@gmail.com',
    phone: '<MASKED_PHONE>',
    avatar: '',
    authId: null,
    customerId: '6721b91375d7c3ceb00c8159',
    walletId: null,
    merchantId: null,
    bankAccount: null,
    clientCode: null,
    bankAccountName: null,
    bankCode: null
  },
  receiver: {
    fullName: '<MASKED_NAME>',
    email: '***@goopay.vn',
    phone: '',
    avatar: '',
    authId: null,
    customerId: '6ac8694d414ceab42c710702',
    credit: '',
    creditName: '',
    bankCode: '',
    userId: '',
    merchantCode: 'LZGDHL',
    merchantId: '',
    masterMerchantId: '',
    clientCode: null,
    bankAccountName: null
  },
  value: NumberInt('30000'),
  transType: 'PAY',
  status: 'SUCCESS',
  error: null,
  noted: 'Capture hold',
  info: {
    merchant: {
      merchantCode: 'LZGDHL',
      name: '<MASKED_NAME>',
      id: '6ac8694d414ceab42c710702'
    },
    transactionParties: {
      payer: {
        _id: '6721b91375d7c3ceb00c8159',
        driverId: 'GP-8840001',
        userId: '6721b91375d7c3ceb00c8159'
      },
      payee: {
        _id: '6ac8694d414ceab42c710702',
        merchantCode: 'LZGDHL',
        name: '<MASKED_NAME>'
      }
    },
    orderInfo: 'Nap tien ky quy LZGDHL',
    transactionType: 'RADA.DEPOSIT_EWALLET',
    transAmount: NumberInt('30000'),
    fee: NumberInt('0'),
    billId: NumberInt('1733'),
    orderId: 'DH697864234972',
    trackingId: '261021034531IKUEJG',
    merchantTransId: 'DEP-01a15c8e58e8742296447b730975b824-1',
    holdingTransId: '261021QOIRAEETAQGDMZKT',
    captureTransId: '261021VLSQCR',
    captureId: 'CONFIRM:DEP-01a15c8e58e8742296447b730975b824-1',
    walletTransId: '127c942d-6349-400d-b3f7-587b0a442560'
  },
  serviceCode: null,
  channelId: 'WALLET',
  createdAt: ISODate('2026-10-21T03:45:33.491Z'),
  updatedAt: ISODate('2026-10-21T03:45:33.999Z'),
  __v: NumberInt('0')
}
```

## 2. Nguồn PostgreSQL Chuẩn (`public.transaction_history` + `public.transaction_history_payload`) & Yêu cầu Kiến trúc Tổng thể `Multi-Shadow Bindings -> 1 Master` (Turn 2 & Turn 3)
User phản biện đề xuất unflatten ở Turn 1 ("giải pháp của mày nó ko còn thuần bản chất cdc nữa rồi"), cung cấp DDL của 2 bảng nguồn PostgreSQL `1:1` (`transaction_history.id = transaction_history_payload.transaction_history_id`), và ở Turn 3 yêu cầu: **"trình bày giải pháp tổng thể lên cho vụ Multi-Shadow Bindings về cùng 1 master"** xuyên suốt toàn bộ kiến trúc Data Hub (`cdc_system`, `cdc-cms-service`, `centralized-data-service` cho cả PostgreSQL Master & MongoDB Master, và `cdc-cms-web`).

1. **`public.transaction_history_payload` (PK: `transaction_history_id`):**
   - `transaction_history_id text NOT NULL` (PK, khớp 1:1 với `transaction_history.id` và `_id` của MongoDB)
   - `info jsonb NULL` (chứa nguyên vẹn sub-document `info`: `merchant`, `transactionParties`, `orderInfo`, `transactionType`, `transAmount`, `fee`, `billId`, `orderId`, `trackingId`, `merchantTransId`, `holdingTransId`, `captureTransId`, `captureId`, `walletTransId`...)
   - `sender jsonb NULL` (chứa nguyên vẹn sub-document `sender`)
   - `receiver jsonb NULL` (chứa nguyên vẹn sub-document `receiver`)
   - `"error" jsonb NULL` (chứa nguyên vẹn trường `error`)
   - `source_field_types jsonb NOT NULL` (chứa bản đồ kiểu BSON & trạng thái hiện diện `@type_presence` của từng trường gốc trên MongoDB: `"objectId"`, `"int32"`, `"long"`, `"decimal128"`, `"string"`, `"boolean"`, `"date"`, `"object"`, `"null"`, và **vắng mặt key nếu trường không tồn tại trong document gốc** như `requestId`, `isDelete`).

2. **`public.transaction_history` (PK: `id`):**
   - **14 cột Scalar gốc** ánh xạ 1:1 sang 14 trường Scalar gốc của MongoDB:
     `id` ($\rightarrow$ `_id`), `trans_id` ($\rightarrow$ `transId`), `original_trans_his_id` ($\rightarrow$ `originalTransHisId`), `request_id` ($\rightarrow$ `requestId`), `trans_type` ($\rightarrow$ `transType`), `status` ($\rightarrow$ `status`), `sys_trans` ($\rightarrow$ `sysTrans`), `value` ($\rightarrow$ `value`), `noted` ($\rightarrow$ `noted`), `service_code` ($\rightarrow$ `serviceCode`), `channel_id` ($\rightarrow$ `channelId`), `is_delete` ($\rightarrow$ `isDelete`), `created_at` ($\rightarrow$ `createdAt`), `updated_at` ($\rightarrow$ `updatedAt`).
   - **18 cột Denormalized Index** (`bill_id`, `order_id`, `sender_customer_id`, `receiver_customer_id`, `sender_phone`, `receiver_phone`, `merchant_info_id`, `receiver_merchant_id`, `fee`, `partner_code`, `master_merchant_code`, `receiver_email`, `receiver_merchant_code`, `receiver_credit`, `sender_full_name`, `receiver_full_name`, `merchant_info_email`, `merchant_info_name`): chỉ sinh ra để đánh B-tree index (`idx_th_*`) trên PostgreSQL. Khi CDC ngược về MongoDB, 18 cột này **BẮT BUỘC loại bỏ (`rejected` / `is_active = false`)**, tuyệt đối không unflatten ngược!

3. **Yêu cầu Kiến trúc Chuẩn Design Pattern & Luồng `1 Master Binding = Multi-Shadow (multi_shadow)` (Turn 4):**
   - **Tuân thủ tuyệt đối Design Pattern gốc & Single-Responsibility (`L006`):** Đặt đúng layer theo Design Pattern gốc của từng repository (`domain`, `app/commands`, `infra/persistence`, `service/master/transmute`, `service/master`, `components`) và tách toàn bộ logic mới ra các file đơn trách nhiệm riêng biệt; tuyệt đối không nhồi nhét thêm hàm lớn vào các God-files (`transmuter.go` 1893 dòng, `master_repo_gorm.go` 1135 dòng, `MasterRegistry.tsx` 2177 dòng, `MasterMappingFieldsPage.tsx` 1415 dòng).
   - **Luồng Nghiệp vụ Thống nhất (Single Master Binding with Multi-Shadow Option):**
     1. **Tạo Master với Option Multi-Shadow (`transform_type = "multi_shadow"`):** Chọn Shadow chính (`transaction_history`) + chọn thêm Shadow phụ (`transaction_history_payload`) ngay trong 1 Master Binding duy nhất (`transform_spec.shadow_sources`).
     2. **Màn hình Mapping Thống nhất (`MasterMappingFieldsPage.tsx`):** Tự động clone & hiển thị **cả 2 bộ field của 2 bảng Shadow** (32 field của `transaction_history` + 6 field của `transaction_history_payload`, kèm badge `shadow_table` chỉ rõ nguồn) trong cùng 1 màn hình Mapping để người dùng tick chọn/approve field mong muốn và reject 18 cột index phụ trợ.
     3. **Thực thi Transmute với Option Multi-Shadow (`multi_shadow`):** Nhấn Transmute 1 lần duy nhất (hoặc tự động kích hoạt khi bất kỳ bảng Shadow nào trong nhóm nhận CDC event), hệ thống chạy chiến lược `multi_shadow` tự động co-fetch & hợp nhất `_raw_data` theo `_source_id` từ cả 2 bảng Shadow, áp `source_field_types` để khôi phục 100% kiểu BSON gốc và ghi xuống Master đích.

## 3. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **L048 (Pure CDC Multi-Table Relational-to-Document Sync):** Khi nguồn quan hệ tách một thực thể gốc thành bảng chính (`transaction_history`) và bảng phụ (`transaction_history_payload` kèm `source_field_types`) theo quan hệ `1:1`, luồng CDC sang Document Store phải hợp nhất trực tiếp từ cả hai bảng nguồn theo khóa định danh chung (`id` = `transaction_history_id` $\rightarrow$ `_id`), dùng trực tiếp các cột `jsonb` (`info`, `sender`, `receiver`, `error`) và `source_field_types` để khôi phục kiểu dữ liệu BSON (`ObjectId`, `NumberInt`, `ISODate`) cùng trạng thái `null` vs `missing`, và loại bỏ hoàn toàn 18 cột phụ trợ chỉ mục (`rejected`).
- **L033 (Immutable Raw Data, BSON Coercion & UTC Timezone):** Trường `_raw_data` ở Shadow là Ground Truth bất biến; khi chuyển đổi Extended JSON / BSON sang đích, phải bảo toàn kiểu BSON (`ObjectId`, `ISODate`/`time.Time`, `NumberInt`/`NumberLong`/`NumberDecimal`) và bật `dec.UseNumber()`.
- **L034 (MongoDB OCC Pipeline Update & Multi-Tenant Lineage):** Trên MongoDB Master, cơ chế BulkWrite Upsert OCC cần nhất quán với khóa chính (`_id`) và điều kiện OCC.
- **L030 (Full Multi-Tenant Identity Tuple & 1-N-M Lineage):** Cho phép nhiều `shadow_binding` (`transaction_history` và `transaction_history_payload`) cùng hội tụ vào một `master_table` (`devtest.transaction_history`) thông qua khóa `(master_connection_id, master_schema, master_table, shadow_binding_id)`.
- **L043 (Spec-Driven PK & Timestamp Field Resolution — Cấm Tự Động Nhận Diện & Phân Tách Rõ Master vs Shadow):** Tuyệt đối cấm cơ chế tự động dò đoán (`auto-detect candidates`) cho `pk` hay `timestamp_field`. Ở tầng Shadow đã có `primary_key_field` và `timestamp_field` riêng của từng bảng Shadow (`source_object_registry`), còn trên tầng Master thì `pk` và `timestamp_field` trong `Spec (JSON)` (`{"pk": "_id", "timestamp_field": "lastUpdatedAt"}`) là của riêng bảng **Master**.
- **Yêu cầu Audit & Tối ưu UI/Spec (Turn 11 & Turn 12 — `TASK-07`):**
  1. Làm rõ nguồn dữ liệu hiển thị của dropdown `Secondary Shadow Tables (Multi-Shadow 1:1) *` (`MultiShadowPicker.tsx:51-64`).
  2. Giữ đúng bản chất phân tầng: `PK` (`spec.pk`) và `Timestamp Field` (`spec.timestamp_field`) trên Modal Tạo/Sửa Master là của bảng **Master**; các bảng Shadow (`Primary` và `Secondary`) đã có `primary_key_field` và `timestamp_field` ở bên tầng Shadow của nó rồi, nên **gỡ bỏ các ô nhập `Timestamp Field` thừa dưới từng dòng `shadow_sources` trong `MultiShadowPicker.tsx`** để giao diện gọn sạch, chuẩn bản chất.
  3. Sắp xếp lại bố cục trực quan trên Modal: đưa **`Spec (JSON)` lên trên** và **`Secondary Shadow Tables (Multi-Shadow 1:1) *` xuống dưới cùng**.
  4. Audit toàn diện nơi sử dụng `Spec (JSON)` (`transform_spec`) và đánh giá ảnh hưởng của `"timestamp_field": "lastUpdatedAt"` trong `{"pk": "_id", "timestamp_field": "lastUpdatedAt"}` trên toàn hệ thống (`MasterDDLGenerator`, `TransmuterModule`, `Recon Segment B`, `Heal Segment B`), đồng thời xử lý triệt để điểm hở đọc `transform_spec.timestamp_field` trong `measureAndResolveWatermarksB` (`recon_tier_b.go:532`) và chuẩn hóa `snake_case <-> camelCase` trên `recon_dest_mongo.go:101,189,409`.
- **Yêu cầu Xử lý Bảng Secondary Shadow không có `updatedAt` (`transaction_history_payload`) & Hợp nhất `multi_shadow` với `join` (Turn 14 — `TASK-08`):**
  1. **Bảng phụ `public.transaction_history_payload` không có cột `updatedAt` / `updated_at` / `lastUpdatedAt`:**
     - Trên thực tế DDL PostgreSQL, bảng `transaction_history_payload` chỉ có 6 cột (`transaction_history_id`, `info`, `sender`, `receiver`, `"error"`, `source_field_types`) và **hoàn toàn không có cột thời gian nghiệp vụ**.
     - Cần mở khóa toàn bộ luồng từ đầu đến cuối cho bảng nguồn/bảng phụ không có `timestamp_field`:
       - **CMS Source Object / Table Registry (`cdc-cms-service` & `cdc-cms-web`):** Cho phép để trống / xóa trắng `timestamp_field` (`""` $\rightarrow$ `NULL` trong DB) khi đăng ký hoặc chỉnh sửa Source Object (`update_source_object_v2.go:61,85-99,131-133`, `update_registry.go:46,88-90`, `registry_handler_update.go:48`, `api/utils.go:43`, `table_registry.go:38`, `TableRegistry.tsx:969-975,1648,1734,1772`).
       - **CMS Master Spec & Transmute 1:1 (`multi_shadow_helper.go:166-194` & `transmuter_multi_shadow.go:237-256`):** Bảng Secondary Shadow (`role = "secondary"`) không cần `timestamp_field` riêng trong `shadow_sources`; ở tầng Transmute, mọi bảng Shadow đều đã có cột hệ thống `_source_ts BIGINT` ghi trực tiếp mốc commit WAL `payload.source.ts_ms` từ Debezium (`kafka_consumer.go:621`, `event_handler.go:463`), nên `fetchMergedShadowStateMap` dùng trực tiếp `sr.SourceTs` (`_source_ts`) cùng hệ quy chiếu WAL UTC làm mốc `st.MaxSourceTs` / `st.MaxDeletedTs` (chỉ fallback sang `extractExplicitTimestampMs` khi `rowTs <= 0`).
       - **Recon Segment A (`recon_tier_a.go:201-260,681,1097`, `recon_hash.go:403-517`, `recon_dest_hash.go:191-273`, `recon_smoke_guard.go:61-69,256-270`):** Khi bảng PostgreSQL nguồn không có cột timestamp (`tsField(entry) == ""` hoặc không tồn tại cột thời gian trên Source), không được trả về mặc định `"updated_at"` gây lỗi `SQLSTATE 42703: column "updated_at" does not exist` và đánh dấu nhầm `sync_status = "source_error"`; thay vào đó trả về `srcTS = ""`, `dstTS = "_id_only"` để `BucketHash` / `RunDeepCheck` đối soát vân tay 256-bucket theo khóa chính (`hashIDPlusTsMs(idStr, 0)`) và `RunHashWindowCheck` tự động chuyển về `RunDeepCheck`.
  2. **Cách hiển thị `join` / `multi_shadow` và Mở rộng Cấu hình Join Đa Bảng (`join như nào, join bao nhiêu bảng, field nào join field nào` — Turn 15):**
     - **Không hợp nhất/đổi tên ở Backend:** Bỏ hẳn `'join'` khỏi mảng `TRANSFORM_TYPES` trên UI (`MasterRegistry.tsx:253`), giữ nguyên giá trị `transform_type = 'multi_shadow'` dưới DB/Backend, và chỉ **hiển thị label của `'multi_shadow'` là `'join'`** trên toàn bộ UI (`MasterRegistry.tsx`, `CreateMasterModal.tsx`).
     - **Mở rộng cơ chế Join tường minh (`1 Primary + N Secondary`, khai báo rõ `field nào join field nào`):**
       - **Số lượng bảng (`join bao nhiêu bảng`):** Cho phép chọn **$N$ bảng Secondary Shadow** trong `spec.shadow_sources` để Left-Join với **1 bảng Primary Shadow** (`1 Primary + N Secondary`).
       - **Cặp trường Join (`field nào join field nào`):** Mỗi bảng Secondary trong `spec.shadow_sources` có thêm 2 trường `primary_join_field` (field bên bảng Primary Shadow) và `secondary_join_field` (field bên bảng Secondary Shadow). Trên UI `MultiShadowPicker.tsx`, khi chọn bảng Secondary sẽ hiển thị rõ dòng cấu hình `[Primary.<primary_join_field>] = [Secondary.<secondary_join_field>]` (mặc định tự điền `primary_key_field` của Primary và `primary_key_field` của Secondary, cho phép chỉnh sửa sang bất kỳ field nào).
       - **Cơ chế thực thi (`join như nào` tại `transmuter_multi_shadow.go`):** Nếu `primary_join_field` / `secondary_join_field` trùng với PK của bảng Shadow thì truy vấn qua cột `_source_id` (có B-Tree index); nếu khác PK thì trích xuất khóa join từ `_raw_data->>primary_join_field` và truy vấn bảng Secondary qua `(_raw_data ->> secondary_join_field) IN (?)` (kèm tra ngược về Primary khi event CDC phát sinh từ bảng Secondary).
  3. **Bảo đảm Toàn vẹn cho Toàn bộ Các Action Phía Sau (`GAP-DOWNSTREAM-1`..`GAP-DOWNSTREAM-5` — Turn 16):**
     - **Chống Miss Value của Secondary khi Consumer lệch nhau (`GAP-DOWNSTREAM-1`):**
       - Trên luồng Realtime NATS: nếu Secondary tới trước thì hoãn chờ Primary (`!st.HasPrimaryActive`); nếu Primary tới trước khi Secondary kịp vào Shadow DB thì `ApplySourceFieldTypesToBSONDoc` (`mongo_bson_restorer.go:43-126`) vẫn phải chuẩn hóa `snake_case -> camelCase`, promote `_id: ObjectId`, và xóa 18 cột index phẳng ngay cả khi chưa có `source_field_types`, đồng thời khi Secondary tới sau thì `enrichMultiShadowEvents` tra ngược về `_source_id` của Primary, hợp nhất đủ Primary + $N$ Secondary với `_source_ts = MAX(_source_ts)` (WAL UTC) và `$unset` sạch mọi cột phẳng/kỹ thuật dư thừa trên MongoDB.
       - Trên luồng Batch / Incremental / Cron (`transmuter.go:1390` & `transmuter_multi_shadow.go`): mở rộng `syncUpdatedShadowRowsBeforeCursor` quét bù cả **các bảng Secondary trong `shadow_sources` có `_updated_at >= cutoff` (hoặc `_source_ts >= cutoffMs`)**, ánh xạ về `_source_id` của Primary để chạy `enrichMultiShadowBatchRows` $\rightarrow$ `processBatch` (đảm bảo dù consumer Secondary lag bao lâu hay rớt NATS event vẫn tự động lấp đầy 100% value của Secondary lên Master).
     - **Đối soát Segment B & `RunRowDiffB` cho Join đa bảng (`GAP-DOWNSTREAM-2`):**
       - Trong `RunHashWindowCheckB` & `RunDeepCheckB` (`recon_tier_b.go`): bổ sung `checkMultiShadowCompletenessInWindow` để phát hiện các bản ghi đã có bên Secondary Shadow nhưng trên Master bị thiếu dữ liệu Secondary (hoặc `Master._source_ts < Secondary._source_ts`), đưa vào `mismatchedIDs` (`status = "drift"`) thay vì chỉ so `(id, ts)` với riêng bảng Primary.
       - Trong `RunRowDiffB` (`recon_tier_b.go:654-713`): tự động co-fetch và merge `_raw_data` của tất cả các bảng Secondary trong `shadow_sources` trước khi so sánh từng mapping rule với Master.
     - **Heal Segment B, Delete đa bảng & Quét Nested JSON (`GAP-DOWNSTREAM-3`..`5`):**
       - Khi chạy Targeted Sync/Heal theo `onlyIDs` (`len(onlyIDs) > 0`), cho phép ghi đè đầy đủ các trường đã merge từ Secondary lên Master mà không bị OCC chặn; phân giải ngược `primary_source_id` khi nhận event Delete từ Secondary; và mở rộng `scan_service.go:247` tìm `master_binding` qua `shadow_sources` khi quét Nested JSON trên bảng Secondary.
- **Yêu cầu Sửa Lỗi Tạo & Quản lý Index trên MongoDB Master (`tạo index trên mongo chưa tạo đc.` — Turn 20 — `TASK-09`):**
  1. **Triệu chứng thực tế:** Khi người dùng vào màn hình Quản lý Index (`TableIndexManager.tsx` tại `MasterMappingFieldsPage.tsx:1079`) trên bảng Master đích MongoDB (hoặc khai báo `indexes` trong `transform_spec` và nhấn Approve DDL) thì **không tạo được index trên MongoDB** (và danh sách index hiện trống hoặc báo thành công giả nhưng thực chất bị lỗi ngầm dưới worker).
  2. **4 Nguyên nhân gốc rễ (`F1`..`F4`) cần xử lý triệt để theo đúng Single-Responsibility (`L008`):**
     - **`F1` (`IndexHandler` & `IndexManager` tại `centralized-data-service` chỉ hỗ trợ PostgreSQL GORM và từ chối kết nối MongoDB Master):**
       - `IndexHandler.resolveDB` (`internal/handler/governance/index_handler.go:34-62`) gọi `h.connMgr.GetMasterDB(ctx, key)`. Với kết nối MongoDB Master (`IsMongoMasterConnection == true`), `GetMasterDB` trả lỗi từ chối (`connection is a MongoDB destination; use GetMasterMongoClient instead of GetMasterDB`).
       - `IndexManager` (`internal/service/governance/index_manager.go:40-166`) dùng thuần SQL PostgreSQL (`pg_index`, `information_schema.columns`, `CREATE INDEX CONCURRENTLY`, `DROP INDEX CONCURRENTLY`) và `sqlutil.IsSafeIdent(col)` chặn dấu chấm `.` của nested BSON path (`sender.customerId`, `info.orderId`).
       - Cần tách module đơn trách nhiệm `internal/service/governance/index_manager_mongo.go` thực thi `ListMongoIndexes`, `CreateMongoIndex`, `DropMongoIndex`, `GetMongoRecommendations` qua `*mongo.Collection` (`coll.Indexes().List`, `coll.Indexes().CreateOne`, `coll.Indexes().DropOne`) và điều hướng trong `IndexHandler` khi `plane == "master" && h.connMgr.IsMongoMasterConnection(ctx, req.ConnectionKey)`.
     - **`F2` (`IntrospectionHandler` tại `cdc-cms-service` nuốt lỗi NATS RPC trả HTTP 200 OK giả & thiếu `table` khi `DropIndex`):**
       - Tại `cdc-cms-service/internal/api/system/introspection_handler.go:466-584` (`ListIndexes`, `CreateIndex`, `DropIndex`), khi worker trả `{"status":"error","error":"..."}`, handler vẫn trả `c.JSON(res)` với HTTP `200 OK` khiến UI `TableIndexManager.tsx` tưởng thành công và hiện toast giả `"Index creation command submitted concurrently."`!
       - Hơn nữa, `DropIndex` (`introspection_handler.go:553-570` và `TableIndexManager.tsx:86-88`) không truyền `table` (tên collection), trong khi MongoDB bắt buộc phải có tên collection (`coll.Indexes().DropOne(ctx, indexName)`) mới xóa được index!
       - Đồng thời `introspection_handler.go` đang là God-file (`584` dòng > 500 dòng) → tách `ListIndexes`, `CreateIndex`, `DropIndex` sang file đơn trách nhiệm `cdc-cms-service/internal/api/system/introspection_index_handler.go` để giảm `introspection_handler.go` xuống `465` dòng (`NOW < HEAD`).
     - **`F3` (`applyMongoCollectionAndIndexes` tại `master_ddl_generator_mongo.go` bỏ sót `transform_spec.indexes` và lệch tên cột `snake_case` vs `camelCase`):**
       - Tại `centralized-data-service/internal/service/master/master_ddl_generator_mongo.go:140-208`, hàm `applyMongoCollectionAndIndexes` chưa đọc `parseIndexesFromSpec(reg.TransformSpec)` (trong khi nhánh PostgreSQL `master_ddl_generator.go:303` có đọc) và `res.FinancialCols` đang lấy tên `target_column` dạng `snake_case` (`trans_id`) thay vì tên field BSON thực tế trên MongoDB (`transId`).
     - **`F4` (UI `TableIndexManager.tsx` & `MasterMappingFieldsPage.tsx` cho MongoDB Master):**
       - Truyền `table` vào `DELETE /api/introspection/indexes/:indexName` (`params: { schema, table, plane, connection_key: connectionKey }`).
       - Cho phép chọn/nhập cả tên field BSON (`source_field`, `source_path`, `camelCase` như `transId`, `sender.customerId`, `info.orderId`) trên `TableIndexManager.tsx` (`mode="tags"`), đồng thời giữ `MasterMappingFieldsPage.tsx` `NOW <= HEAD` (`1406` dòng).
- **Yêu cầu Kiểm tra & Xử lý Triệt để Hiện tượng "Thiếu Trước Hụt Sau" trên Document MongoDB Master `trans_his` (`sao nó cứ thiếu trước hụt sau là sao` — Turn 22 — `TASK-10`, Masked PII theo Rule #20):**
  1. **Dữ liệu thực tế User phản ánh:** User gửi 2 document MongoDB Master thực tế (`_id: ObjectId('6ae314dbe69fdacd4ff3996b')` — `transType: 'PARTNER_RESERVE'` và `_id: ObjectId('6ae2c97e4420823c7550065c')` — `transType: 'INTERNAL_BANK_TRANSFER'`, đã mask PII `<MASKED_NAME>`, `<MASKED_ACCOUNT>`) và hỏi: **`sao nó cứ thiếu trước hụt sau là sao`**.
  2. **5 Nguyên nhân gốc rễ (`F1`..`F5`) cần phân tích rõ và xử lý triệt để theo Single-Responsibility (`L008`):**
     - **`F1` (`source_field_values` từ bảng phụ `transaction_history_payload` bị lọt nguyên cục vào Document MongoDB đích và chưa được bóc tách/strip):**
       - Tại `mongo_bson_restorer.go:64-69`, `ApplySourceFieldTypesToBSONDoc` mới chỉ đọc và xóa `source_field_types` / `sourceFieldTypes`, chưa đọc và xóa `source_field_values` / `sourceFieldValues`. Hậu quả là mỗi document trên MongoDB Master bị dính nguyên một trường `source_field_values: { ... }` lặp lại toàn bộ 16 trường gốc ở dạng Extended JSON (`$numberInt`, `$date`, `$oid`).
     - **`F2` (Các trường số/ngày/ObjectId bên trong `info` và nested JSONB bị giữ nguyên dạng Extended JSON wrapper `{ $numberInt: '15000' }`, `{ $numberInt: '0' }`, `{ $numberInt: '117600' }` thay vì kiểu BSON gốc `NumberInt(...)`):**
       - Tại `mongo_type_converter.go:292-328` (`convertMapToBSONM` & `convertNestedElement`), khi chuyển cột `JSONB` (`info`, `sender`, `receiver`, `error`) sang BSON, code copy thẳng từng key của `map[string]any` sang `bson.M` mà không unwrap các wrapper Extended JSON của Debezium/MongoDB (`$numberInt` $\rightarrow$ `int32`, `$numberLong` $\rightarrow$ `int64`, `$numberDouble` $\rightarrow$ `float64`, `$numberDecimal` $\rightarrow$ `primitive.Decimal128`, `$date` $\rightarrow$ `primitive.DateTime`, `$oid` $\rightarrow$ `primitive.ObjectID`). Kết quả là `info.transAmount`, `info.fee`, `info.senderBalance`, `info.receiverBalance`, `info.bankTransferResponse.fee`, `info.bankTransferResponse.amount`, `info.bankTransferData.amount` bị lưu thành sub-object `{$numberInt: "..."}` thay vì `NumberInt(...)`.
     - **`F3` (`request_id` / `requestId` và `is_delete` / `isDelete` bị xóa mất oan khi có dữ liệu thật, trong khi `idempotency_key` lại bị giữ nguyên `snake_case` và lọt cả `idempotency_key: null` khi vắng mặt):**
       - Tại `mongo_bson_restorer.go:19-37, 79-113`: `snakeToCamelCoreMap` thiếu `request_id` (`requestId`), `is_delete` (`isDelete`), `idempotency_key` (`idempotencyKey`); đồng thời `request_id` và `is_delete` bị đặt trong `denormalizedFlatColumns` và bị kiểm tra `typeMap["request_id"]` (luôn `false` vì `typeMap` lưu `camelCase` `"requestId"`) trước khi đổi sang `camelCase`, làm mất `requestId` / `isDelete` ngay cả khi bản ghi gốc có trường này.
       - Ngược lại, `idempotency_key` không nằm trong `snakeToCamelCoreMap` và khi có giá trị `NULL` trên bảng phẳng PostgreSQL (như ở Doc 2 `6ae2c97e4420823c7550065c`) lại không được xóa bỏ vì `ApplySourceFieldTypesToBSONDoc` chưa xóa các trường top-level có giá trị `nil` mà không tồn tại trong `typeMap` / `source_field_values` (trong khi ở Doc 1 `6ae314dbe69fdacd4ff3996b` có giá trị `'RADA_HOLD:AHFMLU:PF-1793266905416'` nhưng bị giữ tên `snake_case` `idempotency_key` thay vì `idempotencyKey`).
     - **`F4` (Khi `typeMap` rỗng ở nhịp T1 lúc Primary `transaction_history` tới trước `transaction_history_payload`, `ApplySourceFieldTypesToBSONDoc` thoát sớm ở dòng `70-76`):**
       - Làm cho ở nhịp T1 các cột `snake_case` và 18 cột index phẳng chưa được chuẩn hóa/lọc ngay.
     - **`F5` (Đối chiếu sự khác biệt giữa `receiver` 15 keys của Doc 1 `PARTNER_RESERVE` vs 16 keys của Doc 2 `INTERNAL_BANK_TRANSFER`):**
       - Đối chiếu trực tiếp với `source_field_values.receiver` ngay trong 2 document: bản thân dữ liệu gốc trên MongoDB Source của `PARTNER_RESERVE` có 15 keys trong `receiver` (không có `walletId`, vì ví đích nằm ở `info.targetWalletId`), còn `INTERNAL_BANK_TRANSFER` có 16 keys trong `receiver` (có `walletId: null`).
- **Yêu cầu Bổ sung & Audit Lại Plan (`TASK-11 / TASK-10 Revision` — Turn 23, Masked PII theo Rule #20):**
  1. **Vấn đề 1 — Tạo và vận hành bảng Shadow không có cột `updated_at` (`"có 1 vấn đề đó là a tạo cái shadow ko có update at. giải pháp tạo nó như nào."`):**
     - **Ở tầng Đăng ký & Khởi tạo DDL Bảng Shadow (`TableRegistry.tsx`, `register_registry.go`, `schema_ddl_handler.go`):**
       - Trên Modal `Register New Source Object -> Shadow Object` (`TableRegistry.tsx:1647, 1659-1668, 951-962`): Khi chọn connector PostgreSQL (`isPostgres`), không giữ mặc định `'lastUpdatedAt'` của MongoDB; cho phép xóa trắng (`allowClear`) và chuẩn hóa `timestamp_field` rỗng thành `null` ngay trong `handleRegister`.
       - Tại `RegisterRegistryHandler.Handle` (`cdc-cms-service/internal/app/commands/source/register_registry.go:83`): Chuẩn hóa `entry.TimestampField` rỗng thành `nil` để ghi `NULL` xuống `cdc_table_registry.timestamp_field` và `source_object_registry.timestamp_field`.
       - Tại `HandleCreateDefaultColumns` (`centralized-data-service/internal/handler/shadow/schema_ddl_handler.go:384-389`): Khi `reg.TimestampField == nil || strings.TrimSpace(*reg.TimestampField) == ""`, bỏ qua bước tự tạo index trên cột `"updated_at"` (vì bảng Shadow đã có sẵn các cột và B-Tree index thời gian hệ thống `_source_ts` và `_synced_at`).
     - **Ở tầng Đối soát Recon Segment B (`recon_engine_segment_b.go:129-142`):**
       - Tại `resolveSegmentBTSCol` (`recon_engine_segment_b.go:138`): Khi bảng Shadow không có `updated_at`, `resolveSourceAndDestTSFields` trả về `dstTS = "_id_only"` (dành cho Segment A). Segment B bắt buộc phải kiểm tra `if errTS == nil && dstTS != "" && dstTS != "_id_only"` để giữ `tsCol = "_source_ts"`, tránh lỗi `SQLSTATE 42703: column "_id_only" does not exist` khi chạy `HashWindow` / `MaxWindowTs` / `TimeBoundedDiffMissingFromMaster`!
     - **Ở tầng Khôi phục Trường `updatedAt: ISODate(...)` trên MongoDB Master khi Bảng Shadow không có `updated_at` (`mongo_extjson_unwrapper.go` & `mongo_bson_restorer.go`):**
       - Khi bảng Shadow không có cột `updated_at`, trường `updatedAt` trên document MongoDB Master được tạo/khôi phục tự động theo thứ tự ưu tiên:
         (1) Từ `updated_at` / `updatedAt` hoặc `source_field_values.updatedAt` (unwrap `$date` $\rightarrow$ `primitive.DateTime` / `ISODate`);
         (2) Nếu vẫn thiếu (`doc["updatedAt"] == nil`): tự động tổng hợp từ cột hệ thống `_source_ts` (`epochToDateTime(doc["_source_ts"])` — thời điểm commit giao dịch trên WAL nguồn tính bằng `ISODate`), hoặc fallback sang `doc["createdAt"]` / `time.Now().UTC()`!
         (3) Tương tự, nếu `doc["createdAt"] == nil`: lấy từ `source_field_values.createdAt` hoặc thời gian nhúng trong 4 byte đầu của `_id: ObjectId(...)` (`oid.Timestamp()`).
  2. **Vấn đề 2 — Chuẩn Format & Kiểu BSON trên MongoDB Master, Không Thiếu Hụt Trường Core (`"format nó phải như này trên mongo, chỉ format thôi, còn field thì a đang test nên ko 1:1 e đừng quan tâm. các type nó phải như này và ko bị thiếu hụt"`):**
     - Mẫu chuẩn (đã mask PII theo Rule #20): `_id: ObjectId(...)`, `transId: string`, `sysTrans: bool`, `originalTransHisId: null`, `sender: {...}`, `receiver: {...}`, `value: NumberInt(...)`, `transType: string`, `status: string`, `error: null`, `noted: string`, `info: { transAmount: NumberInt(...), fee: NumberInt(...), billId: NumberInt(...), senderBalance: NumberInt(...), serviceCode: null, channelId: null, ... }`, `serviceCode: string|null`, `channelId: string|null`, `createdAt: ISODate(...)`, `updatedAt: ISODate(...)`, `__v: NumberInt('0')`.
     - **Phản biện & Sửa điểm hở `C-AUDIT-1..3` của Plan cũ:**
       - **`C-AUDIT-1` (Chống mất trường core có giá trị `null` khi test subset field):** Không dùng vòng lặp xóa mù mọi `v == nil && !hasTypeMeta(typeMap, k)` (vì khi anh đang test chỉ map một phần hoặc chưa map đủ `typeMap`, vòng lặp đó sẽ xóa mất `originalTransHisId: null`, `error: null`, `serviceCode: null`, `channelId: null`). Thay vào đó, chỉ lược bỏ 3 trường tùy chọn (`requestId`, `idempotencyKey`, `isDelete`) khi chúng `nil`/`false` và không có trong `typeMap`, bảo toàn 100% các trường core nullable!
       - **`C-AUDIT-2` (Bảo đảm `value: NumberInt(...)` và mọi số nguyên trong `info` đều là `NumberInt` kể cả khi test không có `source_field_types`):** Unwrap đệ quy cả `map[string]any` lẫn `bson.M` trong `convertNestedElement`, bổ sung `case json.Number:` vào `detectBsonType`, và khi `doc["value"]` không có `typeMap` chỉ định `long` mà nằm trong dải `int32` thì ép về `int32` (`NumberInt`).
       - **`C-AUDIT-3` (Chạy chuẩn hóa `snake_case -> camelCase`, `_id: ObjectId`, `createdAt`/`updatedAt: ISODate`, `__v: NumberInt(0)` ngay cả khi `typeMap` rỗng):** Bỏ `return` sớm khi thiếu `source_field_types`.
- **G2 (Propose-Only):** Đang ở bước điều tra & thiết kế giải pháp (`S1 -> S3`), tuyệt đối không sửa source code hay dữ liệu DB trước khi có `APPROVE` của User.
