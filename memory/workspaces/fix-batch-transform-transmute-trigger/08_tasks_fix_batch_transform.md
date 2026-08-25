# Tasks — fix-batch-transform-transmute-trigger

**Phase:** Hotfix  
**File duy nhất cần sửa:** `centralized-data-service/internal/handler/shadow/batch_transform_handler.go`

---

## Checklist thực thi

- [/] **T1:** Thêm method `publishTransmuteTrigger` vào `BatchTransformHandler`
- [/] **T2:** Gọi method đó ở cuối `runTransformJob()` khi status = COMPLETED (2 điểm)
- [ ] **T3:** Verify build pass
- [ ] **T4:** Verify existing tests pass (không break)
- [ ] **T5:** Append vào `05_progress.md` kết quả
