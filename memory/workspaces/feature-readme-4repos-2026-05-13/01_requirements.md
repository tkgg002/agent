# 01 Requirements — README refresh 4 repo

## R1 Mỗi repo có README.md (uppercase)
- `cdc-auth-service/README.md`
- `cdc-cms-service/README.md`
- `cdc-cms-web/README.md` (overwrite default Vite template)
- `centralized-data-service/README.md` (mới; giữ `readme.md` cũ chỉ khi chứa nội dung không trùng).

## R2 Mỗi README có tối thiểu các section
1. Tên service + mô tả 1 dòng + vị trí trong CDC pipeline.
2. Tech stack (language, framework, store).
3. Cấu trúc thư mục (mức 1, chỉ thư mục quan trọng).
4. Yêu cầu môi trường (Go version / Node version / Docker).
5. Cách chạy local (Makefile target hoặc lệnh trực tiếp).
6. Cấu hình (env / yaml chính).
7. Endpoints hoặc commands (HTTP/NATS subjects/CLI).
8. Test (cách chạy).
9. Liên kết tham chiếu (`architecture.md`, doc nội bộ).

## R3 Definition of Done
- 4 file `README.md` tồn tại vật lý dưới đường dẫn quy định.
- Mỗi file ≥ 80 dòng, < 400 dòng.
- Đọc lại bằng `Read` tool xác nhận content khớp.
- APPEND `05_progress.md` với entry kèm timestamp + file path.
- Không sửa source code, không commit (chờ Boss `commit docs`).

## R4 Out of scope
- Không tạo `docs/` mới ngoài README cấp repo.
- Không sửa root `cdc-system/README.md` (Boss chưa request).
- Không sửa `agent/redme.md`.
