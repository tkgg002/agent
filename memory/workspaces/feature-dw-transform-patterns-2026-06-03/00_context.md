# 00_context.md — DW Transform Patterns (Sync types cho Report/Đối soát/Metabase)

## Workspace
- **Feature**: Catalog các loại sync (1:1, filter, groupby, join, aggregate, flatten JSON) cho tầng Master = Data Warehouse, phục vụ report / đối soát giao dịch / Metabase.
- **Created**: 2026-06-03
- **Status**: 🟡 Active (Gap analysis xong → Design proposal → Chờ User chọn hướng)
- **Khởi nguồn**: mở rộng từ `feature-masters-page-audit-2026-06-02` (câu hỏi User: "loại sync chưa có pattern; các loại 1:1/filter/groupby/join/aggregate là gì, thực thi ở đâu; thêm flatten JSON; giải pháp cho report/đối soát/Metabase").

## Nguồn ground
- Workflow `map-transmute-capabilities` (7 agent, 2026-06-03). Output: `/private/tmp/.../tasks/w7r95m1me.output`.
- Repo: `/Users/trainguyen/Documents/work/data-hub/{centralized-data-service, cdc-cms-service}`.

## Kết luận 1 dòng
Engine transmute hiện tại **chỉ thực thi `copy_1_to_1` (row-level Go)**. Các `transform_type` filter/aggregate/group_by/join/custom_sql **đã khai báo trong schema enum nhưng KHÔNG có runtime** (im lặng hành xử như copy 1:1). Tầng Master = bản sao 1:1 typed, **không có view/materialized view/mart/aggregate**. Không có Metabase ở đâu cả.

## Artifacts
- `10_gap_analysis.md` — capability map chi tiết (cái gì có, cái gì thiếu, evidence file:line).
- `02_plan.md` — kiến trúc phân tầng đề xuất + map từng loại sync → tầng thực thi + 3 option + rollout.
