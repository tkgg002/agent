# 09 Tasks Solution: Giải pháp Hỗ trợ `Scan Array (Flatten)` Trên Bảng Secondary Shadow (`transaction_history_payload`, `binding_id=77`)

tier: T2
suffix: scan_array_secondary_shadow
date: 2026-11-03

---

## 1. Tóm tắt Trực diện (Direct Summary)
Khi anh bấm **`Scan Array (Flatten) → Tạo rules PENDING để duyệt`** tại `http://localhost:5173/masters/transaction_history/mappings?binding_id=77` cho một trường `JSONB` (như `info`, `sender`, `receiver`, `error`) thuộc bảng phụ **`transaction_history_payload`**, hệ thống thất bại và báo không có dữ liệu trên bảng Shadow vì **4 nguyên nhân gốc rễ (`F1`–`F4`)**:
1. **`F1` (`MasterMappingFieldsPage.tsx:403-415` & `1235-1267`):** Frontend luôn gửi cứng `binding.shadow_table` (bảng chính `transaction_history`) vào `/api/introspection/scan-array/transaction_history` thay vì phân giải bảng Shadow thực tế (`transaction_history_payload`) chứa field cần flatten từ `shadowRules`, và khi `total_keys == 0` (hoặc `status == "skipped"`) lại hiển thị thông báo cứng `"shadow table chưa có data..."`.
2. **`F2` (`scan_service.go:198-215`):** Tại Backend `ScanService.ScanArrayFields`, hễ `masterBindingID != nil` (`77`) là hàm truy vấn `JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id` (chỉ lấy bảng chính `transaction_history`) và **ghi đè vô điều kiện `targetTable = "transaction_history"`**, bỏ qua hoàn toàn các bảng phụ trong `mb.transform_spec->'shadow_sources'` (`transaction_history_payload`). Do trường JSONB cần flatten không nằm trên bảng chính `transaction_history`, truy vấn `_raw_data` trả về 0 dòng (`total_keys = 0`).
3. **`F3` (`scan_service.go:250-258`, `310-311`, `396-409`, `448-455`):**
   - Khi bảng Shadow có Debezium envelope (`_raw_data ? 'after'`), `ScanArrayFields` nối `"after."` trực tiếp vào biến `explodePath` khiến `basePath` bị biến thành `"after.<field>"`, sinh sai `target_column` (`"after_<field>_..."`), `source_field` (`"after.<field>"`), `source_path` (`"after.<field>..."`).
   - Đồng thời, câu truy vấn tìm `parentV2ID` (`scan_service.go:450-454`) chỉ nối `mb.shadow_binding_id = v2.shadow_binding_id` (Primary Shadow) nên với mọi trường thuộc bảng phụ `transaction_history_payload`, `parentV2ID` luôn bằng `0` (`mapping_v2_id = NULL`), làm mất liên kết về bảng phụ và mất badge `transaction_history_payload` trên giao diện Mapping.
4. **`F4` (`scan_service.go:302-309` & `transmuter.go:1768-1781`):**
   - Nếu trường JSONB trong `_raw_data` được Debezium serialize dưới dạng chuỗi JSON (`jsonb_typeof == "string"`), `scan_service.go:302-307` chặn sớm trả về `status: "skipped"` và `isObject = false` thay vì parse chuỗi JSON đó để flatten.
   - Tại `Transmuter.extractColumns` (`transmuter.go:1768-1781`), khi trích xuất `r.SourcePath` (ví dụ `"info.orderId"`), cần hỗ trợ đồng thời cả JSON đã unwrap `"after"` (ở chế độ `multi_shadow` qua `mergeShadowRawJSON`), JSON còn bọc `"after"` (ở chế độ `copy_1_to_1`), và trường hợp trường gốc là chuỗi JSON stringified.

---

## 2. Phân tích Chi tiết Các Phát hiện (`F1` → `F4`)

### Phát hiện `F1` — Frontend `MasterMappingFieldsPage.tsx` (`cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:391-426` & `1235-1273`) gửi cứng bảng chính `binding.shadow_table`
- **Hiện trạng (`MasterMappingFieldsPage.tsx:391-426`):**
  - `shadowRules` trên trang đã nạp đầy đủ các rule từ cả `transaction_history` (Primary) và `transaction_history_payload` (Secondary) thông qua `loadAllBindingShadowRules(binding)` (`ShadowSourceBadge.tsx:60-98`). Mỗi phần tử `r` trong `shadowRules` đều có sẵn `r.shadow_table`, `r.shadow_schema`, `r.shadow_binding_id`, `r.source_field`, `r.target_column`.
  - Tuy nhiên `handleScanArray` (`MasterMappingFieldsPage.tsx:403-406`) lại gọi cứng:
    ```tsx
    const { data: res } = await cmsApi.post(
      `/api/introspection/scan-array/${binding.shadow_table}`,
      { explode_path: path, master_binding_id: masterBindingID },
    );
    ```
  - Ngoài ra, nếu backend trả về `res.status === 'skipped'` hoặc `res.status === 'error'` (kèm `res.reason` hoặc `res.error`), Frontend không kiểm tra mà rơi vào nhánh `newCount === 0 && total === 0` (`MasterMappingFieldsPage.tsx:413`) và báo:
    `Không tìm thấy field nào trong "${path}" — shadow table chưa có data, hoặc field không phải JSON object. Kiểm tra shadow đã ingest data chưa.`

### Phát hiện `F2` — Backend `ScanService.ScanArrayFields` (`centralized-data-service/internal/service/source/scan_service.go:198-215`) ghi đè `targetTable` về Primary Shadow
- **Hiện trạng (`scan_service.go:198-215`):**
  ```go
	if masterBindingID != nil {
		var sb struct {
			ShadowSchema string `gorm:"column:shadow_schema"`
			ShadowTable  string `gorm:"column:shadow_table"`
		}
		if err := s.db.WithContext(ctx).Raw(`
            SELECT sb.shadow_schema, sb.shadow_table
              FROM cdc_system.master_binding mb
              JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
             WHERE mb.id = ?`, *masterBindingID).Scan(&sb).Error; err == nil {
			if strings.TrimSpace(sb.ShadowTable) != "" {
				targetTable = sb.ShadowTable
			}
			if strings.TrimSpace(sb.ShadowSchema) != "" {
				schemaName = sb.ShadowSchema
			}
		}
	}
  ```
- **Hệ quả:** Khi `masterBindingID = 77` (`transaction_history`, `transform_type = 'multi_shadow'`), hàm luôn ép `targetTable = "transaction_history"`, hoàn toàn không kiểm tra xem:
  1. `targetTable` truyền vào (ví dụ `"transaction_history_payload"`) có thuộc danh sách Shadow của `master_binding` (`mb.shadow_binding_id` + `mb.transform_spec->'shadow_sources'`) hay không;
  2. Hoặc trường gốc (`rootField` của `explodePath`, ví dụ `"info"`, `"sender"`, `"receiver"`) thuộc về `shadow_binding` nào trong nhóm `multi_shadow` của `master_binding_id=77`!

