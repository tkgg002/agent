# Solution — Phase 17 cdc_system Namespace Bridge

## Vấn đề

Sau khi dựng màn `Sources & Connectors` giàu nghĩa hơn, CMS backend vẫn còn trỏ `cdc_internal` cho một số backing models quan trọng. Điều này làm UI mới đúng semantics nhưng lại đọc sai namespace vật lý.

## Giải pháp

1. Audit namespace ở model/repo runtime.
2. Chuyển các model/repo đã được migrate sang `cdc_system` về đúng schema:
   - `Source`
   - `WizardSession`
3. Verify lại bằng grep và tests.

## Outcome

- Giảm drift giữa CMS backend và end-state schema.
- Tạo nền sạch hơn cho các màn V2-native tiếp theo.
