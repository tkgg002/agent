# 09 — Giải pháp: Master Workflow + Decision Gate (/laya_router) + Hooks bắt buộc

> Ngày: 2026-10-16 | Vai trò: Brain + Muscle | Trạng thái: **APPROVED** (2026-10-16T10:40+07:00)
> Kế thừa: `13_analysis_process_optimization.md` (U1–U7). Quyết định của User: xem `04_decisions_process_optimization.md`.
> **Revision R1 (10:30)**: (a) gỡ toàn bộ dữ kiện dự án khỏi core → Project Manifest (mục 4.5); (b) tích hợp giao thức gốc *Laya Agentic Debate* của User (mục 3.7); (c) lệnh khởi động Laya thật.

---

## 0. Quyết định của User (2026-10-16)

| Câu hỏi | Quyết định | Tác động lên kế hoạch |
|---|---|---|
| Q1 Claude Code | **Bỏ** | Xoá Rule #8 (sync CLAUDE.md); hooks Claude Code cũ → `hooks/_archive_claude/` |
| Q2 `models.env` | **Không có key thật** | Đóng U7-a, không rotate. Giữ nguyên file |
| Q3 Tier T0–T3 | **Đồng ý** | Áp dụng cho linter + Done-gate |
| Mới | `/laya_router` = **mô hình quyết định**, dùng xuyên suốt: khi đưa giải pháp, khi đưa code demo, khi audit | Thêm **Decision Gate DG1/DG2/DG3** + công cụ `laya_gate.py` + hook cưỡng chế |
| R1 | Core agent phục vụ mọi dự án — **không chứa service/port/ngôn ngữ của dự án nào** | Dữ kiện dự án khai báo trong `<project-root>/.agent-project.json` |

---

## 1. Sự thật về /laya_router (có bằng chứng — không suy diễn)

- **Nguồn gốc (skill của User)**: `~/.agent/skills/laya_router/` — `api.py`, `router.py`, `review_all.py`, `laya_agentic_debate.md`, `SKILL.md`.
- **Khởi động**: `/Users/trainguyen/.venv/bin/python3 /Users/trainguyen/.agent/skills/laya_router/api.py` (User cung cấp). `api.py:15-18` nạp weights ~800MB, cold start ~30s; trong lúc nạp `/v1/predict` trả **HTTP 503** (`api.py:80-81`) → client phải chờ/retry, không được coi là lỗi verdict.
- **Server**: FastAPI `http://127.0.0.1:8200` (`api.py:96`), model `convaiinnovations/laya` subfolder `multilingual` (`api.py:17`). Endpoint `/v1/predict` nhận `{state: dict, questions: dict}` (`api.py:74-77`), trả `{status, result}` với `result.answers.<question>.{choice, confidence}`. Kiểm tra 2026-10-16 10:07: `lsof -nP -iTCP:8200` → LISTEN (PID 13360).
- **Giao thức gốc của User** (`laya_agentic_debate.md`): Vòng 1 `review_all` trên `git diff -U30` (APPROVE / REJECT_LOGIC / REJECT_REDUNDANT) → Vòng 2..3 "Rào đệ quy" cho từng file bị REJECT (agent viết fencing context) → APPROVE > 0.95 = báo động giả; vẫn REJECT sau 3 vòng = lỗi thật.
- **Giao thức claim đã dùng ở các audit**: r1 (CORRECT/INCORRECT/UNSUPPORTED), r2 (thêm feature_objective/function_workflow/diff_lines), r3 (YES/NO).
- **Vấn đề hiện tại của skill** (cần xử lý ở Phase 1): `SKILL.md` không có YAML frontmatter `name/description` và `~/.agent/` không nằm trong đường discovery của Antigravity → `/laya_router` không xuất hiện trong menu. `review_all.py` hardcode `cdc-cms-service`, đường dẫn report trong brain cũ và context theo tên file (cùng lỗi "project leakage").
- **Độ tin cậy đo được**:
  - 2026-10-04: control lật kèo giữa các vòng (K1, K2, K3) — kết luận khi đó: "Laya là router/classifier, không phải verifier; chỉ là tín hiệu tham khảo".
  - **2026-10-16 (chạy cho chính tài liệu này)**: control **sai** K1 bị r2 chấm `CORRECT 0.76`, r3 chấm `YES 0.91` → **sai mà rất tự tin**. r1 trả lời đúng control nhưng confidence chỉ 0.10–0.31. Raw: `06_laya_raw_dg1_r1_master_workflow.json`, `06_laya_raw_dg1_r2_r3_master_workflow.json`.

> [!IMPORTANT]
> **Hệ quả thiết kế**: Laya **được** dùng làm cổng quyết định, nhưng phải là cổng **có hiệu chỉnh bằng control trong từng batch**. Round nào trả lời sai control mà tự tin thì round đó bị loại khỏi batch. Ground truth luôn là code/runtime. Nếu tin Laya trực tiếp (không control) thì batch hôm nay sẽ "xác nhận" một claim sai với độ tự tin 91%.

---

## 2. Master Workflow tổng thể

