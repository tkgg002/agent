# Audit Log & Progress Journal

## Process History
- [2026-08-25T16:45:00+07:00] [Agent:Gemini-3.6-Flash] Task Initialized. Investigated log failure `unknown.hyperverge-face-match` / `shadow_testhecs.hyperverge_face_match` with error `ERROR: column "id" of relation "hyperverge_face_match" does not exist (SQLSTATE 42703)`.
- [2026-08-25T17:14:30+07:00] [Agent:Gemini-3.6-Flash] User Feedback: "ko phải cái lỗi trên, vì PrimaryKeyField đang là _id, nên nó ko về cái id đc."
- [2026-08-25T17:15:00+07:00] [Agent:Gemini-3.6-Flash] Mid-Session Fix executed. Discovered exact root cause: Forced hardcoded overrides `if pkField == "_id" { pgPKField = "id" }` in `event_handler.go` (lines 353, 384) and `bridge_handler.go` (line 281). Appended lesson to `lessons.md` (`#anti-id-override-trap #event-handler-pk-fix`). Updated all workspace documents.
- [2026-08-25T17:16:40+07:00] [Agent:Gemini-3.6-Flash] User approved implementation plan. Beginning execution phase: Modifying `event_handler.go` and `bridge_handler.go` to remove forced `_id -> id` overrides.
- [2026-08-25T17:17:35+07:00] [Agent:Gemini-3.6-Flash] Code modifications complete. Removed forced `_id -> id` overrides in `event_handler.go` and `bridge_handler.go`. Synchronized workspace files (`11_report`, `14_walkthrough`, `implementation_plan.md`). Verified governance linter PASSED 🟢.