### Phát hiện `F3` — Backend `ScanService.ScanArrayFields` (`scan_service.go:250-258`, `310-311`, `396-409`, `448-455`) nối `"after."` làm sai lệch `basePath` và chỉ tìm `parentV2ID` trên `mb.shadow_binding_id`
- **Hiện trạng (`scan_service.go:250-258`, `310-311`, `448-455`):**
  1. Dòng `256-258`:
     ```go
     	if hasAfter && !strings.HasPrefix(explodePath, "after.") && explodePath != "after" {
     		explodePath = "after." + explodePath
     	}
     ```
     Khi `_raw_data` có khóa `'after'`, `explodePath` bị gắn thêm `"after."` (thành `"after.info[*]"`). Ngay sau đó dòng `310` tính `basePath` từ `explodePath` đã bị sửa $\rightarrow$ `basePath = "after.info"`, làm cho:
     - `target_column` sinh ra bị dính tiền tố `after_` (`"after_info_orderId"` thay vì `"info_orderId"`),
     - `source_field` sinh ra là `"after.info"` (thay vì `"info"` — làm gãy cây phân cấp `treeData` trên `MasterMappingFieldsPage.tsx:610-632`),
     - `source_path` sinh ra là `"after.info.orderId"` (thay vì `"info.orderId"`).
  2. Dòng `448-455`:
     ```go
     			var parentV2ID int64
     			tx.Raw(`
                     SELECT v2.id FROM cdc_system.mapping_rule_v2 v2
                        JOIN cdc_system.master_binding mb ON mb.shadow_binding_id = v2.shadow_binding_id
                      WHERE mb.id = ? AND v2.target_column = ? LIMIT 1`,
     				*masterBindingID, basePath).Scan(&parentV2ID)
     ```
     Câu SQL này chỉ tìm trên `mb.shadow_binding_id` (bảng chính `transaction_history`) và so sánh với `basePath` (đã bị dính `"after."` hoặc chứa đường dẫn nhiều cấp). Với trường nằm ở bảng phụ `transaction_history_payload` (`binding.ID`), `parentV2ID` luôn bằng `0` (`mapping_v2_id = NULL`), khiến `masterRuleSelectJoin` (`master_mapping_rule_repo_gorm.go:106-110`) không lấy được `shadow_binding_id` và `shadow_table` (`transaction_history_payload`).

### Phát hiện `F4` — Backend `ScanService.ScanArrayFields` (`scan_service.go:302-309`) & `Transmuter.extractColumns` (`transmuter.go:1768-1781`)
- **Hiện trạng:**
  1. Tại `scan_service.go:302-309`: Khi `pathType == "string"` (xảy ra khi trường JSON được lưu dạng stringified JSON trong `_raw_data`), hàm trả về `status: "skipped"` ngay lập tức thay vì cho phép vòng lặp dòng `337-360` giải mã chuỗi JSON (`json.Unmarshal`) để bóc tách các trường con; đồng thời biến `isObject` chỉ kiểm tra `pathType == "object"` mà chưa xét trường hợp `pathType == "string"` nhưng nội dung giải mã ra là JSON Object/Array (`len(discoveredFields) > 0`).
  2. Tại `transmuter.go:1768-1781` (`extractColumns`): Khi trích xuất rule có `r.SourcePath` (ví dụ `"info.orderId"`):
     - Trên `multi_shadow`, `mergeShadowRawJSON` đã bóc `"after"` ra khỏi `_raw_data`, nhưng trên `copy_1_to_1` thì `_raw_data` vẫn có `"after"`.
     - Nếu `gjson.Get(rawStr, path)` chưa khớp, cần thử `after.<path>` (hoặc bỏ tiền tố `after.` nếu `path` có sẵn `after.`), và nếu segment đầu tiên (`rootSeg`, ví dụ `"info"`) là một chuỗi JSON stringified thì bóc tiếp phần đường dẫn con bên trong chuỗi JSON đó trước khi fallback về `lastSeg`.

---

## 3. Biên bản Phản biện Kiến trúc Layer, Single-Responsibility & Tương thích ngược (Adversarial Plan Audit — S3.5)

### 3.0. Bốn Điểm Hiệu chỉnh Từ Phản biện S3.5 (`CRIT-01` → `CRIT-04`)
1. **`CRIT-01` (Tách Module Đơn Trách Nhiệm — Tuyệt đối không nhồi nhét vào God-files hay làm loãng file cũ):**
   - **Phản biện:**
     - [`scan_service.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/scan_service.go) đang dài **772 dòng**; [`scan_master_binding_resolver.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/scan_master_binding_resolver.go) đang dài **33 dòng** và chỉ có đúng 1 trách nhiệm đơn là `resolveMasterBindingForShadow`. Nếu nhồi 250+ dòng phân giải `ScanArray` vào `scan_master_binding_resolver.go` sẽ phá vỡ Single-Responsibility của file đó.
     - [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go) đang dài **1888 dòng**. Nếu viết thêm 25 dòng fallback GJSON trực tiếp vào `extractColumns` (`transmuter.go:1768-1782`), file `transmuter.go` sẽ phình thêm dòng.
     - [`MasterMappingFieldsPage.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx) đang dài **1403 dòng** và [`ShadowSourceBadge.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/master/ShadowSourceBadge.tsx) là file UI Component (`.tsx`). Đặt logic xử lý chuỗi `explodePath` vào `ShadowSourceBadge.tsx` là sai layer.
   - **Quyết định Kiến trúc Chuẩn SRP & Layer:**
     - Tạo file mới **`centralized-data-service/internal/service/source/scan_array_target_resolver.go`** (Layer: `internal/service/source`) chuyên trách phân giải bảng Shadow đích, làm sạch đường dẫn `explodePath`, giải mã stringified JSON và tra cứu `parentV2ID` cho `ScanArrayFields`. Giữ nguyên `scan_master_binding_resolver.go` (`33` dòng), đồng thời rút gọn `scan_service.go` từ **772 dòng xuống ~748 dòng (-24 dòng)**.
     - Tạo file mới **`centralized-data-service/internal/service/master/transmuter_gjson_path.go`** (Layer: `internal/service/master`) chuyên trách hàm pure `resolveRuleGJSONValue(rawStr, r)`. Trong `transmuter.go:1768-1781`, rút gọn 14 dòng inline thành **1 dòng gọi `gres, path := resolveRuleGJSONValue(rawStr, r)`**, giúp `transmuter.go` giảm từ **1888 dòng xuống 1875 dòng (-13 dòng)**.
     - Tạo file mới **`cdc-cms-web/src/utils/masterScanArray.ts`** (Layer: `src/utils`, khớp convention `apiError.ts`, `datetime.ts`, `actionTrace.ts`) chứa 2 pure helpers `resolveShadowTableForExplodePath` và `buildScanArrayFieldOptions`. Rút gọn IIFE 23 dòng tại `MasterMappingFieldsPage.tsx:1244-1266` thành 1 dòng gọi helper, giúp `MasterMappingFieldsPage.tsx` giảm từ **1403 dòng xuống ~1386 dòng (-17 dòng)**.

