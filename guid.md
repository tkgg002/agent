# 1. Biên dịch lại INDEX.md + topics/01..05_*.md + tripwires.json kèm kiểm tra Global Hygiene
python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py

# 2. Kiểm định bề mặt cấu hình Harness² (GEMINI.md + OPERATOR_MAP.md <= 24KB, compile seluruh .py)
python3 /Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py validate-surface

# 3. Chạy bảng đo KPI Governance tổng thể
bash /Users/trainguyen/Documents/work/agent/tooling/governance_metrics.sh