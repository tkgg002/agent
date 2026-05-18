# 02_plan — Audit config-local.yml

## Phương pháp

1. **Locate config loader** → `centralized-data-service/config/config.go` (viper-based, struct `AppConfig`).
2. **Trích map mapstructure tag → field** từ struct.
3. **Đối chiếu key YAML** với mapstructure tag:
   - Match → field hợp lệ, tiếp tục bước 4.
   - No match → Viper silently drop → key DEAD ngay từ tầng parse.
4. **Grep caller** mỗi field trong `cmd/`, `internal/`, `pkgs/`:
   - Có caller ≥1 ngoài chính `config.go` → ACTIVE.
   - Chỉ có caller trong `config.go` (set qua env, validate, fallback) → mark "set-only, no reader" = DEAD downstream.
5. **Đặc biệt** với map (`Sources`) và struct method (`SourceURL`):
   - Tìm caller method, không phải field.
6. **Đối chiếu lesson 2026-05-05** "validateConfig BEFORE applyFallbacks": confirm pipeline còn đúng order.

## Tools

- `Read` để load `config.go`, `config-local.yml`.
- `grep -rn` (Bash) cho từng struct field.
- `TaskCreate/Update` track progress.

## Risk

- False-negative DEAD: nếu caller dùng reflection hoặc tag inspection. Đã kiểm tra → không có reflection trong worker plane.
- False-positive ACTIVE: nếu caller chỉ là test file. Đã filter trong báo cáo (note rõ "test only" khi áp dụng).
