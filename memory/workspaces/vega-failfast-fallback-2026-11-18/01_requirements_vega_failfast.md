# 01. Requirements — `vega-failfast-fallback-2026-11-18`

tier: T1
- **suffix:** `vega_failfast`
- **Ngày:** 2026-11-18 | **Agent:** Claude Sonnet 5.5

## Bối cảnh (VERIFIED)
- `laya_gate.py run --stage dg3` mất 242s rồi `LAYA_DOWN: timed out` (exit 4). Probe 1 request: `/v1/predict` của Vega `:8222` treo 120s mỗi round dù `/health`=200 (~10ms). `sample` pid 16343: worker kẹt `at::mps ... bmm` (GPU MPS, VEGA_DEVICE=mps tại `vega_systemone_sidecar.py:39`).
- `post_laya` (`laya_gate.py:217-249`): timeout 120s, mọi `OSError` (kể cả timeout) `raise` ngay, **không** fallback Laya. `resolve_gate_url` (`:207-214`) chỉ chọn engine 1 lần lúc đầu theo `/health`.
- Laya `:8200` trả predict 11.5s (1 request, khoẻ).
- Vega đang chạy pid 16343, ppid=1 (không do launchd quản), khởi tạo bởi `ai-orchestrator/sidecar/vegaml/ensure_vega_up.sh` (tự dựng venv, ghi `vega_sidecar.pid`/log vào `.orchestrator/vega-data`).

## User APPROVE (2026-11-18)
1. Khởi động lại Vega (tách riêng, start riêng, không phụ thuộc tiến trình orchestrator khác).
2. (Bỏ) đóng app nặng — User xác định không phải giải pháp.
3. Fail-fast: Vega predict quá **3s** → chuyển sang Laya ngay, **không retry** trên Vega.

## Phạm vi
- `agent/tooling/laya_gate.py` (core agent) — thêm fail-fast + fallback ở tầng `post_laya`.
- Không sửa `ai-orchestrator` (chỉ chạy `ensure_vega_up.sh` sẵn có để start lại).

## Ràng buộc Thực thi
- `L001`: kiểm file tồn tại (đã kiểm, workspace mới).
- Rule #19: restore-point `.bak` trước khi sửa `laya_gate.py`.
- Test: `tests/test_vega_primary_flywheel.py` phải pass lại; thêm test cho fail-fast.

## Definition of Done
- Vega start lại, `/health` 200 **và** `/v1/predict` trả < 3s với 1 claim thật.
- `laya_gate.py` chuyển Laya khi Vega timeout 3s, không retry Vega; test mới + test cũ pass (EV-).
- Chạy lại R5 của `review-export-GP2-4633` suffix `gp2_4633_r2`.
