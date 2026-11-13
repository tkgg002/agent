# 09 Tasks Solution — vega_primary_flywheel_2026_11_12

tier: T2
suffix: vega_primary_flywheel_2026_11_12
date: 2026-11-12

## 0. Direct Summary
Hai thay đổi nhỏ, cô lập: (1) `laya_gate.py` chọn engine qua `resolve_gate_url()` — Vega `:8222` nếu `/health` 200, ngược lại Laya `:8200` + cảnh báo; `post_laya` gửi thêm `"mode":"auto"` (Laya bỏ qua field thừa). (2) Module mới `tooling/hard_negative_feedback.py`; `evidence_run.run()` gọi nó khi `exit != 0`, đẩy các claim `ACCEPT` của DG gần nhất thành `verdict=INCORRECT` về Vega. Flywheel: DG ACCEPT → code chạy FAIL → nhãn âm → `fit()` lần sau học.

## 1. Topology sau thay đổi
```
laya_gate.py run/review ──resolve_gate_url()──┬─ :8222/v1/predict {state,questions,mode:auto}  (Vega TTT, cv 0.893)
                                              └─ :8200/v1/predict {state,questions,mode:auto}  (Laya, fallback, field mode bị bỏ qua)
evidence_run.py --kind unit|integration|runtime -- <cmd>
   └─ exit != 0 ──hard_negative_feedback.collect()── 06_laya_summary_dg*_<latest>.json (ACCEPT, non-control)
                 └─ post() → :8222/v1/systemone/feedback {labels:{verdict:INCORRECT}, auto_fit:false}
```

## 2. Code Demo — `tooling/laya_gate.py` (sửa)
```python
DEFAULT_URL = "http://127.0.0.1:8200/v1/predict"          # Laya — giữ làm fallback
VEGA_URL = os.getenv("VEGA_GATE_URL", "http://127.0.0.1:8222/v1/predict")
LAYA_FALLBACK_URL = os.getenv("LAYA_FALLBACK_URL", DEFAULT_URL)
GATE_MODE = os.getenv("VEGA_GATE_MODE", "auto")


def _health_ok(predict_url: str, timeout: float = 3.0) -> bool:
    """Vega exposes /health; Laya does not (404) — only used for the primary engine."""
    health = predict_url.rsplit("/v1/", 1)[0] + "/health"
    try:
        with urllib.request.urlopen(health, timeout=timeout) as r:
            return r.status == 200
    except (urllib.error.URLError, TimeoutError, OSError):
        return False


def resolve_gate_url(explicit: Optional[str]) -> str:
    """--url wins; else Vega if healthy; else Laya with a loud GATE_FALLBACK warning."""
    if explicit:
        return explicit
    if _health_ok(VEGA_URL):
        return VEGA_URL
    print(f"GATE_FALLBACK: {VEGA_URL} unhealthy -> using {LAYA_FALLBACK_URL}", file=sys.stderr)
    return LAYA_FALLBACK_URL


def post_laya(url, state, questions, retries=6, retry_wait=5.0, timeout=120.0, mode=GATE_MODE):
    payload = {"state": state, "questions": questions, "mode": mode}   # Laya ignores "mode"
    ...
```
Trong `cmd_run`/`cmd_review`: `gate_url = resolve_gate_url(args.url)` dùng thay `args.url`; summary thêm `"engine_url": gate_url`; `--url` default `None`.

## 3. Code Demo — `tooling/hard_negative_feedback.py` (mới)
```python
STAGE = "HARD_NEGATIVE_RUNTIME_FAIL"
FEEDBACK_URL = os.getenv("VEGA_FEEDBACK_URL", "http://127.0.0.1:8222/v1/systemone/feedback")


def latest_summary(ws_dir: Path) -> Optional[Path]:
    files = sorted(ws_dir.glob("06_laya_summary_dg*_*.json"), key=lambda p: p.stat().st_mtime)
    return files[-1] if files else None


def collect_hard_negatives(ws_dir: Path, ev_id: str) -> List[Dict[str, Any]]:
    """ACCEPTed non-control claims of the latest gate run become INCORRECT examples."""
    summ = latest_summary(ws_dir)
    if not summ:
        return []
    data = json.loads(summ.read_text(encoding="utf-8"))
    raw = Path(data.get("raw_file", "")) if data.get("raw_file") else None
    evid = {}
    if raw and raw.is_file():
        for r in json.loads(raw.read_text(encoding="utf-8")):
            evid[r.get("id")] = r
    out = []
    for c in data.get("claims", []):
        if c.get("decision") != "ACCEPT" or str(c.get("kind", "")).startswith("control_"):
            continue
        r = evid.get(c["id"], {})
        out.append({
            "task_id": f"{ev_id}:{c['id']}", "stage": STAGE,
            "state": {"claim": c["claim"], "evidence": r.get("evidence", ""), "flow": r.get("flow", "")},
            "labels": {"verdict": "INCORRECT"}, "auto_fit": False,
        })
    return out


def post_hard_negatives(items, url: str = FEEDBACK_URL, timeout: float = 20.0) -> Dict[str, Any]:
    sent, err = 0, ""
    for it in items:
        req = urllib.request.Request(url, data=json.dumps(it, ensure_ascii=False).encode("utf-8"),
                                     headers={"Content-Type": "application/json"}, method="POST")
        try:
            with urllib.request.urlopen(req, timeout=timeout):
                sent += 1
        except (urllib.error.URLError, TimeoutError, OSError) as e:
            err = str(e); break
    return {"stage": STAGE, "candidates": len(items), "sent": sent, "error": err}
```
`evidence_run.run()` sau khi có `proc`: `if proc.returncode != 0: record["hard_negatives"] = post_hard_negatives(collect_hard_negatives(ws_dir, ev_id))` — exit code **không đổi**.

## 4. Edge cases
- Không có summary / raw thiếu → `candidates=0`, không HTTP.
- Sidecar down → `sent=0, error=...`, exit code giữ nguyên (fail-soft).
- `--url` thủ công → không probe health (giữ hành vi cũ cho test/backtest).
- Laya nhận `"mode"` thừa: `PredictRequest` pydantic v2 mặc định `extra="ignore"` → an toàn.

## 5. Decision Gate
| Stage | Summary | Gate |
|---|---|---|
| DG1 | — | — |
| DG2 | — | — |
