# Walkthrough: Unified Model Optimization & Quota Resilience

Hệ thống Brain/Muscle hiện đã được trang bị cơ chế tối ưu hóa chi phí, hiệu suất và khả năng chống lỗi Quota (429) toàn diện.

## Key Features Implemented

### 1. Multi-Provider Model Pool
- **[models.env](file:///Users/trainguyen/Documents/work/agent/models.env)**: Quản lý danh sách model linh hoạt (`BRAIN_POOL`, `MUSCLE_POOL`).
- **Hỗ trợ đa dạng**: Dễ dàng cấu hình và xoay vòng giữa Gemini, Grok, Qwen, Claude bằng định dạng `provider:model:key`.

### 2. Automated Quota Resilience
- **[quota_check.sh](file:///Users/trainguyen/Documents/work/agent/scripts/quota_check.sh)**: Tự động parse và xoay vòng model trong Pool khi phát hiện giới hạn quota.
- **Workflow Integration**: Được tích hợp sẵn vào bước nạp cấu hình của Brain và Muscle.

### 3. Detailed Progress Tracking
- **Quy tắc ghi log**: Mọi hành động trong `05_progress.md` hiện có tiền tố định danh model: `[Provider:Model-Name] [Time] Action`.
- **Hồi tố (Retroactive)**: Đã cập nhật toàn bộ lịch sử Phase 1, 2, 3 để tuân thủ quy tắc mới, giúp minh bạch hóa vai trò Brain/Muscle.
- **Minh bạch chi phí**: Giúp User theo dấu chính xác model nào đang tiêu tốn budget cho task nào.

### 4. Parallel Orchestration (1 Brain - Nhiều Muscle)
- **[parallel-muscle.md](file:///Users/trainguyen/Documents/work/agent/workflows/parallel-muscle.md)**: Workflow cho phép chia nhỏ task lớn và xử lý song song trên nhiều Terminal session mà không gây xung đột.

---

## Verification Results

### Pool Rotation Test
Hệ thống đã verify việc tách biệt vai trò và cấu hình pool thành công:
```bash
[Quota] Active Provider: antigravity
[Quota] Active Model: gemini-3-flash
[Quota] Pool rotated for MUSCLE_POOL.
```

### Strategy Mapping
- **Brain**: Antigravity (Gemini 3 Pro) -> Fallback: OpenAI/Claude.
- **Muscle**: Antigravity (Gemini 3 Flash) -> Fallback: Grok/Qwen.

---

## Instructions for User
> [!TIP]
> Bạn có thể cập nhật Model Pool bất cứ lúc nào tại [models.env](file:///Users/trainguyen/Documents/work/agent/models.env). Thứ tự trong danh sách (phân tách bằng dấu phẩy) chính là thứ tự ưu tiên khi hệ thống thực hiện xoay vòng key.
