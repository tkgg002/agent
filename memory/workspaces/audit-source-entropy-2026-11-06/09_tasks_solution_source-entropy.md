---
tier: T2
suffix: source-entropy
status: PROPOSED (chờ APPROVE)
---
# 09 Solution — Chặn "feature mới phá feature cũ" & "báo OK láo"

## A. Phát hiện (VERIFIED — bằng chứng runtime 2026-11-06, log task-11)

| # | Phát hiện | Bằng chứng |
|---|---|---|
| F1 | **Baseline đã ĐỎ sẵn** — mọi lần "OK" trước đây là chạy test cục bộ 1 package, không chạy full suite. | `go test ./...`: centralized-data-service FAIL 9 package (`handler/scan`, `handler/source`, `service/master`, `service/source` ×6 test, `pkgs/natsconn`, `test/internal/activity`, `test/internal/service` build failed…); cdc-cms-service FAIL 6 package (`api/system`, `app/commands/master`, `infra/persistence/system`, `natsconn`, `test/internal/app/queries` build failed). |
| F2 | **Refactor đổi signature/xoá type nhưng không sửa caller/test** → chính là "làm cái mới lỗi cái cũ". | `test/internal/service/recon_pipeline_span_test.go:12` dùng `servicerecon.PipelineFlowInfo` (đã bị xoá); `cdc-cms-service/test/internal/app/queries/queries_test.go:343` stub `GetActiveReconJobs(ctx, string)` vs interface mới `[]string`; `docs/server_bk/worker_server_init.go:681` gọi `NewSnapshotRunner` signature cũ. |
| F3 | **`go build ./...` không thể xanh** vì code chết nằm trong module: `docs/server_bk`, `docs/server_bk2`, `docs/sinkworker_bk`, `docs/recon_legacy` (12 file .go), `scratch/` (nhiều `main` trùng). Agent buộc phải build "chọn lọc" → che lỗi thật. | vet: `scratch/check_conn.go:12 main redeclared`; `docs/server_bk2/server.go:47 undefined handlerrecon.DLQHandler`. |
| F4 | **Working tree không kiểm soát**: nhiều task trộn chung, không commit theo task → không biết thay đổi nào làm vỡ cái gì, không rollback được. | centralized-data-service: 66 file `M` + 68 file `??` (1692+/1532-); cdc-cms-service 43 file; cdc-cms-web 19 file. Commit message gộp "fix nat, recon & heal" (×2), "feat:update master mongo" 40 file/4391 dòng. |
| F5 | **Không có hợp đồng kiểm tra dự án**: không tồn tại `.agent-project.json` ở data-hub → gate G3/G4 (`checks`, `runtime_check`) không có lệnh chuẩn, mỗi agent tự chọn test → "OK" tuỳ hứng. | `find data-hub -maxdepth 3 -name .agent-project.json` = rỗng. |
| F6 | **God-files + song song pattern**: logic dồn vào file 900–1900 dòng; tồn tại song song `legacy`/`v2` cùng nghiệp vụ; handler gọi SQL thô thay vì qua repository. | `service/master/transmuter.go` 1874, `recon_tier_a.go` 1258, `recon_smoke.go` 1245; web `NatsMonitor.tsx` 2597, `ActivityManager.tsx` 2083; `recon_legacy.go` + `recon_dest_legacy.go`, `mapping_rule_v2*.go`; 15 file `internal/handler/**` gọi `.Exec(`/`.Raw(`. |
| F7 | **Hai kiến trúc khác nhau giữa 2 service** (cms-service: `api/app/commands/domain/infra` CQRS-hexagonal; data-service: `handler/service/repository` layered) nhưng không có tài liệu ranh giới → agent "sáng tác" pattern mỗi lần. | Cây `internal/` của 2 repo. |

> Ghi chú trung thực: một số test FAIL có tên `LivePostgres`/`RealPostgres`/`NatsClient_EnsureStreams` có thể do thiếu môi trường DB/NATS khi chạy local — **chưa phân loại**. Nhưng các lỗi *build failed* (F2/F3) là lỗi code chắc chắn.

## B. Root cause (1 câu)
Không có **baseline xanh + một lệnh kiểm tra chuẩn bắt buộc + commit theo task**, nên mọi agent chỉ kiểm phần mình sửa, không thấy regression, và tự định nghĩa "OK".

