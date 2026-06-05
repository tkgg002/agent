# 02_plan

## Roadmap
- M0: Workspace setup (đang chạy)
- M1: Option 1 — edit config-local.yml + config-sample.yml (`info: 0.1 → 1.0`)
- M2: Option 3 — extend severityAwareCore (Check + Enabled honor `audit=true`)
- M3: Option 2 — gắn `zap.Bool("audit", true)` vào 5 call site milestone
- M4: Verify build + vet
- M5: Append progress + lesson

## Quyết định
- ADR-01: Audit field key = literal string `"audit"`, value boolean. KHÔNG export constant ra package consumer vì zap field key thường inline. Add comment cho discoverability.
- ADR-02: Audit bypass áp dụng cho cả `Enabled(level)` (để zap không skip Check) — nhưng vì `Enabled` không có entry context (không có fields), Enabled chỉ trả về true khi không muted. Wait: hiện tại `Enabled` trả false nếu ratio<=0. Vì audit log thường là Info (ratio=0.1>0), Enabled vẫn pass. Không cần đụng Enabled. Chỉ cần fix `Check`.
- ADR-03: Audit field KHÔNG bypass mute. Khi degraded, vẫn drop. Lý do: degraded = sink unhealthy, force-push = waste + làm overflow tệ hơn.
- ADR-04: Không đụng config-production (giữ info: 0.05). Để operator chủ động bump nếu cần.
