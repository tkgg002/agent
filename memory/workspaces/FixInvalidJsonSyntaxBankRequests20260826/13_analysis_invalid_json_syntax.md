# 13_analysis_invalid_json_syntax.md: Phân tích Kỹ thuật Chi tiết

## Incident Details
- **Subject / Topic**: `vmg-ekyc-connector-service.bank-requests`
- **Target Table**: `shadow_vmg_ekyc.bank_requests`
- **Document ID**: `6a8e3fc31b3d2729eb078a60`
- **PostgreSQL Exception**: `ERROR: invalid input syntax for type json (SQLSTATE 22P02)`

## Deep Technical Analysis
1. **Lỗi `SQLSTATE 22P02` trong PostgreSQL**:
   PostgreSQL quăng `SQLSTATE 22P02` (invalid_text_representation) khi nhận parameter cho một column có kiểu `JSON` hoặc `JSONB`, nhưng parameter value không khớp với văn phạm (grammar) JSON hợp lệ.
   Ví dụ các trường hợp vi phạm:
   - Truyền chuỗi rỗng `""` (0 bytes text): Trong Postgres JSON syntax, `""` rỗng KHÔNG PHẢI JSON (JSON string đại diện chuỗi rỗng phải là `"\"\""` - 2 ký tự dấu nháy).
   - Truyền raw string unquoted: `"6a8e3fc31b3d2729eb078a60"` nếu không được quote `json.Marshal` thì postgres parser coi `6a8e3fc31b3d2729eb078a60` là token không hợp lệ (không phải number, boolean, null hay string).
   - Truyền raw Go `[]byte` / `bytea` mà driver không format thành JSON text.
   - Bị miss kiểu `JSONB` trong `IsJSONB` do `DataType` mang chữ hoa hoặc khoảng trắng từ `information_schema` (`"JSONB"`/`"jsonb "`), dẫn đến `CoerceValue` không format JSON cho parameter.

2. **Cơ chế Khắc phục**:
   - Normalize `DataType` về chữ thường & trim khoảng trắng.
   - Bắt buộc 100% value đi vào cột `JSON`/`JSONB` phải qua `json.Marshal` nếu chưa phải JSON text hợp lệ.
   - Guard cột `_raw_data` fallback `{}` khi rỗng.
