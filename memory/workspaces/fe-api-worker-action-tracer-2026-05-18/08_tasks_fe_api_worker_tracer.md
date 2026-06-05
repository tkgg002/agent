# Tasks: FE API Worker Action Tracer

- [x] Read lessons and agent governance.
- [x] Create workspace docs.
- [x] Capture git status for FE/API/worker.
- [x] Trace FE click handlers.
- [x] Trace API endpoints and publish path.
- [x] Trace worker subscribers/handlers.
- [x] Fix `Sync Fields to Shadow` (trace_id + scan metrics đính vào log; subscriber đã tồn tại, swallow giữ nguyên).
- [x] Fix `Snapshot Now` (stub subscriber khi reconCore=nil cho 7 recon subject — không còn NATS drop silent).
- [x] Add tracer metadata/logging (FE util + API normalize + worker completeActionTrace; log line "action trace received"/"action trace dispatch" có trace_id ở mọi boundary).
- [x] Run validation (build/vet/test pass 3 repo).
- [x] Write report (báo cáo nằm trong 05_progress.md và workspace docs; runtime cần user restart worker để binary mới hiệu lực).
- [x] Pre-flight check (governance rule audit done in this session).