2. **`CRIT-02` (Đảm bảo Tương thích Ngược 100% cho `copy_1_to_1`, Legacy Rules có tiền tố `after_`, và `ScanArrayAll`):**
   - Với `copy_1_to_1` (chỉ có 1 Primary Shadow): `collectMasterShadowCandidates` trả về 1 phần tử duy nhất $\rightarrow$ chọn ngay phần tử đó mà không cần chạy bất kỳ câu SQL dò `_raw_data` phụ nào.
   - Với các bảng cũ đã từng chạy `ScanArrayFields` và sinh cột có tiền tố `after_<field>_<subkey>` trong `cdc_system.mapping_rule_master`: Khi kiểm tra trùng lặp tại `scan_service.go:400`, kiểm tra đồng thời `if existingMaster[objTarget] || existingMaster["after_"+objTarget] { continue }` để không tạo trùng rule.
   - Với `GetApprovedJSONFieldsByMasterBindingID` (`mapping_rule_repo_gorm.go:359-372`): Nếu `len(shadowIDs) == 0` (ví dụ trong môi trường test chỉ gán `master_binding_id` trên `mapping_rule_v2` mà không tạo `master_binding`), vẫn giữ nguyên câu truy vấn fallback `WHERE master_binding_id = ?` thay vì trả về `nil, nil`.

3. **`CRIT-03` (Đảm bảo Tương thích Ngược thông báo `pathType == "string"`):**
   - Giữ nguyên thông báo gốc `"Dữ liệu đang là 'string' (stringified JSON)."` khi trường có `jsonb_typeof == "string"` nhưng nội dung chuỗi không phải là JSON Object/Array bóc tách được (`len(discoveredFields) == 0`), đồng thời mở khóa bóc tách bình thường khi nội dung chuỗi thực sự là JSON Object/Array (`len(discoveredFields) > 0`).

4. **`CRIT-04` (Bảo toàn `shadow_table` trong `loadAllBindingShadowRules` tại `ShadowSourceBadge.tsx:60-98`):**
   - Trong `loadAllBindingShadowRules`, gắn kèm `fallbackTable` của từng nguồn truy vấn (`binding.shadow_table` cho Primary và `s.shadow_table` cho từng Secondary trong `binding.spec.shadow_sources`) để nếu bản ghi trả về từ `/api/v1/mapping-rules` thiếu `shadow_table` thì tự động điền `r.shadow_table || fallbackTable`.

---

### 3.1. Bảng Kiểm toán Layer, Single-Responsibility & Ngân sách Dòng (`wc -l`)

| STT | Component & Layer | File | Trách nhiệm Đơn (SRP) | `HEAD` (`wc -l`) | Dự kiến `NOW` (`wc -l`) | Biến thiên |
| :---: | :--- | :--- | :--- | ---: | ---: | :---: |
| 1 | `centralized-data-service` / `internal/service/source` | `scan_service.go` | Điều phối luồng `ScanRawData`, `ScanArrayFields`, `Backfill` | `772` | `~748` | **-24 dòng** |
| 2 | `centralized-data-service` / `internal/service/source` | `scan_master_binding_resolver.go` | Phân giải `MasterBindingID` từ `ShadowBindingID` | `33` | `33` | **0 dòng (Giữ nguyên)** |
| 3 | `centralized-data-service` / `internal/service/source` | `scan_array_target_resolver.go` *(Mới)* | Phân giải Shadow Target (Primary/Secondary), `parentV2ID`, `cleanLogicalExplodePath`, `parseScanNodeJSON` cho `ScanArrayFields` | `0` | `~215` | **File mới SRP** |
| 4 | `centralized-data-service` / `internal/service/master` | `transmuter.go` | Điều phối pipeline Transmuter | `1888` | `1875` | **-13 dòng** |
| 5 | `centralized-data-service` / `internal/service/master` | `transmuter_gjson_path.go` *(Mới)* | Pure helper `resolveRuleGJSONValue` trích xuất GJSON đa định dạng (`multi_shadow`, `debezium_after`, stringified JSON, `lastSeg`) | `0` | `~48` | **File mới SRP** |
| 6 | `cdc-cms-service` / `internal/infra/persistence/master` | `mapping_rule_repo_gorm.go` | GORM Repository cho `mapping_rule_v2` (`GetApprovedJSONFieldsByMasterBindingID`) | `372` | `~386` | **+14 dòng (< 400)** |
| 7 | `cdc-cms-web` / `src/pages` | `MasterMappingFieldsPage.tsx` | Page component quản lý Mapping Fields của Master | `1403` | `~1386` | **-17 dòng** |
| 8 | `cdc-cms-web` / `src/components/master` | `ShadowSourceBadge.tsx` | Badge hiển thị nguồn Shadow & loader `loadAllBindingShadowRules` | `110` | `~114` | **+4 dòng (< 120)** |
| 9 | `cdc-cms-web` / `src/utils` | `masterScanArray.ts` *(Mới)* | Pure utils `resolveShadowTableForExplodePath` & `buildScanArrayFieldOptions` | `0` | `~54` | **File mới SRP** |

### 3.2. Bảng Ánh xạ Điểm Gọi Thực thi & Vòng đời Kích hoạt (Lesson `L012`)
| Hạng mục | File thực thi | Hàm / Component | Điểm gọi (`Caller file:line`) | Thời điểm kích hoạt (`Lifecycle / Trigger`) |
| :--- | :--- | :--- | :--- | :--- |
| **Giải pháp `F1`** | `cdc-cms-web/src/utils/masterScanArray.ts`, `ShadowSourceBadge.tsx` & `MasterMappingFieldsPage.tsx` | `resolveShadowTableForExplodePath`, `buildScanArrayFieldOptions`, `handleScanArray` | `MasterMappingFieldsPage.tsx:391-426` & `1244` (`Modal onOk={handleScanArray}`) | Khi User mở Modal `Scan Array (Flatten)`, chọn field JSON từ dropdown và bấm nút **Scan**. |
| **Giải pháp `F2` & `F3`** | `centralized-data-service/internal/service/source/scan_array_target_resolver.go` & `scan_service.go` | `resolveShadowTargetForScanArray`, `resolveParentV2IDForScanArray`, `cleanLogicalExplodePath` | `scan_service.go:196-248` & `448-455` (được gọi từ `ScanHandler.HandleScanArrayFields` tại `scan_handler.go:228` qua NATS `cdc.cmd.scan-array`) | Khi Worker xử lý lệnh `cdc.cmd.scan-array` từ API `/api/introspection/scan-array/:table` hoặc `/api/introspection/scan-array-all`. |
| **Giải pháp `F4`** | `centralized-data-service/internal/service/source/scan_service.go`, `internal/service/master/transmuter_gjson_path.go`, `transmuter.go` & `cdc-cms-service/.../mapping_rule_repo_gorm.go` | `ScanArrayFields`, `resolveRuleGJSONValue`, `GetApprovedJSONFieldsByMasterBindingID` | `scan_service.go:302-360, 396`, `transmuter.go:1768` & `introspection_handler.go:382` | Khi Worker quét JSON/Array trong `_raw_data`, khi Transmuter trích xuất dữ liệu phẳng sang Master, và khi chạy `ScanArrayAll`. |

---

### 3.3. Chi tiết Từng File & Từng Dòng Update (Code Demo Đầy đủ)

#### A. Giải pháp cho `F1` — Tách `cdc-cms-web/src/utils/masterScanArray.ts` & Rút gọn `MasterMappingFieldsPage.tsx`

