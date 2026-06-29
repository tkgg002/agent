# Context: Bug Data Integrity Missing Tables

## Problem Description
User reports that at `http://localhost:5173/data-integrity`, some tables are missing or not showing:
1. "Shadow" tables are not showing.
2. "Master" tables are missing `master_centrallized_export_service.export_jobs` and `master_centrallized_export_service_2.export_jobs`.
3. Needs checking in both `cms-service` (`cdc-cms-service`) and `cms-web` (`cdc-cms-web`).

## Active Workspace
`/Users/trainguyen/Documents/work/agent/memory/workspaces/bug-data-integrity-missing-tables-2026-06-29`

## Environment & Tech Stack
- Frontend: React / Vite (`cdc-cms-web` running on port 5173)
- Backend: Go / Hexagonal Architecture (`cdc-cms-service`)
- Database: Postgres (contains schemas like `master_centrallized_export_service`, etc.)
