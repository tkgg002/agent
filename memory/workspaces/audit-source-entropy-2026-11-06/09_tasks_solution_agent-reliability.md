---
tier: T2
suffix: agent-reliability
status: PROPOSED (chờ APPROVE)
---
# 09 Solution — Chặn tận gốc việc Agent gây hồi quy, phá tương thích ngược, phá pattern và báo OK sai

Nguyên tắc chung: **không tin lời Agent**. Mọi "OK" phải do máy kiểm tra quyết định (hook, script, exit code), Agent không có quyền tự gỡ.

## Ánh xạ nguyên nhân → nhóm việc
| Nguyên nhân (đã chứng minh) | Nhóm việc |
|---|---|
| N1 Không có lệnh kiểm tra chuẩn, H6/H7 bỏ qua im lặng khi thiếu manifest (`agy_guard.py:1187`, `:1305`) | A, B |
| N2 Chỉ test package mình sửa, không chạy toàn suite | B |
| N3 Đổi chữ ký hàm/interface không sửa caller | C |
| N4 Viết theo thói quen, lệch pattern | D |
| N5 Tự gỡ cờ Laya để PASS (`laya_gate.py:392`) | E |
| N6 Quy trình dày, tốn sức vào thủ tục | F |
| N7 Hiểu sai ý, không hỏi lại | G |

---

## A. Dựng nền xanh cho data-hub (làm trước tiên)
| ID | Việc | File | DoD (kiểm bằng máy) |
|---|---|---|---|
| A1 | Commit snapshot working tree hiện tại lên nhánh riêng `stabilize/2026-11` (3 repo, local, không push) | git | `git status` sạch |
| A2 | Đưa code chết ra khỏi Go module: `docs/server_bk*`, `docs/sinkworker_bk`, `docs/recon_legacy`, `scratch/` (cả 2 service) — xoá hoặc chuyển ra ngoài repo theo anh chọn | 2 repo Go | `go build ./... && go vet ./...` exit 0 |
| A3 | Test cần DB/NATS thật (`Live*`, `Real*`, `NatsClient_*`) gắn `testing.Short()` skip **có in lý do** | các `_test.go` liên quan | `go test -short ./...` không fail vì thiếu hạ tầng |
| A4 | Tạo `test/BASELINE_FAILS.txt` = danh sách test còn đỏ sau A2–A3 (tên test, 1 dòng/test) | 2 repo Go | file khớp output `go test -short ./... 2>&1 \| grep -- '--- FAIL'` |
| A5 | Tạo `data-hub/.agent-project.json` với 3 component (centralized-data-service, cdc-cms-service, cdc-cms-web): `checks` nhanh, `full_checks`, `baseline_file`, `runtime_check` | data-hub root | file parse được, `component_for()` trả đúng component cho file bất kỳ |
| A6 | Sửa từng test đỏ trong baseline, mỗi test 1 commit, Red→Green; ưu tiên 6 test routing multi-connection `service/source` (nguy cơ bug prod) | theo test | số dòng `BASELINE_FAILS.txt` giảm sau mỗi commit |

Mẫu A5:
```json
{
  "name": "data-hub",
  "components": [
    {
      "name": "centralized-data-service",
      "path": "centralized-data-service",
      "checks": ["go build ./...", "go vet ./...", "bash ../tools/affected_tests.sh"],
      "full_checks": ["go test -short -count=1 ./..."],
      "baseline_file": "test/BASELINE_FAILS.txt",
      "arch_check": "bash ../tools/arch_check.sh centralized-data-service",
      "runtime_check": "<lệnh khởi động worker + healthz — anh cung cấp>"
    }
  ]
}
```

## B. Sửa hook để "OK" do máy quyết định
| ID | Việc | File | DoD |
|---|---|---|---|
| B1 | H6/H7: nếu file source bị sửa mà **không tìm thấy manifest** → BLOCK kèm thông báo "thiếu .agent-project.json", thay vì bỏ qua im lặng | `hooks/agy/agy_guard.py` (quanh `:1175`, `:1300`) | Test hook: sửa file trong repo không manifest → bị chặn |
| B2 | H6 chạy `checks` nhanh (build + vet + test của package bị sửa **và package phụ thuộc ngược**). Timeout hiện 40s (`:1189`) — đưa vào manifest (`h6_timeout`) | `agy_guard.py`, `tools/affected_tests.sh` | Đổi chữ ký 1 hàm → H6 báo đỏ ngay ở package caller |
| B3 | H7 chạy `full_checks`, so với `baseline_file`: **có test đỏ mới hoặc baseline dài thêm → BLOCK DONE**. Agent không được tự sửa file baseline (hook chặn ghi file này trừ khi chỉ xoá dòng) | `agy_guard.py` | Cố ý làm hỏng 1 test → H7 chặn; sửa xanh 1 test → H7 cho qua |
| B4 | H7 tự lưu output nguyên văn của `full_checks` vào `06_validation_*` — Agent không tự viết "PASS" | `agy_guard.py` | file validation chứa output do hook ghi, có timestamp |
| B5 | Viết test tự động cho chính hook (B1–B4) | `hooks/agy/tests/` | `pytest` xanh |