##### Code Demo 1A — [FILE MỚI SRP] `cdc-cms-web/src/utils/masterScanArray.ts`
```ts
import type { MultiShadowBindingInfo } from '../components/master/ShadowSourceBadge';

export function resolveShadowTableForExplodePath(
  binding: MultiShadowBindingInfo | null | undefined,
  shadowRules: any[],
  explodePath: string,
): string {
  const cleaned = (explodePath || '')
    .trim()
    .replace(/^\$\.?/, '')
    .replace(/^after\./, '');
  const rootSeg = (cleaned.split('.')[0] || '').replace(/\[\*\]$/, '').trim().toLowerCase();
  if (rootSeg && Array.isArray(shadowRules)) {
    const matchRule = (r: any, requireApproved: boolean) => {
      if (!r?.shadow_table) return false;
      if (requireApproved && r?.status !== 'approved') return false;
      const sf = String(r.source_field || '').replace(/^after\./, '').toLowerCase();
      const tc = String(r.target_column || '').toLowerCase();
      return sf === rootSeg || tc === rootSeg;
    };
    const matched =
      shadowRules.find((r) => matchRule(r, true)) ||
      shadowRules.find((r) => matchRule(r, false));
    if (matched?.shadow_table) {
      return String(matched.shadow_table);
    }
  }
  return String(binding?.shadow_table || '');
}

export function buildScanArrayFieldOptions(
  shadowRules: any[],
  primaryShadowTable?: string | null,
): { label: string; value: string }[] {
  const seen = new Set<string>();
  return (shadowRules || [])
    .filter((r) => (r?.data_type || '').toUpperCase().includes('JSON') && r?.status === 'approved')
    .map((r) => {
      const hasRename = r.target_column && r.target_column !== r.source_field;
      const baseLabel = hasRename
        ? `${r.target_column} (gốc: ${r.source_field}) (${r.data_type})`
        : `${r.source_field} (${r.data_type})`;
      const isSecondary =
        r.shadow_table && primaryShadowTable && r.shadow_table !== primaryShadowTable;
      return {
        label: isSecondary ? `${baseLabel} — [${r.shadow_table}]` : baseLabel,
        value: `${r.source_field}[*]`,
      };
    })
    .filter((opt) => {
      if (seen.has(opt.value)) return false;
      seen.add(opt.value);
      return true;
    });
}
```

##### Code Demo 1B — Cập nhật `cdc-cms-web/src/components/master/ShadowSourceBadge.tsx:60-98` (Đảm bảo `shadow_table` luôn có mặt trên từng rule)
```tsx
// ==================== BEFORE (ShadowSourceBadge.tsx:60-98) ====================
export async function loadAllBindingShadowRules(binding: MultiShadowBindingInfo): Promise<any[]> {
  const queries: Record<string, unknown>[] = [];
  if (binding.shadow_binding_id) {
    queries.push({ binding_id: binding.shadow_binding_id, page_size: 200 });
  } else if (binding.shadow_table) {
    queries.push({
      shadow_schema: binding.shadow_schema || 'public',
      shadow_table: binding.shadow_table,
      page_size: 200,
    });
  }
  const extra = Array.isArray((binding.spec as any)?.shadow_sources)
    ? (binding.spec as any).shadow_sources
    : [];
  extra.forEach((s: any) => {
    if (!s) return;
    if (s.shadow_binding_id && s.shadow_binding_id !== binding.shadow_binding_id) {
      queries.push({ binding_id: s.shadow_binding_id, page_size: 200 });
    } else if (s.shadow_table && s.shadow_table !== binding.shadow_table) {
      queries.push({
        shadow_schema: s.shadow_schema || binding.shadow_schema || 'public',
        shadow_table: s.shadow_table,
        page_size: 200,
      });
    }
  });
  const all: any[] = [];
  for (const params of queries) {
    try {
      const { data } = await cmsApi.get('/api/v1/mapping-rules', { params });
      if (Array.isArray(data?.data)) {
        all.push(...data.data);
      }
    } catch {
      // ignore error for individual shadow rule query
    }
  }
  return all;
}

// ==================== AFTER (ShadowSourceBadge.tsx:60-102) ====================
export async function loadAllBindingShadowRules(binding: MultiShadowBindingInfo): Promise<any[]> {
  const queries: { params: Record<string, unknown>; table?: string | null }[] = [];
  if (binding.shadow_binding_id) {
    queries.push({ params: { binding_id: binding.shadow_binding_id, page_size: 200 }, table: binding.shadow_table });
  } else if (binding.shadow_table) {
    queries.push({
      params: { shadow_schema: binding.shadow_schema || 'public', shadow_table: binding.shadow_table, page_size: 200 },
      table: binding.shadow_table,
    });
  }
  const extra = Array.isArray((binding.spec as any)?.shadow_sources)
    ? (binding.spec as any).shadow_sources
    : [];
  extra.forEach((s: any) => {
    if (!s) return;
    if (s.shadow_binding_id && s.shadow_binding_id !== binding.shadow_binding_id) {
      queries.push({ params: { binding_id: s.shadow_binding_id, page_size: 200 }, table: s.shadow_table });
    } else if (s.shadow_table && s.shadow_table !== binding.shadow_table) {
      queries.push({
        params: { shadow_schema: s.shadow_schema || binding.shadow_schema || 'public', shadow_table: s.shadow_table, page_size: 200 },
        table: s.shadow_table,
      });
    }
  });
  const all: any[] = [];
  for (const q of queries) {
    try {
      const { data } = await cmsApi.get('/api/v1/mapping-rules', { params: q.params });
      if (Array.isArray(data?.data)) {
        all.push(...data.data.map((r: any) => (r?.shadow_table || !q.table ? r : { ...r, shadow_table: q.table })));
      }
    } catch {
      // ignore error for individual shadow rule query
    }
  }
  return all;
}
```

