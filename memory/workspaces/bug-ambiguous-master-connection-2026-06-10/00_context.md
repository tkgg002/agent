# 00_context — Fix HTTP 409 ambiguous_master_connection

- **Ngày**: 2026-06-10 · **Vai trò**: Muscle (sửa code — full-loop)
- **Triệu chứng**: `POST https://testing-cdc-cms-api.goopay.vn/api/v1/masters` → **409** `{"error":"ambiguous_master_connection"}`.
- **Root cause (code-cited)**: `create_master.go:240` — `if len(rows) > 1 && code == "" → ErrMasterConnectionAmbiguous`. Resolver query `connection_registry WHERE role_type='master' AND status='active'`; request không gửi `master_connection_code` + DB có **≥2 master active**.
- **Nguồn ≥2 master active (duplicate-provenance)**:
  - `default_master` — CMS Go bootstrap `EnsureDefaultMasterConnection` (ON CONFLICT DO UPDATE → ép active mỗi boot).
  - `master_pg_finance`/`legacy_master_default` — CDS deployment SQL `bootstrap_cdc_system_v2_*.sql`.
  - KHÔNG có constraint chặn nhiều master active.
- **Đã dự báo**: smell này được flag từ phân tích `EnsureDefaultMasterConnection` (workspace inventory 2026-06-08) — nay thành sự cố thật.

## Giải pháp (chặn tái diễn — 2 lớp)
1. **Bootstrap seed-IF-ABSENT**: `master_connection.go` đổi `ON CONFLICT DO UPDATE...active` → `SELECT ... WHERE NOT EXISTS(active master) ON CONFLICT DO NOTHING`. Không tạo master thứ 2, không clobber operator.
2. **Migration 081**: self-heal (giữ 1 master active, ưu tiên connection thật hơn marker `default_master`) + **partial unique index** `uniq_active_master_connection` (role_type WHERE role_type='master' AND status='active') → DB enforce ≤1 active master vĩnh viễn.

## Out-of-scope
- KHÔNG sửa resolver (index đảm bảo không còn >1 → nhánh ambiguous thành dead path).
- Sibling `EnsureDefaultShadowConnection` có cùng pattern → ghi nhận follow-up, chưa sửa (ngoài scope bug này).
