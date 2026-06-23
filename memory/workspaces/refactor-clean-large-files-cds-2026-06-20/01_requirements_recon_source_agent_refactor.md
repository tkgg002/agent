# 01_requirements_recon_source_agent_refactor

Yêu cầu chi tiết cho quá trình phân tách và tối ưu hóa file `internal/service/recon/recon_source_agent.go` (1166 dòng).

## 1. Mục tiêu
- Phân rã file `recon_source_agent.go` thành các file nhỏ chuyên biệt theo chức năng để dễ đọc, bảo trì và kiểm thử.
- Thu hẹp kích thước file gốc xuống dưới 300 dòng.
- Không thay đổi hành vi logic bên trong, không thay đổi cấu hình, đảm bảo tính toàn vẹn của mã nguồn ("Simplicity First, minimal impact").

## 2. Yêu cầu kỹ thuật
- Giữ nguyên `package recon`.
- Không sửa đổi signature của các struct và public method phục vụ bên ngoài, bao gồm:
  - Các struct: `ChunkHash`, `WindowResult`, `BucketHashResult`, `ReconSourceAgentConfig`, `ReconSourceAgent`.
  - Các hằng số mã lỗi: `ErrCodeSrcTimeout`, `ErrCodeSrcConnection`, etc.
  - Các method của `ReconSourceAgent`: `CountDocuments`, `EstimatedCount`, `BucketCounts`, `CountInWindow`, `CountInWindowWithFallback`, `HashWindow`, `BucketHash`, `ListIDsInWindow`, `ListAllIDs`, `StreamAllIDs`, `MaxWindowTs`, `GetChunkHashes`.
  - Các test helpers: `ClassifyMongoErrorForTest`, `HashIDPlusTsForTest`, `HashIDPlusTsMsForTest`, `BucketIndexForTest`.
- Mọi file helper mới phải định nghĩa cùng `package recon` nên có thể truy cập lẫn nhau trực tiếp thông qua con trỏ receiver `(sa *ReconSourceAgent)`.
- Biên dịch thành công 100% bằng `go build ./...` và pass toàn bộ unit tests dự án bằng `go test ./...`.
