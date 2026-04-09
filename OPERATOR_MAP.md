# Operator Workflow Map (Muscle Catalog)

Bản đồ tra cứu nhanh 79+ workflows kỹ thuật từ hạ tầng mới (`agent/workflows/`). Brain sử dụng file này để Dispatch công việc cho Muscle.

## 1. Nhóm Chiến lược & Thực thi (Strategic Execution)
| Workflow | Mục tiêu | Khi nào dùng |
|---|---|---|
| `/prp-prd` | Tạo PRD hoàn chỉnh | Yêu cầu feature mới chưa có spec. |
| `/prp-plan` | Lập kế hoạch chi tiết + trích xuất patterns | Task phức tạp, cần độ chính xác cao. |
| `/prp-implement` | Thực thi plan với validation loop | Sau khi có `.plan.md` từ `/prp-plan`. |
| `/prp-commit` | Commit code kèm context theo chuẩn PRP | Kết thúc task, chuẩn bị PR. |
| `/prp-pr` | Tạo Pull Request chuyên nghiệp | Khi đã sẵn sàng để review. |

## 2. Nhóm Kiểm thử & Chất lượng (TDD & Quality)
| Workflow | Mục tiêu | Khi nào dùng |
|---|---|---|
| `/tdd` | Quy trình Test-Driven Development | Viết tính năng mới, ưu tiên Test trước. |
| `/santa-loop` | Xác minh đối kháng (Adversarial) | Logic nhạy cảm (Tiền tệ, Bảo mật). |
| `/verify` | Kiểm tra tổng thể trạng thái code | Sau khi fix bug hoặc refactor. |
| `/test-coverage` | Đo lường độ bao phủ test | Đánh giá chất lượng bộ test. |
| `/e2e` | Kiểm thử Playwright đầu-cuối | Kiểm tra UI/UX flows. |

## 3. Nhóm Ngôn ngữ & Framework (Custom Tech Muscle)
| Ngôn ngữ | Workflows sẵn có | Chức năng |
|---|---|---|
| **Golang** | `/go-build`, `/go-test`, `/go-review` | Build, test, review code Go. |
| **Python** | `/python-review`, `/python-test` (via `/tdd`) | Xử lý logic Python. |
| **Rust** | `/rust-build`, `/rust-test`, `/rust-review` | Hệ sinh thái Rust. |
| **Frontend** | `/flutter-build`, `/flutter-test`, `/kotlin-build` | App & Mobile development. |
| **C++** | `/cpp-build`, `/cpp-test`, `/cpp-review` | Native development. |

## 4. Nhóm Tối ưu hóa AI (Internal Ops)
| Workflow | Mục tiêu | Khi nào dùng |
|---|---|---|
| `/evolve` | Phân tích và đề xuất tiến hóa cấu trúc | Muốn tối ưu hóa chính Agentic Core. |
| `/skill-create` | Tự đúc kết Skills từ Git history | Muốn lưu lại kinh nghiệm dự án. |
| `/prompt-optimize`| Tối ưu hóa câu lệnh hệ thống | Khi Agent phản hồi kém hiệu quả. |
| `/skill-health` | Kiểm tra "sức khỏe" bộ kỹ năng | Audit lại mớ Skills cũ. |

## 5. Nhóm Tích hợp & Quản trị (Integration & Ops)
| Workflow | Mục tiêu | Khi nào dùng |
|---|---|---|
| `/jira` | Tương tác với vé Jira | Lấy yêu cầu hoặc cập nhật status. |
| `/gh-issue` | Quản lý GitHub Issues | Triage hoặc link active work. |
| `/sessions` | Quản lý lịch sử phiên làm việc | Tra cứu lại quyết định cũ. |
| `/save-session` | Lưu trạng thái phiên hiện tại | Sắp hết context hoặc kết thúc ngày. |

> [!IMPORTANT]
> **Priority**: Nếu Brain có workflow riêng trong `agent/workflows/` (như `/brain-delegate`), hãy dùng nó trước rồi mới Dispatch các lệnh này.
