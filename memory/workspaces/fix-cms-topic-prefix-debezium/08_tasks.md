# 08 - Tasks Breakdown

- [x] **TASK-01:** Rà soát và loại bỏ việc ghép `connector_name` vào MongoDB topic prefix fallback trong `parseConnectionSeed` (`SourceConnectors.tsx`).
- [x] **TASK-02:** Rà soát và loại bỏ việc ghép `connector_name` vào PostgreSQL/MySQL topic prefix fallback trong `parseConnectionSeed` (`SourceConnectors.tsx`).
- [x] **TASK-03:** Cập nhật `useEffect` auto-fill khi mở modal tạo mới: Chỉ gán base prefix cho Debezium, giữ nguyên auto-slugify cho SFTP.
- [x] **TASK-04:** Mở khóa input `topicPrefix` (`disabled={dbKind === 'sftp'}`) và bổ sung Tooltip hướng dẫn xử lý xung đột tên topic.
- [x] **TASK-05:** Verify TypeScript compilation (`npx tsc --noEmit`).
- [x] **TASK-06:** Audit tuân thủ Governance `/agent/GEMINI.md`, ghi bài học vào `lessons.md`, khởi tạo trọn bộ workspace docs.
