# 🧭 LESSONS INDEX (3-Tier Memory Map)
> **Tổng số bài học:** `101` | **Nguồn gốc:** `memory/global/lessons.md` (Read-Only)
> **Cách dùng tại S0 Boot (tiết kiệm token):** (1) Đọc bảng dưới; (2) mở ĐÚNG MỘT file `topics/0X_*.summary.md` khớp domain của task; (3) chỉ khi cần chi tiết mới trích 1 bài theo ID: `grep -A12 'ID: L0xx' topics/0X_*.md`. **CẤM** đọc nguyên `topics/0X_*.md` hay `lessons.md`.

## 1. Bản đồ Chủ đề (Topic Routing)
| Mã | Tier 2 — đọc file này | Số lượng | Khi nào cần đọc |
| :--- | :--- | :---: | :--- |
| `T1` | `topics/01_governance_and_workflow.summary.md` | 49 | Governance, Brain/Muscle, Workspace & Process Discipline |
| `T2` | `topics/02_golang_and_concurrency.summary.md` | 15 | Golang, GORM, Concurrency, Redis & Backend Runtime |
| `T3` | `topics/03_database_and_sql.summary.md` | 13 | Database, PostgreSQL, MongoDB & SQL |
| `T4` | `topics/04_pipeline_cdc_and_kafka.summary.md` | 15 | Data Pipeline, CDC, Debezium, Kafka & External APIs |
| `T5` | `topics/05_frontend_and_ui.summary.md` | 9 | Frontend, React, AntD & UI/UX |

## 2. Top Luật Vàng Bất Biến (Core Tripwires)
- **G1 (No Overwrite):** Cấm `Overwrite=true` lên file memory; chỉ append vào `05_progress`, `04_decisions`, `lessons.md`.
- **G2 (Propose-Only & Audit):** Khi Audit hoặc chưa có `APPROVE` của User sau `09_tasks_solution` -> cấm sửa source code.
- **G3 (Laya Gate):** Mọi giải pháp (DG0/DG1), code demo (DG2), audit (DG3) phải qua `laya_gate.py` (`PASS`).
- **G4 (Runtime Exercise):** Build OK != Done. Phải chạy `checks` và `runtime_check` từ `.agent-project.json`.
- **G5 (Clean Teardown & Locking):** Dừng goroutine nền trước khi đóng DB/Redis; TTL lock phải khớp chu kỳ ticker.
- **G6 (Global Lesson Hygiene):** Mọi bài học append vào `lessons.md` phải <= 12 dòng, đủ 5 trường, 100% Global (cấm chứa `/Users/...`).

## 3. Tra cứu một bài học cụ thể
Danh mục `L001`–`L101` nằm trong các file Tier 2 ở mục 1 (mỗi bài 1 dòng kèm Global Pattern + Tags). Khi đã biết ID, lấy nguyên bản 5 trường bằng:
```bash
grep -A12 'ID: L0xx' /Users/trainguyen/Documents/work/agent/memory/global/lessons/topics/*.md
```
