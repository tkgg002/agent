# 00_context.md — So sánh cdc-control vs hệ thống CDC hiện tại

## Bối cảnh
- **Date**: 2026-05-19
- **Workspace**: `feature-cdc-control-vs-cms-comparison-2026-05-19`
- **Yêu cầu user**: "xem qua cái `/Users/trainguyen/Documents/work/data-hub/cdc-control` rồi so sánh với cdc hiện tại, rồi làm tài liệu xem nó có gì khác, gì chi tiết hơn thôi. ko thực hiện bất cứ dòng code nào"
- **Scope**: PURE DOCUMENTATION — KHÔNG sửa source code 3 repo.

## 3 Repo trong phạm vi

| Repo | Path | Vai trò |
|------|------|---------|
| `cdc-control` | `/Users/trainguyen/Documents/work/data-hub/cdc-control` | Hệ thống cũ — control plane Python cho Kafka Connect (source → shadow MongoDB) |
| `cdc-cms-service` | `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service` | Hệ thống mới — Go backend CMS hexagonal (Postgres `cdc_dw`/`cdc_shadow`) |
| `cdc-cms-web` | `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web` | Hệ thống mới — React FE quản lý connectors + CMS |

## Mục tiêu tài liệu
1. Liệt kê đầy đủ feature `cdc-control` có.
2. Liệt kê đầy đủ feature `cdc-cms-service + cdc-cms-web` có.
3. So sánh từng feature theo bảng — đánh dấu CÓ/KHÔNG/PARTIAL.
4. Chỉ ra **những gì cdc-control chi tiết hơn** so với hệ thống mới.
5. Chỉ ra **những gì hệ thống mới có** mà cdc-control không có.
6. Note design difference (Python vs Go, SSR Jinja2 vs React SPA, Mongo shadow vs Postgres shadow…)

## Non-goals
- Không recommend migration plan.
- Không sửa code source.
- Không quyết định ưu tiên build feature gì.
