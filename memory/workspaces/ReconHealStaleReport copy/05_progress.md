# Progress Log - ReconHealStaleReport

## Governance Audit & Root Cause Analysis

- **[2026-06-29 20:07:00] [Brain] Audit**: Governance checklist followed successfully. Workspace folder `ReconHealStaleReport` initialized at the very beginning of the session.
- **Root Cause**: Lệch hệ quy chiếu timestamp giữa Source Agent (sử dụng domain timestamp `lastUpdatedAt`) và Destination Agent (sử dụng metadata `_source_ts` của Debezium sync event). Khi Debezium chạy snapshot/backfill lại gần đây, cột `_source_ts` của 1.410 bản ghi trong Postgres Shadow bị cập nhật thành ngày 29/06/2026, trong khi trường `lastUpdatedAt` thực tế ở cả 2 bên vẫn là ngày 02/02/2026. Window query của đối soát Tier 1 lọc theo `_source_ts` ở Postgres Shadow quét trúng các bản ghi này, nhưng ở Mongo Source lọc theo `lastUpdatedAt` thì không quét trúng, dẫn đến báo khống 1.410 drift ảo.
- **Remediation**: Cập nhật `ReconDestAgent` để hỗ trợ lọc và hash theo đúng cột domain timestamp (nếu có cấu hình ở source) tương ứng với cấu hình ở Source, thay vì fix cứng `_source_ts`.

## Progress Timeline

- **[2026-06-29 20:07:15] [Brain] Action**: Created workspace and initialized governance documentation (`00_context.md`, `01_todo.md`, `02_plan.md`, `05_progress.md`).
- **[2026-06-29 20:12:00] [Brain] Action**: Query DB control plane, shadow DB, and Mongo local. Confirm that shadow schema is `shadow_test1111`, Mongo collection database is `payment-bill-service`. Confirm that 1,410 records in Postgres Shadow have `_source_ts` of June 29, 2026 but `lastUpdatedAt` of Feb 2, 2026. Mongo database has only 5 test records locally, but schema structure matches.
- **[2026-06-29 20:14:29] [Brain] Action**: Formulated implementation plan and requested user approval.
- **[2026-06-30 09:33:00] [Brain] Action**: Completed execution of all items: synchronized window filtering, dynamic timestamp fields extraction, dynamic projection, and heal Segment B FQN resolution. Passed all tests.
- **[2026-06-30 09:34:00] [Brain] Action**: Restarted CDC worker and successfully triggered a manual heal for `payment_bills`, correcting 68 drifted records. Verified end-to-end functionality. Workspace closed as completed.

