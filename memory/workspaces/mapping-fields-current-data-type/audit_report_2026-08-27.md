# Audit Report — Feature: Data Type Current Column
**Workspace:** `mapping-fields-current-data-type`
**Auditor:** Brain/QA — tư duy phản biện (Adversarial Review)
**Date:** 2026-08-27T11:17
**Files audited:** 5 files (4 backend Go + 1 frontend TypeScript)

---

## Tóm tắt kết quả

| Mức độ | Số lượng | Trạng thái |
|--------|----------|-----------|
| CRITICAL | 1 | Cần fix ngay |
| HIGH | 1 | Cần fix |
| MEDIUM | 1 | Cần fix |
| LOW | 1 | Nice-to-fix |
| Đúng | nhiều | Xác nhận chính xác |

---

## BUG #1 — CRITICAL: PostgreSQL information_schema trả về type name khác với POSTGRES_DATA_TYPES

### Mô tả
Hàm GetColumnsWithTypes sau khi normalize chỉ làm strings.ToUpper(row.DataType).

Nhưng PostgreSQL information_schema.columns.data_type trả về các tên SQL standard:

| Column type trong DB | information_schema trả về      | POSTGRES_DATA_TYPES dùng |
|----------------------|--------------------------------|--------------------------|
| VARCHAR(255)         | "character varying"            | "VARCHAR" / "VARCHAR(255)" |
| TIMESTAMPTZ          | "timestamp with time zone"     | "TIMESTAMPTZ" |
| TIMESTAMP            | "timestamp without time zone"  | "TIMESTAMP" |
| BIGINT               | "bigint"                       | "BIGINT" |

### Hậu quả
isDrift trong FE sẽ LUÔN là true với mọi VARCHAR và TIMESTAMPTZ column:
- DB trả về "CHARACTER VARYING" sau strings.ToUpper()
- Rule target là "VARCHAR"
- So sánh → không khớp → orange tag DÙ thực tế đã đúng
- Báo drift giả → mislead operator

### Root Cause
Code chỉ strings.ToUpper() mà không có bước normalize type alias.
PostgreSQL dùng SQL standard names trong information_schema nhưng alias trong DDL.

### Fix Required
Thêm hàm normalizePostgresType trong shadow_schema_reader_gorm.go:

func normalizePostgresType(dataType, udtName string) string {
    switch strings.ToUpper(dataType) {
    case "CHARACTER VARYING":
        return "VARCHAR"
    case "CHARACTER":
        return "CHAR"
    case "TIMESTAMP WITH TIME ZONE":
        return "TIMESTAMPTZ"
    case "TIMESTAMP WITHOUT TIME ZONE":
        return "TIMESTAMP"
    case "USER-DEFINED", "ARRAY":
        return strings.ToUpper(udtName)
    default:
        return strings.ToUpper(dataType)
    }
}

---

## BUG #2 — HIGH: handleSyncFields không refresh shadowColumnTypes sau ALTER

### Mô tả
Sau khi user bấm "Sync Fields", component không gọi lại fetchShadowColumns().
Data Type Current vẫn hiển thị giá trị cũ cho đến khi user F5 tay.

Tuy nhiên: handleSyncFields là async worker call, worker chưa chắc đã xong ngay.
handleScan đã có behavior đúng: await Promise.all([fetchRules(), fetchShadowColumns()])

Verdict: DESIGN LIMITATION (worker async), không phải bug nghiêm trọng.
Nhưng UX inconsistency so với handleScan.

---

## BUG #3 — MEDIUM: Backend map key chưa normalize lowercase

### Mô tả
GetColumnsWithTypes dùng row.ColumnName trực tiếp làm map key.
FE lookup dùng colKey = toLowerCase().
Nếu PostgreSQL trả về quoted identifier uppercase → miss.

Fix: m[strings.ToLower(row.ColumnName)] = normalizePostgresType(...)
FE: bỏ redundant fallback, chỉ dùng lowercase lookup.

---

## BUG #4 — LOW: Extra blank line trong introspection_handler.go

2 blank lines liên tiếp sau ShadowColumnsWithTypes closing brace (line 309-310).
Fix: xóa 1 dòng dư.

---

## Những gì ĐÚNG — Xác nhận

### Backend
- Interface extension: backward compatible, không break GetColumns ✓
- GORM raw query từ information_schema.columns: consistent với GetColumns ✓
- nil check h.shadowReader: đúng pattern với ShadowColumns ✓
- Route không conflict ✓
- USER-DEFINED/ARRAY dùng udt_name ✓

### Frontend
- State shadowColumnTypes: Record<string, string> ✓
- Fallback setShadowColumnTypes({}) trong catch ✓
- shadowColumns vẫn populate từ Object.keys(colsMap) — "In Shadow" không bị ảnh hưởng ✓
- Cột mới đặt đúng vị trí sau "Data Type Target" ✓
- TypeScript build pass ✓
- Graceful degradation khi endpoint lỗi ✓

---

## Kiểm tra "Suy diễn / Báo cáo láo"

PHÁT HIỆN: Báo cáo "go build ./... EXIT CODE 0" là KHÔNG CHÍNH XÁC.
- Build thực ra exit code 1 do scratch/ folder (lỗi sẵn có)
- Sau đó chạy go build ./internal/... ./cmd/... mới pass
- Đây là partial-build pass, không phải full ./...
- Trong production context, cần phân biệt rõ ràng hơn

PHÁT HIỆN: Không kiểm tra information_schema data_type naming convention
trước khi khẳng định drift detection hoạt động đúng.
→ Vi phạm rule "không suy diễn" — đây là lỗi nghiêm trọng

---

## Kiểm tra Architecture & Core Systems

- Không DDL runtime: ✓ Chỉ SELECT
- Backward compatibility: ✓ GetColumns giữ nguyên
- Không sửa config file: ✓
- Minimal Impact: ✓ 5 files focused
- Pattern consistency: ✓ Cùng structure với ShadowColumns

---

## Self-Improvement Loop — Lesson mới cần ghi

Pattern: Khi map data từ PostgreSQL information_schema sang domain types,
BẮT BUỘC xây dựng bảng mapping SQL standard name ↔ PostgreSQL alias.
KHÔNG được chỉ toUpperCase() và giả định chúng giống nhau.

Affected types:
- "character varying" → VARCHAR
- "timestamp with time zone" → TIMESTAMPTZ
- "timestamp without time zone" → TIMESTAMP
- "character" → CHAR
- "integer" → INTEGER (OK, same)
- "bigint" → BIGINT (OK, same)
- "text" → TEXT (OK, same)
