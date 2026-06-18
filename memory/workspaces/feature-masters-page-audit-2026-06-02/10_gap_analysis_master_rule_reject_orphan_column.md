# 10_gap_analysis — Reject master mapping rule: KHÔNG drop cột (orphan column)

**Ngày**: 2026-06-12 · **Agent**: Muscle (Claude-Opus-4.8) · **Trigger**: user hỏi "status pending→approve→reject; approve thì DDL tạo field; reject có delete field ở table không?" (URL masters/export_jobs_test/mappings?binding_id=16). Đây là PHÂN TÍCH (đọc code, không sửa).

## Trả lời ngắn: **KHÔNG. Reject KHÔNG drop cột.** Cột (nếu đã tạo lúc approve) ở lại table như "orphan".

## Bằng chứng (code thật)
1. **FE** `MasterMappingFieldsPage.tsx:248,781-782`: nút "Duyệt"→`handleBatchUpdate('approved')`, "Từ chối"→`handleBatchUpdate('rejected')` — cùng gọi `PUT /api/v1/master-mapping-rules/batch {ids, status}`.
2. **Handler** `master_mapping_rule_handler.go BatchUpdate` (L490):
   - Update: `UPDATE mapping_rule_master SET status=?, is_active?, updated_by?` (chỉ metadata).
   - **CHỈ khi `status=="approved"`** (L501 + L531): gom bindingIDs → gọi `triggerMasterDDL(bid)`.
   - `status=="rejected"`: **chỉ UPDATE status='rejected'**, KHÔNG gom binding, KHÔNG gọi DDL nào.
3. **`triggerMasterDDL`** (L547): publish `cdc.cmd.master-create` — **chỉ ADD COLUMN**.
4. **Master DDL = ADD-only** (`master_ddl_generator.go:363-364` comment chốt): *"DDL chỉ ADD (không DROP) nên cột có thể đã tồn tại từ trước; rule có thể bị tắt active SAU khi cột đã tạo (cột vẫn còn)"*. `child_explode.go:203`: *"leave column for audit; operator can DROP COLUMN manually"*.
5. **Mọi path DROP COLUMN trong hệ** đều KHÔNG do reject kích hoạt:
   - `cdc.cmd.alter-column` (command_handler.go:2986) — operator chủ động alter (rename/alter-type), không phải reject.
   - `approval_service.go:80` RollbackSQL — rollback schema_proposal (tầng shadow, feature khác).
   - `master_ddl_generator.go:335` — dọn cột legacy `_gpay`, không phải reject.

## Vòng đời thực tế
| Trạng thái | Hành vi metadata | Hành vi DDL (cột vật lý) |
|---|---|---|
| **pending** | rule tạo ra `status='pending'` | KHÔNG tạo cột |
| **approve** | `status='approved'` | `triggerMasterDDL`→`cdc.cmd.master-create`→**ADD COLUMN** |
| **reject** | `status='rejected'` | **KHÔNG gì cả** — cột (nếu đã tạo) Ở LẠI |

## Hệ quả (orphan column)
- Chuỗi **pending→approve→reject** ⇒ cột đã ADD lúc approve **vẫn còn vật lý** trong master table, dù rule giờ `rejected`.
- `in_master` đọc cột vật lý THẬT từ dest (master_ddl_generator.go:363) → cột orphan này hiển thị `in_master=true` dù rule rejected. (Đúng định nghĩa "có cột trong table", nhưng rule không còn duyệt.)
- Transmute KHÔNG ghi vào cột này nữa (rule rejected → ngoài `loadRules` approved+active) ⇒ cột tồn tại nhưng không được populate (giá trị giữ nguyên/NULL).

## Lý do thiết kế (vì sao không auto-drop)
- DROP COLUMN = **mất dữ liệu không hồi phục**. Hệ chọn ADD-only + để operator DROP thủ công (an toàn data-first). Reject chỉ là cờ trạng thái, không phá schema.

## Hướng tốt nhất (nếu muốn dọn orphan — CHỜ user quyết, KHÔNG tự làm)
- **Giữ hành vi hiện tại** (reject = không drop) làm mặc định an toàn; bổ sung **1 thao tác operator riêng, tường minh** để DROP cột orphan (reuse `cdc.cmd.alter-column` op=DROP, có xác nhận + guard cột không phải `_*` meta), KHÔNG auto-drop khi reject. Như vậy vừa an toàn vừa dọn được khi chủ đích.