##### Code Demo 1C — Rút gọn `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:391-426` & `1244-1266` (Giảm ~17 dòng)
```tsx
// ==================== BEFORE (MasterMappingFieldsPage.tsx:391-426 & 1244-1266) ====================
  const handleScanArray = async () => {
    if (!binding?.shadow_table) {
      message.warning('Binding chưa có shadow_table');
      return;
    }
    const path = explodePath.trim();
    if (!path) {
      message.warning('Nhập explode_path (vd: after.items[*])');
      return;
    }
    setScanningArray(true);
    try {
      const { data: res } = await cmsApi.post(
        `/api/introspection/scan-array/${binding.shadow_table}`,
        { explode_path: path, master_binding_id: masterBindingID },
      );
      const newCount = (res.new_rules || []).length;
      const total = Number(res.total_keys ?? 0);
      if (newCount === 0) {
        if (total > 0) {
          message.info(`Đã quét ${total} field trong "${path}" — tất cả đã được map rồi (không có field mới). Muốn map lại thì xoá field cũ trước.`);
        } else {
          message.warning(`Không tìm thấy field nào trong "${path}" — shadow table chưa có data, hoặc field không phải JSON object. Kiểm tra shadow đã ingest data chưa.`);
        }
      } else {
        message.success(`Quét xong: đã tạo ${newCount} mapping rule mới (status pending) — vào bảng để review và Approve DDL.`);
        setFlattenOpen(false);
      }
      setExplodePath('');
      fetchRules();
    } catch (err) {
      message.error(humanizeApiError(err, 'Scan array thất bại'));
    } finally {
      setScanningArray(false);
    }
  };
...
          options={(() => {
            const seen = new Set<string>();
            return shadowRules
              // Chỉ field JSON ĐÃ APPROVED trên shadow (chưa approve thì chưa nên flatten sang master).
              .filter((r) => (r.data_type || '').toUpperCase().includes('JSON') && r.status === 'approved')
              .map((r) => {
                const hasRename = r.target_column && r.target_column !== r.source_field;
                const labelText = hasRename
                  ? `${r.target_column} (gốc: ${r.source_field}) (${r.data_type})`
                  : `${r.source_field} (${r.data_type})`;
                return {
                  label: labelText,
                  value: `${r.source_field}[*]`,
                };
              })
              .filter((opt) => {
                if (seen.has(opt.value)) {
                  return false;
                }
                seen.add(opt.value);
                return true;
              });
          })()}

// ==================== AFTER (MasterMappingFieldsPage.tsx:391-430 & 1244) ====================
  const handleScanArray = async () => {
    const path = explodePath.trim();
    if (!path) {
      message.warning('Nhập explode_path (vd: after.items[*])');
      return;
    }
    const targetShadowTable = resolveShadowTableForExplodePath(binding, shadowRules, path);
    if (!targetShadowTable) {
      message.warning('Binding chưa có shadow_table');
      return;
    }
    setScanningArray(true);
    try {
      const { data: res } = await cmsApi.post(
        `/api/introspection/scan-array/${targetShadowTable}`,
        { explode_path: path, master_binding_id: masterBindingID },
      );
      if (res?.status === 'error' || res?.status === 'skipped') {
        message.warning(String(res.reason || res.error || `Không thể quét "${path}" trên bảng ${targetShadowTable}`));
        return;
      }
      const newCount = (res.new_rules || []).length;
      const total = Number(res.total_keys ?? 0);
      const scannedTbl = res?.table || targetShadowTable;
      if (newCount === 0) {
        if (total > 0) {
          message.info(`Đã quét ${total} field trong "${path}" (${scannedTbl}) — tất cả đã được map rồi (không có field mới). Muốn map lại thì xoá field cũ trước.`);
        } else {
          message.warning(`Không tìm thấy field nào trong "${path}" trên bảng "${scannedTbl}" — shadow table chưa có data, hoặc field không phải JSON object.`);
        }
      } else {
        message.success(`Quét xong trên "${scannedTbl}": đã tạo ${newCount} mapping rule mới (status pending) — vào bảng để review và Approve DDL.`);
        setFlattenOpen(false);
      }
      setExplodePath('');
      fetchRules();
    } catch (err) {
      message.error(humanizeApiError(err, 'Scan array thất bại'));
    } finally {
      setScanningArray(false);
    }
  };
...
          options={buildScanArrayFieldOptions(shadowRules, binding?.shadow_table)}
```

---

#### B. Giải pháp cho `F2`, `F3`, `F4` (Phần Scan) — Tách File Mới `scan_array_target_resolver.go` & Rút gọn `scan_service.go` (Giữ nguyên `scan_master_binding_resolver.go` = 33 dòng)

