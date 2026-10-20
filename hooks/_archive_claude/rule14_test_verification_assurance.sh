#!/usr/bin/env bash
# Rule 14 (Test Verification Assurance)
# Ngăn chặn việc đánh tráo khái niệm giữa Unit Test Mock và Test Thực Tế.
set -euo pipefail

echo '{"systemMessage":"⚠️  TEST VERIFICATION ASSURANCE (Rule 14): Bạn phải phân biệt RÕ RÀNG giữa (1) Unit Test Mock (chạy offline bằng sqlmock/mock) và (2) Real Integration Test / QC Thực Tế (chọc DB thật, bắn message NATS thật trên container/staging). TUYỆT ĐỐI CẤM báo cáo láo/đánh tráo khái niệm chỉ chạy test mock rồi ghi nhận là đã test thực tế thành công. Phải có bằng chứng vật lý riêng cho từng loại test!"}'
exit 0
