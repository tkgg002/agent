# Context: Bug Sync Master Mapping Rules From Shadow 500 Error

## Overview
Endpoint `POST /api/v1/master-mapping-rules/sync-from-shadow?master_binding_id=1` trả về lỗi `500 Internal Server Error` khi client cố gắng đồng bộ mapping rules từ shadow tables.

Task này nhằm xác định nguyên nhân gốc rễ (Root Cause) của lỗi 500 này, sửa lỗi và đảm bảo quy trình đồng bộ mapping rules hoạt động trơn tru.

## Key Goals
1. **Root Cause Analysis (RCA)**: Tìm chính xác file và dòng code nào gây ra lỗi 500 trong `cdc-cms-service` (hoặc service xử lý route này, có port 8083 là `cdc-cms-service`).
2. **Fix & Refactor**: Sửa lỗi để API hoạt động đúng, tuân thủ các quy tắc thiết kế hệ thống.
3. **Verify**: Kiểm tra lại thông qua unit tests hoặc gọi API trực tiếp để xác thực kết quả.
