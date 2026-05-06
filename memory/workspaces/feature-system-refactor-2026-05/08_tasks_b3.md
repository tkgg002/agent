# B3 — Tasks Table

| # | Task | File loại | Owner | Status | Notes |
|---|---|---|---|---|---|
| 103 | Revive Redpanda Console | docker-compose | Brain | ✅ done | image v2.7.2, port 18088 |
| 104 | Full Doc Set workspace B3 | md | Brain | 🟡 in_progress | this file |
| 105 | cms `system_health_collector.go` /health → /healthz | go | Muscle | pending | line 267 |
| 106 | cms `prom_client.go` /metrics 401 graceful | go | Muscle | pending | line 200 |
| 107 | cms-service `Makefile` migrate target | mk | Muscle | pending | inspect first |
| 108 | cms `config-local.yml` `kafkaExporterUrl=""` | yml | Brain | pending | clear stub |
| 109 | worker `schema_adapter.go` auto-CREATE shadow V1 | go | Muscle | pending | per plan curried-waddling-spindle P2 |
| 110 | `036_prune_legacy_v1.sql` write + apply | sql | Brain+Muscle | pending | per plan P3 |
| 111 | Operator add-source-DB inventory doc | md | Brain | pending | `03_implementation_b3_operator_flow.md` |
| 112 | Smoke 3 engine Mongo / MariaDB / PG | bash | Brain | pending | E2E real data |
| 113 | Verify zero-error post-fix | bash | Brain | pending | AC-B3-1..9 |
| 114 | Report + APPEND + lesson | md | Brain | pending | `report_phase_b3_completed_*.md` |

## Dependencies

```
103 → done
104 → in_progress (will close right after writing 4 doc files)
107, 108 → can run parallel (low-risk)
105, 106 → after cms restart capacity (atomic restart)
109 → worker rebuild needed (last cms-side fix)
110 → independent SQL apply
111 → research only (no exec)
112 → after 109 + 110 (need shadow auto-create + clean registry)
113 → after all fixes
114 → last
```

## Verify checklist (per task)

- 105: `curl :8083/api/system/health | jq .infrastructure.worker?.status` → "up" hoặc absent (now `/healthz` parsed OK)
- 106: 401 không gây overall=critical
- 107: `make migrate` exit 0
- 108: alert "kafka exporter unreachable" gone từ alerts list
- 109: drop shadow → insert source → wait 60s → shadow re-created với 8+ V1 cols
- 110: `psql -c "SELECT count(*) ... legacy_% AND is_active=true"` → 0
- 111: file viết đầy đủ 11-step path mỗi engine
- 112: 3 evidence block mỗi engine: source row → shadow row → master row
- 113: AC list từ 01_requirements_b3.md
- 114: report file + APPEND + lesson + commit
