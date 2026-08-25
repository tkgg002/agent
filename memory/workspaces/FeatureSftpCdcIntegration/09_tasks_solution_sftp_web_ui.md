# 09_tasks_solution_sftp_web_ui.md — Kế hoạch Triển khai Bổ sung SFTP vào cdc-cms-web UI

> **Ngày tạo**: 2026-08-07  
> **Mục tiêu**: Bổ sung lựa chọn "SFTP Source" vào dropdown Database Type và xây dựng Form cấu hình SFTP Source Connector trên giao diện `cdc-cms-web` (`http://localhost:5173/sources`).

---

## 1. Phân tích Nguyên nhân & Phạm vi Thiếu sót

- **Nguyên nhân**: Các bước trước đây chỉ tập trung vào Backend (`cdc-cms-service` và `centralized-data-service`), chưa cập nhật giao diện Frontend React (`cdc-cms-web`).
- **Hậu quả**: Khi mở Modal "New Connect" tại `http://localhost:5173/sources`, dropdown **Database Type** chỉ có 3 lựa chọn: `MongoDB`, `MySQL`, `PostgreSQL`. Không có `SFTP`.

---

## 2. Giải pháp Kỹ thuật Chi tiết (Frontend React + TypeScript)

### A. File `cdc-cms-web/src/types/index.ts`
- Mở rộng type union `source_type` từ `'mongodb' | 'mysql' | 'postgresql'` thành `'mongodb' | 'mysql' | 'postgresql' | 'sftp'`.

### B. File `cdc-cms-web/src/pages/SourceConnectors.tsx`
1. **Types & Options**:
   - `type DbKind = 'mongodb' | 'mysql' | 'postgresql' | 'sftp';`
   - Bổ sung `DB_OPTIONS`: `{ label: 'SFTP Source', value: 'sftp' }`
   - `TOPIC_PREFIX_BY_DB`: `sftp: 'sftp.reconcile'`
2. **Detection & Build Config**:
   - `detectDbKind`: Nhận diện `sftp` từ `sourceType` hoặc `SftpSourceConnector` từ `connector_class`.
   - `buildConnectorConfig`: Khi `dbKind === 'sftp'`, sinh JSON config chuẩn cho `io.confluent.connect.sftp.SftpSourceConnector`:
     - `connector.class`: `"io.confluent.connect.sftp.SftpSourceConnector"`
     - `tasks.max`: `"1"`
     - `sftp.host`, `sftp.port` (default 2022), `sftp.username`, `sftp.password`
     - `host.key.verify`: `"false"`
     - `input.path`: ví dụ `/goopay/reconcile_final`
     - `input.file.pattern`: ví dụ `^reconcile_final_.*\.csv$`
     - `format.class`: `"io.confluent.connect.sftp.parser.CsvFormat"`
     - `csv.first.row.as.header`: `"true"`
     - `topic.prefix`: `sftp.reconcile.final`
     - `behavior.on.error`: `"IGNORE"`
   - `extractFormValues`: Reverse mapping để khi bấm **Edit Config**, Form điền sẵn các giá trị SFTP cũ.
3. **Form Component UI**:
   - Khi `dbKind === 'sftp'`, render các trường:
     - SFTP Host & Port (mặc định 2022)
     - Username & Password
     - Input Path (Thư mục SFTP chứa file)
     - Input File Pattern (Regex nhận diện file CSV)
     - Finished Path & Error Path

### C. File `cdc-cms-web/src/pages/TableRegistry.tsx`
- Thêm tag hiển thị loại DB `sftp` màu `purple` và option filter cho `sftp`.

---

## 3. Kế hoạch Phân công Thực thi (Brain / Muscle Protocol)

1. **Brain**: Trình bày Plan này và chờ lệnh **APPROVE** của User.
2. **Muscle**: Sau khi có lệnh Approve, Muscle subagent sẽ áp dụng code vào `cdc-cms-web`.
3. **Verification**: 
   - Verify build frontend với `npm run build` trong `cdc-cms-web`.
   - Mở modal New Connect trên UI để xác nhận dropdown hiển thị "SFTP Source" và form nhập liệu hoạt động chuẩn.
