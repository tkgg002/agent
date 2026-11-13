# 08 Tasks — source-entropy
- [ ] P0.1 Freeze feature, tạo nhánh `stabilize/2026-11`, commit snapshot local (DoD: `git status` sạch, không push)
- [ ] P1.1 `.agent-project.json` cho 3 repo (DoD: file tồn tại, lệnh chạy được)
- [ ] P1.2 Di dời `docs/*_bk*`, `docs/recon_legacy`, `scratch/` khỏi module (DoD: `go build ./...` + `go vet ./...` exit 0)
- [ ] P1.3 Gắn `-short` skip có log cho test Live/Real (DoD: `go test -short ./...` không còn fail do thiếu hạ tầng)
- [ ] P1.4 Ghi `BASELINE_FAILS.txt` (DoD: khớp output `go test -short ./...`)
- [ ] P2.x Mỗi test đỏ 1 task Red→Green (DoD: baseline giảm 1 dòng/commit)
- [ ] P3.1 `ARCHITECTURE.md` + arch-lint gate (DoD: gate chạy trong `checks`)
- [ ] P4.1 Đề xuất hook core chạy manifest `checks` trước DONE
