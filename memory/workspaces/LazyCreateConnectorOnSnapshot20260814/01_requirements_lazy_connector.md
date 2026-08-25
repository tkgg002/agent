# Requirements: Lazy Create Connector On Snapshot (SFTP ONLY Scope)

## Overview
Chuyển đổi kiến trúc khởi tạo Kafka Connectors từ "Eager Creation" sang "Lazy Creation" **CHỈ ÁP DỤNG DUY NHẤT CHO NGUỒN SFTP / FILE STREAM**.
TẤT CẢ các Nguồn dữ liệu Database SQL / NoSQL khác (MongoDB, PostgreSQL, MySQL, Oracle, SQL Server...) **GIỮ NGUYÊN 100% LUỒNG TẠO CONNECTOR CŨ (Eager Creation)**.

## Detail Requirements
1. **Phân tách Scope kết nối (Strict Scope Isolation):**
   - Kiểm tra `isSFTP`: `sourceType == "sftp"` HOẶC `connector.class` chứa `FsSourceConnector` / `Sftp`.

2. **Luồng cho SFTP (`isSFTP == true`):**
   - **Tạo Connection:** Chỉ validate và lưu cấu hình/fingerprint vào DB (`sourceRepo.Upsert`) với `Status = "configured"`. KHÔNG gọi Kafka Connect REST API `Create`.
   - **Comment out code cũ:** Giữ nguyên (comment out) mã nguồn tạo SFTP Connector trực tiếp và mã nguồn `pause`/`resume` với annotation `// LEGACY SFTP EAGER & PAUSE FLOW - PRESERVED`.
   - **Snapshot / Ingestion:** Khi kích hoạt Snapshot / Ingest file SFTP, helper `EnsureSFTPConnectorCreated` kiểm tra nếu `testsftpXX` chưa có trên Kafka Connect -> Lấy config từ DB ra và khởi tạo Connector trên Kafka Connect ngay lúc đó.

3. **Luồng cho Database Types khác (`isSFTP == false`):**
   - **GIỮ NGUYÊN 100% LUỒNG CŨ**: Gọi `h.writer.Create(ctx, cmd.Name, cmd.Config)` ngay lập tức khi đăng ký Connector từ UI/API, upsert fingerprint audit trail như cũ. KHÔNG BỊ ẢNH HƯỞNG BỞI CODE MỚI.
