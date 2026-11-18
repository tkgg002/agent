tier: T1
# 01. Requirements — vega-sidecar-fit-fix-2026-11-18 (suffix vega_fitfix)
- Ngày: 2026-11-18 | Agent: Claude Sonnet 5.5
## Bối cảnh (VERIFIED)
- fit_state.json last_fit_ts = 2026-11-13; sau restart 24 phút /health báo fit=not_run, CPU 0%, mọi thread chờ lock (sample).
- vega_systemone_sidecar.py: _load_model_blocking chạy v.fit() KHÔNG giữ model_lock; /health báo ok ngay khi vega_model gán; predict giữ model_lock; log không flush.
## User APPROVE (2026-11-18): "Restart Vega, Sửa sidecar => ok"
## Phạm vi
- sidecar/vegaml/vega_systemone_sidecar.py: startup fit + warm-up chạy dưới model_lock; /health status=ok chỉ sau khi xong; predict chờ model_ready_event; print flush.
- sidecar/vegaml/ensure_vega_up.sh: PYTHONUNBUFFERED=1.
- Không đặt lịch 12:15 (chưa biết thời lượng fit).
## DoD
- py_compile + bash -n OK; restart Vega; log hiện "Online ... Startup TTT"; fit_state/health fit.status khác not_run.
