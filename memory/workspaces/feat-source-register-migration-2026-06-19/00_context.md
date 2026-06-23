# Workspace Context: Source Register Migration & Refactor

## Scope & Objective
- Di chuyển handler đăng ký Source Object (`source_register.go`) từ `internal/admin` sang đúng vị trí kiến trúc tại `internal/handler/source/source_register.go`.
- Áp dụng các refactoring cho truy vấn PostgreSQL `RETURNING` để tránh N+1 query.
- Tích hợp context (`WithContext`) đầy đủ cho các câu lệnh GORM.
- Đảm bảo tính cô lập và cấu trúc Screaming Architecture.

## Tech Stack & Dependencies
- Go (Golang)
- Gin Gonic HTTP web framework
- GORM (PostgreSQL)
- NATS Go client
