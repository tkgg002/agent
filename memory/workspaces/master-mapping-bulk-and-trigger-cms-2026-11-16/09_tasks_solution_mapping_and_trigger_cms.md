---
tier: T2
suffix: mapping_and_trigger_cms
workspace: master-mapping-bulk-and-trigger-cms-2026-11-16
---
# 09 Tasks Solution — Giải Pháp Bulk Spec Master Mapping (Task 1) & Quản Trị Controlled Downstream Trigger Trên CMS (Task 2)

## 0. Bảng Tổng Hợp Đối Chiếu Phát Hiện & Đề Xuất (`F1..F3` — Executive Summary)

| Mã | Hạng mục | Hiện trạng (`file:line` / SQL User) | Vấn đề gốc rễ | Giải pháp Tối ưu Duy nhất |
| :-: | :--- | :--- | :--- | :--- |
| **`F0`**<br/>*(DG0 Critique)* | **Đối chiếu chéo Mapping `binding_id=95` vs SQL Trigger `payment_bill` & `payment_bill_payload`** | Danh sách 101 Mapping Rules + 3 khối SQL PL/pgSQL trong prompt của User | **5 lỗi chí mạng sẽ gây nổ runtime `42703` hoặc sai lệch dữ liệu:**<br/>1. Payload trigger đọc `NEW._id` / `OLD._id`, nhưng `cdc_payment_bill` chỉ có cột `id` (`_id -> _id` đang `pending`/`not yet`).<br/>2. Payload trigger đọc `NEW.merchant_info`, `NEW.embed_data`... (`snake_case`), nhưng trên `cdc_payment_bill` 9 cột JSONB đang `pending` + `camelCase` (`merchantInfo`...) và `embedData` đang `camelCase` (Trigger 1 đọc `NEW."embedData"` mâu thuẫn với Trigger 2 đọc `NEW.embed_data`).<br/>3. Bỏ sót hoàn toàn cột #11 `refunded_operation_ids` (`TEXT[]` từ `refundedOperationIds`) ở cả Mapping lẫn Trigger.<br/>4. Gán cứng `source_field_types = '{}'::jsonb` làm mất metadata BSON (trong khi `data-hub` đã có `@type_presence:{...}`), dư hàm `fn_sync_bill_derived_fields()` (đọc `"embedData"` không tồn tại trên `payment_bill`), thiếu `CREATE TRIGGER` cho payload, và tách 2 trigger dễ lỗi thứ tự FK / lệch OCC.<br/>5. Gán `v_temp.is_delete := NEW._deleted` làm **mất toàn bộ cờ xóa mềm `NEW.is_delete = TRUE`** từ MongoDB (`isDelete`) và dùng `DROP TRIGGER` rời làm hở cửa sổ rơi bản tin CDC. | Chuẩn hóa 61 cột `snake_case` trên `cdc_payment_bill` (34 scalar + 1 `refunded_operation_ids JSONB` + 10 nested + 11 payload `JSONB` + 1 `source_field_types` `@type_presence:{...}` + 4 cột hệ thống `_gpay_id/_source_ts/_deleted/_updated_at`) và gộp vào **1 Trigger Function hợp nhất `trg_sync_cdc_to_payment_bill_all()`** đảm bảo đúng thứ tự FK, đồng nhất OCC và bảo toàn soft-delete. |
| **`F1`**<br/>*(Task 1)* | **Cấu hình hàng loạt trên `/masters/:name/mappings`** | [`MasterMappingFieldsPage.tsx:316-367,466-514,777-802`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx#L316-L367)<br/>[`master_mapping_rule_handler.go:313-333`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_mapping_rule_handler.go#L313-L333) | `PUT /batch` hiện chỉ đổi `status` (`approved`/`rejected`). Mọi thao tác đổi `camelCase -> snake_case`, đổi `data_type` (`NUMERIC`, `TIMESTAMPTZ`), gắn `transform_fn` và tạo cột nested `a.b -> x_y` đều bắt gọi API từng dòng 1 + reload toàn bảng. | Bổ sung API **`POST /api/v1/master-mapping-rules/bulk-apply-spec`** (có `dry_run` preview diff) + Modal SRP **`MasterBulkMappingModal.tsx`**: hỗ trợ 1-click **Auto `snake_case` / Bulk Type** VÀ **Dán Spec TSV/JSON** (tự map scalar, tự sinh nested & `@type_presence`, tự stage rename/type cho cột `in_master`, tự reject cột thừa ngoài spec chỉ trong 1 transaction). |
| **`F2`**<br/>*(Task 2)* | **Quản trị & Thực thi Trigger `Master CDC -> Master Business` trên CMS ("Có nên không?")** | [`MasterMappingFieldsPage.tsx:1074-1083`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx#L1074-L1083)<br/>[`master_ddl_handler.go:231-286`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/master_ddl_handler.go#L231-L286) | **KHÔNG NÊN** mở Raw SQL Console tự do (nguy cơ SQLi/RCE, `AccessExclusiveLock` treo bảng, chạy mù).<br/>**RẤT NÊN** quản trị dưới dạng **Controlled Downstream Trigger Governance** gắn theo `master_binding_id`, vì hiện tại chạy tay ngoài DB bị mù trạng thái, không kiểm chéo với Mapping Rules, dễ rớt bản tin khi `DROP TRIGGER`, và bị **mất sạch trigger khi `SwapMasterTable`**. | Bổ sung bảng `cdc_system.master_downstream_triggers` + Component **`MasterTriggerManager.tsx`** (đặt cạnh `TableIndexManager`) đi qua 6 chốt chặn: Whitelist Function/Trigger theo đúng bảng Master, Static Check `NEW.<col>`/`OLD.<col>` khớp `mapping_rule_master`, Dry-Run compile trong `BEGIN ... ROLLBACK`, Zero-Lock `CREATE OR REPLACE FUNCTION` + `SET LOCAL lock_timeout = '5s'`, Live Drift Check (`pg_trigger`), và tự gắn lại khi `SwapMasterTable`. |

---

## 1. [S2.5 Pre-Plan Critique — DG0] 5 Lỗi Chí Mạng Khi Đối Chiếu Mapping `binding_id=95` Với SQL Trigger Của User

Trước khi bàn chuyện công cụ tự động hóa, đối chiếu vật lý giữa **101 Mapping Rules (`binding_id=95`)**, **Đặc tả 48 cột `payment_bill` + 13 cột `payment_bill_payload`** và **Đoạn SQL ở Task 2** cho thấy **5 lỗi chắc chắn nổ hoặc làm sai dữ liệu** nếu chạy nguyên trạng:

1. **Lỗi #1 — Nổ `ERROR: record "new" has no field "_id" (SQLSTATE 42703)` ở `trg_sync_cdc_to_payment_bill_payload()`:**
   - Tại [`master_ddl_generator.go:148-163`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L148-L163), bảng Master PostgreSQL chỉ khởi tạo 4 cột hệ thống (`_gpay_id`, `_source_ts`, `_deleted`, `_updated_at`) và **không tự tạo cột `_id`**.
   - Trên `binding_id=95`, khóa `_id` đã được duyệt sang cột **`id` (`TEXT`, `in master`)**, còn dòng `_id -> _id (INTEGER)` đang `pending / not yet` (và sai kiểu `INTEGER` vì `id` chứa cả UUIDv7).
   - Do đó `OLD._id` và `NEW._id` trong `trg_sync_cdc_to_payment_bill_payload()` không tồn tại $\rightarrow$ phải đổi thành **`OLD.id` / `NEW.id`**, đồng thời **Reject** rule `_id -> _id (INTEGER)`.
2. **Lỗi #2 — Nổ `SQLSTATE 42703` trên 10 cột JSONB Payload do lệch `camelCase` vs `snake_case` & chưa Approve:**
   - Trong `trg_sync_cdc_to_payment_bill_payload()`, SQL đọc `NEW.merchant_info`, `NEW.payment_ids`, `NEW.payment_method`, `NEW.merchant_customer`, `NEW.embed_data`, `NEW.extra_info`, `NEW.routing_info`, `NEW.transaction_parties`, `NEW.device_info`.
   - Nhưng trên `binding_id=95`, 9 cột này đang có tên đích **`camelCase`** (`merchantInfo`, `paymentIds`...) và trạng thái **`pending` (`not yet` in master)**! Riêng `embedData` đã `in master` với tên `"embedData"` (Trigger 1 viết `NEW."embedData"`, nhưng Trigger 2 lại viết `NEW.embed_data` — tự mâu thuẫn ngay giữa 2 hàm!).
   - **Khắc phục:** Đổi toàn bộ 11 cột JSONB trên `cdc_payment_bill` sang `snake_case` (`embed_data`, `merchant_info`, `payments`, `payment_ids`, `payment_method`, `item`, `merchant_customer`, `extra_info`, `routing_info`, `transaction_parties`, `device_info`) và Approve DDL.
3. **Lỗi #3 — Bỏ sót cột #11 `refunded_operation_ids` (`TEXT[]`) ở cả Mapping lẫn Trigger:**
   - Đặc tả `payment_bill` cột #11 quy định `refunded_operation_ids TEXT[]` (nguồn Mongo `refundedOperationIds`, dùng để chặn hoàn tiền trùng).
   - Theo chuẩn kiến trúc `data-hub` (**Lesson `L003`**), ở tầng `cdc_payment_bill` trường mảng được map thành `refunded_operation_ids` kiểu **`JSONB`**, sau đó trong Trigger chuyển an toàn từ mảng `JSONB` sang `TEXT[]`:
     ```sql
     ARRAY(SELECT jsonb_array_elements_text(
         CASE WHEN jsonb_typeof(NEW.refunded_operation_ids) = 'array' THEN NEW.refunded_operation_ids ELSE '[]'::jsonb END
     ))
     ```
4. **Lỗi #4 — Mất `source_field_types`, dư hàm `fn_sync_bill_derived_fields()`, thiếu `CREATE TRIGGER` payload & nguy cơ lệch FK/OCC:**
   - Cột `source_field_types` đang bị gán cứng `'{}'::jsonb` trong khi `MasterMappingFieldsPage.tsx:533-544` và `transmuter.go:1728-1733` đã hỗ trợ sẵn rule `@type_presence:{_id,trackingId,...}` sinh tự động bản đồ kiểu/hiện diện BSON!
   - Hàm số 1 `fn_sync_bill_derived_fields()` đọc `NEW."embedData"` (cột không hề tồn tại trên `payment_bill`!), thiếu `COALESCE(..., false)` và trùng 100% logic với `trg_sync_cdc_to_payment_bill()` $\rightarrow$ **bỏ hoàn toàn hàm số 1**.
   - Việc tách 2 trigger rời cho `payment_bill` và `payment_bill_payload` vừa tốn gấp đôi chi phí gọi PL/pgSQL trên mỗi dòng, vừa thiếu chặn OCC `WHERE` ở bảng payload (khi gói tin CDC cũ đến trễ, `payment_bill` giữ bản mới nhưng `payment_bill_payload` bị đè lùi về bản cũ!), vừa có nguy cơ vi phạm Foreign Key nếu thứ tự bảng chữ cái tên trigger chạy payload trước core.
5. **Lỗi #5 — Hồi sinh nhầm mọi bản ghi đã xóa mềm (`isDelete = true`) & hở cửa sổ mất dữ liệu khi `DROP TRIGGER`:**
   - Dòng `v_temp.is_delete := NEW._deleted;` bỏ qua cột `NEW.is_delete` (nguồn Mongo `isDelete`). Một bill có `isDelete = true` trên Mongo (nhưng chưa xóa vật lý nên `_deleted = false`) sẽ bị ghi thành `is_delete = false` trên `payment_bill`! Phải sửa thành `COALESCE(NEW.is_delete, FALSE) OR COALESCE(NEW._deleted, FALSE)`.
   - Lệnh `DROP TRIGGER IF EXISTS ...; CREATE TRIGGER ...` nếu chạy rời ngoài transaction sẽ làm các dòng CDC ghi vào `cdc_payment_bill` đúng mili-giây ở giữa bị lọt lưới không sang `payment_bill`.

---

## 2. Giải Pháp Tối Ưu Nhất Cho Task 1 (`F1`): Bulk Spec Apply & Smart Normalize Trên Master Mapping

### 2.1. Thiết kế Kiến trúc (Đúng chuẩn `system_overview_and_design_pattern.md` Mục 2.2, 2.3 & 5)
- **Nhóm chức năng & Phân tầng:** Nhóm `master` — mở rộng 100% tương thích ngược (Open-Closed, Blast-Radius lên `CP-1..CP-5` = **0**):
  1. **`cdc-cms-web`:** Tạo component riêng `src/components/master/MasterBulkMappingModal.tsx` (không phình `MasterMappingFieldsPage.tsx`), gắn 1 nút **`Bulk Spec / Chuẩn hóa`** trên thanh công cụ `MasterMappingFieldsPage.tsx:988`.
  2. **`cdc-cms-service`:**
     - Delivery: `internal/api/master/master_mapping_rule_bulk_handler.go` (`POST /api/v1/master-mapping-rules/bulk-apply-spec`).
     - Application Command: `internal/app/commands/master/bulk_apply_mapping_spec.go` (`BulkApplyMappingSpecHandler`).
     - Persistence: `internal/infra/persistence/master/master_mapping_rule_bulk_repo_gorm.go`.
- **Cách vận hành giải quyết triệt để nỗi đau "làm từng cái 1":**
  - **Tab 1 — Chuẩn hóa Nhanh (1-Click cho các dòng đã chọn):**
    - Đổi hàng loạt `target_column` từ `camelCase` sang `snake_case` (`merchantCustomer -> merchant_customer`, `embedData -> embed_data`), tự kiểm tra trùng tên và tự bật cờ staging `pending_rename_from` nếu cột đã `in_master`.
    - Đổi hàng loạt `data_type` / `transform_fn` cho nhiều cột cùng lúc (`amount, paid_amount, refunded_amount -> NUMERIC`).
  - **Tab 2 — Dán Spec Bảng Đích (TSV / JSON — Xong 60+ cột trong 1 lần bấm):**
    - Cho phép dán danh sách dòng định dạng `target_column | data_type | source_expr | transform_fn` (copy trực tiếp từ bảng tài liệu thiết kế!):
      - Nếu `source_expr` là field đơn (`trackingId`, `merchantInfo`, `refundedOperationIds`): khớp tự động với rule `mapping_rule_v2`, đổi `target_column`, `data_type`, bật `status = 'approved'`, `is_active = true`.
      - Nếu `source_expr` là nested path (`merchantInfo.merchantCode`, `transactionParties.payer.merchantCode`, `embedData.collectResend.nextAt`): tự tra `parentRule` ở `mapping_rule_v2` để gắn `mapping_v2_id = parentRule.id`, `source_field = parentField`, `source_path = source_expr` và upsert vào `mapping_rule_master`.
      - Nếu `source_expr` bắt đầu bằng `@type_presence:{...}` hoặc `{...}`: tạo luôn rule JSON Subset (`mapping_v2_id = 0`).
      - Checkbox **`Tự động Reject & tắt Active các rule ngoài Spec`** (`auto_reject_unlisted = true`): tự dọn sạch `__v`, `_id -> _id`, `bizDomain`, `bizType`, `currencies`, `isActive`, `sourceType`, `tokenCreate`... trong cùng 1 transaction!
    - Luôn chạy **`dry_run = true` trước** để hiển thị bảng Preview Diff (`Tạo mới` / `Đổi tên` / `Đổi kiểu` / `Reject`), bấm **Xác nhận Áp dụng** $\rightarrow$ bấm **Approve All DDL** là hoàn tất toàn bộ bảng!

### 2.2. Code Demo 1 — Use-Case `BulkApplyMappingSpecHandler` (`cdc-cms-service/internal/app/commands/master/bulk_apply_mapping_spec.go`)

```go
package master

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"cdc-cms-service/internal/domain/governance"
	"cdc-cms-service/internal/domain/master"
	"cdc-cms-service/pkgs/utils/dbutil"
)

type BulkSpecItem struct {
	TargetColumn string  `json:"target_column"`
	DataType     string  `json:"data_type"`
	SourceExpr   string  `json:"source_expr"` // vd: "merchantTransId", "merchantInfo.id", "@type_presence:{...}"
	TransformFn  *string `json:"transform_fn,omitempty"`
}

type BulkApplyMappingSpecCommand struct {
	MasterBindingID    int64          `json:"master_binding_id"`
	DryRun             bool           `json:"dry_run"`
	AutoSnakeCaseIDs   []int64        `json:"auto_snake_case_ids,omitempty"`
	BulkDataType       *string        `json:"bulk_data_type,omitempty"`
	SpecItems          []BulkSpecItem `json:"spec_items,omitempty"`
	AutoRejectUnlisted bool           `json:"auto_reject_unlisted"`
	Actor              string         `json:"actor"`
}

type BulkDiffEntry struct {
	Action       string `json:"action"` // "create" | "update" | "stage_ddl" | "reject" | "noop"
	RuleID       int64  `json:"rule_id,omitempty"`
	SourceExpr   string `json:"source_expr"`
	OldColumn    string `json:"old_column,omitempty"`
	TargetColumn string `json:"target_column"`
	OldDataType  string `json:"old_data_type,omitempty"`
	DataType     string `json:"data_type"`
}

type BulkApplyMappingSpecResult struct {
	DryRun  bool            `json:"dry_run"`
	Summary map[string]int  `json:"summary"`
	Diffs   []BulkDiffEntry `json:"diffs"`
}

var camelBoundaryRe = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func ToSnakeCaseIdent(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || strings.HasPrefix(trimmed, "_") {
		return trimmed
	}
	snake := camelBoundaryRe.ReplaceAllString(trimmed, "${1}_${2}")
	snake = strings.ReplaceAll(snake, ".", "_")
	return strings.ToLower(snake)
}

type BulkApplyMappingSpecHandler struct {
	repo      master.MasterRuleRepository
	shadowRep governance.MappingRuleRepository
	publisher master.MasterDDLPublisher
}

func NewBulkApplyMappingSpecHandler(
	repo master.MasterRuleRepository,
	shadowRep governance.MappingRuleRepository,
	publisher master.MasterDDLPublisher,
) *BulkApplyMappingSpecHandler {
	return &BulkApplyMappingSpecHandler{repo: repo, shadowRep: shadowRep, publisher: publisher}
}

func (h *BulkApplyMappingSpecHandler) Handle(ctx context.Context, cmd BulkApplyMappingSpecCommand) (*BulkApplyMappingSpecResult, error) {
	existingRules, err := h.repo.List(ctx, governance.MasterRuleFilter{MasterBindingID: cmd.MasterBindingID})
	if err != nil {
		return nil, err
	}
	shadowID, _ := h.repo.GetShadowBindingID(ctx, cmd.MasterBindingID)
	shadowRules, _ := h.shadowRep.ListByShadowBindingID(ctx, shadowID)

	shadowByField := make(map[string]*governance.MappingRule, len(shadowRules))
	for _, sr := range shadowRules {
		shadowByField[sr.SourceField] = sr
	}

	existingBySource := make(map[string]*governance.MasterRule, len(existingRules))
	existingByTarget := make(map[string]*governance.MasterRule, len(existingRules))
	for _, r := range existingRules {
		key := r.SourceField
		if r.SourcePath != nil && *r.SourcePath != "" {
			key = *r.SourcePath
		}
		// Ưu tiên giữ rule đã in_master nếu trùng source_field (như _id -> id vs _id -> _id)
		if prev, ok := existingBySource[key]; !ok || (!prev.InMaster && r.InMaster) {
			existingBySource[key] = r
		}
		existingByTarget[r.TargetColumn] = r
	}

	res := &BulkApplyMappingSpecResult{
		DryRun:  cmd.DryRun,
		Summary: map[string]int{"create": 0, "update": 0, "stage_ddl": 0, "reject": 0, "noop": 0},
	}
	keepRuleIDs := make(map[int64]bool)
	plannedOps := make([]func(ctx context.Context) error, 0, len(cmd.SpecItems))

	for _, item := range cmd.SpecItems {
		targetCol := strings.TrimSpace(item.TargetColumn)
		dt := dbutil.NormalizeMasterDataType(item.DataType)
		srcExpr := strings.TrimSpace(item.SourceExpr)
		if !dbutil.IsValidColumnName(targetCol) || dbutil.IsSystemColumn(targetCol) || !dbutil.IsValidDataType(dt) {
			return nil, fmt.Errorf("spec không hợp lệ tại cột %q (%s)", targetCol, dt)
		}

		matched := existingByTarget[targetCol]
		if matched == nil {
			matched = existingBySource[srcExpr]
		}

		if matched != nil {
			keepRuleIDs[matched.ID] = true
			needRename := matched.TargetColumn != targetCol
			needType := !strings.EqualFold(strings.TrimSpace(matched.DataType), dt)
			needApprove := matched.Status != "approved" || !matched.IsActive
			if !needRename && !needType && !needApprove {
				res.Summary["noop"]++
				continue
			}
			action := "update"
			if matched.InMaster && (needRename || needType) {
				action = "stage_ddl"
			}
			res.Summary[action]++
			res.Diffs = append(res.Diffs, BulkDiffEntry{
				Action: action, RuleID: matched.ID, SourceExpr: srcExpr,
				OldColumn: matched.TargetColumn, TargetColumn: targetCol,
				OldDataType: matched.DataType, DataType: dt,
			})
			ruleCopy := matched
			tfCopy := item.TransformFn
			plannedOps = append(plannedOps, func(txCtx context.Context) error {
				updates := map[string]interface{}{
					"target_column": targetCol,
					"data_type":     dt,
					"status":        "approved",
					"is_active":     true,
					"updated_by":    cmd.Actor,
				}
				if tfCopy != nil {
					updates["transform_fn"] = strings.TrimSpace(*tfCopy)
				}
				if ruleCopy.InMaster && needRename {
					updates["pending_rename_from"] = ruleCopy.TargetColumn
				}
				if ruleCopy.InMaster && needType {
					updates["pending_type_change"] = true
				}
				return h.repo.UpdateColumnMetadata(txCtx, ruleCopy.ID, updates)
			})
			continue
		}

		// Tạo mới rule (Nested a.b.c, JSON Subset @type_presence:{...}, hoặc Scalar mới)
		res.Summary["create"]++
		res.Diffs = append(res.Diffs, BulkDiffEntry{
			Action: "create", SourceExpr: srcExpr, TargetColumn: targetCol, DataType: dt,
		})
		var v2ID int64
		sourceField := srcExpr
		var sourcePath *string
		if !strings.HasPrefix(srcExpr, "@") && !strings.HasPrefix(srcExpr, "{") && strings.Contains(srcExpr, ".") {
			parentField := strings.SplitN(srcExpr, ".", 2)[0]
			sourceField = parentField
			sp := srcExpr
			sourcePath = &sp
			if pRule := shadowByField[parentField]; pRule != nil {
				v2ID = pRule.ID
			}
		} else if sr := shadowByField[srcExpr]; sr != nil {
			v2ID = sr.ID
		}
		tfCopy := item.TransformFn
		plannedOps = append(plannedOps, func(txCtx context.Context) error {
			return h.repo.Save(txCtx, &governance.MasterRule{
				MasterBindingID: cmd.MasterBindingID,
				MappingV2ID:     v2ID,
				TargetColumn:    targetCol,
				SourceField:     sourceField,
				SourcePath:      sourcePath,
				DataType:        dt,
				TransformFn:     tfCopy,
				IsActive:        true,
				Status:          "approved",
				CreatedBy:       &cmd.Actor,
				UpdatedBy:       &cmd.Actor,
			})
		})
	}

	if cmd.AutoRejectUnlisted && len(cmd.SpecItems) > 0 {
		var rejectIDs []int64
		for _, r := range existingRules {
			if !keepRuleIDs[r.ID] && (r.Status != "rejected" || r.IsActive) {
				rejectIDs = append(rejectIDs, r.ID)
				res.Summary["reject"]++
				res.Diffs = append(res.Diffs, BulkDiffEntry{
					Action: "reject", RuleID: r.ID, SourceExpr: r.SourceField,
					OldColumn: r.TargetColumn, TargetColumn: r.TargetColumn, DataType: r.DataType,
				})
			}
		}
		if len(rejectIDs) > 0 {
			inactive := false
			plannedOps = append(plannedOps, func(txCtx context.Context) error {
				return h.repo.BatchUpdateStatus(txCtx, rejectIDs, "rejected", &inactive, cmd.Actor)
			})
		}
	}

	if cmd.DryRun {
		return res, nil
	}
	for _, op := range plannedOps {
		if err := op(ctx); err != nil {
			return nil, err
		}
	}
	if masterTable, err := h.repo.GetMasterTableByBindingID(ctx, cmd.MasterBindingID); err == nil && masterTable != "" {
		_ = h.publisher.PublishMasterCreate(ctx, masterTable, cmd.Actor)
	}
	return res, nil
}
```

---

## 3. Đánh Giá Kiến Trúc & Giải Pháp Tối Ưu Nhất Cho Task 2 (`F2`): Quản Trị Downstream Trigger Trên CMS ("Có Nên Không?")

### 3.1. Phân tích Khách quan: "Có Nên Đưa Lên CMS Không?"

1. **Phần TUYỆT ĐỐI KHÔNG NÊN làm:**
   - **Không bao giờ mở "Ô chạy SQL tự do" (`Execute Raw SQL`) trên CMS!** Nếu cho phép gửi chuỗi SQL bất kỳ qua Worker vào DB Production (`payment_bill`), một câu `DROP` / `TRUNCATE` nhầm hoặc lộ JWT Admin sẽ phá hủy trực tiếp dữ liệu giao dịch thanh toán, hoặc gây treo toàn bộ DB do `AccessExclusiveLock` không có `lock_timeout`.

2. **Phần RẤT NÊN LÀM — "Controlled Master Downstream Trigger Governance" (Gắn chặt vào `master_binding_id`, cạnh `TableIndexManager`):**
   - Trong kiến trúc `data-hub` hiện hữu, mô hình `Shadow -> Master CDC (cdc_<table>) -> Trigger -> Master Business (<table> + <table>_payload)` là **kiến trúc chuẩn bắt buộc** khi bảng Business không mang cột nội bộ `_gpay_id / _source_ts / _deleted / _updated_at` của CDC và có các cột dẫn xuất riêng (`is_atomic`, `payment_channel_ids`...).
   - Hiện tại để dev/ops tự vào DB chạy tay script PL/pgSQL gây ra **4 rủi ro vận hành nghiêm trọng**:
     1. **Mù trạng thái trên Control Plane:** Nhìn vào CMS không ai biết `cdc_payment_bill` đang đẩy tiếp sang `payment_bill` và `payment_bill_payload`, hàm nào đang chạy, phiên bản mấy, đang bật hay tắt.
     2. **Lệch Schema gây nổ batch CDC (`42703`):** Sửa tên cột trên CMS nhưng quên sửa Trigger (hoặc ngược lại như lỗi `NEW._id` và `NEW.merchant_info` ở mục 1).
     3. **Rơi bản tin hoặc treo bảng khi cập nhật Trigger:** Chạy tay `DROP TRIGGER IF EXISTS ...; CREATE TRIGGER ...` vừa đòi `AccessExclusiveLock` không giới hạn thời gian chờ, vừa tạo khoảng trống mili-giây làm lọt bản tin CDC đang ghi liên tục.
     4. **Mất sạch Trigger sau khi `SwapMasterTable`:** Tại [`master_ddl_handler.go:231-286`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/master_ddl_handler.go#L231-L286), khi hoán đổi bảng staging sang bảng chính, bảng mới không có trigger $\rightarrow$ đứt đồng bộ sang `payment_bill` trong im lặng!

### 3.2. Thiết kế Kiến trúc Giải pháp Duy nhất: `MasterTriggerManager` + 6 Chốt Chặn An Toàn

Kế thừa đúng khuôn mẫu chuẩn đang chạy ổn định của `TableIndexManager` ([`TableIndexManager.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/TableIndexManager.tsx) $\rightarrow$ [`introspection_index_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_index_handler.go) $\rightarrow$ NATS RPC $\rightarrow$ Worker), bổ sung:

1. **Bảng Metadata trên Control Plane (`cdc_system.master_downstream_triggers`):**
   - Lưu `id`, `master_binding_id`, `trigger_name`, `function_name`, `target_tables TEXT[]`, `events TEXT[]` (`INSERT`, `UPDATE`, `DELETE`), `function_body TEXT`, `checksum TEXT`, `version INT`, `status` (`draft` | `validated` | `applied` | `disabled` | `failed`), `last_applied_at`, `last_error`, `created_by`, `updated_by` + bảng lịch sử phiên bản `cdc_system.master_downstream_trigger_revisions` để Rollback 1-click!
2. **6 Chốt Chặn Cưỡng Chế An Toàn (Safety Guards):**
   - **Guard 1 (No Raw DDL — Structured Template Only):** User chỉ nhập `function_name`, `trigger_name`, `target_tables` và thân hàm PL/pgSQL (`DECLARE ... BEGIN ... END;`). Hệ thống tự sinh header `CREATE OR REPLACE FUNCTION <schema>.<fn>() RETURNS TRIGGER LANGUAGE plpgsql AS $cdc_fn$ ... $cdc_fn$;` (mặc định `SECURITY INVOKER`, chặn từ khóa nguy hiểm: `DROP TABLE`, `TRUNCATE`, `ALTER SYSTEM`, `COPY`, `GRANT`, `REVOKE`, `SECURITY DEFINER`, `EXECUTE` SQL động). Trigger chỉ được gắn duy nhất lên chính `<master_schema>.<master_table>` của binding đó!
   - **Guard 2 (Static Column Cross-Check với `mapping_rule_master`):** Tự động trích xuất mọi tham chiếu `NEW.<col>` / `OLD.<col>` (kể cả `NEW."col"`) trong thân hàm và đối chiếu với tập `{4 cột hệ thống _gpay_id, _source_ts, _deleted, _updated_at} ∪ {approved + in_master columns của binding}`. Nếu thiếu hoặc đang `pending` (như `NEW._id`, `NEW.merchant_info`), **chặn ngay trên CMS và chỉ rõ dòng/tên cột sai**!
   - **Guard 3 (Transactional Dry-Run Compile trên Worker — `cdc.cmd.master-trigger.validate`):** Worker mở transaction trên PostgreSQL đích: `BEGIN; SET LOCAL lock_timeout = '3s'; CREATE OR REPLACE FUNCTION ...; CREATE OR REPLACE TRIGGER ...; ROLLBACK;` để trình biên dịch PL/pgSQL và catalog thật kiểm tra `%ROWTYPE` của `payment_bill` / `payment_bill_payload` mà không lưu bất kỳ thay đổi nào!
   - **Guard 4 (Zero-Lock Function Hot-Swap & Lock-Timeout Guard — `cdc.cmd.master-trigger.apply`):**
     - Worker kiểm tra `pg_trigger` trước: nếu trigger `<trigger_name>` đã tồn tại, đang `ENABLED` và đã trỏ đúng vào `<function_name>`, Worker **CHỈ chạy `CREATE OR REPLACE FUNCTION`** (chỉ khóa dòng trong `pg_proc`, **0 `AccessExclusiveLock` trên `cdc_payment_bill` và 0 mili-giây gián đoạn CDC!**).
     - Chỉ khi tạo mới lần đầu mới chạy `CREATE OR REPLACE TRIGGER` (PostgreSQL 14+, không bao giờ dùng `DROP TRIGGER`!) bọc trong `BEGIN; SET LOCAL lock_timeout = '5s'; ... COMMIT;`.
   - **Guard 5 (Live Introspection & Drift Detection — `cdc.cmd.master-trigger.list`):** Đọc trực tiếp `pg_trigger` + `pg_proc` từ DB đích, băm SHA-256 nội dung hàm thực tế đối chiếu với `checksum` trên CMS để báo trạng thái thực (`In Sync v3`, `DB Drift`, `Disabled`, `Missing on DB`).
   - **Guard 6 (Swap-Safe Re-attach):** Trong `MasterDDLGenerator.Swap`, sau khi rename bảng staging thành bảng chính, tự động nạp lại các trigger có `status = 'applied'` của `master_binding_id` trong cùng transaction swap!

### 3.3. Code Demo 2 — Bộ Kiểm Định Tĩnh `NEW/OLD` & Thực Thi Zero-Lock Trigger (`cdc-cms-service` & `centralized-data-service`)

```go
// 1. Guard 1 & Guard 2 tại cdc-cms-service/internal/domain/master/trigger_validator.go
package master

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	recordColRefRe   = regexp.MustCompile(`(?i)\b(?:NEW|OLD)\.(?:"([^"]+)"|([a-zA-Z_][a-zA-Z0-9_]*))`)
	forbiddenPLSQLRe = regexp.MustCompile(`(?i)\b(DROP\s+TABLE|TRUNCATE\b|ALTER\s+SYSTEM|SECURITY\s+DEFINER|COPY\s+|GRANT\s+|REVOKE\s+| pg_read_file| dblink)`)
)

func ValidateTriggerPLpgSQLAgainstMasterCols(plpgsqlBody string, approvedInMasterCols map[string]bool) error {
	if loc := forbiddenPLSQLRe.FindString(plpgsqlBody); loc != "" {
		return fmt.Errorf("từ khóa bị cấm trong downstream trigger: %q", strings.TrimSpace(loc))
	}
	allowed := map[string]bool{
		"_gpay_id": true, "_source_ts": true, "_deleted": true, "_updated_at": true,
	}
	for col := range approvedInMasterCols {
		allowed[col] = true
	}

	missingSet := make(map[string]struct{})
	for _, m := range recordColRefRe.FindAllStringSubmatch(plpgsqlBody, -1) {
		col := m[1]
		if col == "" {
			col = strings.ToLower(m[2]) // unquoted identifier trong PG luôn fold về lowercase
		}
		if !allowed[col] {
			missingSet[col] = struct{}{}
		}
	}
	if len(missingSet) > 0 {
		missing := make([]string, 0, len(missingSet))
		for c := range missingSet {
			missing = append(missing, c)
		}
		sort.Strings(missing)
		return fmt.Errorf("trigger tham chiếu cột chưa tồn tại (hoặc chưa in_master) trên bảng Master CDC: %s", strings.Join(missing, ", "))
	}
	return nil
}
```

```go
// 2. Guard 3 & Guard 4 tại centralized-data-service/internal/service/master/master_trigger_manager.go
package master

import (
	"context"
	"fmt"
	"gorm.io/gorm"
)

type ApplyTriggerSpec struct {
	MasterSchema string
	MasterTable  string
	FunctionName string
	TriggerName  string
	PLpgSQLBody  string // chỉ chứa khối DECLARE ... BEGIN ... END;
	DryRun       bool
}

func ApplyMasterDownstreamTriggerSafe(ctx context.Context, destDB *gorm.DB, spec ApplyTriggerSpec) (hotSwappedFnOnly bool, err error) {
	if !ddlIdentRe.MatchString(spec.FunctionName) || !ddlIdentRe.MatchString(spec.TriggerName) {
		return false, fmt.Errorf("invalid function or trigger identifier")
	}
	qualTable := quoteDDLQualified(spec.MasterSchema, spec.MasterTable)
	qualFn := quoteDDLQualified(spec.MasterSchema, spec.FunctionName)

	createFnSQL := fmt.Sprintf(
		"CREATE OR REPLACE FUNCTION %s()\nRETURNS TRIGGER\nLANGUAGE plpgsql\nAS $cdc_trigger_fn$\n%s\n$cdc_trigger_fn$;",
		qualFn, spec.PLpgSQLBody,
	)
	createTrgSQL := fmt.Sprintf(
		"CREATE OR REPLACE TRIGGER %s\nAFTER INSERT OR UPDATE OR DELETE ON %s\nFOR EACH ROW\nEXECUTE FUNCTION %s();",
		quoteDDLIdent(spec.TriggerName), qualTable, qualFn,
	)

	err = destDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL lock_timeout = '5s'").Error; err != nil {
			return err
		}
		if err := tx.Exec(createFnSQL).Error; err != nil {
			return fmt.Errorf("compile function failed: %w", err)
		}

		if spec.DryRun {
			if err := tx.Exec(createTrgSQL).Error; err != nil {
				return fmt.Errorf("validate trigger failed: %w", err)
			}
			// Trả về sentinel để GORM luôn ROLLBACK khi Dry-Run
			return errDryRunRollback
		}

		// Guard 4: Nếu trigger đã tồn tại, đang enabled ('O') và trỏ đúng hàm → KHÔNG đụng pg_trigger
		// để tránh hoàn toàn AccessExclusiveLock trên bảng Master CDC đang nhận tải cao!
		var existsAndEnabled bool
		_ = tx.Raw(`
			SELECT EXISTS (
				SELECT 1
				  FROM pg_trigger t
				  JOIN pg_class c ON c.oid = t.tgrelid
				  JOIN pg_namespace n ON n.oid = c.relnamespace
				  JOIN pg_proc p ON p.oid = t.tgfoid
				 WHERE n.nspname = ? AND c.relname = ?
				   AND t.tgname = ? AND p.proname = ?
				   AND t.tgenabled = 'O' AND NOT t.tgisinternal
			)`, spec.MasterSchema, spec.MasterTable, spec.TriggerName, spec.FunctionName).Scan(&existsAndEnabled).Error

		if existsAndEnabled {
			hotSwappedFnOnly = true
			return nil
		}
		return tx.Exec(createTrgSQL).Error
	})
	if spec.DryRun && err == errDryRunRollback {
		return false, nil
	}
	return hotSwappedFnOnly, err
}
```

---

## 4. Code Demo 3 — Bản SQL Trigger Hợp Nhất Đã Vá Đủ 5 Lỗi Cho `cdc_payment_bill` (Dùng Ngay Được)

Sau khi chuẩn hóa tên cột `snake_case` trên `cdc_payment_bill` (bao gồm thêm `refunded_operation_ids JSONB`, 11 cột payload `snake_case` và `source_field_types JSONB` với chế độ `@type_presence:{...}`), chỉ cần **1 Function + 1 Trigger hợp nhất** dưới đây cho cả `payment_bill` và `payment_bill_payload`:

```sql
CREATE OR REPLACE FUNCTION public.trg_sync_cdc_to_payment_bill_all()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $cdc_trigger_fn$
DECLARE
    v_payments       jsonb;
    v_embed          jsonb;
    v_refund_ops     jsonb;
    v_embed_flow     text;
    v_is_delete      boolean;
    v_channel_ids    text[];
    v_refund_op_ids  text[];
    v_is_atomic      boolean;
    v_is_atomic_coll boolean;
    v_has_unapplied  boolean;
