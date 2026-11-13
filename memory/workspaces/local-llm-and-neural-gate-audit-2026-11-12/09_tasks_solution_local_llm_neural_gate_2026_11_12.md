# 09 Tasks Solution — local_llm_neural_gate_2026_11_12

tier: T2
suffix: local_llm_neural_gate_2026_11_12
date: 2026-11-12
status: PROPOSE-ONLY — chờ `APPROVE`

## 0. Direct Summary
VegaML `:8222` không học được vì **tập feedback 1 chiều và quá mỏng**: 49/51 dòng là seed `verdict` (không có APPROVE nào), 2 head còn lại chỉ 1 mẫu. Trong khi đó trên đĩa đã có **437 control claims có nhãn ground-truth** (`agent/memory/workspaces/*/06_laya_claims_*.json`) và **10 task đã `.user-approved`** (`ai-orchestrator/.orchestrator/tasks/*/subtasks/*/`) chưa bao giờ được replay vào Vega. Giải pháp: **1 script replay đơn trách nhiệm** đẩy dữ liệu thật này qua `POST /v1/systemone/feedback` (`auto_fit=false`) rồi `POST /v1/systemone/fit` 1 lần. LLM local `qwen3.5:9b` **không dùng để bịa nhãn** — chỉ làm Muscle phụ (mục A).

## 1. Macro Runtime Topology (VERIFIED)
```
Human Gate (WebUI server.go:1373 APPROVE / telegram_bridge.go:1959 REVISE / subtask_wave_engine.go:238)
   └─ emitVegaHumanGateFeedbackAsync (vega_human_gate.go:187-210)
        └─ buildVegaHumanGateFeedbackPayload (vega_human_gate.go:100-129)
             labels = {c_req_check: approved, dg0_dg1_routing: PASS|REJECT}  AutoFit: true
             └─ POST :8222/v1/systemone/feedback (vega_systemone_sidecar.py:362-394)
                  ├─ append human_gate_feedback.jsonl (:386-387)
                  └─ run_ttt_fit_sync() (:86-110) → vega_model.fit(examples, questions, min_examples=6) (:102)
laya_gate.py (agent) ──POST :8200/v1/predict (laya_gate.py:42)──► Laya (convaiinnovations/laya) — KHÔNG đi qua :8222
```

## 2. Findings `F1..F5` (mỗi finding có REF)
| # | Finding | REF | Hệ quả |
|---|---|---|---|
| F1 | `fit()` chỉ giữ head khi CV accuracy > chance **và** `auto` chỉ dùng head khi CV > engine. Hiện `verdict` 0.592 < 0.612 → `used_by_auto=false`. | `vegaml 0.14.0 Vega.fit.__doc__`; `/health` runtime | Adapter đã train nhưng vô dụng ở `mode=auto` |
| F2 | 49 mẫu `verdict` là seed `CALIBRATION_GROUND_TRUTH` tạo cùng 1 timestamp (`1793329407.57–.65`), 24/49 thiếu `evidence`/`flow` → state nghèo feature. | `human_gate_feedback.jsonl` (thống kê python) | Head học trên noise |
| F3 | `c_req_check`/`dg0_dg1_routing` chỉ có **2 mẫu REVISE, 0 mẫu APPROVE** dù code đã nối `HUMAN_GATE_S3_APPROVE` tại `server.go:1373`. 10 subtask có `.user-approved` nhưng không có dòng nào `approved=true` trong jsonl → đường APPROVE chưa từng bắn thành công (sidecar offline lúc đó hoặc `isTestTaskOrTempRoot` chặn, `vega_human_gate.go:196-199`). | `server.go:1373`, `vega_human_gate.go:196-199`, `find .orchestrator/tasks -name .user-approved` = 10 | Head boolean/choice không thể fit (1 lớp) |
| F4 | Mỗi lần feedback đều `AutoFit: true` (`vega_human_gate.go:127`) → fit lại toàn bộ (294s lần gần nhất, `/health.fit.elapsed_sec=293.989`) trong `model_lock` → chặn `/v1/predict` ~5 phút mỗi lần APPROVE. | `vega_human_gate.go:127`, `sidecar.py:390-392`, `/health` | Latency gate tăng vọt sau mỗi APPROVE |
| F5 | `agent/` core không có đường nào đến `:8222` (`laya_gate.py:42` hardcode `:8200`; grep `8222` trong `agent/` = 0). 437 control claims đã có ground truth nhưng chưa từng được dùng làm training data cho Vega. | `laya_gate.py:42`; `glob agent/memory/workspaces/*/06_laya_claims_*.json` = 437 control | Mỏ dữ liệu thật bị bỏ phí |


## 3. Giải pháp duy nhất (S3)

