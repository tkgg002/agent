# 01_requirements — Audit CDC QA Process

## Definition of Done

### DoD-1: 5 nhóm tiêu chí được rate L0..L4
- Functional & Correctness (3 tiêu chí)
- Stability & Resilience (4 tiêu chí)
- Performance & Scalability (4 tiêu chí)
- Resource Utilization (2 tiêu chí)
- Metric Monitor (4+ tiêu chí)

### DoD-2: Mỗi tiêu chí có evidence
- Tối thiểu 1 file:line reference cho L≥1.
- Tối thiểu 1 test file reference cho L≥3.
- Tối thiểu 1 metric/dashboard reference cho L≥4.

### DoD-3: Gap analysis có priority
- P0 (blocker production-ready)
- P1 (cần làm trước release)
- P2 (nice-to-have)

### DoD-4: Recommend test harness có code demo
- Mỗi gap P0/P1 → code skeleton hoặc command demo (không apply code, chỉ paste vào report).

### DoD-5: Memory governance
- Workspace files: 00_context, 01_requirements, 02_plan, 05_progress, 06_validation, 07_status_report, 10_gap_analysis, report_*.md
- Append lesson global (nếu audit phát hiện pattern mới).
- Append active_plans Done entry.

## Non-Goals
- KHÔNG fix gap (chỉ identify + recommend).
- KHÔNG triển khai test harness (chỉ thiết kế).
- KHÔNG bench performance thực tế (môi trường không có infra live).
- KHÔNG sửa source code.