## C. Phương án duy nhất — "Đóng băng feature, dựng hàng rào, rồi mới dọn"

### Phase 0 — Đóng băng & chụp hiện trạng (map F4)
1. Dừng mọi feature mới trên `recon-heal`.
2. Commit local restore-point toàn bộ working tree hiện tại thành 1 commit `wip(snapshot): pre-stabilize` ở **nhánh riêng** `stabilize/2026-11` (không push). Từ đây mọi thay đổi = 1 task = 1 commit.

### Phase 1 — Hàng rào bắt buộc (map F1, F3, F5)
1. Tạo `data-hub/<repo>/.agent-project.json` khai báo lệnh chuẩn (agent không được tự chọn):
```json
{
  "checks": [
    "go build ./...",
    "go vet ./...",
    "go test ./... -count=1 -short"
  ],
  "integration_checks": ["go test ./... -count=1 -run 'Live|Real' (cần DB/NATS thật)"],
  "runtime_check": "khởi động binary + gọi healthz + grep log ERROR = 0",
  "baseline_file": "test/BASELINE_FAILS.txt"
}
```
2. Đưa code chết ra khỏi module: `git mv docs/*_bk* docs/recon_legacy scratch/` → thư mục ngoài repo hoặc thêm `//go:build ignore`. Mục tiêu: `go build ./...` + `go vet ./...` XANH.
3. Tách test cần hạ tầng: test `Live*/Real*` gắn `if testing.Short() { t.Skip(...) }` **có log lý do** (không skip im lặng — L015).
4. Ghi `BASELINE_FAILS.txt` = danh sách test đang đỏ còn lại. **Luật: số dòng file này chỉ được giảm.** Thêm test đỏ mới = task bị từ chối.

### Phase 2 — Sửa đỏ về xanh (map F2)
Mỗi test đỏ = 1 task riêng, Red→Green, 1 commit: `PipelineFlowInfo`, stub `GetActiveReconJobs`, 6 test `service/source` multi-conn routing, `TestKnownTriggeredByMatchesPersistedTaxonomy`, `TestHandleScanArrayFields_*`… Ưu tiên test phản ánh hành vi nghiệp vụ (routing multi-connection có thể là bug thật đang chạy prod).

### Phase 3 — Chốt kiến trúc thành văn bản + kiểm tự động (map F6, F7)
1. Viết `ARCHITECTURE.md` mỗi service: layer nào được import layer nào; handler **không** gọi `db.Exec/Raw`; cấm tạo `*_v2`/`*_legacy` song song (thay thế thì xoá cái cũ trong cùng PR).
2. Kiểm tự động bằng script đặt trong `checks` (vd `go-arch-lint` hoặc grep-gate): fail nếu `internal/handler` có `.Exec(`/`.Raw(`, nếu file mới > 600 dòng.
3. **Không refactor big-bang.** Chỉ tách god-file khi có task chạm vào nó (boy-scout), và phải có test phủ trước khi tách.

### Phase 4 — Siết quy trình agent (map "báo OK láo")
1. "Done" chỉ hợp lệ khi đính kèm output nguyên văn của **toàn bộ** `checks` + diff `BASELINE_FAILS.txt` (không tăng).
2. Mỗi task: `git diff --stat` phải nằm trong scope khai báo ở `08_tasks`; file ngoài scope → dừng hỏi User.
3. Đề xuất bổ sung vào core `agent/`: hook đọc `.agent-project.json` và tự chạy `checks` trước khi cho phép append `DONE` (core đang có H7 nhưng không có manifest để chạy).

## D. Thứ tự & ước lượng
Phase 0 (0.5 ngày) → Phase 1 (1 ngày) → Phase 2 (theo số test đỏ, ~2–3 ngày) → Phase 3–4 song song với feature tiếp theo.

## E. Trạng thái Gate
- DG1 Laya: **PASS** sau resolve 4 FLAG bằng output go build/vet thật (`06_laya_summary_dg1_20261106T162733.json`, `04_decisions_*`). Laya r3 lệch với code ở cả 4 claim → kết luận dựa trên code/runtime, không dựa Laya.
- Chờ User APPROVE Phase 0–1.
