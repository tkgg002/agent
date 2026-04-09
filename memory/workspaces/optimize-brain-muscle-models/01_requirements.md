# 01_requirements.md - Requirements

## Trigger
> User: "tôi muốn brain và Muscle dùng những model AI khác nhau để tối ưu chi phí + tối ưu hiêu suất"

## Functional Requirements
1. **Model Differentiation**: Xác định rõ model nào dùng cho Brain, model nào dùng cho Muscle.
2. **Flexible Configuration**: Cả Brain và Muscle đều có thể được cấu hình để sử dụng model nội bộ Antigravity hoặc **External API Key** (OpenAI, Anthropic, etc.).
3. **Cost & Performance Balance**: Mặc định sử dụng model rẻ (`gemini-3-flash`) cho Muscle và model mạnh (`gemini-3-pro-high`) cho Brain, nhưng cho phép ghi đè linh hoạt qua cấu hình.
4. **Transparency**: Brain phải báo cáo model/provider nào đang được sử dụng trong mọi phản hồi và ghi nhận vào `05_progress.md`.
5. **Quota Resilience**: Hỗ trợ cơ chế xoay vòng (rotation) API Keys hoặc fallback provider khi gặp lỗi "Quota Limit".
6. **Multi-Muscle Strategy**: Nghiên cứu và đề xuất cơ chế 1 Brain điều phối nhiều Muscle song song.

## Constraints
- Không phá vỡ cấu trúc Brain/Muscle hiện tại.
- Muscle vẫn phải đảm bảo hoàn thành task code/debug dù dùng model yếu hơn.
