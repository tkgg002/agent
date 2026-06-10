# 00_context — Inventory cấu trúc + dò file chưa dùng `cdc-cms-service`

- **Ngày**: 2026-06-08 · **Vai trò**: Muscle (phân tích, KHÔNG sửa code)
- **Yêu cầu User**: "xuất lại cấu trúc thư mục hiện tại + chức năng chi tiết từng file + đề xuất sắp xếp lại; mục tiêu khi sắp xếp lại sẽ lộ ra những file CHƯA DÙNG tới."
- **Target**: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service` (module `cdc-cms-service`, Go 1.26.1)
- **Liên quan**: nối tiếp `feature-cdc-cms-service-restructure-2026-05-19` (v1) + `feature-cdc-cms-hexagonal-refactor-2026-06-01` (v2, 8 Bounded Context). Workspace này CHỈ làm inventory + dead-code, KHÔNG ghi đè v1/v2.

## Số liệu thực đo (recon)
- 250 .go (213 non-test + 37 test). ~34K LOC non-test.
- `go build ./...` = PASS. `deadcode` chạy được offline → **75 hàm unreachable**.
- **Reachability**: 42 package non-test, 32 reachable từ 2 binary (cmd/server, cmd/sync_v2). UNREACHABLE: `internal/infra/cache` (dead thật, 0 import) + 9 package `test/*` (bình thường).
- Test KHÔNG co-located: nằm ở cây riêng `test/internal/...` (37 file).

## Phương pháp dò "file chưa dùng" (3 tầng, dữ liệu thật)
1. **Reachability** (`go list -deps`): package không vào binary nào.
2. **deadcode tool**: hàm unreachable từ main → file có TẤT CẢ hàm chết = file thừa.
3. **grep reference**: symbol exported 0 tham chiếu ngoài file (chú ý handler wired qua router/bus).
