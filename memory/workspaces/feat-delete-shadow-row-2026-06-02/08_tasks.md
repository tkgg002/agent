# 08_tasks — Delete /shadow Row (Task Checklist)

> Plan-only. Mỗi task chỉ được set `in_progress`/`completed` sau khi user duyệt 02_plan + 03_implementation.

## T1 — NEW `internal/app/commands/delete_source_object_v2.go`
- **Owner**: Muscle
- **Phụ thuộc**: Phase 0 audit (đã xong trong 02_plan)
- **DoD**:
  - File tồn tại với struct `DeleteSourceObjectV2Command` + handler `DeleteSourceObjectV2Handler`.
  - Validate: ID > 0 + reason ≥ 10 char.
  - Handle: SELECT identity → SELECT bindings → TX cleanup legacy + DELETE registry → best-effort DROP ngoài TX.
  - Return JSON `{status, source_object_id, object_code, dropped_tables, skipped_drops}`.
  - `go build ./internal/app/commands/...` EXIT=0.
- **Verify**: `go vet ./...` + đọc lại file đối chiếu 03_implementation.

## T2 — Wire CQRS bus
- **Owner**: Muscle
- **Phụ thuộc**: T1
- **DoD**:
  - `internal/server/server.go` có dòng `cmdBus.RegisterSync("source.delete-v2", commands.NewDeleteSourceObjectV2Handler(db, logger))` ngay sau dòng `source.update-v2`.
  - `go build ./...` EXIT=0.

## T3 — EDIT `internal/api/source_objects_handler.go`
- **Owner**: Muscle
- **Phụ thuộc**: T2
- **DoD**:
  - Constructor `NewSourceObjectsHandler` thêm param `bus messaging.CommandBus`.
  - Struct field `bus` thêm vào.
  - Method `Delete(c *fiber.Ctx) error` thêm vào (snippet trong 03_implementation §3.2).
  - Imports thêm: `errors`, `cdc-cms-service/internal/app/commands`, `cdc-cms-service/internal/messaging`, `cdc-cms-service/internal/middleware`.
- **Verify**: `go build ./internal/api/...` EXIT=0.

## T4 — EDIT `internal/server/server.go` constructor call
- **Owner**: Muscle
- **Phụ thuộc**: T3
- **DoD**:
  - Tại nơi gọi `api.NewSourceObjectsHandler(db, logger, listSourceObjectsQ, mappingCxQ)` (grep để tìm dòng chính xác), pass thêm `cmdBus` làm arg thứ 3.
  - `go build ./...` EXIT=0.

## T5 — EDIT `internal/router/router.go` mount DELETE
- **Owner**: Muscle
- **Phụ thuộc**: T3 (handler tồn tại)
- **DoD**:
  - Block manual mount thêm vào sau dòng 322 (sau `shared.Get("/v1/shadow-bindings", ...)`).
  - Pattern y hệt block `/v1/system/connectors/:name` (dòng 211-215).
  - `go build ./...` EXIT=0.

## T6 — EDIT `cdc-cms-web/src/pages/TableRegistry.tsx`
- **Owner**: Muscle
- **Phụ thuộc**: T5 (BE endpoint sẵn sàng)
- **DoD**:
  - Import `DeleteOutlined` thêm vào dòng 3.
  - State `deletePending` + `deleting` + handler `handleDelete` thêm vào sau dòng 308.
  - Button `Xoá` thêm vào cell `Thao tác` (dòng 856-895), danger + tooltip.
  - 1 `ConfirmDestructiveModal` thêm vào cuối JSX (trước `</div>` cuối cùng).
  - `npx tsc --noEmit -p tsconfig.app.json` EXIT=0.

## T7 — Verify + Report
- **Owner**: Muscle
- **Phụ thuộc**: T6
- **DoD**:
  - `cd data-hub/cdc-cms-service && go build ./...` EXIT=0.
  - `cd data-hub/cdc-cms-web && npx tsc --noEmit -p tsconfig.app.json` EXIT=0.
  - Smoke test theo 02_plan §Phase 4.2:
    - Tạo source_object test.
    - Click "Xoá" + nhập reason ≥10 → confirm.
    - Verify 7 acceptance criteria A1-A7 ở 01_requirements.md.
  - Tạo `report_2026-06-02_delete-shadow-row.md`:
    - Bảng "Files changed" với từng file + LOC delta thực tế.
    - Output `go build` + `tsc` (EXIT code).
    - Smoke test result (mỗi A1-A7 PASS/FAIL với evidence).
    - Skipped drops nếu có (kèm lý do).

## Anti-tasks (KHÔNG làm)
- ❌ Không tạo migration mới — schema đã có FK CASCADE sẵn từ 030/031/032/033/034.
- ❌ Không sửa worker — replay-safe đã được handle ở worker side.
- ❌ Không thêm middleware mới — destructiveChain đã đủ.
- ❌ Không touch `ConfirmDestructiveModal` — component đã đủ feature (reason ≥ 10, danger, loading).
- ❌ Không refactor `SourceObjectsHandler` ngoài việc thêm `bus` + `Delete` method (Simplicity First §6).

## Escalation
- Nếu T1 hoặc T3 build fail > 3 lần → dừng, ghi entry vào `05_progress.md`, báo Brain re-plan.
- Nếu smoke test A2 fail (re-register vẫn 23505) → có FK/constraint chưa cleanup → grep schema migration mới phát hiện, không tự "DROP CONSTRAINT" mà escalate.
