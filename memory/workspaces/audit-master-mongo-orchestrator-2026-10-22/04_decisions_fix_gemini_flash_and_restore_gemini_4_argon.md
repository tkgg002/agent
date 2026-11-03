---
task_id: FIX-GEMINI-FLASH-DOWNGRADE
title: Quyết định kỹ thuật giải quyết FLAG DG1
workspace: audit-master-mongo-orchestrator-2026-10-22
date: 2026-10-27
---

# 04_decisions: Quyết Định Kỹ Thuật

- **D-01 (Resolution S-RC):** Xác nhận `internal/webui/task_llm_bridge.go:235-238` chứa đoạn ép model `gemini-2.5-flash` khi `modelName` chứa `argon`. Quyết định gỡ bỏ 100% việc ép model này và giữ nguyên `gemini-4-argon (antigravity)`.
- **D-02 (Resolution S-FIX):** Xác nhận `internal/runner/antigravity.go:192-200` chứa đoạn gán `chosenModel = "gemini-2.5-flash"` và append `-m gemini-2.5-flash`. Quyết định loại bỏ triệt để flash model, giữ nguyên mô hình được chọn.
