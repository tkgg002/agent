---
tier: T2
mode: standard
workspace: audit-consumer-retry-and-smoke-recon-2026-11-05
suffix: 2026_11_05_consumer_miss_smoke_recon
---

# 01. Requirements — Kiểm tra & Sửa lỗi Luồng Nút Heal (`source_conn` & `ExecuteHealModal`) (`2026_11_05_consumer_miss_smoke_recon`)

## 1. Yêu cầu của User
- Kiểm tra cơ chế chạy lại consumer/DLQ và luồng Smoke Recon 2h.
- Truy vết nguyên nhân nút **Heal** (`POST /api/reconciliation/execute-heal`) trả về `rows_affected = 0` trên các connector 5-part (`topic_prefix` riêng như `cdc.goopay.core_trans_proxy_history_service`).
- **Yêu cầu thực thi (`APPROVE` — "fix sửa ngay cho tao"):** Sửa dứt điểm lỗi thiếu `"source_conn"` và `WithBindingScope` trong `recon_heal_fetch.go` (`centralized-data-service`) và lỗi thiếu `shadow_schema`/`shadow_table` trong `ExecuteHealModal.tsx` & `useReconStatus.ts` (`cdc-cms-web`).

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **Frontend Call-Site Tracing (`L001`):** Đồng bộ đầy đủ payload từ `DataIntegrity.tsx` → `ExecuteHealModal.tsx` → `useReconStatus.ts` → `POST /api/reconciliation/execute-heal`.
- **Deployed-vs-Local Diff Grounding (`L002`):** Đảm bảo `recon_heal_fetch.go` truyền `"source_conn"` và `WithBindingScope` giống hệt `snapshot_runner_handler.go:560, 821-822` để không bị `ResolveSourceRoutes` (`metadata_registry_service.go:486-500`) lọc bỏ route.
- **Zero-Mockup Evidence (`L032`):** Chạy test/build thật qua `evidence_run.py` để lấy biên lai `EV-...` trước khi báo Done.
- **Source-Engine Dispatch In Heal Fetch & No Connection-Bypass Unit Testing (`L001`):** Bắt buộc kiểm tra loại nguồn (`entry.SourceType` / `entry.SourceURL`) trong `FetchAndWriteByIDsWithAccepted` để phân nhánh sang trình đọc PostgreSQL (`GetPGClientForURL`) khi nguồn là `postgres://` / `postgresql://` (như `shadow_testcoretrans.transaction_history`); tuyệt đối cấm tự viết unit test bỏ qua bước mở kết nối nguồn rồi báo cáo láo đã pass.
