# 09_tasks_solution_phase_p0 — Hồ sơ giải pháp P0

## M-1: Migration mask_strategy
- **Root cause**: `cdc_mapping_rules` không có per-field strategy → bị ép 1 chiến lược duy nhất (`"***"`).
- **Solution**: ENUM TYPE + 3 column + 2 audit table.
- **Lý do ENUM**: ADR-007 — type-safe ở DB layer, ép constraint kể cả khi bypass app.
- **Seed default**: classify theo tên field, fallback DROP (an toàn nhất).

## M-2: Strategy interface
- **Root cause**: Logic masking nằm rải rác trong `masking_service.go` với hardcode `"***"`.
- **Solution**: Strategy pattern + Registry.
- **Lý do 4 strategy mà không nhiều hơn**:
  - NONE: cần cho field không nhạy cảm.
  - DROP: theo NĐ 356 "loại bỏ dữ liệu không cần thiết".
  - HASH_HMAC: theo Luật 91/2025 "De-identification" + giữ tính đối soát.
  - PARTIAL: theo VBHN 25 "hiển thị một phần cho audit".
- **Tokenize defer**: cần FPE vault, scope sau.

## M-3: HMAC key vault
- **Root cause**: Hash mà không có secret key → rainbow table tấn công được.
- **Solution**: `KeyProvider` interface + env-based loader ban đầu.
- **Lý do env vs Vault**: ADR-002 — đủ compliance hiện tại, không over-engineer.
- **Anti-pattern tránh**: hardcode key trong source — fail nếu key < 32 chars, force operator inject đúng.

## M-4: MaskingService refactor
- **Root cause**: 3 nơi hardcode `"***"` (lines 91, 133, 152-153) + 1 path log (acceptable).
- **Solution**: Dispatch qua Registry, DROP set `nil` thay vì literal.
- **Lý do giữ text_sanitizer.go**: ADR-008 — log không phải DB persistence, không vi phạm Điều 13 Accuracy.
- **Audit emit non-blocking**: tránh block hot path nếu DB chậm. Channel buffered + dropped khi full.

## M-5: Unit test
- **Root cause**: Test hiện tại đều assert `"***"` literal → confirm anti-pattern.
- **Solution**: Test mới assert `NotEqual("***", ...)` để khóa regression.
- **Coverage ≥ 90%**: bắt buộc vì compliance critical.
- **Deterministic HMAC**: assert same input → same output, chứng minh tính đối soát.

## Tổng impact P0
- Loại bỏ hoàn toàn `"***"` khỏi DB write-path.
- Strategy engine sẵn sàng cho 4 use case ngành Fintech VN.
- Foundation cho P1 (worker integration) + P2 (UI + backfill).