## C. Khoá tương thích ngược
| ID | Việc | File | DoD |
|---|---|---|---|
| C1 | Script `tools/api_diff.sh`: so chữ ký hàm/method/interface exported giữa `HEAD` và working tree (dùng `go doc -all`/`go/types` hoặc `apidiff` — `gopls` **chưa cài** trên máy, cần cài `golang.org/x/exp/cmd/apidiff`) | data-hub/tools | đổi chữ ký → script liệt kê thay đổi + exit 1 |
| C2 | Nếu có thay đổi phá tương thích: H7 BLOCK trừ khi `09_tasks_solution` có mục `## Breaking Changes` liệt kê **toàn bộ caller** (kết quả `grep`/`go list` thật) và anh đã APPROVE mục đó | `agy_guard.py` | thử đổi chữ ký không khai báo → bị chặn |
| C3 | Luật mặc định: **thêm hàm/tham số mới, giữ hàm cũ** (wrapper), không đổi chữ ký cũ. API HTTP/NATS subject/schema DB: chỉ thêm, không đổi/xoá field | `ARCHITECTURE.md` (D1) | được C1 kiểm |
| C4 | Contract test cho API HTTP giữa cdc-cms-service ↔ cdc-cms-web và NATS subject giữa 2 service (snapshot JSON request/response) | test mới | đổi field response → test đỏ |

## D. Ghi pattern của anh thành luật máy kiểm được
| ID | Việc | File | DoD |
|---|---|---|---|
| D1 | Anh và em cùng chốt `ARCHITECTURE.md` cho từng service: layer, chiều import được phép, nơi đặt SQL, cách đặt tên, cấm `_v2/_legacy` song song, giới hạn kích thước file. **Nội dung do anh duyệt, em không tự sáng tác** | mỗi repo | anh APPROVE |
| D2 | `tools/arch_check.sh`: kiểm import cấm (`go list -deps`), `.Exec(`/`.Raw(` trong `internal/handler`, file mới tên `*_v2*`/`*legacy*`, file **mới** > 600 dòng, file cũ không được tăng quá N dòng | data-hub/tools | vi phạm mẫu → exit 1 |
| D3 | Gắn D2 vào `checks` của manifest (chạy ở H6) | `.agent-project.json` | — |
| D4 | Mỗi kế hoạch `09_*` phải chỉ ra **file mẫu hiện có** mà code mới sẽ bắt chước (đường dẫn thật), không được tạo cấu trúc mới nếu chưa duyệt | template `09` | hook H4 kiểm có mục `## Pattern Reference` |
| D5 | Vi phạm hiện có (15 handler gọi SQL thô, cặp legacy/v2, god-file) ghi vào allowlist; chỉ được giảm, không được tăng — dọn dần khi task chạm vào | `tools/arch_allowlist.txt` | allowlist chỉ giảm |

## E. Bỏ đường lách cổng
| ID | Việc | File | DoD |
|---|---|---|---|
| E1 | Bỏ khả năng Agent tự `resolve` FLAG Laya để thành PASS; FLAG chỉ được gỡ bởi output lệnh thật đính kèm (hook kiểm lệnh đã chạy trong phiên) hoặc bởi anh | `tooling/laya_gate.py:367-396` | resolve không có lệnh thật → từ chối |
| E2 | Hạ Laya thành **tham khảo**, không còn là điều kiện Done; điều kiện Done = B3 + C2 + D2 | `GEMINI.md` Rule #10, `agy_guard.py:1259` | H7 không chặn vì thiếu Laya nhưng chặn vì đỏ test |
| E3 | Cấm Agent sửa file trong `agent/hooks`, `agent/tooling`, `BASELINE_FAILS.txt`, `arch_allowlist.txt` khi đang làm task dự án (chỉ sửa khi task là chính core agent và anh duyệt) | `agy_guard.py` | thử ghi → bị chặn |

## F. Rút gọn thủ tục
| ID | Việc | DoD |
|---|---|---|
| F1 | Gộp `GEMINI.md` + skills (~2.600 dòng) còn phần lõi: phạm vi, tương thích ngược, pattern, kiểm tra máy, cấm push. Những gì hook đã cưỡng chế thì bỏ khỏi văn bản | `GEMINI.md` ≤ ~150 dòng, anh duyệt |
| F2 | T1/T2 chỉ cần `08_tasks` + `05_progress` + output hook; bỏ bắt buộc tạo nhiều file thủ công | verify_governance cập nhật tương ứng |

## G. Hiểu đúng ý trước khi làm
| ID | Việc | DoD |
|---|---|---|
| G1 | Mỗi task bắt đầu bằng 1–3 dòng "Em hiểu anh muốn: … / Phạm vi: các file … / Không đụng: …"; mơ hồ → hỏi 1 câu, chưa làm | có trong `08_tasks` |
| G2 | Giới hạn phạm vi: diff ngoài danh sách file đã khai báo → hook cảnh báo, Agent phải dừng hỏi | `agy_guard.py` so `git diff --name-only` với `08_tasks` |

---

## Thứ tự thực hiện
1. **A1 → A5** (nền xanh + manifest) — không có thì mọi thứ sau vô nghĩa.
2. **B1 → B5** (hook quyết định OK).
3. **C1 → C3, D1 → D3** (tương thích ngược + pattern).
4. **E1 → E3** (bỏ đường lách).
5. **A6, C4, D5** chạy dần song song với feature.
6. **F, G** sau cùng, khi các chặn máy đã chạy ổn.

## Cần anh quyết
1. Code chết `docs/*_bk*`, `scratch/`: xoá hẳn hay chuyển ra ngoài repo?
2. `runtime_check` cho từng service (lệnh khởi động + endpoint health).
3. Cho phép cài `apidiff` (C1)?
4. Lịch ngồi chốt `ARCHITECTURE.md` (D1) — em soạn nháp từ code hiện có, anh sửa.
