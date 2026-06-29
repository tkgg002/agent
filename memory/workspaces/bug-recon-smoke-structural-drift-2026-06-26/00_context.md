# Context: Khắc phục lệch kiến trúc Recon Smoke

## Vấn đề hiện tại
- Bảng `recon_smoke_tables` (và các logic đi kèm trong `recon_smoke_model.go`, `recon_smoke_repo.go`, `recon_smoke.go`) đang có các lệch lạc về mặt kiến trúc:
  1. SQL migration `002_recon_smoke_tables.sql` chưa có giao dịch transaction (`BEGIN`/`COMMIT`), kiểu dữ liệu chưa viết hoa chuẩn và khoá ngoại `fk_smoke_result_cycle` đang khai báo inline cũ.
  2. Struct model `SmokeResult` và `CycleSummary` đang dùng kiểu dữ liệu không đồng nhất cho trường `ID` và `CycleID` (cần chuẩn hoá sang `uint64` và `*uint64`).
  3. Repository layer `LinkSmokeResultsToCycle` nhận kiểu `uint64` không khớp.
  4. Khởi tạo `ReconSmokeRepo` đang được thực hiện cục bộ bên trong service `recon_smoke.go` thay vì thông qua cơ chế Dependency Injection (DI) tại constructor `NewReconCoreWithConfig`.

## Mục tiêu
- Chuẩn hoá schema SQL migration và struct model.
- Áp dụng Dependency Injection cho `ReconSmokeRepo` vào `ReconCore` tại file khởi tạo server.
- Đảm bảo hệ thống biên dịch thành công không lỗi compile.
