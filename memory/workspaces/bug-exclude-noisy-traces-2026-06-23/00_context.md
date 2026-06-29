# Context: Exclude Noisy Traces from Telemetry

## Overview
Task này tập trung vào việc loại bỏ 2 trace spans gây nhiễu và spam hệ thống telemetry (SigNoz):
1. `cdc.batchbuffer.flush` (thuộc `centralized-data-service`)
2. `cdc.cms.stuck_job_reaper` (thuộc `cdc-cms-service`)

Việc loại bỏ các trace này giúp giảm tải cho collector, giảm kích thước lưu trữ của SigNoz/ClickHouse và giúp lập trình viên tập trung vào các trace nghiệp vụ thực tế mà không bị nhiễu bởi các background jobs chạy với tần suất cao.

## Key Goals
1. **Identify & Remove**: Định vị chính xác nơi phát sinh hai trace span trên và loại bỏ/vô hiệu hóa việc tạo span telemetry đối với chúng.
2. **System Stability & Compliance**: Đảm bảo việc loại bỏ spans không ảnh hưởng đến logic nghiệp vụ cốt lõi và không làm hỏng trace context propagation của các luồng liên quan (nếu có).
3. **Verify**: Biên dịch thành công và kiểm thử chạy cục bộ đảm bảo hệ thống ổn định.