```mermaid
flowchart TD
    S0["S0 BOOT - hook H1: đọc Core + lessons INDEX, check Laya"] --> T{"Phân loại"}
    T -->|"T0 hỏi đáp"| A0["Trả lời dựa trên bằng chứng, không tạo workspace"]
    A0 --> E["Kết thúc"]
    T -->|"Audit theo yêu cầu User"| AU["S2 điều tra READ-ONLY"]
    AU --> DG3A{"DG3 laya_gate"}
    DG3A -->|"BLOCK"| AU
    DG3A -->|"PASS"| RP["audit_report: lỗ hổng + gốc rễ + giải pháp đề xuất"]
    RP --> W
    T -->|"T1 / T2 / T3"| S1["S1 INTAKE: workspace, 01 có tier, 05 START, 08"]
    S1 --> S2["S2 INVESTIGATE: evidence file:line, log, runtime, reproduce bug G2"]
    S2 --> S3["S3 SOLUTION: 09_tasks_solution + code demo"]
    S3 --> DG12{"DG1 giải pháp + DG2 code demo"}
    DG12 -->|"BLOCK"| S2
    DG12 -->|"FLAG"| RS["Giải trình vào 04_decisions qua laya_gate resolve"]
    RS --> DG12
    DG12 -->|"PASS"| W["Trình User - DỪNG chờ APPROVE"]
    W -->|"Yêu cầu chỉnh"| S3
    W -->|"APPROVE"| S4["S4 EXECUTE: restore-point, code qua hook H4, test thật, runtime verify"]
    S4 --> S5["S5 SELF-AUDIT: DG3 + security + QA, Propose-only"]
    S5 -->|"Có finding"| W
    S5 -->|"Sạch"| S6["S6 CLOSE: 06_validation, 11_report, 05 DONE, lesson, linter"]
    S6 --> H7{"Hook H7 Done-gate"}
    H7 -->|"Thiếu bước"| S6
    H7 -->|"Đạt"| E
```

### 2.1 Chi tiết từng chặng

| Chặng | Ai | Làm gì | Đầu ra bắt buộc | Cưỡng chế |
|---|---|---|---|---|
| S0 BOOT | Brain | Đọc `GEMINI.md` Core + `lessons/INDEX.md`; phân tier; xác định workspace | — | H1 |
| S1 INTAKE (T1+) | Brain | Tạo/chọn workspace; `01_requirements_*` có dòng `tier: T1/T2/T3`; append `START` | 01, 05, 08 | H4 (chặn sửa code nếu chưa có) |
| S2 INVESTIGATE | Brain/Debugger | Chỉ đọc: code (file:line), log, DB read-only, runtime. Bug → reproduce (test đỏ hoặc log) | Ghi chú evidence: VERIFIED / HYPOTHESIS | H5 (chặn ghi DB trực tiếp) |
| S3 SOLUTION | Brain | 1 phương án duy nhất: root cause → fix → impact → code demo | `09_tasks_solution_*` | — |
| DG1 + DG2 | Brain | Chạy `laya_gate.py run --stage dg1` (giải pháp) và `--stage dg2` (code demo) | `06_laya_raw_*`, `06_laya_summary_*` | H7 |
| Chờ APPROVE | User | Đọc 09 + bảng Decision Gate | Lời APPROVE của User | H4 (force_ask nếu chưa thấy APPROVE) |
| S4 EXECUTE | Muscle | Append `START` → `git commit` local → code → `checks` + `test` của component (manifest) → `runtime_check` → request thật | Code + test | H2, H3, H4, H5, H6 |
| S5 SELF-AUDIT | QA/Security | DG3 trên từng requirement + từng thay đổi; finding → **đề xuất**, không tự sửa ngoài phạm vi đã duyệt | `06_laya_summary_dg3_*`, `audit_report_*` | H7 |
| S6 CLOSE | Brain | `06_validation`, `11_report`, `05 DONE`, rà lesson, `verify_governance.py --workspace` | Đủ file theo tier | H7 |

**Ranh giới S4 / S5 (codify lesson 2026-10-14):** trong S4 được sửa để test pass *trong thiết kế đã APPROVE*. Từ S5 trở đi, mọi phát hiện → báo cáo + đề xuất, chờ APPROVE mới sửa.

