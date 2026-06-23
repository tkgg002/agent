# Refactor Plan: `internal/` → 7 Domain Groups

> **Phương pháp**: Inside-Out — Model → Repo Interface → Service → Handler  
> **Nguyên tắc**: Move files, không rewrite. Compile-check sau mỗi domain.

## Thứ tự thực hiện (dependency order)

| Phase | Domain | Lý do ưu tiên |
|---|---|---|
| P1 | `source` | Không phụ thuộc domain nào — foundation |
| P2 | `schema` | Phụ thuộc `source` (source_object_id) |
| P3 | `ingestion` | Phụ thuộc `source` + `schema` |
| P4 | `discovery` | Phụ thuộc `source` + `schema` |
| P5 | `transmute` | Phụ thuộc `schema` + `ingestion` |
| P6 | `recon` | Phụ thuộc tất cả domain trên |
| P7 | `platform` | Cross-cutting — làm cuối |

## Cấu trúc mỗi domain folder

```
internal/<domain>/
├── model.go           ← GORM entity structs (move từ model/)
├── repository.go      ← Port interface (mù GORM) — TẠO MỚI
├── repository/
│   └── gorm_<entity>_repo.go  ← GORM implementation (move từ repository/)
├── service/
│   └── *.go           ← business logic (move từ service/)
└── handler/
    └── *.go           ← NATS handlers (move từ handler/)
```

## Quy tắc move

1. **Package name**: giữ nguyên package cũ trong từng file (ví dụ `package service`) → chỉ thay import paths
2. **Không rewrite logic**: chỉ di chuyển file + cập nhật import
3. **Compile gate**: sau mỗi domain, chạy `go build ./...` trước khi tiếp tục
4. **Test gate**: sau mỗi domain, chạy `go test ./internal/<domain>/...`

## Files chi tiết

- [01_source.md](./01_source.md)
- [02_schema.md](./02_schema.md)
- [03_ingestion.md](./03_ingestion.md)
- [04_discovery.md](./04_discovery.md)
- [05_transmute.md](./05_transmute.md)
- [06_recon.md](./06_recon.md)
- [07_platform.md](./07_platform.md)