BEGIN
    -- 1. Hard DELETE trên bảng CDC: xóa bảng con payload trước rồi xóa bảng cha payment_bill (an toàn FK)
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.payment_bill_payload WHERE payment_bill_id = OLD.id;
        DELETE FROM public.payment_bill WHERE id = OLD.id;
        RETURN OLD;
    END IF;

    -- Bỏ qua an toàn nếu khóa chính id rỗng/null
    IF NEW.id IS NULL OR NEW.id = '' THEN
        RETURN NEW;
    END IF;

    -- 2. CDC Tombstone (_deleted = TRUE): đánh dấu is_delete = TRUE trên payment_bill và dọn payload
    IF NEW._deleted = TRUE THEN
        DELETE FROM public.payment_bill_payload WHERE payment_bill_id = NEW.id;
        UPDATE public.payment_bill
           SET is_delete = TRUE,
               last_updated_at = COALESCE(NEW.last_updated_at, NEW._updated_at)
         WHERE id = NEW.id;
        RETURN NEW;
    END IF;

    -- 3. Chuẩn hóa cờ xóa mềm (kết hợp cả field nghiệp vụ isDelete từ Mongo lẫn _deleted của CDC)
    v_is_delete := COALESCE(NEW.is_delete, FALSE) OR COALESCE(NEW._deleted, FALSE);

    -- 4. Chuẩn hóa JSONB & tính 4 cột dẫn xuất (DV) + cột mảng refunded_operation_ids (#11)
    v_payments   := CASE WHEN jsonb_typeof(NEW.payments) = 'array' THEN NEW.payments ELSE '[]'::jsonb END;
    v_embed      := CASE WHEN jsonb_typeof(NEW.embed_data) = 'object' THEN NEW.embed_data ELSE '{}'::jsonb END;
    v_refund_ops := CASE WHEN jsonb_typeof(NEW.refunded_operation_ids) = 'array' THEN NEW.refunded_operation_ids ELSE '[]'::jsonb END;
    v_embed_flow := v_embed->>'flow';

    v_refund_op_ids := COALESCE(
        ARRAY(SELECT elem FROM jsonb_array_elements_text(v_refund_ops) AS elem WHERE elem IS NOT NULL),
        '{}'::text[]
    );

    v_channel_ids := COALESCE(
        ARRAY(
            SELECT elem->>'channelID'
              FROM jsonb_array_elements(v_payments) AS elem
             WHERE elem->>'channelID' IS NOT NULL
        ),
        '{}'::text[]
    );

    v_is_atomic := COALESCE((
        v_embed_flow = 'ATOMIC'
        AND EXISTS (
            SELECT 1 FROM jsonb_array_elements(v_payments) AS p
             WHERE p->>'flow' = 'ATOMIC' OR p->>'channelID' = 'WALLET'
        )
    ), FALSE);

    v_is_atomic_coll := COALESCE((
        v_embed_flow = 'ATOMIC'
        AND EXISTS (
            SELECT 1 FROM jsonb_array_elements(v_payments) AS p
             WHERE p->>'flow' = 'ATOMIC' AND p->>'channelID' = 'VIET_QR'
        )
    ), FALSE);

    v_has_unapplied := COALESCE(EXISTS (
        SELECT 1
          FROM jsonb_array_elements(
              CASE WHEN jsonb_typeof(v_embed->'unappliedPayments') = 'array'
                   THEN v_embed->'unappliedPayments'
                   ELSE '[]'::jsonb
              END
          ) AS u
         WHERE u->>'reason' IS NOT NULL
    ), FALSE);

    -- 5. Upsert bảng cha payment_bill (đủ 48 cột chuẩn đặc tả) có chốt chặn OCC thời gian
    INSERT INTO public.payment_bill (
        id, tracking_id, merchant_trans_id, order_id, state, holding_step,
        pos_pay_holding_step, amount, paid_amount, refunded_amount, refunded_operation_ids,
        currency, order_info, merchant_return_url, merchant_cancel_url, link_payment_gateway,
        qrdata, code, payment_code, payment_bill_holding_id_ref, trans_holding_id,
        connector, service_code, api_type, created_by, created_by_user, last_updated_by,
        utm, extension_time, expire_time, completed_at, created_at, last_updated_at,
        is_delete, merchant_info_id, merchant_info_merchant_code, merchant_info_email,
        payer_merchant_code, payee_merchant_code, embed_transaction_type, embed_trans_id,
        embed_collection_trans_id, collect_resend_next_at, collect_resend_gave_up_at,
        is_atomic, is_atomic_collection, payment_channel_ids, has_unapplied_lock_unavailable
    )
    VALUES (
        NEW.id, NEW.tracking_id, NEW.merchant_trans_id, NEW.order_id, NEW.state, NEW.holding_step,
        NEW.pos_pay_holding_step, NEW.amount, NEW.paid_amount, NEW.refunded_amount, v_refund_op_ids,
        NEW.currency, NEW.order_info, NEW.merchant_return_url, NEW.merchant_cancel_url, NEW.link_payment_gateway,
        NEW.qrdata, NEW.code, NEW.payment_code, NEW.payment_bill_holding_id_ref, NEW.trans_holding_id,
        NEW.connector, NEW.service_code, NEW.api_type, NEW.created_by, NEW.created_by_user, NEW.last_updated_by,
        NEW.utm, NEW.extension_time, NEW.expire_time, NEW.completed_at, NEW.created_at, NEW.last_updated_at,
        v_is_delete, NEW.merchant_info_id, NEW.merchant_info_merchant_code, NEW.merchant_info_email,
        NEW.payer_merchant_code, NEW.payee_merchant_code, NEW.embed_transaction_type, NEW.embed_trans_id,
        NEW.embed_collection_trans_id, NEW.collect_resend_next_at, NEW.collect_resend_gave_up_at,
        v_is_atomic, v_is_atomic_coll, v_channel_ids, v_has_unapplied
    )
    ON CONFLICT (id) DO UPDATE SET
        tracking_id                    = EXCLUDED.tracking_id,
        merchant_trans_id              = EXCLUDED.merchant_trans_id,
        order_id                       = EXCLUDED.order_id,
        state                          = EXCLUDED.state,
        holding_step                   = EXCLUDED.holding_step,
        pos_pay_holding_step           = EXCLUDED.pos_pay_holding_step,
        amount                         = EXCLUDED.amount,
        paid_amount                    = EXCLUDED.paid_amount,
        refunded_amount                = EXCLUDED.refunded_amount,
        refunded_operation_ids         = EXCLUDED.refunded_operation_ids,
        currency                       = EXCLUDED.currency,
        order_info                     = EXCLUDED.order_info,
        merchant_return_url            = EXCLUDED.merchant_return_url,
        merchant_cancel_url            = EXCLUDED.merchant_cancel_url,
        link_payment_gateway           = EXCLUDED.link_payment_gateway,
        qrdata                         = EXCLUDED.qrdata,
        code                           = EXCLUDED.code,
        payment_code                   = EXCLUDED.payment_code,
        payment_bill_holding_id_ref    = EXCLUDED.payment_bill_holding_id_ref,
        trans_holding_id               = EXCLUDED.trans_holding_id,
        connector                      = EXCLUDED.connector,
        service_code                   = EXCLUDED.service_code,
        api_type                       = EXCLUDED.api_type,
        created_by                     = EXCLUDED.created_by,
        created_by_user                = EXCLUDED.created_by_user,
        last_updated_by                = EXCLUDED.last_updated_by,
        utm                            = EXCLUDED.utm,
        extension_time                 = EXCLUDED.extension_time,
        expire_time                    = EXCLUDED.expire_time,
        completed_at                   = EXCLUDED.completed_at,
        created_at                     = EXCLUDED.created_at,
        last_updated_at                = EXCLUDED.last_updated_at,
        is_delete                      = EXCLUDED.is_delete,
        merchant_info_id               = EXCLUDED.merchant_info_id,
        merchant_info_merchant_code    = EXCLUDED.merchant_info_merchant_code,
        merchant_info_email            = EXCLUDED.merchant_info_email,
        payer_merchant_code            = EXCLUDED.payer_merchant_code,
        payee_merchant_code            = EXCLUDED.payee_merchant_code,
        embed_transaction_type         = EXCLUDED.embed_transaction_type,
        embed_trans_id                 = EXCLUDED.embed_trans_id,
        embed_collection_trans_id      = EXCLUDED.embed_collection_trans_id,
        collect_resend_next_at         = EXCLUDED.collect_resend_next_at,
        collect_resend_gave_up_at      = EXCLUDED.collect_resend_gave_up_at,
        is_atomic                      = EXCLUDED.is_atomic,
        is_atomic_collection           = EXCLUDED.is_atomic_collection,
        payment_channel_ids            = EXCLUDED.payment_channel_ids,
        has_unapplied_lock_unavailable = EXCLUDED.has_unapplied_lock_unavailable
    WHERE COALESCE(EXCLUDED.last_updated_at, EXCLUDED.created_at, '1970-01-01'::timestamptz)
       >= COALESCE(public.payment_bill.last_updated_at, public.payment_bill.created_at, '1970-01-01'::timestamptz);

    -- 6. Upsert bảng con payment_bill_payload (chỉ ghi khi FOUND ở trên hoặc dòng mới/hợp lệ OCC)
    IF FOUND THEN
        INSERT INTO public.payment_bill_payload (
            payment_bill_id,
            merchant_info, payments, payment_ids, payment_method, item,
            merchant_customer, embed_data, extra_info, routing_info,
            transaction_parties, device_info, source_field_types
        )
        VALUES (
            NEW.id,
            NEW.merchant_info, NEW.payments, NEW.payment_ids, NEW.payment_method, NEW.item,
            NEW.merchant_customer, NEW.embed_data, NEW.extra_info, NEW.routing_info,
            NEW.transaction_parties, NEW.device_info,
            COALESCE(NEW.source_field_types, '{}'::jsonb)
        )
        ON CONFLICT (payment_bill_id) DO UPDATE SET
            merchant_info       = EXCLUDED.merchant_info,
            payments            = EXCLUDED.payments,
            payment_ids         = EXCLUDED.payment_ids,
            payment_method      = EXCLUDED.payment_method,
            item                = EXCLUDED.item,
            merchant_customer   = EXCLUDED.merchant_customer,
            embed_data          = EXCLUDED.embed_data,
            extra_info          = EXCLUDED.extra_info,
            routing_info        = EXCLUDED.routing_info,
            transaction_parties = EXCLUDED.transaction_parties,
            device_info         = EXCLUDED.device_info,
            source_field_types  = EXCLUDED.source_field_types;
    END IF;

    RETURN NEW;
END;
$cdc_trigger_fn$;

-- Dùng CREATE OR REPLACE TRIGGER (PG 14+) trong 1 transaction thay vì DROP TRIGGER rời
BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE OR REPLACE TRIGGER trg_sync_cdc_payment_bill
AFTER INSERT OR UPDATE OR DELETE ON public.cdc_payment_bill
FOR EACH ROW
EXECUTE FUNCTION public.trg_sync_cdc_to_payment_bill_all();
COMMIT;
```

---

## 5. Kiểm Tra 5 Bước Trước Khi Sửa Code (`system_overview_and_design_pattern.md` Mục 5)

1. **Nhóm chức năng & Tầng:**
   - Thuộc nhóm `master` xuyên suốt 3 repo: `cdc-cms-web/src/components/master/{MasterBulkMappingModal,MasterTriggerManager}.tsx` $\rightarrow$ `cdc-cms-service/internal/{api,app/commands,app/queries,domain,infra/persistence}/master/` $\rightarrow$ `centralized-data-service/internal/{handler,service}/master/master_trigger_*.go`.
2. **File mẫu chuẩn (Canonical Pattern Reference):**
   - Luồng Mapping Staging DDL: theo đúng [`update_column_metadata.go:28-92`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/update_column_metadata.go#L28-L92) và [`approve_ddl_all.go:29-48`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/approve_ddl_all.go#L29-L48).
   - Luồng CMS $\leftrightarrow$ Worker RPC quản trị DDL đối tượng phụ thuộc trên bảng Master: theo đúng [`TableIndexManager.tsx:45-109`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/TableIndexManager.tsx#L45-L109) và [`introspection_index_handler.go:22-162`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_index_handler.go#L22-L162).
3. **Đối chiếu 5 Shared Chokepoints (`CP-1..CP-5`) & Ma trận Đa hình 3 Chiều:**
   - **Không chạm** vào `CP-1` (`ResolveSourceRoutes`), `CP-2` (`EventHandler`), `CP-3` (`BatchBuffer`), `CP-4` (`TransmuterModule`), `CP-5` (`ReconCore`).
   - Tại `MasterTriggerManager`: chỉ hiển thị khi `engine_type == 'postgresql'` (ẩn tự động với `mongodb`, `clickhouse`, `mysql`), bảo toàn 100% tương thích ngược cho mọi tuyến khác (**Blast-Radius = 0**).
4. **Kiểm tra Hợp đồng Liên Service:**
   - NATS RPC mới tách biệt hoàn toàn: `cdc.cmd.master-trigger.list`, `cdc.cmd.master-trigger.validate`, `cdc.cmd.master-trigger.apply`, `cdc.cmd.master-trigger.toggle`, không sửa bất kỳ payload NATS hiện hữu nào.
5. **Kế hoạch Kiểm thử sau `APPROVE` (S4/S5):**
   - Unit test cho `BulkApplyMappingSpecHandler` (scalar rename, nested `a.b.c` parent link, `@type_presence`, staging `in_master`, `auto_reject_unlisted`) và `ValidateTriggerPLpgSQLAgainstMasterCols` + integration test transaction Dry-Run / Hot-Swap Function trên PostgreSQL.
   - Bổ sung tích hợp `MasterDDLGenerator.Swap` ([`master_ddl_generator.go:705-719`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L705-L719)) tự động gỡ trigger trên `oldName` và gắn lại trên bảng mới vừa swap.

---

## Phụ Lục A — Spec Chuẩn 56 Rules Cho `binding_id=95` (`cdc_payment_bill`) Để Dán Vào `MasterBulkMappingModal`

> *(Gồm: 34 cột scalar #1–#34 — trong đó #11 `refunded_operation_ids` là `JSONB` theo Lesson `L003`; 10 cột nested #35–#44; 11 cột payload `JSONB` `snake_case`; và 1 cột `source_field_types` `@type_presence:{...}`. Các trường ngoài danh sách như `__v`, `_id -> _id`, `bizDomain`, `bizType`, `currencies`, `isActive`, `sourceType`, `tokenCreate` sẽ tự chuyển sang `rejected` + `is_active=false`).*

```text
id	TEXT	_id
tracking_id	TEXT	trackingId
merchant_trans_id	TEXT	merchantTransId
order_id	TEXT	orderId
state	TEXT	state
holding_step	TEXT	holdingStep
pos_pay_holding_step	TEXT	posPayHoldingStep
amount	NUMERIC	amount
paid_amount	NUMERIC	paidAmount
refunded_amount	NUMERIC	refundedAmount
refunded_operation_ids	JSONB	refundedOperationIds
currency	TEXT	currency
order_info	TEXT	orderInfo
merchant_return_url	TEXT	merchantReturnUrl
merchant_cancel_url	TEXT	merchantCancelUrl
link_payment_gateway	TEXT	linkPaymentGateway
qrdata	TEXT	qrdata
code	TEXT	code
payment_code	TEXT	paymentCode
payment_bill_holding_id_ref	TEXT	paymentBillHoldingIdRef
trans_holding_id	TEXT	transHoldingId
connector	TEXT	connector
service_code	TEXT	serviceCode
api_type	TEXT	apiType
created_by	TEXT	createdBy
created_by_user	TEXT	createdByUser
last_updated_by	TEXT	lastUpdatedBy
utm	TEXT	utm
extension_time	INTEGER	extensionTime
expire_time	TIMESTAMPTZ	expireTime
completed_at	TIMESTAMPTZ	completedAt
created_at	TIMESTAMPTZ	createdAt
last_updated_at	TIMESTAMPTZ	lastUpdatedAt
is_delete	BOOLEAN	isDelete
merchant_info_id	TEXT	merchantInfo.id
merchant_info_merchant_code	TEXT	merchantInfo.merchantCode
merchant_info_email	TEXT	merchantInfo.email
payer_merchant_code	TEXT	transactionParties.payer.merchantCode
payee_merchant_code	TEXT	transactionParties.payee.merchantCode
embed_transaction_type	TEXT	embedData.transactionType
embed_trans_id	TEXT	embedData.transId
embed_collection_trans_id	TEXT	embedData.collectionTransId
collect_resend_next_at	TIMESTAMPTZ	embedData.collectResend.nextAt
collect_resend_gave_up_at	TIMESTAMPTZ	embedData.collectResend.gaveUpAt
merchant_info	JSONB	merchantInfo
payments	JSONB	payments
payment_ids	JSONB	paymentIds
payment_method	JSONB	paymentMethod
item	JSONB	item
merchant_customer	JSONB	merchantCustomer
embed_data	JSONB	embedData
extra_info	JSONB	extraInfo
routing_info	JSONB	routingInfo
transaction_parties	JSONB	transactionParties
device_info	JSONB	deviceInfo
source_field_types	JSONB	@type_presence:{_id,trackingId,merchantTransId,orderId,state,holdingStep,posPayHoldingStep,amount,paidAmount,refundedAmount,refundedOperationIds,currency,orderInfo,merchantReturnUrl,merchantCancelUrl,linkPaymentGateway,qrdata,code,paymentCode,paymentBillHoldingIdRef,transHoldingId,connector,serviceCode,apiType,createdBy,createdByUser,lastUpdatedBy,utm,extensionTime,expireTime,completedAt,createdAt,lastUpdatedAt,isDelete,merchantInfo,payments,paymentIds,paymentMethod,item,merchantCustomer,embedData,extraInfo,routingInfo,transactionParties,deviceInfo,bizDomain,bizType,currencies,isActive,sourceType,tokenCreate}
```
