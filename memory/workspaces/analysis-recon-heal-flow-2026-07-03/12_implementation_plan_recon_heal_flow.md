# Kế Hoạch Triển Khai AI — Phân Tích Luồng Đối Soát & Chữa Lành

## Mục tiêu
- Phân tích kiến trúc toàn bộ luồng Recon & Heal (3 giai đoạn) từ code thực tế
- Tạo tài liệu sequence diagram + bảng mapping kỹ thuật chi tiết

## Phương pháp thực hiện

### Bước 1: Khởi động (Pre-flight)
- Đọc `GEMINI.md`, `lessons.md`
- Đọc workspace memory liên quan: `ReconInteractiveHeal`, `feat-recon-heal-optimization-2026-06-30`
- Đọc project context

### Bước 2: Nghiên cứu code (Research — chỉ đọc, KHÔNG sửa)
- **Worker** (`centralized-data-service`):
  - `recon_handler_run.go`: `HandleReconCheck()`, `HandleReconHeal()` — Giai đoạn 0 + Luồng heal cũ
  - `recon_execute_heal.go`: `HandleExecuteHeal()`, `executeHealSegA()`, `executeHealSegB()` — Giai đoạn 2
  - `recon_heal_v4.go`: `healSegmentA()`, `healSegmentB()`, constants — Luồng heal cũ chi tiết
  - `server_setup.go`: NATS subscription registration
- **API Gateway** (`cdc-cms-service`):
  - `reconciliation_handler_commands.go`: `TriggerCheck()`, `TriggerCheckAll()`, `TriggerPrune()`
  - `reconciliation_handler_heal.go`: `TriggerHeal()`
  - `reconciliation_handler_execute_heal.go`: `TriggerExecuteHeal()`, `GetUnhealedReports()`
  - `recon_async.go`: Command definitions (`ReconCheckCommand`, `ReconHealCommand`, `ExecuteHealCommand`)
  - `router.go`: Route registration
- **Frontend** (`cdc-cms-web`):
  - `DataIntegrity.tsx`: `openHeal()`, `openExecuteHeal()`, modal rendering
  - `ExecuteHealModal.tsx`: Execute heal modal component
  - `useReconStatus.ts`: Mutations (`useHealMutation`, `useExecuteHealMutation`)

### Bước 3: Tổng hợp tài liệu
- Tạo `13_analysis_recon_heal_flow.md` với:
  - Sơ đồ kiến trúc tổng quan (Mermaid graph)
  - Sequence diagram cho từng giai đoạn (Mermaid sequenceDiagram)
  - Bảng mapping kỹ thuật (File → Function → NATS Subject)
  - So sánh luồng cũ vs mới
  - Known issues / TODOs
  - Schema table `cdc_reconciliation_report`

### Bước 4: Đồng bộ workspace (Post-flight)
- Tạo workspace folder theo naming convention
- Tạo đủ bộ docs tối thiểu: `00_context.md`, `05_progress.md`, `12_implementation_plan_*.md`, `13_analysis_*.md`
- Rà soát lessons cuối phiên

## Skills sử dụng
1. Brain (Architect) — phân tích kiến trúc
2. Context Manager — đọc workspace memory
3. Code Review — đối chiếu code đa repo

---

## Revision Log

| Thời gian | Thay đổi |
|---|---|
| 2026-07-03 16:53 | Bổ sung luồng `full_diff` mode trong `healSegmentA` + sequence diagram `healSegmentB` |
| 2026-07-03 16:56 | Sửa payload Giai đoạn 0 (Recon Check): route `/check` không phải `/run`, data flow 3 tầng FE→API→Worker, phân biệt wire payload vs reserved fields |
| 2026-07-03 17:01 | Sửa payload Background Heal: FE chỉ gửi `{table,segment}`, API map 2 field, Worker có reserved fields không được expose. Phát hiện GAP kiến trúc |
| 2026-07-06 10:28 | Rà soát 5 rủi ro vận hành: Race Condition (🔴), OOM SegA (🟡), Partial Failure (🟡), Query Logic (🟢 ĐÃ VÁ), Safety Gate (🔴). Chi tiết tại `10_gap_analysis.md` |

## Phát hiện GAP Kiến trúc

> [!IMPORTANT]
> **GAP: full_diff mode không expose qua API Gateway**
> - **Hiện trạng:** Worker (`recon_heal_v4.go:293`) đã implement đầy đủ nhánh `full_diff` với `TimeBoundedDiffMissingFromShadow()`, hỗ trợ quét time-range tối đa 30 ngày.
> - **Vấn đề:** `ReconHealCommand` struct (`recon_async.go:18`) chỉ có 2 field `Table` + `Segment`. Không có `mode`, `start_time`, `end_time`, `lookback`.
> - **Hệ quả:** FE **KHÔNG THỂ** trigger `full_diff` mode qua UI. Muốn dùng phải gửi NATS message trực tiếp.
> - **Đề xuất:** Bổ sung fields `Mode`, `StartTime`, `EndTime`, `Lookback` vào `ReconHealCommand` và `TriggerHeal` handler.

## Rủi Ro Vận Hành (Gap Analysis)

> Chi tiết đầy đủ: [10_gap_analysis.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/analysis-recon-heal-flow-2026-07-03/10_gap_analysis.md)

| # | Rủi ro | Mức độ | Hành động |
|---|---|---|---|
| 1 | Race Condition — 2 luồng heal cùng table | 🔴 Cao | Thêm `FOR UPDATE SKIP LOCKED` hoặc cột `heal_status` |
| 2 | OOM SegA — MongoDB `$in` chưa chunk | 🟡 Trung bình | Chunk IDs trước khi gọi `FetchAndWriteByIDs()` |
| 3 | Partial Failure — Worker crash giữa chừng | 🟡 Trung bình | Chấp nhận idempotent hoặc cleanup JSONB |
| 4 | Query Unhealed trả report "sạch" | 🟢 ĐÃ VÁ | Không cần — đã có guard `count > 0` |
| 5 | Interactive Heal thiếu Safety Gate | 🔴 Cao | Thêm threshold check + `force_heal` flag |
