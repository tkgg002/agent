# 00_context — Source Provisioning Mode (Auto vs Manual)

## 1. Bối cảnh dự án
`centralized-data-service` (CDC worker) sau khi xong **Track D Hardening** (P1+P2+P3+P4 + 045 + 046 schema-drift sweep) đã đạt trạng thái: source MongoDB/PostgreSQL → Kafka/Debezium → Shadow → Master chạy thông end-to-end trên 4 PG container isolation. Boot xanh, không còn lỗi `42703`.

## 2. Vấn đề hiện tại
Flow đăng ký 1 source mới hiện gồm **6+ bước rời rạc**, mỗi bước là 1 NATS command độc lập, KHÔNG có orchestrator điều phối:

| # | NATS subject hiện tại | Việc | Trạng thái lưu ở đâu |
|---|----------------------|------|----------------------|
| 1 | `cdc.cmd.scan-fields` | Scan field metadata source | `source_object_registry.profile_status` (`pending`→`profiled`) |
| 2 | `cdc.cmd.scan-source` | Tạo `source_object_registry` row | (insert mới) |
| 3 | (manual/manifest) | Tạo `shadow_binding` + DDL shadow table | `shadow_binding.ddl_status` |
| 4 | `cdc.cmd.discover` | Sinh `mapping_rule_v2` từ shadow | `mapping_rule_v2.status` |
| 5 | (manual API) | Phê duyệt master schema | `master_binding.schema_status` (`pending`→`approved`) |
| 6 | (config DB) | Bật `transmute_schedule.is_enabled` | `transmute_schedule.last_status` |

**Thiếu**:
- Không có cột nào nói "source này đang ở bước thứ mấy của flow tổng".
- Operator phải biết command/API nào gọi tiếp theo — không có UI flow.
- Không có lựa chọn "tự chạy hết" — mỗi lần thêm source phải gõ ≥6 command bằng tay, dễ sót / sai thứ tự.
- Không có audit log "ai click bước nào lúc nào".
- Khi 1 bước fail, không có cơ chế retry chuẩn — operator phải đọc log đoán.

## 3. Yêu cầu user (nguyên văn)
> "tao muốn có 1 cái trạng thái khi thêm sources, check auto thì nó mới chạy auto các bước tiếp theo của flow. ko thì manager cms vào click action từng phần. luôn đảm bảo 2 trạng thái auto, manual đều đc kiểm soat thật kỹ từng bước."

→ Trích yêu cầu chốt:
- **Mode toggle** ngay khi tạo source: `auto` hoặc `manual`.
- **Auto mode**: orchestrator tự bước tiếp khi bước trước thành công.
- **Manual mode**: dừng tại mỗi bước, manager CMS click action để advance.
- **Cả 2 mode đều phải kiểm soát kỹ từng bước** — đây là điểm khác biệt với "auto = cứ chạy đại". Có nghĩa: mode auto vẫn audit từng transition, vẫn pause được, vẫn rollback được.

## 4. Constraint kế thừa
- Không phá kiến trúc hiện có (`SourceObjectRegistry`, `ShadowBinding`, `MasterBinding`, `MappingRuleV2`, `TransmuteSchedule`).
- Idempotent migration (rule §11 immutable + ADD COLUMN IF NOT EXISTS pattern).
- Không Brain-touch source code (CLAUDE.md §12) — Brain plan, Muscle thực thi, sau khi user duyệt plan.
- Tuân thủ event-driven pattern đã chốt ở P4 (D-39.A): không direct UPDATE cross-domain table; mỗi bước emit `cdc.evt.X.completed`, monitor consume → UPDATE state.

## 5. Out of scope (giai đoạn 1)
- CMS frontend UI — chỉ define API contract, FE workspace riêng (`feature-cms-fe-overhaul/` đã có).
- Bulk import (multiple sources cùng lúc) — phase 2.
- Schema rejection workflow chi tiết — đã có `master_binding.rejection_reason`, không mở rộng.
- Cross-source dependency (source A phải xong rồi source B mới chạy) — không cần ở phase này.

## 6. Định nghĩa thành công (Definition of Done)
1. Tạo source mới qua API → có thể chọn `provisioning_mode` ('auto'|'manual'); default = 'manual' (an toàn).
2. Mode `auto` → từ `draft` → `running` không cần click thêm; mỗi transition log JSONB; tổng < 30s cho source size demo.
3. Mode `manual` → dừng tại mỗi state; CMS gọi `POST /api/cms/sources/:id/provisioning/advance` để bước tiếp; mỗi click cũng log JSONB.
4. Cả 2 mode đều có `pause`, `resume`, `retry`, `archive` action.
5. Idempotent: replay event = no-op (WHERE state guard).
6. Audit JSONB log đầy đủ: `[{step, from_state, to_state, actor, timestamp, success, error}]`.
7. Migration mới (047_*.sql) thêm 4 cột vào `source_object_registry`, idempotent re-run.
8. Worker boot xanh, không regression P1..P4.