##### Code Demo 2A — [FILE MỚI SRP] `centralized-data-service/internal/service/source/scan_array_target_resolver.go`
```go
package source

import (
	"centralized-data-service/internal/service/master/transmute"
	"centralized-data-service/internal/service/metadata"
	"centralized-data-service/pkgs/sqlutil"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type resolvedShadowScanTarget struct {
	ShadowBindingID int64
	SourceObjectID  int64
	ShadowSchema    string
	ShadowTable     string
}

// cleanLogicalExplodePath returns the logical explode path without leading "$." or "after.",
// its base path (without trailing "[*]" or "."), and the top-level root field name.
func cleanLogicalExplodePath(rawPath string) (logicalExplode, logicalBase, rootField string) {
	p := strings.TrimSpace(rawPath)
	p = strings.TrimPrefix(p, "$")
	p = strings.TrimPrefix(p, ".")
	p = strings.TrimPrefix(p, "after.")
	logicalExplode = p
	logicalBase = strings.TrimSuffix(strings.TrimSuffix(p, "[*]"), ".")
	firstSeg := strings.Split(logicalBase, ".")[0]
	rootField = strings.TrimSuffix(strings.TrimSpace(firstSeg), "[*]")
	return logicalExplode, logicalBase, rootField
}

// parseScanNodeJSON unmarshals a sampled JSON node and automatically unwraps double-encoded JSON strings.
func parseScanNodeJSON(rawStr string) (interface{}, bool) {
	trimmed := strings.TrimSpace(rawStr)
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return nil, false
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, false
	}
	if innerStr, ok := parsed.(string); ok {
		innerTrimmed := strings.TrimSpace(innerStr)
		if (strings.HasPrefix(innerTrimmed, "{") && strings.HasSuffix(innerTrimmed, "}")) ||
			(strings.HasPrefix(innerTrimmed, "[") && strings.HasSuffix(innerTrimmed, "]")) {
			var innerParsed interface{}
			if err := json.Unmarshal([]byte(innerTrimmed), &innerParsed); err == nil {
				return innerParsed, true
			}
		}
		return nil, false
	}
	return parsed, true
}

func (s *ScanService) collectMasterShadowCandidates(ctx context.Context, masterBindingID int64) []resolvedShadowScanTarget {
	if s == nil || s.db == nil || masterBindingID <= 0 {
		return nil
	}
	var mbRow struct {
		ShadowBindingID int64  `gorm:"column:shadow_binding_id"`
		TransformSpec   []byte `gorm:"column:transform_spec"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(shadow_binding_id, 0) AS shadow_binding_id,
		       COALESCE(transform_spec, '{}'::jsonb) AS transform_spec
		  FROM cdc_system.master_binding
		 WHERE id = ? LIMIT 1`, masterBindingID).Scan(&mbRow).Error; err != nil {
		return nil
	}

	var candidates []resolvedShadowScanTarget
	seenID := make(map[int64]bool)
	seenTbl := make(map[string]bool)

	addByBindingID := func(sbID int64) {
		if sbID <= 0 || seenID[sbID] {
			return
		}
		var row resolvedShadowScanTarget
		if err := s.db.WithContext(ctx).Raw(`
			SELECT id AS shadow_binding_id, COALESCE(source_object_id, 0) AS source_object_id,
			       COALESCE(shadow_schema, '') AS shadow_schema, COALESCE(shadow_table, '') AS shadow_table
			  FROM cdc_system.shadow_binding WHERE id = ? LIMIT 1`, sbID).Scan(&row).Error; err == nil && row.ShadowTable != "" {
			seenID[row.ShadowBindingID] = true
			seenTbl[strings.ToLower(row.ShadowSchema+"."+row.ShadowTable)] = true
			candidates = append(candidates, row)
		}
	}

	addByTable := func(schema, table string) {
		table = strings.TrimSpace(table)
		schema = strings.TrimSpace(schema)
		if table == "" {
			return
		}
		var row resolvedShadowScanTarget
		q := `SELECT id AS shadow_binding_id, COALESCE(source_object_id, 0) AS source_object_id,
		             COALESCE(shadow_schema, '') AS shadow_schema, COALESCE(shadow_table, '') AS shadow_table
		        FROM cdc_system.shadow_binding WHERE shadow_table = ?`
		args := []interface{}{table}
		if schema != "" {
			q += ` AND shadow_schema = ?`
			args = append(args, schema)
		}
		q += ` ORDER BY is_active DESC, id ASC LIMIT 1`
		if err := s.db.WithContext(ctx).Raw(q, args...).Scan(&row).Error; err == nil && row.ShadowTable != "" {
			if !seenID[row.ShadowBindingID] {
				seenID[row.ShadowBindingID] = true
				seenTbl[strings.ToLower(row.ShadowSchema+"."+row.ShadowTable)] = true
				candidates = append(candidates, row)
			}
			return
		}
		if schema == "" {
			schema = metadata.ResolveTargetSchema(s.metadataRegistry, table)
		}
		key := strings.ToLower(schema + "." + table)
		if !seenTbl[key] {
			seenTbl[key] = true
			candidates = append(candidates, resolvedShadowScanTarget{
				ShadowSchema: schema,
				ShadowTable:  table,
			})
		}
	}

	addByBindingID(mbRow.ShadowBindingID)
	sources, _ := transmute.ParseShadowSources(mbRow.TransformSpec)
	for _, src := range sources {
		if src.ShadowBindingID > 0 {
			addByBindingID(src.ShadowBindingID)
		} else if src.ShadowTable != "" {
			addByTable(src.ShadowSchema, src.ShadowTable)
		}
	}
	return candidates
}

// resolveShadowTargetForScanArray resolves the exact Shadow (schema, table, shadow_binding_id, source_object_id)
// that owns the target JSON field being flattened, supporting both single-shadow and multi_shadow bindings.
func (s *ScanService) resolveShadowTargetForScanArray(
	ctx context.Context,
	execDB *gorm.DB,
	requestedTable string,
	explodePath string,
	masterBindingID *int64,
) resolvedShadowScanTarget {
	_, _, rootField := cleanLogicalExplodePath(explodePath)
	defaultSchema := metadata.ResolveTargetSchema(s.metadataRegistry, requestedTable)
	fallback := resolvedShadowScanTarget{
		ShadowSchema: defaultSchema,
		ShadowTable:  requestedTable,
	}
	if requestedTable != "" && s != nil && s.db != nil {
		s.db.WithContext(ctx).Raw(`
			SELECT id AS shadow_binding_id, COALESCE(source_object_id, 0) AS source_object_id,
			       COALESCE(NULLIF(TRIM(shadow_schema), ''), ?) AS shadow_schema,
			       COALESCE(NULLIF(TRIM(shadow_table), ''), ?) AS shadow_table
			  FROM cdc_system.shadow_binding
			 WHERE shadow_table = ?
			 ORDER BY is_active DESC, id ASC LIMIT 1`,
			defaultSchema, requestedTable, requestedTable,
		).Scan(&fallback)
	}
	if masterBindingID == nil || *masterBindingID <= 0 {
		return fallback
	}

	candidates := s.collectMasterShadowCandidates(ctx, *masterBindingID)
	if len(candidates) == 0 {
		return fallback
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	// 1. Nếu requestedTable khớp với 1 bảng Secondary và bảng đó có rule/dữ liệu của rootField -> chọn ngay.
	if requestedTable != "" {
		for idx, c := range candidates {
			if idx > 0 && strings.EqualFold(c.ShadowTable, requestedTable) {
				if rootField == "" || s.candidateHasFieldRuleOrData(ctx, execDB, c, rootField) {
					return c
				}
			}
		}
	}

	// 2. Kiểm tra bảng nào trong candidates có mapping_rule_v2 khớp với rootField.
	if rootField != "" {
		for _, c := range candidates {
			if c.ShadowBindingID <= 0 {
				continue
			}
			var cnt int64
			s.db.WithContext(ctx).Raw(`
				SELECT COUNT(1) FROM cdc_system.mapping_rule_v2
				 WHERE shadow_binding_id = ?
				   AND (LOWER(target_column) = LOWER(?) OR LOWER(source_field) = LOWER(?) OR LOWER(source_field) = LOWER('after.' || ?))
				   AND status <> 'rejected'`,
				c.ShadowBindingID, rootField, rootField, rootField,
			).Scan(&cnt)
			if cnt > 0 {
				return c
			}
		}

		// 3. Nếu chưa có trong mapping_rule_v2, kiểm tra trực tiếp bảng Shadow vật lý nào có chứa key rootField trong _raw_data.
		for _, c := range candidates {
			if s.shadowTableHasRawKey(ctx, execDB, c.ShadowSchema, c.ShadowTable, rootField) {
				return c
			}
		}
	}

	// 4. Nếu requestedTable khớp bất kỳ candidate nào trong danh sách của master_binding -> giữ candidate đó.
	for _, c := range candidates {
		if strings.EqualFold(c.ShadowTable, requestedTable) {
			return c
		}
	}

	return candidates[0]
}

func (s *ScanService) candidateHasFieldRuleOrData(ctx context.Context, execDB *gorm.DB, c resolvedShadowScanTarget, rootField string) bool {
	if c.ShadowBindingID > 0 && s.db != nil {
		var cnt int64
		s.db.WithContext(ctx).Raw(`
			SELECT COUNT(1) FROM cdc_system.mapping_rule_v2
			 WHERE shadow_binding_id = ?
			   AND (LOWER(target_column) = LOWER(?) OR LOWER(source_field) = LOWER(?) OR LOWER(source_field) = LOWER('after.' || ?))`,
			c.ShadowBindingID, rootField, rootField, rootField,
		).Scan(&cnt)
		if cnt > 0 {
			return true
		}
	}
	return s.shadowTableHasRawKey(ctx, execDB, c.ShadowSchema, c.ShadowTable, rootField)
}

func (s *ScanService) shadowTableHasRawKey(ctx context.Context, execDB *gorm.DB, schemaName, tableName, rootField string) bool {
	if execDB == nil || tableName == "" || rootField == "" {
		return false
	}
	if !s.TableExistsInSchema(ctx, execDB, schemaName, tableName) || !s.HasColumnInSchema(ctx, execDB, schemaName, tableName, "_raw_data") {
		return false
	}
	var exists bool
	_ = execDB.WithContext(ctx).Raw(fmt.Sprintf(
		`SELECT EXISTS(SELECT 1 FROM %s WHERE _raw_data ? ? OR (_raw_data ? 'after' AND (_raw_data->'after') ? ?) LIMIT 1)`,
		sqlutil.QualifiedTable(schemaName, tableName)),
		rootField, rootField,
	).Scan(&exists).Error
	return exists
}

func (s *ScanService) resolveParentV2IDForScanArray(
	ctx context.Context,
	tx *gorm.DB,
	masterBindingID int64,
	shadowBindingID int64,
	rootField string,
	logicalBasePath string,
) int64 {
	if tx == nil || rootField == "" {
		return 0
	}
	var parentV2ID int64
	if shadowBindingID > 0 {
		_ = tx.WithContext(ctx).Raw(`
			SELECT id FROM cdc_system.mapping_rule_v2
			 WHERE shadow_binding_id = ?
			   AND (LOWER(target_column) = LOWER(?) OR LOWER(source_field) = LOWER(?) OR LOWER(source_field) = LOWER('after.' || ?) OR LOWER(target_column) = LOWER(?))
			 ORDER BY (status = 'approved') DESC, id ASC LIMIT 1`,
			shadowBindingID, rootField, rootField, rootField, logicalBasePath,
		).Scan(&parentV2ID).Error
		if parentV2ID > 0 {
			return parentV2ID
		}
	}
	candidates := s.collectMasterShadowCandidates(ctx, masterBindingID)
	sbIDs := make([]int64, 0, len(candidates))
	for _, c := range candidates {
		if c.ShadowBindingID > 0 {
			sbIDs = append(sbIDs, c.ShadowBindingID)
		}
	}
	if len(sbIDs) > 0 {
		_ = tx.WithContext(ctx).Raw(`
			SELECT id FROM cdc_system.mapping_rule_v2
			 WHERE shadow_binding_id IN (?)
			   AND (LOWER(target_column) = LOWER(?) OR LOWER(source_field) = LOWER(?) OR LOWER(source_field) = LOWER('after.' || ?) OR LOWER(target_column) = LOWER(?))
			 ORDER BY (status = 'approved') DESC, id ASC LIMIT 1`,
			sbIDs, rootField, rootField, rootField, logicalBasePath,
		).Scan(&parentV2ID).Error
	}
	return parentV2ID
}
```

