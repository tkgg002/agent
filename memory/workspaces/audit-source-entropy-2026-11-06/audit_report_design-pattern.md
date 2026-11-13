---
tier: T2
mode: audit
suffix: design-pattern
status: PROPOSED — KHÔNG đụng nhánh recon-heal cho tới khi User nói
---
# Audit — Design pattern của data-hub góp phần thế nào vào việc Agent làm hỏng code cũ

## Ràng buộc User (2026-11-06 16:46)
- `recon-heal` User đang tự làm và buộc phải lên prod → **cấm sửa, cấm commit, cấm tạo nhánh trên đó** cho tới khi User nói.

## Bằng chứng (đếm bằng grep/find, 2026-11-06)
| # | Hiện trạng | Số liệu | Vì sao làm Agent sửa sai |
|---|---|---|---|
| P1 | **Repository layer bị bỏ qua** ở centralized-data-service | 56 file `internal/service` gọi `.Exec/.Raw` trực tiếp; chỉ có 19 file `internal/repository` | Truy vấn cùng một bảng nằm rải rác ở nhiều service → Agent sửa 1 chỗ, các chỗ khác vẫn logic cũ → lệch hành vi |
| P2 | **Handler vượt tầng** | 21/69 file handler nhận `*gorm.DB`; 17 file handler import thẳng `repository`; 15 file handler tự viết SQL | Không có một nơi duy nhất chứa luật nghiệp vụ → Agent không biết sửa ở tầng nào, thường sửa ở chỗ mình đang đứng |
| P3 | **Hexagonal bị xuyên thủng** ở cdc-cms-service | 33/74 file `internal/api` import thẳng `internal/infra`; 5 file api tự viết SQL | Port/interface bị bỏ qua → đổi interface (vd `GetActiveReconJobs`) không bắt được hết chỗ dùng |
| P4 | **Hai mô hình cũ/mới chạy song song** | cms: `internal/model` (16 file) và `internal/domain` (16 file); data-service: `recon_legacy.go` + `recon_dest_legacy.go`, `mapping_rule_v2*`, `GetByLegacyRegistryID` vẫn được gọi ở 3 handler | Agent không biết bản nào là chuẩn → sửa nhầm bản, hoặc tạo thêm bản thứ ba |
| P5 | **Logic trùng lặp** | 9 hàm `resolve/detect/infer…PK/TS` nằm ở 7 file khác nhau | Sửa cách xác định PK/timestamp ở 1 hàm, 8 hàm kia không đổi → recon/heal/transmute ra kết quả khác nhau |
| P6 | **Package và file quá to** | `service/recon` 31 file, `handler/shadow` 28, `service/master` 27; `transmuter.go` 1874 dòng; web `NatsMonitor.tsx` 2597 dòng | Agent chỉ đọc được một phần → không thấy chỗ bị ảnh hưởng |
| P7 | **Hai service theo hai kiến trúc khác nhau, không có tài liệu** | data-service: handler/service/repository; cms: api/app/domain/infra + model cũ | Agent mang thói quen của service này sang service kia |
| P8 | **Test tách khỏi package** | 43 file test nằm ở `test/internal/...` | Agent sửa package, chạy test trong package, **không chạy** test ở `test/` → bể mà không biết (đúng ca `PipelineFlowInfo`, stub `GetActiveReconJobs`) |

## Kết luận trung thực
- **Đúng, design pattern lộn xộn là một phần nguyên nhân**: nó khiến một thay đổi phải đụng nhiều chỗ mà không có chỗ nào báo lỗi (P1, P2, P3, P5, P8).
- **Nhưng không đổ hết cho nó**: lỗi vẫn là của Agent — không tìm hết chỗ dùng, không chạy hết test, tạo thêm `legacy/v2`/SQL thô thay vì theo chuẩn. Commit gắn mã "F1-F5", "AF-01..AF-05" cho thấy một phần sự lộn xộn này do chính Agent tạo ra qua các phiên trước. Hai thứ nuôi nhau: code càng rối → Agent càng sửa sai → code càng rối.

## Hướng giải quyết (CHƯA làm — chờ User cho phép, không đụng recon-heal)
1. **Chốt 1 chuẩn duy nhất cho mỗi service** (`ARCHITECTURE.md`, User duyệt): tầng nào được gọi DB, chiều import, bản nào là chuẩn giữa model/domain, legacy/v2.
2. **Bản đồ nguồn sự thật**: mỗi bảng/khái niệm (PK, timestamp, registry, binding) chỉ có 1 hàm/repository sở hữu; liệt kê 9 hàm PK/TS → chọn 1, các hàm kia gọi lại nó.
3. **Máy kiểm chuẩn** (`arch_check`): cấm thêm SQL ngoài repository, cấm handler/api import infra/repository mới, cấm file `*_v2/*legacy*` mới. Vi phạm hiện có vào allowlist, chỉ được giảm.
4. **Chạy cả `test/internal/...`** trong mọi lần kiểm (P8).
5. **Dọn dần theo task chạm vào** (không big-bang, không đụng recon-heal): mỗi lần chạm file nằm trong allowlist thì chuyển SQL về repository và xoá 1 mục allowlist.
6. Thứ tự: sau khi recon-heal lên prod xong → làm 1 → 3 → 4 → 2 → 5.

## Six-step QC
- **(a) Line-by-line / git diff:** Không có thay đổi source code; chỉ đọc cây thư mục và đếm bằng grep/find.
- **(b) Anti smoke-test / runtime trace:** Số liệu là đếm tĩnh, không suy diễn hành vi runtime; tác động lên prod của P5 (lệch PK/TS) là HYPOTHESIS, cần trace runtime thật.
- **(c) Concurrency / timeout-heartbeat / continuity:** Không áp dụng cho phân tích cấu trúc này.
- **(d) Suy diễn & trung thực:** Đã nêu rõ phần lỗi của Agent, không đổ hết cho design pattern.
- **(e) Self-improvement / lessons:** Lesson "Identify Criticized Subject" đã ghi ở lượt trước; ràng buộc "không đụng recon-heal" ghi vào 01_requirements.
- **(f) DG3 (laya_gate):** Dùng DG3 của workspace (`06_laya_summary_dg3_20261106T163753.json`, PASS sau tự resolve — không phải xác nhận độc lập). Số liệu P1–P8 dựa trên lệnh đếm thật.
