# Project Context

> **Last Updated**: 2026-05-04
> **Maintained by**: Brain (Antigravity) qua workspace `feature-system-refactor-2026-05`

## Overview

**cdc-system** — Hệ thống Change Data Capture (CDC) đồng bộ dữ liệu thay đổi từ
nguồn vận hành (MongoDB / PostgreSQL / MariaDB) sang Data Warehouse PostgreSQL,
qua 2 tầng `shadow` → `master`, có schema evolution kiểm soát, reconciliation,
DLQ retry, và operator UI.

- **Scale**: monorepo 4 service (3 Go + 1 TS), ~250 .go file + 22 .tsx (~7600 LOC FE).
- **Users**: operator/admin nội bộ qua CMS UI; developer Devops ops backend.
- **Stage**: Development / Local smoke (chưa staging hoặc production deploy).
- **Repo**: 1 git monorepo, branch `main`, ngưỡng 6 commit (lịch sử ngắn).