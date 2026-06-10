# 05_progress — Inventory cdc-cms-service (APPEND-ONLY)

## [2026-06-08] Phân tích cấu trúc + dò file chưa dùng
- Recon: 250 .go (213 non-test), Go 1.26.1, `go build ./...` PASS, `deadcode` chạy offline → 75 hàm unreachable.
- Reachability (`go list -deps` từ cmd/server+cmd/sync_v2): 42 pkg, 32 reachable; UNREACHABLE non-test = `internal/infra/cache` (dead).
- Fan-out 7 sub-agent (Rule 20) phân tích chức năng + status 213 file → 7 part-file `/tmp/inv_*.md`, mỗi agent trả compact dead-list.
- Assembly → `11_current_structure.md` (493 dòng, 274 file-row). Tổng hợp → `12_unused_files.md`. Đề xuất reorg → `13_proposed_structure.md`.
- KẾT QUẢ DEAD: nhóm A 8 file/575 LOC (cache/doc, master_swap 192, registry_mirror 260, model/cdc_event, ports/query_bus, ports/publisher, utils/hash, utils/type_inference); nhóm B 2 (cmd/sync_v2 one-shot, api/registry_handler_read 2 endpoint không mount); nhóm C ~30 hàm (*ForTest do test không co-located + .Type() do query-bus chưa wire); nhóm D 4 cặp trùng model/↔domain/.
- Ràng buộc: KHÔNG sửa .go (chỉ phân tích + đề xuất). Xoá file = chờ user duyệt → 1 PR "remove dead code" + verify go build/deadcode lại.