### A. Local LLM `qwen3.5:9b` — vai trò Muscle phụ (KHÔNG sinh nhãn)
- Ollama `0.40.16` + `qwen3.5:9b` (6.6–7.6 GB). Dùng cho: autocomplete/Cline (`http://127.0.0.1:11434/v1`), tóm tắt log, draft `08_tasks`, pre-review diff trước khi lên Brain.
- Đề xuất sửa `agent/models.env` (sau APPROVE): `MUSCLE_POOL` thay slot `qwen:qwen-2.5-coder:sk-muscle2` → `ollama:qwen3.5:9b:local`.
- Lý do không dùng LLM sinh nhãn: head `verdict` cần nhãn **đúng tuyệt đối** (CORRECT/INCORRECT/UNSUPPORTED); dữ liệu control đã có nhãn tuyệt đối theo cấu trúc, còn LLM 9B sẽ đưa noise vào đúng chỗ Vega đang yếu (F1).

### B. Script replay `sidecar/vegaml/vega_replay_ground_truth.py` (file mới, đơn trách nhiệm)
Nguồn → nhãn (quy tắc cứng, không suy diễn):
1. `06_laya_claims_*.json`: `kind=control_true` → `verdict=CORRECT`; `kind=control_false` → `verdict=INCORRECT`. State = `{claim, evidence, flow}` (đúng schema seed hiện có). Chỉ lấy claim có đủ `claim` + `evidence` (437 mẫu).
2. `.orchestrator/tasks/**/.user-approved` tồn tại → `c_req_check=true`, `dg0_dg1_routing=PASS_TO_HUMAN_GATE`; state = `{task_key, title, preplan_critique_md, tasks_solution_md, existing_file_snapshot, human_feedback:""}` đọc đúng 3 file như `buildVegaHumanGateFeedbackPayload` (`vega_human_gate.go:107-115`, cắt 24000/32000/16000 bytes). → 10 mẫu APPROVE, cân bằng với 2 REVISE sẵn có.
3. Gửi từng mẫu `POST /v1/systemone/feedback` với `auto_fit=false` (tránh F4), cuối cùng `POST /v1/systemone/fit` **1 lần**; in `report` (`cv_accuracy`, `engine_accuracy`, `used_by_auto`).
4. `--dry-run` mặc định: ghi ra `.orchestrator/vega-data/replay_preview.jsonl`, **không chạm** `human_gate_feedback.jsonl`; `--apply` mới gửi HTTP. Trước `--apply` tự `cp human_gate_feedback.jsonl human_gate_feedback.jsonl.bak_<ts>` (pattern đã có: `.bak_20261029T1621`).

### C. Đề xuất sửa nhỏ sau khi có dữ liệu (tách task riêng, KHÔNG nằm trong APPROVE này)
- `vega_human_gate.go:127` `AutoFit: true` → `false` + fit theo lô qua `/v1/systemone/fit` (xử lý F4).