##### Code Demo 2B — Rút gọn `centralized-data-service/internal/service/source/scan_service.go:191-475` (Giảm ~24 dòng)
```go
// ==================== AFTER (scan_service.go:191-255) ====================
func (s *ScanService) ScanArrayFields(ctx context.Context, targetTable, explodePath string, masterBindingID *int64, mode string) (map[string]interface{}, error) {
	if mode == "" {
		mode = "sample"
	}

	execDB := s.db
	if s.shadowDB != nil {
		execDB = s.shadowDB
	}

	logicalExplodePath, logicalBasePath, rootField := cleanLogicalExplodePath(explodePath)
	resolvedTarget := s.resolveShadowTargetForScanArray(ctx, execDB, targetTable, explodePath, masterBindingID)
	if resolvedTarget.ShadowTable != "" {
		targetTable = resolvedTarget.ShadowTable
	}
	schemaName := resolvedTarget.ShadowSchema
	if strings.TrimSpace(schemaName) == "" {
		schemaName = metadata.ResolveTargetSchema(s.metadataRegistry, targetTable)
	}

	if !s.TableExistsInSchema(ctx, execDB, schemaName, targetTable) || !s.HasColumnInSchema(ctx, execDB, schemaName, targetTable, "_raw_data") {
		return map[string]interface{}{
			"status": "skipped",
			"reason": "table or _raw_data column not found",
		}, nil
	}

	binding := struct {
		ID             int64
		SourceObjectID int64
	}{ID: resolvedTarget.ShadowBindingID, SourceObjectID: resolvedTarget.SourceObjectID}
	if binding.ID <= 0 {
		s.db.WithContext(ctx).Table("cdc_system.shadow_binding").
			Select("id, source_object_id").
			Where("shadow_table = ?", targetTable).
			Order("is_active DESC, id ASC").
			Limit(1).Scan(&binding)
	}

	if masterBindingID == nil && (binding.ID > 0 || targetTable != "") {
		masterBindingID = s.resolveMasterBindingForShadow(ctx, binding.ID, targetTable)
	}

	var hasAfter bool
	_ = execDB.WithContext(ctx).Raw(fmt.Sprintf(
		`SELECT EXISTS(SELECT 1 FROM %s WHERE _raw_data ? 'after')`,
		sqlutil.QualifiedTable(schemaName, targetTable)),
	).Scan(&hasAfter).Error

	dbExplodePath := logicalExplodePath
	if hasAfter && !strings.HasPrefix(dbExplodePath, "after.") && dbExplodePath != "after" {
		dbExplodePath = "after." + dbExplodePath
	}

	pgPath, isArrayExplode, err := explodePathToPGPath(dbExplodePath)
	if err != nil {
		return nil, fmt.Errorf("invalid_explode_path: %w", err)
	}
```
Và phần xử lý `pathType`, `masterRows`, `parentV2ID` tại `scan_service.go:294-455`:
```go
	isObject := pathType == "object" || pathType == "array"
	basePath := logicalBasePath
	if basePath == "" {
		basePath = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(explodePath), "[*]"), ".")
	}
	if rootField == "" {
		rootField = basePath
	}
...
		if nodeData.Valid && nodeData.String != "" && nodeData.String != "{}" {
			parsed, ok := parseScanNodeJSON(nodeData.String)
			if !ok {
				continue
			}

			scannedCount++
			isObject = true
			if isArray, ok := parsed.([]interface{}); ok {
				for _, item := range isArray {
					flattenJSONWithTypes("", item, discoveredFields, isArrayExplode)
				}
			} else {
				flattenJSONWithTypes("", parsed, discoveredFields, isArrayExplode)
			}
		}
	}

	delete(discoveredFields, "")
	if pathType == "string" && len(discoveredFields) == 0 {
		return map[string]interface{}{
			"status": "skipped",
			"reason": "Dữ liệu đang là 'string' (stringified JSON).",
		}, nil
	}
...
		if (isObject || isArrayExplode) && masterBindingID != nil {
			objTarget := strings.ReplaceAll(basePath+"."+relPath, ".", "_")
			objTarget = strings.ReplaceAll(objTarget, "[*]", "")

			if existingMaster[objTarget] || existingMaster["after_"+objTarget] {
				continue
			}
			newFields = append(newFields, relPath)
			masterRows = append(masterRows, masterNestedRow{
				target:   objTarget,
				srcField: rootField,
				srcPath:  basePath + "." + relPath,
				dataType: pgType,
			})
		}
...
		if len(masterRows) > 0 && masterBindingID != nil {
			parentV2ID := s.resolveParentV2IDForScanArray(ctx, tx, *masterBindingID, binding.ID, rootField, basePath)
```

---

#### C. Giải pháp cho `F4` (Phần Transmuter & `ScanArrayAll`) — Tách File Mới `transmuter_gjson_path.go`, Rút gọn `transmuter.go` (-13 dòng) & Mở rộng `GetApprovedJSONFieldsByMasterBindingID`

##### Code Demo 3A — [FILE MỚI SRP] `centralized-data-service/internal/service/master/transmuter_gjson_path.go`
```go
package master

import (
	"strings"

	"centralized-data-service/internal/service/master/transmute"
	"github.com/tidwall/gjson"
)

// resolveRuleGJSONValue resolves the effective GJSON path and extracts the value for a mapping rule.
// Supports unwrapped multi_shadow JSON, debezium_after envelopes, legacy "after."-prefixed rules,
// stringified JSON root objects, and leaf-segment fallback.
func resolveRuleGJSONValue(rawStr string, r transmute.MappingRule) (gjson.Result, string) {
	path := r.SourceField
	if r.SourcePath != nil && *r.SourcePath != "" {
		path = *r.SourcePath
	} else if r.SourceFormat == "debezium_after" {
		path = "after." + r.SourceField
	}
	gres := gjson.Get(rawStr, path)
	if !gres.Exists() {
		if !strings.HasPrefix(path, "after.") {
			gres = gjson.Get(rawStr, "after."+path)
		} else {
			gres = gjson.Get(rawStr, strings.TrimPrefix(path, "after."))
		}
	}
	if !gres.Exists() && strings.Contains(path, ".") {
		cleanPath := strings.TrimPrefix(path, "after.")
		if dotIdx := strings.IndexByte(cleanPath, '.'); dotIdx > 0 {
			rootSeg, subPath := cleanPath[:dotIdx], cleanPath[dotIdx+1:]
			rootRes := gjson.Get(rawStr, rootSeg)
			if !rootRes.Exists() {
				rootRes = gjson.Get(rawStr, "after."+rootSeg)
			}
			if rootRes.Type == gjson.String {
				strVal := strings.TrimSpace(rootRes.Str)
				if strings.HasPrefix(strVal, "{") || strings.HasPrefix(strVal, "[") {
					gres = gjson.Get(strVal, subPath)
				}
			}
		}
	}
	if !gres.Exists() && strings.Contains(path, ".") {
		lastSeg := path[strings.LastIndex(path, ".")+1:]
		lastSeg = strings.TrimSuffix(lastSeg, "]")
		if lastSeg != "_id" || path == "_id" || path == "after._id" {
			gres = gjson.Get(rawStr, lastSeg)
		}
	}
	return gres, path
}
```

