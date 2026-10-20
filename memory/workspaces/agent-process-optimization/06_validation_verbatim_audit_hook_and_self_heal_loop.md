---
tier: T2
suffix: verbatim_audit_hook_and_self_heal_loop
workspace: agent-process-optimization
created_at: 2026-10-20T17:03:15+07:00
---

# BẰNG CHỨNG KIỂM THỬ & RUNTIME (`06_validation_verbatim_audit_hook_and_self_heal_loop.md`)

## 1. Kết quả Unit Tests (`hooks/agy/tests` & `tooling/tests`)

- **Lệnh thực thi:**
  ```bash
  python3 -m unittest discover -s hooks/agy/tests -p 'test_*.py' -v && python3 -m unittest discover -s tooling/tests -p 'test_*.py' -v
  ```
- **Kết quả:**
  - `hooks/agy/tests/test_agy_guard.py`: `Ran 31 tests` -> `OK` (bao gồm `test_29_h4b_blocks_source_edit_after_s5_audit_until_s3_replan_and_new_approve` và `test_31_h6_and_h7_enforce_verbatim_user_audit_prompt_and_first_stop_bounce`).
  - `tooling/tests`: `Ran 32 tests` -> `OK`.

## 2. Kết quả Kiểm thử Đột biến 3 Bản Hỏng (`mutation_gate.py`)

- **Lệnh thực thi:**
  ```bash
  python3 /Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py \
    --workspace agent-process-optimization \
    --suffix verbatim_audit_hook_and_self_heal_loop \
    --mutations /Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/06_mutations_verbatim_audit_hook_and_self_heal_loop.json
  ```
- **Biên bản vật lý:** `06_mutation_summary_verbatim_audit_hook_and_self_heal_loop_20261020T170929.json`
- **Kết quả:** `gate: "PASS"`, `killed_count: 3/3`, `survived_count: 0`, `all_restored_sha256: true` (`d82c3cd3a4956998fb8516a5df78400e9e6fad4585bb2cbf70cd05837d52f302`).

## 3. Kết quả Decision Gate (`DG1`, `DG2`, `DG3`)

- **DG1:** `06_laya_summary_dg1_20261020T164016.json` -> `gate: "PASS"`.
- **DG2:** `06_laya_summary_dg2_20261020T164032.json` -> `gate: "PASS"`.
- **DG3:** `06_laya_summary_dg3_20261020T170931.json` -> `gate: "PASS"`.

