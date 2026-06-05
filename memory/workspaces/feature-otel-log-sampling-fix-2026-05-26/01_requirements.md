# 01_requirements

## Functional
- R1: Local dev: 100% Info log forward sang SigNoz (parity với stdout).
- R2: Production: giữ Info sampling 0.05 nhưng milestone audit log KHÔNG bị drop.
- R3: Audit log identification = field `audit=true` (boolean zap field).
- R4: Mute (fallback degraded) VẪN applies cho audit log — degraded mode đồng nghĩa OTel sink unhealthy, không thể bypass.

## Non-functional
- N1: KHÔNG thay đổi public API của observability package.
- N2: Backward compat: log không có field `audit` vẫn hoạt động như cũ.
- N3: Performance: thêm 1 lần iterate `[]zapcore.Field` ở Check → O(n) với n thường < 10. Acceptable.
- N4: KHÔNG log raw URI/password (theo L-2026-05-19).

## DoD
- A1: `go build ./... && go vet ./...` exit 0.
- A2: Config local/sample sửa, production giữ nguyên.
- A3: 5 call site milestone gắn `zap.Bool("audit", true)`.
- A4: `severityAwareCore.Check` + `Enabled` honor audit field.
- A5: Append `05_progress.md` + lesson global.