##### Code Demo 3B — Rút gọn `centralized-data-service/internal/service/master/transmuter.go:1768-1782` (Từ 14 dòng xuống 1 dòng)
```go
// ==================== BEFORE (transmuter.go:1768-1782) ====================
		path := r.SourceField
		if r.SourcePath != nil && *r.SourcePath != "" {
			path = *r.SourcePath
		} else if r.SourceFormat == "debezium_after" {
			path = "after." + r.SourceField
		}
		gres := gjson.Get(rawStr, path)
		if !gres.Exists() && strings.Contains(path, ".") {
			lastSeg := path[strings.LastIndex(path, ".")+1:]
			lastSeg = strings.TrimSuffix(lastSeg, "]")
			if lastSeg != "_id" || path == "_id" || path == "after._id" {
				gres = gjson.Get(rawStr, lastSeg)
			}
		}

// ==================== AFTER (transmuter.go:1768) ====================
		gres, path := resolveRuleGJSONValue(rawStr, r)
```

##### Code Demo 3C — Cập nhật `cdc-cms-service/internal/infra/persistence/master/mapping_rule_repo_gorm.go:359-372` (Tương thích ngược 100%)
```go
// ==================== BEFORE (mapping_rule_repo_gorm.go:359-372) ====================
func (r *mappingRuleRepoGorm) GetApprovedJSONFieldsByMasterBindingID(ctx context.Context, masterBindingID int64) ([]string, error) {
	var fields []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT target_column FROM cdc_system.mapping_rule_v2
		WHERE master_binding_id = ?
		  AND status = 'approved'
		  AND is_active = TRUE
		  AND UPPER(data_type) IN ('JSON', 'JSONB')
	`, masterBindingID).Scan(&fields).Error
	if err != nil {
		return nil, err
	}
	return fields, nil
}

// ==================== AFTER (mapping_rule_repo_gorm.go:359-385) ====================
func (r *mappingRuleRepoGorm) GetApprovedJSONFieldsByMasterBindingID(ctx context.Context, masterBindingID int64) ([]string, error) {
	var primaryShadowID int64
	_ = r.db.WithContext(ctx).Raw(`SELECT COALESCE(shadow_binding_id, 0) FROM cdc_system.master_binding WHERE id = ?`, masterBindingID).Scan(&primaryShadowID).Error
	shadowIDs := collectShadowIDsFromSpec(ctx, r.db, masterBindingID, primaryShadowID)

	var fields []string
	var err error
	if len(shadowIDs) > 0 {
		err = r.db.WithContext(ctx).Raw(`
			SELECT DISTINCT target_column FROM cdc_system.mapping_rule_v2
			WHERE (shadow_binding_id IN (?) OR master_binding_id = ?)
			  AND status = 'approved'
			  AND is_active = TRUE
			  AND UPPER(data_type) IN ('JSON', 'JSONB')
			ORDER BY target_column
		`, shadowIDs, masterBindingID).Scan(&fields).Error
	} else {
		err = r.db.WithContext(ctx).Raw(`
			SELECT DISTINCT target_column FROM cdc_system.mapping_rule_v2
			WHERE master_binding_id = ?
			  AND status = 'approved'
			  AND is_active = TRUE
			  AND UPPER(data_type) IN ('JSON', 'JSONB')
			ORDER BY target_column
		`, masterBindingID).Scan(&fields).Error
	}
	if err != nil {
		return nil, err
	}
	return fields, nil
}
```

---

## 4. Kế hoạch Kiểm thử & Nghiệm thu (Sau khi có `APPROVE`)
1. **Unit & Integration Test (`centralized-data-service` & `cdc-cms-service`):**
   - Viết test kiểm chứng `ScanArrayFields` với Master Binding `multi_shadow` (`primary = transaction_history`, `secondary = transaction_history_payload`):
     - Khi gọi `ScanArrayFields(ctx, "transaction_history", "info[*]", &masterBindingID, "full")` hoặc `ScanArrayFields(ctx, "transaction_history_payload", "info[*]", &masterBindingID, "full")`, hàm tự động định tuyến sang bảng phụ `transaction_history_payload`, bóc tách các trường con trong `info`, tạo `mapping_rule_master` trạng thái `pending` với:
       - `mapping_v2_id` = ID của rule `info` thuộc `transaction_history_payload`,
       - `source_field = "info"`,
       - `source_path = "info.<subField>"`,
       - `target_column = "info_<subField>"`.
   - Kiểm chứng `extractColumns` trích xuất đúng giá trị `info.<subField>` trên cả JSON đã merge (`multi_shadow`) lẫn JSON còn bọc `after` (`copy_1_to_1`) và stringified JSON.
2. **Build & Type-Check Frontend (`cdc-cms-web`):**
   - Chạy `npm run build` (`tsc -b && vite build`) đảm bảo `exit code = 0`.
3. **Line-Budget Guard (`wc -l`):**
   - Đảm bảo `scan_service.go <= 772` dòng, `MasterMappingFieldsPage.tsx <= 1403` dòng, `transmuter.go <= 1888` dòng.

---

## 5. Bổ sung Phản tỉnh & Tự chữa lành (`S5.5 Self-Improvement Loop`)
1. **`CRIT-S5-01` (`scan_array_target_resolver.go:262-266`):** Thay toán tử JSONB `?` trong `execDB.Raw(...)` bằng `jsonb_extract_path(_raw_data, ?) IS NOT NULL OR jsonb_extract_path(_raw_data, 'after', ?) IS NOT NULL` để tránh xung đột với placeholder `?` của GORM.
2. **`CRIT-S5-02` (`scan_array_target_resolver.go:133-147`):** Khi `transform_spec.shadow_sources` không khai báo `shadow_schema`, tự động kế thừa `primarySchema` của bảng chính (`mb.shadow_binding_id`) đồng bộ với `transmuter_multi_shadow.go:187-189`.
3. **`CRIT-S5-03` (`scan_service.go:270-273`):** Bổ sung `AND jsonb_typeof(_raw_data #> ?::text[]) != 'null'` vào câu truy vấn lấy `pathType` để khi các bản ghi đầu tiên trên bảng lớn (`251,242` dòng của `transaction_history_payload`) có giá trị JSON literal `null` (như `"error": null`), truy vấn vẫn tìm đúng dòng có kiểu `"object"` / `"array"` / `"string"`.
4. **`CRIT-S5-04` (`scan_service.go:296-301`):** Giới hạn `limitClause := "LIMIT 5000"` khi `mode == "full"` trên tập dòng khác `null` để tránh quét toàn bộ hàng trăm nghìn dòng JSONB gây vượt ngưỡng timeout 15 giây của NATS RPC (`introspection_handler.go:341`).