## 4. Code Demo (DG2) — `ai-orchestrator/sidecar/vegaml/vega_replay_ground_truth.py` (file MỚI)
```python
#!/usr/bin/env python3
"""Replay ground-truth đã có trên đĩa vào VegaML (:8222) để fit TTT heads. Dry-run mặc định."""
from __future__ import annotations
import argparse, glob, json, shutil, time, urllib.request
from pathlib import Path

ORCH_ROOT = Path(__file__).resolve().parents[2]
DATA_DIR = ORCH_ROOT / ".orchestrator" / "vega-data"
FEEDBACK_FILE = DATA_DIR / "human_gate_feedback.jsonl"
VERDICT_BY_KIND = {"control_true": "CORRECT", "control_false": "INCORRECT"}
TRUNC = {"10_preplan_critique.md": 24000, "09_tasks_solution.md": 32000, "diff.patch": 16000}

def _read_trunc(p: Path, n: int) -> str:
    return p.read_text(encoding="utf-8", errors="ignore")[:n] if p.is_file() else ""

def iter_claim_examples(agent_ws_root: Path):
    for f in sorted(glob.glob(str(agent_ws_root / "*" / "06_laya_claims_*.json"))):
        try:
            rows = json.loads(Path(f).read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError):
            continue
        for c in rows if isinstance(rows, list) else []:
            if not isinstance(c, dict) or c.get("kind") not in VERDICT_BY_KIND:
                continue
            if not (c.get("claim") and c.get("evidence")):
                continue
            yield {
                "task_id": f"replay-claim-{Path(f).parent.name}-{c.get('id', '?')}",
                "stage": "CALIBRATION_GROUND_TRUTH_REPLAY",
                "state": {"claim": c["claim"], "evidence": c["evidence"], "flow": c.get("flow", "")},
                "labels": {"verdict": VERDICT_BY_KIND[c["kind"]]},
                "auto_fit": False,
            }

def iter_approved_examples(tasks_root: Path):
    for marker in sorted(tasks_root.rglob(".user-approved")):
        d = marker.parent
        yield {
            "task_id": f"replay-approved-{d.name}",
            "stage": "HUMAN_GATE_S3_APPROVE_REPLAY",
            "state": {
                "task_key": d.name, "title": d.name,
                "preplan_critique_md": _read_trunc(d / "10_preplan_critique.md", TRUNC["10_preplan_critique.md"]),
                "tasks_solution_md": _read_trunc(d / "09_tasks_solution.md", TRUNC["09_tasks_solution.md"]),
                "existing_file_snapshot": _read_trunc(d / "diff.patch", TRUNC["diff.patch"]),
                "human_feedback": "",
            },
            "labels": {"c_req_check": True, "dg0_dg1_routing": "PASS_TO_HUMAN_GATE"},
            "auto_fit": False,
        }

def post_json(url: str, payload: dict, timeout: float = 60.0) -> dict:
    req = urllib.request.Request(url, data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
                                 headers={"Content-Type": "application/json"}, method="POST")
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.loads(r.read().decode("utf-8"))

def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--agent-ws-root", required=True)
    ap.add_argument("--base-url", default="http://127.0.0.1:8222")
    ap.add_argument("--apply", action="store_true", help="Gửi HTTP thật; mặc định dry-run")
    a = ap.parse_args()
    examples = list(iter_claim_examples(Path(a.agent_ws_root))) + \
               list(iter_approved_examples(ORCH_ROOT / ".orchestrator" / "tasks"))
    if not examples:
        print("no examples"); return 3
    if not a.apply:
        out = DATA_DIR / "replay_preview.jsonl"
        out.write_text("".join(json.dumps(e, ensure_ascii=False) + "\n" for e in examples), encoding="utf-8")
        print(f"DRY-RUN: {len(examples)} examples -> {out}"); return 0
    if FEEDBACK_FILE.exists():
        shutil.copy2(FEEDBACK_FILE, FEEDBACK_FILE.with_name(FEEDBACK_FILE.name + f".bak_{time.strftime('%Y%m%dT%H%M')}"))
    for e in examples:
        post_json(a.base_url + "/v1/systemone/feedback", e)
    rep = post_json(a.base_url + "/v1/systemone/fit", {}, timeout=1800.0)
    print(json.dumps(rep.get("report", rep), ensure_ascii=False, indent=1)); return 0

if __name__ == "__main__":
    raise SystemExit(main())
```

## 5. Bảng Decision Gate (điền sau khi chạy `laya_gate.py`)
| Stage | Claims file | Summary | Gate |
|---|---|---|---|
| DG1 (v1) | `06_laya_claims_dg1_local_llm_neural_gate_2026_11_12.json` | `06_laya_summary_dg1_20261112T165242.json` | `NEEDS_RESOLUTION` (claim ghép nhiều ý → Laya không phân biệt; chỉ r2 được tin) |
| DG1 (v2) | `06_laya_claims_dg1_v2_local_llm_neural_gate_2026_11_12.json` (tách 1 ý/claim, 3 control_false + 1 control_true) | `06_laya_summary_dg1_20261112T165553.json` | 4 ACCEPT + 2 FLAG → `resolve` S-RC-01, S-IMPACT-01 kèm `REF:` vào `04_decisions_*` → **`PASS`** 🟢 |
| DG2 (v1) | `06_laya_claims_dg2_...json` (REF `:176` sai dòng) | `06_laya_summary_dg2_20261112T165330.json` | `BLOCK` (evidence không khớp đĩa — gate hoạt động đúng) |
| DG2 (v2) | cùng file, sửa REF → `:175` | `06_laya_summary_dg2_20261112T165540.json` | 3/3 ACCEPT → **`PASS`** 🟢 |

**Ghi chú trung thực:** Laya `:8200` chỉ tin được 1 round mỗi lần (r2 ở DG1, r3 ở DG2), K2 (`AutoFit: false`) bị chấm `UNSUPPORTED` thay vì `INCORRECT` ở cả 2 lần → năng lực phân biệt của engine với claim boolean đảo ngược là yếu. Ground truth cuối cùng = code trên đĩa (đã `sed` từng dòng).

## 6. ⏸️ DỪNG — Chờ anh `APPROVE`
Khi `APPROVE`, S4 sẽ: (1) tạo file **mới** `ai-orchestrator/sidecar/vegaml/vega_replay_ground_truth.py` đúng Code Demo §4; (2) chạy `--dry-run` → kiểm `replay_preview.jsonl` (kỳ vọng 447 dòng); (3) chạy `--apply` (fit ~5–30 phút, chặn `:8222` trong lúc đó); (4) `curl /health` xác nhận `c_req_check`/`dg0_dg1_routing` `fitted=true`; (5) DG3 + `audit_report_*` + `verify_governance.py`. **Không** sửa `vega_human_gate.go:127` trong lượt này.