**Bị User sửa lưng ở bất kỳ chặng nào:** DỪNG → ghi lesson (Rule #5) → quay lại chặng tương ứng.

---

## 3. Decision Gate — /laya_router làm "mô hình quyết định"

### 3.1 Ba điểm áp dụng

| Gate | Khi nào | Claim bắt buộc (mỗi claim 1 câu, khẳng định, có REF) |
|---|---|---|
| **DG1 — Giải pháp** | Trước khi trình `09_tasks_solution` | **S-RC** Root cause là X tại REF · **S-FIX** Thay đổi Y loại bỏ X (REF dòng sẽ sửa) · **S-IMPACT** Caller/consumer của symbol là {list} (REF grep), không cái nào bị break · **S-PATTERN** Theo pattern có sẵn P (REF ví dụ) |
| **DG2 — Code demo** | Mỗi khối code demo trong 09 | **D-BEHAV** DEMO thực hiện hành vi B · **D-EDGE** 1 claim/edge case (null, rỗng, trùng, lỗi DB/mạng) · **D-API** DEMO chỉ dùng hàm/field có thật (REF định nghĩa) — chống bịa field (lesson 2026-10-14) |
| **DG3 — Audit** | Audit theo yêu cầu User + self-audit S5 | **A-F** 1 claim/finding (REF) · **A-R** Requirement Rn được đáp ứng tại REF · **A-SCOPE** Thay đổi tại REF không đổi hành vi ngoài phạm vi |

**Control bắt buộc mỗi batch:** ≥ 2 `control_false` (đảo 1 chi tiết của claim thật: đổi tên hàm, đảo điều kiện, đổi số) + ≥ 1 `control_true` (trích nguyên văn hiển nhiên). Thiếu control → gate `INVALID_INPUT`.

### 3.2 Quy tắc quyết định (deterministic, không cảm tính)

1. **Kiểm evidence trước khi gọi Laya**: mỗi `REF: <path>:<start>-<end>` phải trỏ tới file có thật và từng dòng trích phải khớp nội dung file. Sai → claim `BLOCK` (chống bịa code/dòng).
2. **Hiệu chỉnh round theo control**: round r1/r2/r3 chỉ được tính nếu (a) không có control nào bị trả lời **sai** với confidence ≥ 0.5 **và** (b) có ≥ 1 control trả lời **đúng** với confidence ≥ 0.5.
3. **Quyết định từng claim** (chỉ dùng các round được tin):

| Điều kiện | Quyết định | Hành động |
|---|---|---|
| Evidence không khớp file | **BLOCK** | Quay lại S2 |
| Có round tin cậy trả lời phủ định, conf ≥ 0.8 | **BLOCK** | Điều tra lại, sửa giải pháp, chạy lại |
| Có round tin cậy phủ định, conf 0.5–0.8 | **FLAG** | `resolve` kèm lý do + REF vào 04_decisions |
| Có round tin cậy khẳng định, conf ≥ 0.5 | **ACCEPT** | — |
| Không round nào được tin | **UNVALIDATED** | Báo rõ "Laya không phân biệt được ở batch này — kết luận dựa trên code" |
| Còn lại | **FLAG** | Như trên |

4. `basis: hypothesis` → luôn hiển thị **HYPOTHESIS**, kể cả khi Laya ACCEPT. Laya không bao giờ biến giả thuyết thành sự thật.
5. **Gate của batch**: có BLOCK → `BLOCK` (exit 2); còn FLAG chưa resolve → `NEEDS_RESOLUTION` (exit 2); ngược lại → `PASS` (exit 0). Laya không chạy → `LAYA_DOWN` (exit 4).

**Backtest thủ công trên dữ liệu thật:**
- Batch 2026-10-16: r1 không có control decisive, r2/r3 sai K1 decisive → không round nào được tin → toàn bộ **UNVALIDATED**. Khớp với phân tích ở mục 1.
- Batch 2026-10-04: r1 bị loại (K1 sai 0.54); r2, r3 được tin. C3, C4, C6, C8 → ACCEPT; C1, C2, C10, C11 → FLAG. Khớp với kết luận do người đọc code khi đó (những claim mà Laya mâu thuẫn đều phải giải trình bằng code).

> Ngưỡng 0.5 / 0.8 là ngưỡng khởi đầu từ 2 batch thực tế. Sau mỗi 5 batch, xem lại tỉ lệ control sai để hiệu chỉnh (ghi vào 04_decisions).

### 3.3 Quy tắc viết claim (giảm nhiễu classifier)
- 1 câu, khẳng định, 1 ý. Tránh phủ định kép ("không thể… không có cách…").
- Evidence ≤ 40 dòng, bắt đầu bằng `REF:`; code demo bắt đầu bằng `DEMO:`.
- *Giả thuyết cần kiểm chứng*: K1 hôm nay là câu phủ định và bị r2/r3 chấm sai → theo dõi qua control các batch sau.

### 3.4 Code demo — `agent/tooling/laya_gate.py` (gộp 2 client scratch đã dùng thật)

```python
#!/usr/bin/env python3
"""Decision Gate dùng /laya_router (POST http://127.0.0.1:8200/v1/predict).
  laya_gate.py run --stage dg1|dg2|dg3 --workspace <ws> --claims claims.json [--objective "..."]
  laya_gate.py resolve --workspace <ws> --stage dg1 --id S3 --reason "... REF ..."
Exit: 0 PASS | 2 BLOCK/NEEDS_RESOLUTION | 3 INVALID_INPUT | 4 LAYA_DOWN
Verdict CHỈ lấy từ HTTP response — CẤM hardcode.
"""
import argparse, json, re, sys, time, urllib.request
from pathlib import Path

URL = "http://127.0.0.1:8200/v1/predict"
WS_ROOT = Path(__file__).resolve().parents[1] / "memory" / "workspaces"
DECISIVE, BLOCK_CONF = 0.5, 0.8
CRIT3 = {"CORRECT": "...", "INCORRECT": "...", "UNSUPPORTED": "..."}      # giữ nguyên prompt r1/r2 đã dùng
Q = {
    "r1": {"type": "choice", "instructions": "You are a strict code auditor ...", "criteria": CRIT3},
    "r2": {"type": "choice", "instructions": "Read the function workflow and diff lines ...", "criteria": CRIT3},
    "r3": {"type": "choice", "instructions": "Answer strictly YES or NO ...",
           "criteria": {"YES": "claim is true for the code shown", "NO": "claim is false for the code shown"}},
}
POS = {"r1": "CORRECT", "r2": "CORRECT", "r3": "YES"}
NEG = {"r1": "INCORRECT", "r2": "INCORRECT", "r3": "NO"}
REF = re.compile(r"^REF:\s*(\S+):(\d+)(?:-(\d+))?\s*$", re.M)
norm = lambda s: re.sub(r"\s+", " ", s).strip()

def state(rnd, c, objective):
    if rnd == "r1": return {"claim": c["claim"], "evidence": c["evidence"], "flow": c.get("flow", "")}
    if rnd == "r2": return {"feature_objective": objective, "function_workflow": c.get("flow", ""),
                            "diff_lines": c["evidence"], "claim": c["claim"]}
    return {"code_excerpt": c["evidence"], "claim": c["claim"]}

def verify_evidence(c):
    """Chống bịa: REF phải tồn tại và từng dòng trích phải khớp nội dung file."""
    blocks = re.split(r"(?m)^(?=REF:|DEMO:)", c["evidence"])
    refs = [b for b in blocks if b.startswith("REF:")]
    if not refs and c["kind"] != "demo": return False, "thiếu REF file:line"
    for b in refs:
        path, s, e = REF.match(b.splitlines()[0]).groups()
        p = Path(path)
        if not p.is_file(): return False, f"không tồn tại {path}"
        actual = {norm(l) for l in p.read_text(errors="ignore").splitlines()[int(s) - 1:int(e or s)]}
        bad = [norm(l) for l in b.splitlines()[1:] if norm(l) and norm(l) not in actual]
        if bad: return False, f"trích dẫn không khớp {path}:{s}-{e or s}: {bad[0][:60]}"
    return True, ""

def ask(rnd, c, objective):
    body = {"state": state(rnd, c, objective), "questions": {"verdict": Q[rnd]}}
    req = urllib.request.Request(URL, json.dumps(body).encode(), {"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=120) as r:
        resp = json.loads(r.read())
    a = resp.get("result", {}).get("answers", {}).get("verdict", {})
    return {"choice": a.get("choice"), "confidence": a.get("confidence") or 0.0, "raw": resp, "request": body}

def trusted(rows, rnd):
    hit = False
    for r in (x for x in rows if x["kind"].startswith("control_")):
        want = POS[rnd] if r["kind"] == "control_true" else NEG[rnd]
        v = r[rnd]
        if v["confidence"] >= DECISIVE and v["choice"] != want: return False
        hit |= v["confidence"] >= DECISIVE and v["choice"] == want
    return hit

def decide(r, rounds):
    if not r["evidence_ok"]: return "BLOCK"
    if not rounds: return "UNVALIDATED"
    vs = [(n, r[n]) for n in rounds]
    if any(v["choice"] == NEG[n] and v["confidence"] >= BLOCK_CONF for n, v in vs): return "BLOCK"
    if any(v["choice"] == NEG[n] and v["confidence"] >= DECISIVE for n, v in vs): return "FLAG"
    if any(v["choice"] == POS[n] and v["confidence"] >= DECISIVE for n, v in vs): return "ACCEPT"
    return "FLAG"

def run(a):
    claims = json.load(open(a.claims))
    kinds = [c["kind"] for c in claims]
    if kinds.count("control_false") < 2 or kinds.count("control_true") < 1:
        print("INVALID_INPUT: cần ≥2 control_false + ≥1 control_true"); return 3
    rows = []
    try:
        for c in claims:
            ok, why = verify_evidence(c)
            row = {**c, "evidence_ok": ok, "evidence_error": why}
            for rnd in ("r1", "r2", "r3"):
                row[rnd] = ask(rnd, c, a.objective)
            rows.append(row)
    except OSError as e:
        print(f"LAYA_DOWN: {e}"); return 4
    rounds = [n for n in ("r1", "r2", "r3") if trusted(rows, n)]
    for r in rows:
        if not r["kind"].startswith("control_"):
            r["decision"] = "HYPOTHESIS" if r.get("basis") == "hypothesis" else decide(r, rounds)
    ts = time.strftime("%Y%m%dT%H%M%S")
    ws = WS_ROOT / a.workspace
    (ws / f"06_laya_raw_{a.stage}_{ts}.json").write_text(json.dumps(rows, ensure_ascii=False, indent=2))
    summary = {"stage": a.stage, "ts": ts, "trusted_rounds": rounds,
               "claims": [{"id": r["id"], "decision": r["decision"], "resolved": None,
                           "evidence_error": r["evidence_error"]} for r in rows if "decision" in r]}
    summary["gate"] = gate(summary)
    (ws / f"06_laya_summary_{a.stage}_{ts}.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2))
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if summary["gate"] == "PASS" else 2

def gate(s):
    ds = [c["decision"] for c in s["claims"]]
    if "BLOCK" in ds: return "BLOCK"
    if any(c["decision"] == "FLAG" and not c["resolved"] for c in s["claims"]): return "NEEDS_RESOLUTION"
    return "PASS"

# resolve(): mở summary mới nhất của stage, gán claims[id].resolved = reason (bắt buộc có "REF:"),
# APPEND 1 dòng vào 04_decisions_*.md: "- [ts] [Agent:laya_gate] <stage> <id> FLAG resolved: <reason>",
# tính lại gate. BLOCK KHÔNG resolve được — phải sửa giải pháp/claim và chạy lại.
```

### 3.5 Mẫu claims (DG2 — code demo)

```json
[
  {"id": "D1", "kind": "demo", "basis": "code", "flow": "Shutdown chờ goroutine nền",
   "claim": "Shutdown waits for all background goroutines to exit before closing the SQL connection.",
   "evidence": "DEMO:\nfunc (s *Server) Shutdown() { s.cancel(); s.wg.Wait(); s.sqlDB.Close() }\nREF: <project-root>/<component>/server.go:<start>-<end>\n<các dòng hiện tại, trích nguyên văn>"},
  {"id": "K1", "kind": "control_false", "claim": "Shutdown closes the SQL connection before cancelling the context.", "evidence": "DEMO:\n..."},
  {"id": "K2", "kind": "control_false", "claim": "Shutdown waits on a channel named doneCh.", "evidence": "DEMO:\n..."},
  {"id": "K3", "kind": "control_true", "claim": "Shutdown calls s.cancel().", "evidence": "DEMO:\n..."}
]
```
*(Ví dụ minh hoạ định dạng — số dòng thật sẽ lấy khi làm task thật.)*

### 3.6 Trình bày cho User (bắt buộc trong 09 / audit_report)

```markdown
#### Decision Gate DG1 — PASS | trusted rounds: r2, r3 | raw: 06_laya_raw_dg1_<ts>.json
| ID | Claim | Basis | Decision | Ghi chú |
|----|-------|-------|----------|---------|
| S1 | Root cause ... | code | ACCEPT | r3 YES 0.86 |
| S3 | Không caller nào bị break | code | FLAG → resolved | 04_decisions: REF grep ... |
| S4 | Lag tăng khi ... | hypothesis | HYPOTHESIS | cần log prod |
```

### 3.7 Hai chế độ của Decision Gate — tích hợp giao thức *Agentic Debate* gốc

| Chế độ | Dùng cho | Câu hỏi gửi Laya | Nguồn |
|---|---|---|---|
| **A — Claim** | DG1 (giải pháp), DG2 (code demo), DG3 phần *finding/requirement* | `verdict`: r1 CORRECT/INCORRECT/UNSUPPORTED → r2 có ngữ cảnh → r3 YES/NO | Mục 3.1–3.4 |
| **B — Debate review** | DG3 phần *review code đã sửa* (S5) + khi User yêu cầu "dùng laya review code" | `review_result`: APPROVE / REJECT_LOGIC / REJECT_REDUNDANT | `laya_agentic_debate.md` của User |

**Chế độ B — giữ nguyên giao thức của anh, bổ sung 3 điểm:**

1. **Vòng 1** (`review_all`): chạy trên `git diff -U30` của **repo đang sửa** (lấy từ đường dẫn file bị sửa, không hardcode service). `feature_objective`/`function_workflow` lấy từ `01_requirements` + `09_tasks_solution` của workspace (không map theo tên file như `review_all.py` hiện tại).
2. **Vòng 2..3** (rào đệ quy) cho từng file REJECT — **thêm control chống "bị thuyết phục"**: cùng fencing context đó, gửi kèm **1 biến thể cố ý lỗi** của diff (vd bỏ guard, đảo điều kiện). Nếu Laya cũng APPROVE biến thể lỗi với confidence ≥ 0.5 → vòng đó **không phân biệt được** → không được dùng để đóng cờ đỏ; kết luận dựa trên đọc code (ghi rõ trong báo cáo).
   *Lý do*: luật sư viết context để thuyết phục; hôm nay Laya đã xác nhận câu sai với 0.91 → rủi ro "thuyết phục được cả lỗi thật" là có thật.
3. **Phán quyết**: APPROVE > 0.95 **và** control lỗi bị REJECT → báo động giả, đóng. Vẫn REJECT sau 3 vòng → **lỗi thật** → xử lý theo chặng:
   - Đang ở **S4** (trong phạm vi đã APPROVE): sửa, rồi chạy lại từ Vòng 1.
   - Ở **S5 / audit theo yêu cầu**: **chỉ đề xuất** cách sửa trong `audit_report`, chờ APPROVE (thay cho câu "bắt buộc phải sửa lại Source Code" trong tài liệu gốc — để khớp Rule #13 và lesson 2026-10-14).

`laya_gate.py` có thêm lệnh: `laya_gate.py review --workspace <ws> --repo <path> [--max-rounds 3]` → ghi `06_laya_raw_dg3review_<ts>.json` + `06_laya_summary_dg3review_<ts>.json` (cùng định dạng gate PASS/BLOCK/NEEDS_RESOLUTION).

---

## 4. hooks.json — các script bắt buộc

### 4.1 Danh sách & lý do

| Hook | Sự kiện | Chặn cái gì | Rule | Lesson ngăn tái phạm |
|---|---|---|---|---|
| **H1 boot** | `PreInvocation` (lần đầu của hội thoại) | Bơm checklist S0 + trạng thái Laya (UP/DOWN + lệnh khởi động nếu DOWN) | 0, 4 | Bỏ qua đọc lesson đầu phiên |
| **H2 memory_guard** | `PreToolUse` write/replace | `deny` Overwrite file memory; `deny` sửa file append-only (05_progress, 04_decisions, lessons) mà không giữ nguyên TargetContent | 4, 17 | Data destruction |
| **H3 secret_guard** | `PreToolUse` write/replace | Phát hiện token/key/URI có mật khẩu: `deny` nếu ghi vào memory, `force_ask` nếu ghi vào source | 20 | #audit-log-secret-masking |
| **H4 change_gate** | `PreToolUse` write/replace vào source (đuôi file theo manifest, mặc định danh sách chung) | Chưa có workspace → `deny`; chưa append 05_progress → `deny`; DG1 chưa PASS → `deny`; chưa thấy User APPROVE sau 09 trong phiên → `force_ask` | 4, 13, DG | #audit-propose-only, #no-unauthorized-fix |
| **H5 command_guard** | `PreToolUse` run_command | `deny` `git push`, `git reset --hard`, `git clean -f`, `rm -rf` memory/.git; `force_ask` lệnh ghi DB trực tiếp (psql/mongosh UPDATE/DELETE/DROP…) | 12, 15, 19 | Cheat DB, push trái phép |
| **H6 fast_feedback** | `PostInvocation` | Sau mỗi lượt có sửa source: tìm manifest của file → chạy `checks` của **component** chứa file đó, bơm lỗi ngay. Không có manifest → bỏ qua | 14-G3 | #build-ok-not-runtime-ok |
| **H7 done_gate** | `Stop` | Chặn kết thúc nếu: viết 09 mà thiếu DG1/DG2 PASS; viết audit mà thiếu DG3 PASS; sửa source mà thiếu 05 sau lần sửa cuối, thiếu 06_validation, thiếu DG3; component có khai báo `runtime_check` trong manifest mà chưa chạy lệnh đó sau lần sửa cuối; `verify_governance.py` FAIL. Tối đa 2 lần chặn/hội thoại | 14, 22, DG | #premature-done-on-port-conflict, #stale-process-blindspot |

*Không dùng `PostToolUse` để phản hồi: contract quy định output của nó là `{}` (docs/hooks.md:235) — không bơm được thông điệp. Phản hồi sau khi sửa file đặt ở `PostInvocation` (cho phép `injectSteps`, docs/hooks.md:273-291).*

### 4.2 `~/.gemini/config/hooks.json`

```json
{
  "agy-boot": {
    "PreInvocation": [
      { "command": "python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py pre-invocation", "timeout": 10 }
    ]
  },
  "agy-pre-write": {
    "PreToolUse": [
      { "matcher": "write_to_file\\|replace_file_content",
        "hooks": [{ "command": "python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py pre-write", "timeout": 10 }] }
    ]
  },
  "agy-pre-command": {
    "PreToolUse": [
      { "matcher": "run_command",
        "hooks": [{ "command": "python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py pre-command", "timeout": 5 }] }
    ]
  },
  "agy-fast-feedback": {
    "PostInvocation": [
      { "command": "python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py post-invocation", "timeout": 90 }
    ]
  },
  "agy-done-gate": {
    "Stop": [
      { "command": "python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py stop", "timeout": 60 }
    ]
  }
}
```

### 4.3 Code demo — `agent/hooks/agy/agy_guard.py` (1 dispatcher, dễ test)

```python
#!/usr/bin/env python3
"""Antigravity hook dispatcher. stdin JSON -> stdout JSON.
Usage: agy_guard.py pre-invocation|pre-write|pre-command|post-invocation|stop"""
import json, re, subprocess, sys
from pathlib import Path

AGENT = Path(__file__).resolve().parents[2]
MEM, WS_ROOT, STATE = AGENT / "memory", AGENT / "memory" / "workspaces", Path("/tmp/agy")
SRC_EXT = {".go", ".ts", ".tsx", ".js", ".py", ".sql"}
APPEND_ONLY = re.compile(r"(05_progress[^/]*|04_decisions[^/]*|lessons)\.md$")
SECRET = re.compile(r"(AKIA[0-9A-Z]{16}|sk-[A-Za-z0-9]{20,}|ATATT[\w\-]{20,}|ghp_\w{30,}"
                    r"|Bearer\s+[\w.\-]{20,}|(mongodb|postgres(ql)?)://[^*\s:/]+:[^*\s@]+@"
                    r"|password\s*[:=]\s*[^\s*'\"]{6,})", re.I)
APPROVE = re.compile(r"(approve|duyệt|ok (sửa|làm|fix)|làm đi|sửa đi|fix đi|triển khai)", re.I)
MANIFEST = ".agent-project.json"          # dữ kiện dự án nằm trong repo dự án, KHÔNG nằm trong core
LAYA_START = "/Users/trainguyen/.venv/bin/python3 /Users/trainguyen/.agent/skills/laya_router/api.py"
DENY_CMD = [(r"\bgit\s+push\b", "Rule 15: CẤM git push — User tự quyết định push."),
            (r"\bgit\s+(reset\s+--hard|clean\s+-\w*f)", "Rule 19: lệnh huỷ thay đổi chưa commit."),
            (r"\brm\s+-\w*r\w*\b.*(/memory\b|\.git\b)", "Rule 17: CẤM xoá memory/.git.")]
ASK_CMD = [(r"\b(psql|mongosh|mysql)\b.*\b(UPDATE|DELETE|INSERT|DROP|TRUNCATE|ALTER|deleteMany|updateMany)\b",
            "Rule 12: ghi trực tiếp vào DB (cheat DB) — cần User xác nhận.")]
BOOT = ("BOOT S0: (1) Đọc agent/GEMINI.md + memory/global/lessons/INDEX.md. (2) Phân loại T0/Audit/T1-T3. "
        "(3) T1+: workspace + 05 START trước khi sửa code. (4) Giải pháp/code demo/audit → laya_gate.py "
        "(Laya: {laya}). (5) Không sửa source trước APPROVE. (6) Không git push.")

def out(o): print(json.dumps(o, ensure_ascii=False)); sys.exit(0)
def steps(p):
    try: return [json.loads(l) for l in open(p.get("transcriptPath", ""), encoding="utf-8") if l.strip()]
    except OSError: return []
def calls(st):
    for i, s in enumerate(st):
        for tc in s.get("tool_calls") or []:
            yield i, tc.get("name"), tc.get("args") or {}
def tgt(a): return a.get("TargetFile") or ""
def is_src(f): return Path(f).suffix in SRC_EXT and str(MEM) not in f
def touched(st, pat, after=-1):
    return any(i > after and re.search(pat, json.dumps(a, ensure_ascii=False)) for i, _, a in calls(st))
def src_edits(st):
    return [(i, tgt(a)) for i, n, a in calls(st) if n in ("write_to_file", "replace_file_content") and is_src(tgt(a))]
def active_ws(st):
    for _, _, a in reversed(list(calls(st))):
        m = re.search(re.escape(str(WS_ROOT)) + r"/([^/\"]+)/", json.dumps(a, ensure_ascii=False))
        if m: return m.group(1)
def laya_gate(ws, stage):
    s = sorted((WS_ROOT / ws).glob(f"06_laya_summary_{stage}_*.json")) if ws else []
    return json.loads(s[-1].read_text())["gate"] if s else None
def manifest_for(f):
    """Tìm .agent-project.json gần nhất, đi từ thư mục của file bị sửa lên trên. Không có → (None, None)."""
    for d in Path(f).resolve().parents:
        if (d / MANIFEST).is_file(): return d, json.loads((d / MANIFEST).read_text())
    return None, None
def component_for(f):
    """Component có path dài nhất chứa file f (vd 'services/api'). Không khai báo → None."""
    root, mf = manifest_for(f)
    if not mf: return None, None
    rel = Path(f).resolve().relative_to(root).as_posix()
    hits = [c for c in mf.get("components", []) if rel.startswith(c["path"].rstrip("/") + "/")]
    return root, max(hits, key=lambda c: len(c["path"]), default=None)

def pre_invocation(p):
    flag = STATE / f"{p['conversationId']}.boot"
    if flag.exists(): out({})
    STATE.mkdir(exist_ok=True); flag.touch()
    up = subprocess.run(["lsof", "-nP", "-iTCP:8200", "-sTCP:LISTEN"], capture_output=True).returncode == 0
    laya = "UP" if up else f"DOWN — nhờ User chạy: {LAYA_START} (cold start ~30s, 503 khi đang nạp)"
    out({"injectSteps": [{"ephemeralMessage": BOOT.format(laya=laya)}]})

def pre_write(p):
    a = p["toolCall"]["args"]; f = tgt(a)
    body = a.get("CodeContent", "") + a.get("ReplacementContent", "")
    if str(MEM) in f:                                                     # H2
        if a.get("Overwrite") and Path(f).exists():
            out({"decision": "deny", "reason": f"Rule 17: CẤM Overwrite memory file {f}. Chỉ APPEND."})
        if APPEND_ONLY.search(f) and a.get("TargetContent") and a["TargetContent"] not in a.get("ReplacementContent", ""):
            out({"decision": "deny", "reason": f"Rule 4/17: {Path(f).name} append-only — phải giữ nguyên TargetContent."})
    if (m := SECRET.search(body)):                                        # H3
        out({"decision": "deny" if str(MEM) in f else "force_ask",
             "reason": f"Rule 20: phát hiện chuỗi giống secret ({m.group(0)[:10]}***). Mask trước khi ghi."})
    if is_src(f):                                                         # H4
        st = steps(p); ws = active_ws(st)
        if not ws: out({"decision": "deny", "reason": "Rule 4: chưa có workspace (01/05/08) cho thay đổi code."})
        if not touched(st, r"05_progress"):
            out({"decision": "deny", "reason": f"Rule 4: append START vào {ws}/05_progress_*.md trước khi sửa code."})
        if laya_gate(ws, "dg1") != "PASS":
            out({"decision": "deny", "reason": "DG1: giải pháp chưa qua laya_gate (PASS) — không được sửa code."})
        sol = [i for i, _, x in calls(st) if "09_tasks_solution" in tgt(x)]
        ok = any(s.get("type") == "USER_INPUT" and APPROVE.search(s.get("content", ""))
                 for s in st[(sol[-1] + 1 if sol else 0):])
        if not ok:
            out({"decision": "force_ask", "reason": "Rule 13: chưa thấy User APPROVE trong phiên này. Cho phép sửa code?"})
    out({"decision": "ask"})                     # passthrough — chốt lại ở Phase 0 (ask vs allow)

def pre_command(p):
    cmd = p["toolCall"]["args"].get("CommandLine", "")
    for pat, why in DENY_CMD:
        if re.search(pat, cmd): out({"decision": "deny", "reason": why})
    for pat, why in ASK_CMD:
        if re.search(pat, cmd, re.I): out({"decision": "force_ask", "reason": why})
    out({"decision": "ask"})

def post_invocation(p):
    st = steps(p); mark = STATE / f"{p['conversationId']}.fb"
    last = int(mark.read_text()) if mark.exists() else -1
    comps = {}
    for i, f in src_edits(st):
        if i > last:
            root, c = component_for(f)
            if c: comps[(str(root), c["name"])] = (root / c["path"], c)
    STATE.mkdir(exist_ok=True); mark.write_text(str(len(st) - 1))
    errs = []
    for (_, name), (cwd, c) in sorted(comps.items()):
        for cmd in c.get("checks", []):          # lệnh do chính dự án khai báo; phải exit != 0 khi lỗi
            r = subprocess.run(cmd, shell=True, cwd=cwd, capture_output=True, text=True, timeout=40)
            if r.returncode:
                errs.append(f"[{name}] $ {cmd}\n{(r.stdout + r.stderr)[-1200:]}")
    out({"injectSteps": [{"ephemeralMessage": "⚠️ H6 fast-feedback:\n" + "\n".join(errs)}]} if errs else {})

def stop(p):
    st = steps(p); cnt = STATE / f"{p['conversationId']}.stop"
    n = int(cnt.read_text()) if cnt.exists() else 0
    if n >= 2: out({})
    ws, src, miss = active_ws(st), src_edits(st), []
    if touched(st, r"09_tasks_solution"):
        for g in ("dg1", "dg2"):
            if laya_gate(ws, g) != "PASS": miss.append(f"{g.upper()}: laya_gate chưa PASS cho giải pháp/code demo")
    if touched(st, r"(audit_report|13_analysis)") and laya_gate(ws, "dg3") != "PASS":
        miss.append("DG3: laya_gate chưa PASS cho audit")
    if src:
        last = src[-1][0]
        if not touched(st, r"05_progress", last): miss.append("05_progress: chưa append kết quả sau lần sửa code cuối")
        if not touched(st, r"06_validation", last): miss.append("06_validation: chưa ghi bằng chứng test/runtime")
        if laya_gate(ws, "dg3") != "PASS": miss.append("DG3: chưa self-audit thay đổi bằng laya_gate")
        seen = set()
        for _, f in src:
            _, c = component_for(f)
            rc = c.get("runtime_check") if c else None
            if rc and c["name"] not in seen and not touched(st, re.escape(rc), last):
                seen.add(c["name"])
                miss.append(f"Runtime: sửa component `{c['name']}` nhưng chưa chạy `{rc}` sau lần sửa cuối")
    if ws and (src or touched(st, r"workspaces/")):
        r = subprocess.run([sys.executable, str(AGENT / "tooling/verify_governance.py"), "--workspace", ws],
                           capture_output=True, text=True, timeout=30)
        if r.returncode: miss.append("verify_governance.py FAILED:\n" + (r.stdout + r.stderr)[-600:])
    if not miss: out({})
    STATE.mkdir(exist_ok=True); cnt.write_text(str(n + 1))
    out({"decision": "continue", "reason": "⛔ Done-gate chưa đạt:\n- " + "\n- ".join(miss)})

if __name__ == "__main__":
    payload = json.load(sys.stdin)
    {"pre-invocation": pre_invocation, "pre-write": pre_write, "pre-command": pre_command,
     "post-invocation": post_invocation, "stop": stop}[sys.argv[1]](payload)
```

### 4.4 Kịch bản test (fixture JSON → assert output) — `agent/hooks/agy/tests/test_agy_guard.py`

| # | Kịch bản | Kỳ vọng |
|---|---|---|
| 1 | write_to_file Overwrite=true vào `memory/global/lessons.md` (đã tồn tại) | `deny` |
| 2 | replace_file_content vào `05_progress_x.md`, ReplacementContent bỏ mất TargetContent | `deny` |
| 3 | replace_file_content vào `05_progress_x.md`, ReplacementContent = TargetContent + dòng mới | `ask` |
| 4 | Ghi `mongodb://admin:Secr3t@host` vào memory | `deny` |
| 5 | Ghi `mongodb://***:***@host` vào memory | `ask` |
| 6 | Sửa `.go` khi transcript chưa có workspace | `deny` |
| 7 | Sửa `.go`, có workspace + 05, DG1 = BLOCK | `deny` |
| 8 | Sửa `.go`, DG1 PASS, không có USER_INPUT approve sau 09 | `force_ask` |
| 9 | Sửa `.go`, DG1 PASS, có "ok sửa đi" sau 09 | `ask` |
| 10 | run_command `git push origin main` | `deny` |
| 11 | run_command `psql -c "DELETE FROM t"` | `force_ask` |
| 12 | Stop: sửa file thuộc component có `runtime_check` trong manifest fixture (repo giả `tests/fixtures/proj/`), chưa chạy lệnh đó sau lần sửa cuối | `continue` + lý do Runtime |
| 13 | Stop lần thứ 3 trong cùng hội thoại | `{}` (chống vòng lặp) |
| 14 | Stop: hội thoại T0 không ghi file | `{}` |
| 15 | Sửa file trong repo **không có** manifest | H6 không chạy gì; H7 chỉ áp yêu cầu chung (05, 06_validation, DG3) |
| 16 | Component lồng nhau (`services/` và `services/api/`) | Chọn component có path dài nhất |

*Fixture dùng repo giả trong `tests/fixtures/` — test của core không phụ thuộc bất kỳ dự án thật nào.*

### 4.5 Project Manifest — dữ kiện dự án nằm ở dự án, không nằm ở core

Core (`agent/`) chỉ chứa **cơ chế**. Mỗi dự án muốn bật fast-feedback (H6) và kiểm runtime (H7) tự khai báo file `<project-root>/.agent-project.json`, commit cùng repo dự án. Hook đi từ file bị sửa ngược lên để tìm manifest gần nhất. Không có manifest → chỉ áp yêu cầu chung.

```json
{
  "name": "<tên dự án>",
  "components": [
    {
      "name": "<tên component>",
      "path": "<đường dẫn tương đối từ project root>",
      "checks": ["<lệnh kiểm tra nhanh, exit != 0 khi lỗi>"],
      "test": "<lệnh test đầy đủ — Muscle chạy ở S4>",
      "runtime_check": "<lệnh chứng minh process mới đang chạy — H7 yêu cầu sau lần sửa cuối>"
    }
  ]
}
```

- `checks` phải trả exit code ≠ 0 khi lỗi (vd với Go: `test -z "$(gofmt -l .)"` thay vì `gofmt -l .`, vì lệnh sau luôn exit 0).
- Lệnh trong manifest chạy bằng shell trong thư mục component → manifest chỉ được đặt trong repo do User kiểm soát.
- Chính repo `agent/` cũng có manifest riêng (component `hooks` với `checks: ["python3 -m pytest hooks/agy/tests -q"]`) — core tự kiểm chính nó theo cùng cơ chế.
- Việc khai báo manifest cho từng dự án (vd data-hub) là **task của dự án đó**, làm trong workspace của dự án, không thuộc phạm vi core.

---

## 5. Thay đổi trong GEMINI.md Core (bản nháp đoạn mới)

```markdown
### 9. Master Workflow
S0 Boot → (T0: trả lời) | (Audit: S2 read-only → DG3 → báo cáo) | (T1–T3: S1 Intake → S2 Investigate →
S3 Solution → DG1+DG2 → DỪNG chờ APPROVE → S4 Execute → S5 Self-audit DG3 → S6 Close → Done-gate).
Từ S5: mọi finding chỉ đề xuất, không tự sửa ngoài phạm vi đã APPROVE.

### 10. Decision Gate (/laya_router)
- Giải pháp (DG1), code demo (DG2), audit/kết luận (DG3) BẮT BUỘC chạy `agent/tooling/laya_gate.py` trước khi trình User.
- Claim 1 câu khẳng định + `REF: path:line` trích nguyên văn; mỗi batch ≥2 control sai + ≥1 control đúng.
- BLOCK → điều tra lại. FLAG → `laya_gate.py resolve` kèm REF. UNVALIDATED → nói rõ "Laya không phân biệt được".
- Ground truth = code/runtime. Laya không biến HYPOTHESIS thành sự thật. CẤM hardcode verdict.
- Báo cáo ghi: gate, trusted rounds, bảng claim → decision, đường dẫn raw JSON.
```

Bỏ: Rule #8 (CLAUDE.md). Rule #22 đổi thành: "Linter chạy cho T1–T3, do hook H7 tự gọi".

---

## 6. Lộ trình (cập nhật — mỗi phase cần APPROVE riêng)

| Phase | Nội dung | Tiêu chí PASS |
|---|---|---|
| 0 | `git commit` local `agent/` (restore-point); cài hook đo `matcher: "*"` ghi stdin → xác minh: tên tool thật, khoá `toolCall.args`, định dạng transcript, `invocationNum`, passthrough `ask`/`allow`, hook có bị sandbox chặn đọc `memory/` không | File log có đủ payload của 5 sự kiện |
| 1 | GEMINI Core ≤ 12KB (thêm mục 9–10, bỏ Rule 8); global rule → pointer; `skills.json` gồm `agent/skills` **và `~/.agent/skills`**; skills lõi; thêm YAML frontmatter cho `~/.agent/skills/laya_router/SKILL.md` + phần hướng dẫn DG (viết claim, control, gọi gate); archive hooks Claude | Không còn `<truncated`; `/laya_router` xuất hiện trong menu `/` (nếu tên có `_` không được nhận → đổi `laya-router`) |
| 2 | `laya_gate.py` (chế độ A claim + chế độ B debate review, retry khi 503) + test (fake HTTP server cố định verdict; 1 lần chạy live có control) | Test pass; backtest batch 2026-10-04 và 2026-10-16 ra đúng kết quả mục 3.2 |
| 3 | `agy_guard.py` + `hooks.json` + 16 test fixture (repo giả) + manifest của chính repo `agent/` | 16/16 PASS; thử thật trên hội thoại nháp: kịch bản 1, 10, 12 bị chặn đúng |
| 4 | lessons INDEX + topics | Số entry INDEX = số entry gốc; INDEX ≤ 15KB |
| 5 | Linter theo tier + bắt buộc `--workspace` + chấp nhận `12_implementation_plan*`; archive workflow & skill ngoài domain | Linter PASS trên 1 workspace T1 thật |

## 7. Trạng thái câu hỏi

- ~~Lệnh khởi động Laya~~ → `/Users/trainguyen/.venv/bin/python3 /Users/trainguyen/.agent/skills/laya_router/api.py` (User, 10:20). Đã đưa vào H1.
- ~~Danh sách service/port~~ → **huỷ câu hỏi**: không thuộc core (R1). Dự án nào cần thì tự khai báo `.agent-project.json`.
- **Còn lại**: APPROVE Phase 0.
