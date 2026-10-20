# Final Adversarial Audit Report: Hợp Nhất Đệ Quy 2 Tầng `Harness²` (`harness2_recursive_integration`)

> **Workspace:** `feature-harness2-recursive-integration-2026-10-20` | **Tier:** `T2` | **Suffix:** `harness2_recursive_integration`
> **Trạng thái Kiểm định:** `PASS 🟢` (Hoàn tất 6 phần phản biện bắt buộc theo Rule #9 S5 & Rule #14 G7/G8)

---

## 1. Line-by-Line Diff & Task Critique (Phản biện từng dòng code, từng file & từng task trên `git diff`)

| Task / File:Line | Thay đổi thực tế | Phản biện (Adversarial Question: Cách phá / Edge case) | Bằng chứng Code / Runtime thật | Kết luận |
| :--- | :--- | :--- | :--- | :---: |
| **Task 1 (`F1`)** — `tooling/orchestrator/harness2_engine.py:441-536, 539-751` | Hiện thực hóa `DomainHarnessLibrary` (`record_winning_harness`), `can_commit_adopted_harness_to_domain_library`, và `RecursiveHarnessEngine.execute_task_recursion` hỗ trợ cả `mode="agg"` (song song) và `mode="seq"` (tuần tự). | Điều gì xảy ra nếu nhiều tiến trình cùng ghi vào `<domain>/INDEX.md` hoặc nếu một ứng viên chưa qua cổng `DG3` / `3-Mutant Gate` cố tình ghi đè vào các file `APPEND_ONLY` (`lessons.md`, `05_progress`)? | Dòng `448-453` chặn cứng `APPEND_ONLY`, `CLEAN_ROOM_MODE_RE`, `CLEAN_ROOM_FORBIDDEN_RE`; dòng `462-467` bắt buộc `dg3_conf >= DECISIVE_CONF` và `mutants_killed >= MIN_MUTANTS`; dòng `517-535` dùng `fcntl.flock(lf, fcntl.LOCK_EX)` + `os.replace()` khi cập nhật `<domain>/INDEX.md`. Đã kiểm chứng trong `test_06_f1_two_level_recursion_agg_and_seq_with_domain_library_and_cache`. | **PASS** 🟢 |
| **Task 2 (`F2`)** — `tooling/orchestrator/harness2_engine.py:188-226, 571-578` | Hiện thực hóa `validate_orthogonal_axis_edit()` phân tách trực giao `INV` vs `DOM`, quy tắc `KEEP_BASE`, và kiểm tra trần `always_on_max`. | Nếu gọi `autotrigger_agent_skills()` trước khi gọi `validate_orthogonal_axis_edit()`, liệu `DOM` Proposer thêm skill mới có bị báo lỗi oan vì `OPERATOR_MAP.md` bị sửa tự động không? | Đã phát hiện và khắc phục ngay tại dòng `571-577`: chạy `validate_orthogonal_axis_edit()` trên cây `proposed` nguyên bản của Proposer **trước**, sau đó mới gọi `autotrigger_agent_skills()` và `validate_agent_surface()`. Đã kiểm chứng bởi Mutant `M2_bypass_orthogonal_inv_axis_fence` (`KILLED`). | **PASS** 🟢 |
| **Task 3 (`F3`)** — `tooling/orchestrator/harness2_engine.py:78-181` | Hiện thực hóa `AgentSurfaceSpec`, `load_agent_surface_tree()`, `validate_agent_surface()`, và `autotrigger_agent_skills()`. | Nếu trong `GEMINI.md` hoặc `skills/*/SKILL.md` gọi lệnh `python3 /Users/.../agent/tooling/x.py` hoặc `python3 agent/tooling/x.py`, regex có nhận diện đúng về đường dẫn tương đối `tooling/x.py` không? | Dòng `122-124` chuẩn hóa cả `/Users/\S+/agent/` và `agent/` về đường dẫn tương đối `tooling/*.py` hoặc `hooks/agy/*.py`, đồng thời dòng `141-145` chạy `compile(src, rel, "exec")` bắt mọi lỗi cú pháp Python. Chạy CLI `validate-surface` trên `/Users/trainguyen/Documents/work/agent` quét 26 file thật đạt `status: "VALIDATED"`, `errors: []`. Mutant `M1_bypass_always_on_budget` bị `KILLED`. | **PASS** 🟢 |
| **Task 4 (`F4`)** — `tooling/orchestrator/harness2_engine.py:233-306` | Hiện thực hóa `assert_pre_flight_clean_room()` và `build_evidence_context_tree()` kèm `EVIDENCE_BOUNDARY_HEADER`. | Nếu đường dẫn cấm trong `forbidden_globs` có dạng `./legacy_router.go` trong khi file nằm ở thư mục con `deliverables/legacy_router.go`, bộ lọc có để lọt không? | Dòng `251-260` chuẩn hóa `clean_pat = pat.strip().lstrip("./")` và kiểm tra đồng thời `fnmatch(rel, clean_pat)`, `fnmatch(rel, f"*/{clean_pat}")`, `rel == clean_pat`, và `rel.endswith(f"/{clean_pat}")`, kết hợp quét `GRADER_VERDICT_MARKERS` (`dòng 262-264`), `SECRET` (`dòng 265-266`) và `contaminated_clean_room_reads` (`dòng 240`). | **PASS** 🟢 |
| **Task 5 (`F5`)** — `tooling/orchestrator/harness2_engine.py:313-345` | Hiện thực hóa `assign_anonymous_candidate_ids()` và `compute_tree_diversity()`. | Nếu `labels` truyền vào theo thứ tự ngẫu nhiên giữa các lần gọi, liệu bảng ánh xạ ẩn danh `A, B, C...` có bị thay đổi không? | Dòng `318-320` sắp xếp `ordered = sorted(labels)` trước khi `random.Random(seed).shuffle(ordered)` với seed lấy từ `SHA-1(f"{task_id}|{k}|{mode}")[:12]`, đảm bảo tính tất định 100%. | **PASS** 🟢 |
| **Task 6 (`F6`)** — `tooling/orchestrator/harness2_engine.py:353-435` | Hiện thực hóa `DeterministicRecursionLedger.assert_identity()`, `call_fingerprint_sha256()`, `atomic_flock_write_json()`, và `atomic_flock_merge_summary()`. | Khi nhiều tiến trình cùng ghi file tạm `.json.tmp`, liệu tên file tạm có thể đụng độ hoặc hỏng dữ liệu nếu không gắn PID không? | Dòng `407` và `429` đặt tên file tạm kèm `os.getpid()` (`.{os.getpid()}.json.tmp`) bên trong vùng khóa độc quyền `fcntl.flock(lf, fcntl.LOCK_EX)` trước khi gọi `os.replace()`. Mutant `M3_corrupt_ledger_budget_identity` bị `KILLED`. | **PASS** 🟢 |

---

## 2. Anti-Smoke-Speculation & Real Runtime Trace Verification (Chống Mockup, Smoke Test & Hallucination)

- **Kiểm tra Production Mock Pollution:**
  - Quét toàn bộ `tooling/orchestrator/__init__.py` và `tooling/orchestrator/harness2_engine.py`: **0 hàm mock/closure giả** trong code production (Hook `H4` kiểm tra tự động và cho phép ghi).
- **Bằng chứng Test, 3-Mutant Gate & Runtime CLI thật:**
  1. **Real CLI Runtime Execution (`validate-surface` trên cây `/Users/trainguyen/Documents/work/agent` thật):**
     - Lệnh: `python3 /Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py validate-surface`
     - Kết quả runtime thực tế: `{"status": "VALIDATED", "agent_root": "/Users/trainguyen/Documents/work/agent", "files_scanned": 26, "always_on_chars": 18091, "always_on_max": 24000, "errors": []}` (`exit code 0`).
  2. **3-Mutant Verification Gate (`mutation_gate.py`):**
     - Lệnh: `python3 /Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py --workspace feature-harness2-recursive-integration-2026-10-20 --suffix harness2_recursive_integration --mutations .../06_mutations_harness2_recursive_integration.json`
     - Kết quả: `gate: "PASS"`, `killed_count: 3/3` (`M1_bypass_always_on_budget`, `M2_bypass_orthogonal_inv_axis_fence`, `M3_corrupt_ledger_budget_identity` đều `KILLED`), `all_restored_sha256: true`, lưu tại `06_mutation_summary_harness2_recursive_integration_20261020T153533.json`.
  3. **Full Regression & Integration Suite:**
     - `python3 -m unittest discover -s tooling/tests -p "test_*.py" -v`: **Ran 31 tests in 1.611s — OK**.
     - `python3 -m unittest discover -s hooks/agy/tests -p "test_*.py" -v`: **Ran 30 tests in 0.149s — OK**.
- **Điểm cần phối hợp User trace runtime value (nếu có):**
  - Toàn bộ bề mặt `/Users/trainguyen/Documents/work/agent` nằm trực tiếp trên máy local và đã được chạy trace thật (`26 files scanned`, `18,091 chars`). Khi tích hợp với daemon gọi LLM trả phí thực tế ở dự án ngoài (`claude -p` / `codex exec`), cần phối hợp với User cung cấp API key môi trường tương ứng.

---

## 3. Concurrency, Timeout/Heartbeat & Data Continuity Audit (Kiểm tra Đa luồng, Timeout & Liên tục dữ liệu)

- **Concurrency / Multi-worker / Batching:**
  - Trong `RecursiveHarnessEngine.execute_task_recursion(mode="agg")` (`harness2_engine.py:625-637`), các nhánh `INV` và `DOM` chạy song song qua `ThreadPoolExecutor(max_workers=max(2, 2 * k))`.
  - Trong `test_05_f6_ledger_identity_and_concurrent_flock_summary` (`test_harness2_engine.py:182-195`), đã kiểm thử thực tế **8 worker song song (`ThreadPoolExecutor(max_workers=8)`)** cùng ghi đồng thời vào `concurrent_summary.json` qua `atomic_flock_merge_summary()` (`fcntl.flock(LOCK_EX)` + `os.replace`): bảo toàn đủ `8/8` task, 0 race condition, 0 mất mát dữ liệu.
- **Timeout / Heartbeat / AckWait:**
  - Mọi tiến trình kiểm thử đột biến trong `mutation_gate.py` đều được bọc `timeout_sec=60` và trap `SIGTERM` để khôi phục `SHA-256` nguyên trạng (`timed_out: false` trên cả 3 mutants).
- **Data Continuity / Watermark / Idempotency:**
  - Sổ cái `DeterministicRecursionLedger` (`harness2_engine.py:368-378`) cưỡng chế tính liên tục và chống trùng lặp nhãn (`dupes = 0`, `R = 4k`, `P = 3k`).
  - Bộ nhớ đệm phiên chạy theo mã băm `call_fingerprint_sha256` (`harness2_engine.py:381-397`) đảm bảo tính idempotency tuyệt đối: khi chạy lại cùng một cấu hình (`test_06_f1_two_level_recursion_agg_and_seq_with_domain_library_and_cache`), `cache_hits == 6/6` phiên (`3k` với `k=2`), và `<domain>/INDEX.md` không bao giờ chèn trùng dòng của cùng một `task_id` (`harness2_engine.py:529-530`).

---

## 4. Anti-Speculation & Honesty Check (Kiểm tra Suy diễn & Báo cáo láo)

- Mọi số liệu trong báo cáo đều đối chiếu 1-1 với `file:line` và kết quả chạy lệnh thật:
  - Số file cấu hình `/agent` được quét bởi `validate-surface`: `26` file, tổng độ dài Always-On: `18,091` ký tự (`<= 24,000`).
  - Số lượng unit/integration test đã chạy: `31` tests trong `tooling/tests` + `30` tests trong `hooks/agy/tests` = `61` tests PASS.
  - Số lượng mutants bị tiêu diệt bởi `mutation_gate.py`: `3/3 KILLED` (`06_mutation_summary_harness2_recursive_integration_20261020T153533.json`).
  - Không có bất kỳ khẳng định nào dựa trên suy đoán (Zero Speculation, Zero Hallucination).

---

## 5. Self-Improvement Loop (Vòng lặp Phản tỉnh & Khắc phục — Lessons Audit)

- Đối chiếu với `memory/global/lessons/INDEX.md`:
  - **Lesson `L001` & `L002` (`topics/01_governance_and_workflow.md:3-18`):** Đã mở đầu tài liệu `13_analysis` và `09_tasks_solution` bằng `Macro Runtime & Integration Topology` và ánh xạ giải pháp 1:1 theo từng Finding `F1 → F6`.
  - **Lesson `L009` (`Skill Context-Bloat`):** Đã hiện thực hóa trực tiếp thành kiểm định tự động trong `validate_agent_surface()` (`always_on_max = 24000` và kiểm tra dòng Trigger cho mọi `SKILL.md`).
  - **Lesson `L010` (`Dedicated Suffix & Separate DG1/DG2/DG3 Claims`):** Đã sử dụng suffix riêng `harness2_recursive_integration` và các file claims tách biệt cho `DG1`, `DG2`, `DG3`.
- Không vi phạm bất kỳ lesson nào trong phiên.

---

## 6. Decision Gate `/laya_router` DG3 Final Verdict

- **Summary file (Post-Edit S5):** [`06_laya_summary_dg3_20261020T153819.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feature-harness2-recursive-integration-2026-10-20/06_laya_summary_dg3_20261020T153819.json) | **Gate:** **`PASS`** 🟢 (`trusted_rounds: ["r3"]`)
- **Chi tiết kết quả các claim:**
  - `A-F3-SURFACE` (`harness2_engine.py:78-85`): **`ACCEPT`** (`r3: YES, conf=0.9374`)
  - `A-F4-MARKERS` (`harness2_engine.py:50-56`): **`ACCEPT`** (`r3: YES, conf=0.7921`)
  - `A-F6-LEDGER` (`harness2_engine.py:368-378`): **`FLAG -> Resolved`** (`r1: CORRECT 0.6782`, `r2: CORRECT 0.2547`, `r3: YES 0.3358`, đã giải trình kèm `REF:` và bằng chứng Mutant `M3 KILLED` qua `laya_gate.py resolve` vào `04_decisions_feature-harness2-recursive-integration-2026-10-20.md`).
  - Controls: `A-K1` (`control_false`), `A-K2` (`control_false`), `A-K3` (`control_true`) đều phân tách chính xác ở vòng `r3`.
