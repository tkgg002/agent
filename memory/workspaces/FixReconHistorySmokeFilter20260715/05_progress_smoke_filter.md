# Nhật ký tiến độ - Lọc lịch sử đối soát theo smoke check

- [2026-07-15 09:13:00] [Brain:Gemini-Pro] Khởi tạo workspace và tài liệu thiết kế.
- [2026-07-15 09:27:00] [Muscle:Claude-Sonnet] Thực thi: thêm checkTypeFilter vào GetTableHistory (recon_read_repo_gorm.go). excludeSmoke=false → IN ('smoke','segment_b_smoke'); excludeSmoke=true → NOT IN.
- [2026-07-15 09:30:00] [Muscle:Claude-Sonnet] Verify: go vet ./internal/... → OK. Không lỗi biên dịch.
