---
name: laya-router
description: >-
  Laya Universal Decision Engine & Decision Gate (/laya_router) — mandatory decision
  verification for technical solutions (DG1), code demos (DG2), and audits/diff reviews
  (DG3) using Mode A (calibrated claim verification with controls) and Mode B (Agentic
  Debate recursive fencing with false-variant controls).
---

# Laya Router & Decision Gate Skill (`/laya_router`)

> **Quy tắc cốt lõi:** Rule #9 (Master Workflow), Rule #10 (Decision Gate `DG1`, `DG2`, `DG3`), ADR-04, ADR-08.
> **Công cụ thực thi:** `/Users/trainguyen/Documents/work/agent/tooling/laya_gate.py`
> **Server Laya:** FastAPI `http://127.0.0.1:8200/v1/predict`
> **Lệnh khởi động Server (khi DOWN):**
> `/Users/trainguyen/.venv/bin/python3 /Users/trainguyen/.agent/skills/laya_router/api.py`
> *(Lưu ý: Cold start nạp model ~800MB mất ~30s; trong lúc nạp server trả HTTP 503 — `laya_gate.py` tự động chờ/retry).*

## 1. Ba Cổng Quyết Định Bắt Buộc Trong Master Workflow

| Gate | Thời điểm chạy | Chế độ | Các Claim bắt buộc |
|------|----------------|--------|--------------------|
| **DG1 — Giải pháp** | Trước khi trình `09_tasks_solution_*.md` (cuối S3) | Mode A (`run --stage dg1`) | `S-RC` (Root cause tại `REF:`), `S-FIX` (Thay đổi loại bỏ lỗi), `S-IMPACT` (Caller/consumer không vỡ), `S-PATTERN` (Đúng pattern dự án) |
| **DG2 — Code Demo** | Cho các khối code demo trong `09_tasks_solution_*.md` | Mode A (`run --stage dg2`) | `D-BEHAV` (Hành vi chính), `D-EDGE` (Xử lý biên/lỗi), `D-API` (Chỉ gọi hàm/field có thật tại `REF:`) |
| **DG3 — Audit / Self-Audit** | Khi Audit theo yêu cầu User hoặc ở chặng `S5 SELF-AUDIT` | Mode A (`run --stage dg3`) & Mode B (`review`) | `A-F` (Mỗi finding có `REF:`), `A-R` (Mỗi requirement `R1..Rn` đạt tại `REF:`), `A-SCOPE` (Không vượt phạm vi duyệt) + Review `git diff -U30` |

## 2. Chế độ A — Kiểm chứng Claim Có Hiệu chỉnh Control (`laya_gate.py run`)

### Cấu trúc file `claims.json`
Mỗi batch **BẮT BUỘC** có:
- Các claim thật (`kind: "solution" | "demo" | "audit"`), mỗi claim là **1 câu khẳng định đơn ý**, kèm `REF: <absolute_or_repo_path>:<start>-<end>` và các dòng code trích nguyên văn từ file thật.
- **≥ 2 `control_false`** (đảo ngược 1 chi tiết của claim thật: đổi tên hàm, đảo điều kiện, đổi giá trị trả về).
- **≥ 1 `control_true`** (khẳng định hiển nhiên khớp nguyên văn dòng code trích dẫn).

```json
[
  {
    "id": "S1",
    "kind": "solution",
    "basis": "code",
    "flow": "Mô tả luồng hàm",
    "claim": "Hàm X trả về lỗi khi tham số Y rỗng.",
    "evidence": "REF: /abs/path/to/file.py:10-14\ndef func_x(y):\n    if not y:\n        raise ValueError('empty')"
  },
  {
    "id": "K1",
    "kind": "control_false",
    "claim": "Hàm X trả về True khi tham số Y rỗng.",
    "evidence": "REF: /abs/path/to/file.py:10-14\ndef func_x(y):\n    if not y:\n        raise ValueError('empty')"
  },
  {
    "id": "K2",
    "kind": "control_false",
    "claim": "Hàm X có tên là func_z.",
    "evidence": "REF: /abs/path/to/file.py:10-14\ndef func_x(y):\n    if not y:\n        raise ValueError('empty')"
  },
  {
    "id": "K3",
    "kind": "control_true",
    "claim": "Hàm func_x nhận tham số y.",
    "evidence": "REF: /abs/path/to/file.py:10-14\ndef func_x(y):\n    if not y:\n        raise ValueError('empty')"
  }
]
```

### Lệnh chạy & Giải trình
```bash
# Chạy cổng quyết định (DG1 / DG2 / DG3)
python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run \
  --stage dg1 --workspace <workspace_name> --claims <claims.json> --objective "<mục tiêu>"

# Giải trình một claim bị FLAG (tự động append vào 04_decisions_*.md kèm REF:)
python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py resolve \
  --workspace <workspace_name> --stage dg1 --id S1 --reason "Đã kiểm chứng tại REF: /path/file.py:10-14"
```

## 3. Chế độ B — Laya Agentic Debate Review (`laya_gate.py review`)

Thực thi giao thức **Rào Đệ Quy (Recursive Context Refinement)** trên `git diff -U30` của repo bất kỳ (không hardcode dự án):
1. **Vòng 1 (`review_all`):** Quét các file thay đổi trong `git diff -U30` với ngữ cảnh từ `01_requirements_*` & `09_tasks_solution_*`. Phân loại `APPROVE`, `REJECT_LOGIC`, `REJECT_REDUNDANT`.
2. **Vòng 2..3 (Rào đệ quy + Control chống bị thuyết phục):**
   - Với mỗi file bị `REJECT`, gửi kèm `fencing_context` giải thích kiến trúc **VÀ** 1 biến thể diff cố ý cài lỗi (`control_corrupted_diff`).
   - Nếu Laya chấm `APPROVE > 0.95` cho code thật **VÀ** `REJECT` biến thể lỗi → Báo động giả (False Positive), đóng cờ đỏ.
   - Nếu Laya cũng `APPROVE` cả biến thể lỗi → Vòng đó bị đánh dấu `UNVALIDATED` (bị thuyết phục mù), phải kết luận bằng đọc code trực tiếp.
   - Nếu sau 3 vòng vẫn `REJECT` → Lỗi thật (`BLOCK`): sửa nếu đang ở S4; đề xuất trong báo cáo nếu đang ở S5/Audit.

```bash
python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py review \
  --workspace <workspace_name> --repo <path_to_git_repo> --max-rounds 3
```
