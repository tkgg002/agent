# 00_context — Fix recon stale-runs + source-URI warn noise

**Ngày**: 2026-06-11 · **Agent**: Muscle (Claude-Opus-4.8) · **Trigger**: user "làm đi" (đồng ý xử lý 2 vấn đề pre-existing phát hiện khi verify orphan-prune).

## Bối cảnh phát sinh
Khi verify endpoint orphan-prune (workspace `feature-masters-page-audit-2026-06-02`), đọc log worker `/tmp/cdc-worker-recon-p4g.log` lộ 2 vấn đề pre-existing (KHÔNG do prune):
1. **`recon_runs` treo**: 7 dòng `status='running'` không bao giờ đóng → unique index `recon_runs_one_running` (theo `table_name`) chặn run mới của bảng đó → log `tier1 beginRun failed: duplicate key ... 23505` lặp lại; dashboard hiển thị Source=NULL cho các bảng kẹt.
2. **Warn lặp mỗi ~60s**: `connection_registry: cannot resolve source URI; recon/snapshot will skip sources bound to this connection` cho `default_master` (engine=postgresql) — "no usable DSN in secret_ref nor host/port fields".

## Hệ thống liên quan
- Service: `centralized-data-service` (CDC worker, port 8082), binary chạy `/tmp/cdc-worker-recon-p4g` (PID 90730, detached ppid=1, log `.log` cùng tên).
- Control DB: `gpay-postgres-cdc` localhost:5433 db=`cdc_dw` schema `cdc_system` (creds gpay_admin/*** — masked).
- Bảng: `cdc_system.recon_runs`, `cdc_system.connection_registry`, `cdc_system.source_object_registry`.

## Ràng buộc (standing)
- KHÔNG commit/push. KHÔNG cheat DB / fabricate config để fake kết quả. Trả lời tiếng Việt. APPEND-only memory. Mask secret/PII (Rule 19). Full-loop tự chủ (Rule 2).
