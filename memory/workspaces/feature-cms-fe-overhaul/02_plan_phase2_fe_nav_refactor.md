# Plan — Phase 2 FE Nav Refactor

1. Audit lại route tree và các label đang lệch với V2.
2. Refactor `App.tsx`:
   - bỏ import/page `CDCInternalRegistry` khỏi menu chính
   - thêm redirect mềm `/cdc-internal -> /registry`
   - regroup menu thành `Setup / Operate / Advanced`
3. Dọn operator text ở wizard, registry, connector, master, operations.
4. Chạy build thực tế để bắt lỗi type/runtime bundle.
5. Ghi lại kết quả, residual gaps, và bước tiếp theo trong workspace.
