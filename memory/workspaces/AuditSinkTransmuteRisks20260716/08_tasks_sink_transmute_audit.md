# Tasks - Audit Sink & Transmute Risks

## Phase 1: Research & Analysis
- [x] T1.1: Research luồng Sink (subagent Sink Flow Researcher)
- [x] T1.2: Research luồng Transmute (subagent Transmute Flow Researcher)
- [x] T1.3: Tổng hợp Error Patterns lịch sử (subagent Error Patterns Researcher)

## Phase 2: Tổng hợp & Báo cáo
- [x] T2.1: Tổng hợp kết quả từ 3 subagent
- [x] T2.2: Phân loại rủi ro theo Severity
- [x] T2.3: Tạo artifact báo cáo audit tổng quan
- [x] T2.4: Recommendations ưu tiên fix

## Phase 3: User Review & Corrections
- [x] T3.1: Xác nhận 2 sink paths → đúng 2 nhưng chỉ 1 active
- [x] T3.2: Kiểm tra Kafka Consumer có dùng ở prod → CÓ (env vars inject brokers)
- [x] T3.3: Xác nhận activity log → Kafka Consumer = PRIMARY cả prod+local
- [x] T3.4: Xác nhận Sink Worker = LEGACY, không chạy
- [x] T3.5: Viết lại toàn bộ báo cáo với vai trò đúng
- [x] T3.6: Trace git history V1→V2 evolution
- [x] T3.7: Thêm section design pattern issue (handler/shadow SRP violation)
- [x] T3.8: User APPROVED báo cáo cuối cùng
