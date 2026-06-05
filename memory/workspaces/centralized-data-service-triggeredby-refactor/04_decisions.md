# Decisions

## 2026-05-18
- Use a minimal core package (`internal/activity`) instead of a large middleware rewrite for this pass. Reason: user asked to make TriggeredBy manageable/debuggable and extend kafka-consumer; broad refactor risks unrelated behavior.
- Keep `recon-healer` as a sub-event TriggeredBy constant because existing source already persists it in recon heal flow, but do not promote it to a fourth user-facing root trigger.

