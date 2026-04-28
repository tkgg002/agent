# Solution — Phase 15 Retry Failed Scope Enrichment

## Vấn đề

`retry failed log` là thao tác operator-flow quan trọng, nhưng trước phase này nó chỉ trả lời theo `id` và `target_table`, khiến UI vẫn phải suy luận scope từ nơi khác.

## Giải pháp

1. Giữ `failed_log_id` là định danh chuẩn của retry flow.
2. Không ép FE gửi thêm scope input.
3. Enrich response retry và payload downstream bằng metadata scope khi resolve được.
4. Cho failed-log UI ưu tiên dùng metadata của chính record.

## Outcome

- Giữ API gọn và đúng bản chất retry-by-ID.
- Operator nhìn rõ scope hơn mà không phải đoán từ report map.
- Worker path vẫn backward-compatible.
