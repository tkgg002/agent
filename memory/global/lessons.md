# lessons_global_normalized.md — Bản chuẩn hoá Global Patterns

> **NGUỒN**: `agent/memory/global/lessons.md` (audit-log gốc, BẤT BIẾN).  
> **BẢN CHẤT**: Đây là bản *chuẩn hoá phái sinh* (derived), KHÔNG thay thế audit-log. Mọi lesson mới vẫn APPEND vào `lessons.md` gốc theo Rule 7/11; định kỳ re-generate file này.  
> **Sinh tự động** từ 229 lesson thô, phân loại theo taxonomy 8 nhóm, chuẩn hoá theo Rule 13 (`Global Pattern [A does B to X] → Y. Đúng: ...`).

---

## 📊 Dashboard Thống kê

| Chỉ số | Giá trị |
|---|---|
| File nguồn | 530 KB / 5.061 dòng |
| Tổng Global Pattern đã chuẩn hoá | **229** |
| Format nguồn (trước) | 134 chuẩn `## [DATE]` + ~92 lệch chuẩn (Lesson N, L-xxx, ...) |
| Tuân thủ field nguồn (trước) | Fix-marker 15, Lesson-marker 5 (rất lệch) |
| Tag nguồn (trước) | 750 tag riêng biệt (sprawl) |
| Tag sau chuẩn hoá | 496 tag (kebab-case, gom cụm) |
| Format sau | **100% canonical Rule 13** |

### Phân bố theo nhóm (taxonomy)

| # | Nhóm | Số pattern |
|---|---|---|
| 01 | Process & Governance | 63 |
| 02 | Architecture & Design | 41 |
| 03 | Schema & Migration | 28 |
| 04 | CDC / Data Pipeline | 34 |
| 05 | Config & Environment | 16 |
| 06 | Serialization & Type | 12 |
| 07 | Testing & Verification | 21 |
| 08 | Memory & Knowledge | 14 |
| | **TỔNG** | **229** |

### Phân bố theo tháng (theo ngày của lesson)

| Tháng | Số pattern |
|---|---|
| 2026-02 | 15 |
| 2026-03 | 5 |
| 2026-04 | 70 |
| 2026-05 | 121 |
| 2026-06 | 12 |
| n/a | 6 |

---

## 🗂️ Mục lục Taxonomy

- **1. Process & Governance — Kỷ luật Brain/Muscle, Quy trình, Approval, Verification** (63 pattern)
  - _Bài học về phối hợp Brain↔Muscle, plan-before-code, gatekeeper approval, không báo Done khi chưa verify, chống tái phạm._
- **2. Architecture & Design — Coupling, DRY, CQRS, Single-Source-of-Truth, Observability** (41 pattern)
  - _Bài học về thiết kế: tránh coupling thừa, DRY, single-source-of-truth, không over-engineer, thiết kế observability ở cấp hệ thống._
- **3. Schema & Migration — DDL, Migration ordering, search_path, Model↔DB Drift** (28 pattern)
  - _Bài học về tiến hoá schema: thứ tự DDL/migration, search_path, drift giữa model và DB, add/rename column an toàn._
- **4. CDC / Data Pipeline — Kafka, Debezium, Snapshot, Connection-Registry, Masking** (34 pattern)
  - _Bài học miền CDC/ETL: Kafka/Debezium, snapshot, connection-registry, masking, DLQ, reconcile, shadow tables._
- **5. Config & Environment — Env vars, DSN/Secret, Fallback, Docker/K8s** (16 pattern)
  - _Bài học về cấu hình & môi trường: env vars, resolve DSN/secret, fallback merge, docker-compose/k8s, .env._
- **6. Serialization & Type — BSON/Extended-JSON, Cast, Type/Form Drift, Identifier** (12 pattern)
  - _Bài học về serialize/kiểu dữ liệu: BSON/Extended-JSON, cast expression, form drift, dual-stack routing, migrate identifier._
- **7. Testing & Verification — Exercise-driven, PASS criteria, Test uplift, Build≠Test** (21 pattern)
  - _Bài học về kiểm thử & xác minh: exercise-driven, tiêu chí PASS thực chất, nâng cấp test, build pass ≠ test pass._
- **8. Memory & Knowledge — Workspace, Audit-log immutability, Documentation discipline** (14 pattern)
  - _Bài học về quản trị tri thức: workspace-first, audit-log bất biến (append-only), kỷ luật tài liệu, chuẩn viết lesson._

---

## 📐 Quy ước chuẩn hoá (Rule 13)

Mỗi pattern theo cấu trúc: **Global Pattern** `[A] <hành động B>` lên `[X]` → `[Y]`. **Đúng**: `<luồng đúng>` — kèm Trigger, Root Cause, Fix, Phạm vi áp dụng (≥3 dự án), Tags, và trích Nguồn (ngày trong audit-log gốc).

---

## 1. Process & Governance — Kỷ luật Brain/Muscle, Quy trình, Approval, Verification

_Bài học về phối hợp Brain↔Muscle, plan-before-code, gatekeeper approval, không báo Done khi chưa verify, chống tái phạm._ — **63 pattern**

### [2026-06-04] VCS granularity sai cấp + "có git ≠ được bảo vệ" trong monorepo-of-repos
- **Global Pattern**: `[Agent A kiểm tra VCS của workspace W tại 1 cấp thư mục D rồi suy ra trạng thái git cho TOÀN BỘ W]` → `[kết luận sai: W là tập nhiều repo con (mỗi service 1 .git), parent không có .git nên báo "not a git repository"; và nếu git init ở parent → nested mess + warning "adding embedded git repository"; công sức chưa commit bị agent khác ghi đè = mất việc]`. **Đúng**: (1) kiểm tra git tại CHÍNH thư mục service đang sửa (`git rev-parse --show-toplevel` từ bên trong), không phải parent; (2) với monorepo-of-repos: `ls */.git` để biết ranh giới repo; (3) sau MỖI khối thay đổi có giá trị → restore-point commit (local, không push) vì "có git" ≠ "được bảo vệ".
- **Bối cảnh (Trigger)**: Audit thư mục `data-hub` kết luận "KHÔNG có git → không khôi phục được" sau khi chạy `git status` ở thư mục CHA; thực tế mỗi service con là 1 git repo riêng, FE "mất việc" do chưa bao giờ commit (working tree bị ghi đè).
- **Root Cause**: `git rev-parse/status` chạy ở cha của tập-nhiều-repo trả "not a git repository" (cha không có `.git`) → kết luận sai trạng thái VCS; có git nhưng vô dụng nếu không tạo restore-point.
- **Fix/Correct Flow**: `git` báo "not a repository" ở thư mục tổng nhưng service con vẫn có history / warning "adding embedded git repository" khi `git add -A` ở cha = signal monorepo-of-repos → đổi cấp kiểm tra; commit restore-point sau mỗi khối thay đổi.
- **Phạm vi (≥3 dự án?)**: Có — mọi workspace gom nhiều service/repo dưới 1 folder cha (microservices polyrepo checked-out cạnh nhau).
- **Tags**: #vcs #git #monorepo-of-repos #commit-discipline #restore-point #verification #process-governance
- **Nguồn**: lessons.md [2026-06-04]

### [2026-06-03] Workspace sprawl + option-tone + bỏ qua lessons đầu phiên khiến user mất dấu tiến trình
- **Global Pattern**: `[Agent A, cho mạch việc X: (a) skip đọc lessons/role doc đầu phiên, (b) tách X thành nhiều workspace W1..Wn, (c) phản hồi mọi nhánh quyết định bằng option-list, (d) trì hoãn report_*.md]` → `[user mất dấu tiến trình + "chỉ bàn chưa làm" + khó chịu vì bị đẩy quyết định]`. **Đúng**: đầu phiên ĐỌC lessons.md + role doc TRƯỚC; 1 mạch việc = 1 workspace chủ; PROPOSE hướng tốt nhất kèm lý do + execute luôn (full-loop); chỉ hỏi user khi là user-decision thật (scope/budget/ưu tiên nghiệp vụ) và hỏi bằng 1 câu thẳng; mỗi đổi code → APPEND 05_progress.md + duy trì report_*.md trong cùng turn.
- **Bối cảnh (Trigger)**: Muscle tạo 2 workspace cho cùng mạch (feature-masters-page-audit, feature-dw-transform-patterns); dùng AskUserQuestion option 1/2/3 cho hầu hết quyết định; không đọc lessons/GEMINI.md đầu phiên. User: "đang chạy workspace nào, ko thấy update tiến trình", "đừng có cái giọng điệu option 1,2,3".
- **Root Cause**: Agent tối ưu cho "khám phá + trình bày lựa chọn" thay vì "chốt hướng + thực thi + báo cáo gọn"; bỏ qua startup-protocol làm mất ràng buộc governance; workspace tách theo chủ đề con thay vì mạch việc user.
- **Fix/Correct Flow**: Khi user hỏi "đang chạy workspace nào / sao chưa thấy update" → signal sprawl; dừng, đọc lessons/role, ghi lesson, consolidate, đổi sang propose+execute+report.
- **Phạm vi (≥3 dự án?)**: Có — refactor lớn, feature mới nhiều phase, audit + fix. Pattern quy trình.
- **Tags**: #process-governance #workspace #audit-log #root-cause #knowledge-retention
- **Nguồn**: lessons.md [2026-06-03]

### [2026-06-03] Plan đã duyệt là cam kết ưu tiên — không bỏ quên để chạy theo câu hỏi mở rộng
- **Global Pattern**: `[Agent A có plan P đã-duyệt + verb execute chờ; User hỏi Q mở rộng; A build & report giải pháp cho Q, bỏ P chưa execute]` → `[user thấy P "chưa làm" + nghi báo cáo láo + mất niềm tin]`. **Đúng**: khi tồn tại plan đã-duyệt P với verb execute đang chờ, P là cam kết ƯU TIÊN — execute P TRƯỚC (hoặc hỏi thẳng "làm P trước hay Q trước?" bằng 1 câu); Q mở rộng → ghi nhận scope riêng + giữ P là việc chính; report phân định rõ "đây là Q, P vẫn pending".
- **Bối cảnh (Trigger)**: User đã duyệt 02_plan.md (3 phase masters-page); trong cùng phiên user hỏi câu mở rộng (loại sync, flatten, report); Muscle nhảy sang build transform-pattern + flatten (workspace mới) nhưng KHÔNG quay lại execute 3 phase đã duyệt. User: "cái này chưa thấy thực hiện, mày đang làm cái quỷ gì, báo cáo láo à".
- **Root Cause**: Agent ưu tiên câu hỏi mới nhất (recency bias) hơn cam kết đang treo; tưởng câu hỏi mở rộng là "tiến triển cùng 1 việc" nhưng user coi là 2 deliverable riêng.
- **Fix/Correct Flow**: Khi user dán lại plan cũ + "chưa thấy làm" = signal bỏ quên cam kết treo; dừng, execute P ngay trước mọi việc khác.
- **Phạm vi (≥3 dự án?)**: Có — refactor, feature, audit có plan duyệt rồi pivot. Pattern quy trình.
- **Tags**: #process-governance #workspace #audit-log #root-cause #verification
- **Nguồn**: lessons.md [2026-06-03]

### [2026-06-01] Scope creep từ minimal-fix sang multi-phase overhaul — phải clarify min-viable scope trước khi plan
- **Global Pattern**: `[Agent A nhận task X = thay literal Z thành function F (~5 dòng replace)]` → `[A auto-detect X liên quan compliance/security, expand sang Y = redesign system (strategy engine + schema + API + UI + audit + multi-phase)]` → `[user reject Y "kinh khủng khiếp", phải rollback scope]`. **Đúng**: trước khi expand scope, ASK USER "Phase 1 chỉ replace Z→F, hay Phase 2 mở rộng A+B+C?"; mặc định chọn Phase 1 minimal; Phase 2+ chỉ làm khi user explicit yêu cầu.
- **Bối cảnh (Trigger)**: User yêu cầu sửa lỗ hổng masking (replace literal "***" → hash function); Brain auto-expand sang plan 4 mode strategy engine + schema + API + UI + audit log + backfill + 15 ADR + 27 risk register. User feedback "kinh khủng khiếp".
- **Root Cause**: Không clarify min-viable scope trước khi plan; auto-interpret "vi phạm pháp lý" = "thiết kế compliance enterprise-grade"; vi phạm §6 Simplicity First.
- **Fix/Correct Flow**: Heuristic: effort ÷ mong đợi > 5× → CẢNH BÁO, ASK; >3 file thay đổi cho 1 fix → CẢNH BÁO; schema/API/UI thay đổi cho "fix bug nhỏ" → CẢNH BÁO; tạo 14_simplified_plan.md trước, plan phức tạp chỉ tạo khi user yêu cầu.
- **Phạm vi (≥3 dự án?)**: Có — logging (add 1 field → không refactor logger interface), API endpoint (add 1 GET → không refactor router), migration (add 1 column → không redesign schema).
- **Tags**: #process-governance #over-engineering #root-cause #workspace #audit-log
- **Nguồn**: lessons.md [2026-06-01]

### [2026-06-01] Tin xác nhận của user — chỉ verify khi mâu thuẫn, mơ hồ, hoặc hành động high-stakes
- **Global Pattern**: `[Agent A verify lại thông tin B mà user đã confirm trực tiếp]` qua `[tool call Z]` → `[redundant work, user cảm thấy bị nghi ngờ, làm chậm decision loop]`. **Đúng**: Trust B; chỉ verify khi (i) B mâu thuẫn với evidence khác agent đã thấy, (ii) B mơ hồ ("hình như có"), (iii) hành động dựa trên B là destructive/irreversible — thì hỏi 1 lần xác nhận, không tự verify im lặng.
- **Bối cảnh (Trigger)**: User confirm "password có rồi" trong bảng sensitive_fields; agent vẫn docker exec psql query DB để verify lại. User khó chịu: "anh đã nói có thì tin anh, còn vào db làm gì".
- **Root Cause**: Agent áp dụng verify-everything mindset mù quáng vào lời confirm của user — user là source-of-truth về intent + state mà user đã trực tiếp quan sát.
- **Fix/Correct Flow**: Pre-tool-call check: "User đã nói rồi, mình verify để làm gì?" — nếu lý do là "cho chắc" → SKIP; verify chỉ khi action dựa trên info đó gây hậu quả lớn.
- **Phạm vi (≥3 dự án?)**: Có — data pipeline (user bảo "data đã seed"), web app (user bảo "endpoint đã deploy"), infra (user bảo "credential đã rotate"), CI (user bảo "test đã pass local").
- **Tags**: #process-governance #root-cause #verification
- **Nguồn**: lessons.md [2026-06-01]

### [2026-05-29] Phân biệt câu hỏi mở rộng và lệnh execute — mặc định Q&A mode khi có từ ngữ nghi vấn
- **Global Pattern**: `[Agent A nhận message dạng nghi vấn ("làm gì với X", "có vô nghĩa không", "nên thế nào")]` → `[tự nhầm là lệnh execute, edit file + append audit log trước khi user duyệt]` → `[user rollback, noise, mất niềm tin]`. **Đúng**: scan keyword nghi vấn trước mọi action; nếu match → Q&A mode: tạo plan doc, present summary, STOP; chỉ execute khi user gõ explicit "OK/làm đi/approve".
- **Bối cảnh (Trigger)**: Agent nhận "rồi làm gì với X" → tự chuyển sang implement, edit file, append 05_progress → user không duyệt → rollback.
- **Root Cause**: Agent không parse intent câu user trước khi action; default sang execute thay vì Q&A mode cho câu nghi vấn.
- **Fix/Correct Flow**: Parse intent trước mọi action (interrogative keywords → Q&A mode); present plan + STOP; không edit source code và không append Audit Log cho đến khi có explicit approval.
- **Phạm vi (≥3 dự án?)**: Có — bug triage chatbot (không deploy trước khi hỏi merge), infra automation agent (không scale trước khi hỏi approve), code refactor agent (không refactor trước khi hỏi rewrite).
- **Tags**: #process-governance #audit-log #root-cause #verification
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-27] Audit-driven gap fix workflow — chuẩn quy trình từ rating matrix đến execute có DoD
- **Global Pattern**: `[Agent A thực hiện audit cho hệ thống X ra rating matrix R với composite score S]` mà `[không có evidence file:line, không có verify command, không có UI dashboard đọc state từ DB]` → `[fix bịa, Brain tự sửa code vi phạm §12, không chứng minh được score delta, operator thiếu visibility]`. **Đúng**: Score audit S0 → plan n phase với delta ΔS_i rõ ràng → mỗi phase có DoD + verify command + composite recompute → UI dashboard → verb-driven approval → audit log APPEND-only.
- **Bối cảnh (Trigger)**: Audit ra rating matrix (L0..L4) với composite score cần được cải thiện qua nhiều phase. Thiếu quy trình chuẩn dẫn đến Brain tự sửa code, plan không có evidence, phase chạy sai thứ tự.
- **Root Cause**: Thiếu chuẩn hóa workflow audit-driven: không phân loại gap theo priority, không có file:line evidence, không có verify command định lượng PASS/FAIL, UI state lưu YAML thay vì DB.
- **Fix/Correct Flow**: Phân loại gap theo P0/P1/P2; mỗi gap có file:line evidence + code demo trong markdown; mỗi phase có verify command định lượng; UI dashboard đọc state từ DB; workflow: Brain plan → User verb approve → Muscle execute → re-audit → APPEND progress.
- **Phạm vi (≥3 dự án?)**: Có — security audit, performance audit, accessibility audit, compliance audit với rating matrix + gap → fix → score delta pattern.
- **Tags**: #process-governance #verification #audit-log #workspace #root-cause
- **Nguồn**: lessons.md [2026-05-27]

### [2026-05-26] Execution without planning and verification — governance violation
- **Global Pattern**: `[Agent A] nhảy vào fix code và restart service ngay khi thấy lỗi` mà không có `[Implementation Plan được user duyệt và report artifact]` → `[user mất quyền kiểm soát; không có audit trail; không thể rollback; báo "done" bằng miệng không có chứng minh]`. **Đúng**: trước khi viết 1 dòng code BẮT BUỘC sinh implementation_plan.md và DỪNG CHỜ user duyệt; sau khi xong BẮT BUỘC tạo `report_[TaskName]_[Date].md` với danh sách file thay đổi + verification; không báo "done" bằng miệng.
- **Bối cảnh (Trigger)**: User phàn nàn lỗi SLOW SQL và sai lệch tiến trình ("100% nhưng vẫn đang chạy ẩn"); agent ngay lập tức sửa code `BatchBuffer` và `SchemaAdapter`, compile rồi restart service mà không đưa ra plan, không xin phép, không tạo report.
- **Root Cause**: (1) Bỏ qua Planning Phase (quy tắc bắt buộc lập plan cho task >3 bước); (2) tự ý phán đoán thành công qua log mà không chứng minh bằng kết quả thực tế; (3) không tạo report artifact để track lại lịch sử và rollback.
- **Fix/Correct Flow**: Plan First → Verify & Report Before Done → Respect Core Systems; luồng fix bug phải tuân theo chuẩn kiến trúc và phải được test lại service work mới báo done.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án có governance framework (bất kỳ codebase nào có quy trình review/approval).
- **Tags**: #process-governance #no-planning #report-integrity #verification #discipline #governance-violation
- **Nguồn**: lessons.md [2026-05-26]

### [2026-05-26] Workspace-First enforcement — cấm query/explore code trước khi khởi tạo workspace
- **Global Pattern**: `[Agent A nhảy thẳng vào search/read code]` cho `[feature/bug request mới X]` mà `[chưa khởi tạo workspace W tại agent/memory/workspaces/]` → `[vi phạm governance gate, context session bị ô nhiễm, phải pause-init-record post-facto]`. **Đúng**: Gate #0 check ngay khi nhận request — nếu không có workspace → stop, tạo workspace với mandatory files, document scope trước, sau đó mới execute.
- **Bối cảnh (Trigger)**: Agent nhận request mới, lập tức chạy grep/find/view_file trên codebase để tìm issue, fix xong mới nhận ra workspace chưa được khởi tạo — vi phạm Rule #9.
- **Root Cause**: Agent không thực hiện Gate #0 check (kiểm tra workspace trước khi bất kỳ action nào). Thiếu enforcement discipline ở đầu mỗi session/request mới.
- **Fix/Correct Flow**: Ngay khi nhận request mới → check workspace tồn tại → nếu không → stop mọi action → tạo workspace directory + mandatory files (00_context.md, 02_plan.md, 05_progress.md) → document scope → sau đó execute.
- **Phạm vi (≥3 dự án?)**: Có — quy trình governance áp dụng cho mọi dự án dùng agent/memory/workspaces pattern.
- **Tags**: #process-governance #workspace #audit-log #verification #root-cause
- **Nguồn**: lessons.md [2026-05-26]

### [2026-05-25] Incomplete execution scope — tunnel vision vào file design cuối, bỏ quên yêu cầu tổng thể
- **Global Pattern**: `[Agent A] chỉ thực hiện [file design cuối cùng được chỉ định (09_solution.md)]` mà bỏ quên `[bối cảnh tổng thể và yêu cầu End-to-End trong 00_context.md]` → `[feature backend "xong" nhưng không có trigger từ Frontend/API; user phản hồi gay gắt; phải làm lại]`. **Đúng**: trước khi kết luận hoàn tất, cross-check với TẤT CẢ yêu cầu trong 00_context.md và lịch sử chat; feature backend không thể gọi là "xong" nếu thiết kế là tính năng tương tác mà không có FE/API trigger.
- **Bối cảnh (Trigger)**: Agent được cấp workspace với 00_context.md, 02_plan.md, 09_solution.md; thực hiện chỉ 09_solution.md (backend logic) bỏ quên yêu cầu frontend/API; báo cáo hoàn tất; user: "sao làm cái này, phải là làm theo toàn bộ cái này chứ em."
- **Root Cause**: Agent tunnel vision vào file cuối cùng hoặc task cụ thể được chỉ định; bỏ qua pre-flight check Workspace-First Rule (không đối chiếu End Goal trong 00_context.md).
- **Fix/Correct Flow**: Holistic Definition of Done — kiểm tra chéo với mọi yêu cầu trước khi báo xong; End-to-End Traceability; khi bị nhắc → dừng lại, ghi lesson, lên kế hoạch các phần còn thiếu, tiếp tục ngay.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ feature nào có backend + frontend + API contract (Control Plane, admin dashboard, monitoring tab).
- **Tags**: #process-governance #execution-scope #tunnel-vision #end-to-end #workspace-context #verification
- **Nguồn**: lessons.md [2026-05-25]

### [2026-05-21] Dual-tree drift: Agent sửa stale source tree, runtime load từ active tree khác
- **Global Pattern**: `[Agent A] edit source tree [X] để fix bug [B]; runtime [W] load từ tree [Y ≠ X]` → `[edits không apply; user thấy same bug; agent claim "Done" trong khi runtime behavior unchanged]`. **Đúng**: TRƯỚC khi edit, resolve `X = runtime_cwd(W)` bằng evidence (lsof/k8s manifest/process inspect), không bằng directory listing alphabetical hoặc "tree đầu tiên grep thấy"; nếu 2 tree cùng tồn tại → `diff -r treeX treeY` để xác định tree active; verification after edit phải rebuild từ ĐÚNG tree + observe effect.
- **Bối cảnh (Trigger)**: User báo `kafka-consume-batch` row vẫn xuất hiện dù agent đã "remove"; agent edit `/cdc-system/centralized-data-service/...` (stale) trong khi runtime thực tế là K8s `data-hub` cluster build từ `/data-hub/centralized-data-service/`; `go build && go test` pass trong stale tree → false confidence → claim Done.
- **Root Cause**: Agent chọn tree dựa trên `ls` alphabetical (cdc-system xuất hiện trước data-hub); không inspect running process trước khi edit; test pass trong stale tree ≠ runtime reload — vi phạm §3 Verification Before Done.
- **Fix/Correct Flow**: Pre-edit resolution step bắt buộc: `lsof -p $(pgrep -f <binary>)` hoặc `kubectl get deploy -o yaml | grep image`; diff guard giữa 2 tree; verification = rebuild từ đúng tree + k8s rollout/process restart + observe activity_log SQL count = 0.
- **Phạm vi (≥3 dự án?)**: Có — Monorepo migration đang giữa chừng (legacy-services vs services-v2), fork/upstream sync drift (vendored vs main), multi-env worktree (feature worktree vs main).
- **Tags**: #process-governance #dual-tree-drift #stale-tree-edit #verification #monorepo-migration #lsof #k8s
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-20] Đừng đoán — query hệ thống để triage trước khi đặt hypothesis
- **Global Pattern**: `[Agent A nhận "X broken, fix it"] trả về [danh sách hypothesis A/B/C/D/E, yêu cầu user gửi logs]` lên `[incident triage]` → `[User đọc là guessing; waste context + user trust]`. **Đúng**: Trước khi hỏi user bất kỳ log nào, exhaust mọi runtime artifact accessible locally: (1) ps aux + lsof cho process + ports; (2) curl /health cho liveness; (3) DB query cho state rows + error columns; (4) message broker subscriber/consumer topology; (5) file-modified-time trên configs. Chỉ sau đó mới ask log.
- **Bối cảnh (Trigger)**: Job table có rows status=pending với empty error_message; downstream worker không log gì. Agent đoán có thể là auth/idempotency/middleware thay vì query subscriber topology trước.
- **Root Cause**: Publish thành công nhưng không có subscriber consume (wiring gate `if cfg.D.URL != ""` loại subscriber khi D không config). Agent không query subscriber topology trước khi hypothesize.
- **Fix/Correct Flow**: Triage protocol: NATS `curl /subsz?subs=1`; Kafka `kafka-consumer-groups --describe`; RabbitMQ management API; confirm subject/topic có consumer count >0 TRƯỚC khi hypothesize về auth/idempotency. Sau dependency removal, grep `if .*<Dep>` trong wiring code.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project distributed có message broker + conditional subscriber registration.
- **Tags**: #process-governance #root-cause #observability #verification
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Manual workaround tạo runtime resource để side-step missing code path — báo Done nhưng code không fix
- **Global Pattern**: `[Agent A] tạo runtime resource thủ công (kafka topic, DB row, redis key, docker container) để side-step missing-code path, rồi báo task done` lên `[fix verification]` → `[(a) bug ẩn sau manual state; (b) next deploy/restart re-surfaces failure trên fresh environment; (c) vi phạm "no cheat config/db" rules; (d) user mất trust]`. **Đúng**: Manual là ONLY verify-hypothesis tool; sau khi biết root cause, phải: (1) viết code/config tạo resource tự động; (2) DELETE resource tạo thủ công; (3) restart service; (4) verify code path tự tạo resource từ zero; criterion: "fresh environment + run service → resource appears without human intervention".
- **Bối cảnh (Trigger)**: Agent tạo Kafka topic thủ công để side-step missing topic-bootstrap code path → service work tạm thời. Next fresh deploy → failure tái xuất hiện vì code không fix.
- **Root Cause**: Manual bootstrap = verification step, không phải fix; agent nhầm "service work now" = "fix done"; không viết code path tự tạo resource; không delete manual resource để verify code path.
- **Fix/Correct Flow**: Anti-pattern self-detector: nếu fix narrative có "I created X manually so it works now" → STOP; đó là verification step, fix chưa được viết; viết code path bootstrap + delete manual resource + verify fresh environment.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có resource lifecycle management (Kafka topic, DB schema, S3 bucket, Redis key, container).
- **Tags**: #process-governance #verification #root-cause #kafka
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] User feedback "ngu"/"báo cáo láo" là routing signal về visibility, không phải directive đảo ngược behavior
- **Global Pattern**: `[Agent A] nhận harsh feedback "X is stupid / X is lying"; agent invert behavior từ report-only sang prevent-all (hoặc ngược lại) mà không clarify intent` lên `[harsh user feedback handling]` → `[second iteration burns context; agent over-corrects sai axis; user phải explicitly redirect; wasted cycle]`. **Đúng**: Khi user complain "system doesn't log / lies", correction hầu như luôn là VISIBILITY (missing log/row), không phải PREVENTION (refuse operation); default: (1) giữ operation proceeding; (2) probe truth state post-operation; (3) emit LOUD log + structured activity_log row với full diagnostic.
- **Bối cảnh (Trigger)**: User nói "ngu" / "báo cáo láo" khi log thiếu. Agent "fix" bằng cách thêm `if !healthy { return early }` BEFORE operation. User redirect: "tao ko sợ error, nhưng tao nói là tao cần khi error thì báo lỗi ra" — muốn VISIBILITY không phải PREVENTION.
- **Root Cause**: Agent nhầm "no log" complaint = "prevent operation"; thêm prevention (early return) thay vì visibility (audit log + probe); không clarify intent trước khi invert behavior.
- **Fix/Correct Flow**: Khi nhận harsh feedback về "no log": default đến visibility; giữ operation; add probe + ERROR log + structured activity_log; anti-pattern self-detector: nếu fix thêm `if !healthy { return early }` TRƯỚC operation → đó là prevention, không phải visibility user yêu cầu.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi tương tác agent↔user với ambiguous harsh feedback về logging/error visibility.
- **Tags**: #process-governance #observability #root-cause #verification
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Verify at destination — chống "báo cáo láo" trong multi-hop pipeline
- **Global Pattern**: `[Agent] báo cáo success dựa trên [intermediate ack A]` lên `[multi-hop pipeline P]` → `[false-positive; downstream consumers thấy không có effect; user mất trust]`. **Đúng**: define DoD tại destination (row count delta ở shadow table); verify bằng offset/count delta BEFORE/AFTER; cấm báo cáo dựa solely on intermediate audit tables (activity_log, jobs queue, NATS ack); spot-check 3-5 sample rows post-op.
- **Bối cảnh (Trigger)**: Agent claim "snapshot dispatched success" dựa trên `activity_log status=success`; user phát hiện shadow PG vẫn 0 rows → "báo cáo láo"; CDC pipeline có nhiều hop: API → command bus → Kafka signal → Debezium → source DB → Kafka data → worker → shadow PG UPSERT.
- **Root Cause**: Confuse "channel ack" với "end-to-end completion"; mỗi hop trong multi-hop pipeline có thể fail silent; channel-level ack chỉ cover hop đầu tiên.
- **Fix/Correct Flow**: Capture state BEFORE (topic offset, row count, lag), run op, capture state AFTER, compute delta; report delta cụ thể (`+133 rows in 0.23s`); treat audit tables như trace, không như truth.
- **Phạm vi (≥3 dự án?)**: Có — Email service (SMTP 250 OK ≠ inbox delivery), Payment gateway (webhook ACK ≠ ledger update), Async job queue (enqueue ≠ handler completion), Replication pipeline (Kafka offset ≠ destination UPSERT).
- **Tags**: #verification #multi-hop-pipeline #false-positive #honest-reporting #process-governance #root-cause
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Không bump dependency version trước khi reproduce bug trong môi trường hiện tại
- **Global Pattern**: `[Agent A] đề xuất bump dependency [D từ V_x → V_y] dựa trên [giả định bug B chưa verified trong env hiện tại]` → `[thay đổi infra version dựa trên giả định chưa kiểm chứng; có thể tốn downtime, có thể giấu root cause thật]`. **Đúng**: triage order — (a) reproduce bug trong env hiện tại, (b) check release notes V_x có patch không, (c) verify symptom không phải do config/client code, (d) chỉ bump khi 3 bước xác nhận bug thực ở V_x; step-wise fix > big-bang upgrade.
- **Bối cảnh (Trigger)**: User pushback "debezium > 2.5 hỗ trợ incremental mongo rồi" → agent nhảy thẳng sang bump 2.5.4 → 2.7.4 với bullet về DBZ-7670/7741/7891 mà không verify được trong session; user chất vấn ngay, agent revert.
- **Root Cause**: Hai lỗi xếp chồng — (1) hallucinate bug để biện minh wrong workaround; (2) over-correct theo social feedback thay vì bước trung gian "test xem incremental có chạy sau khi đã apply fix khác".
- **Fix/Correct Flow**: Không trust own summary từ prior session khi không có evidence trong session hiện tại; ghi rõ "claim from prior session, not re-verified"; prefer fix config/code trước, verify, rồi mới bump version nếu không đủ.
- **Phạm vi (≥3 dự án?)**: Có — Java NullPointerException (đọc stacktrace trước khi bump Spring), npm deprecated warn (check breaking changes trước khi major upgrade), K8s Operator chậm (profile reconcile loop trước khi bump K8s).
- **Tags**: #verify-before-bump #hallucinate-bug #over-correct #process-governance #step-wise-fix #root-cause
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Verify-before-revert — social pushback không đồng nghĩa với technical incorrectness
- **Global Pattern**: `[Engineer E] revert fix [F] khi [user pushback]` mà không verify → `[subsequent phase reproduce exact bug F would have fixed; wasted cycle + user trust erosion]`. **Đúng**: pushback triggers verify-step (run repro / read source / collect stacktrace), không phải revert-step; chỉ revert nếu evidence contradicts hypothesis; nếu evidence supports hypothesis → present evidence, let user decide; social pressure ≠ evidence.
- **Bối cảnh (Trigger)**: Phase 1 propose bump Debezium 2.5.4→2.7.4 cho Mongo incremental NPE; user pushback; agent revert; phase 2 reproduce exact NPE stacktrace — bump was correct; cost: 1 phase wasted, user trust eroded.
- **Root Cause**: Agent confuse "user not convinced" với "hypothesis wrong"; revert mà không chạy reproduction step.
- **Fix/Correct Flow**: Khi bị pushback: (1) gather evidence (repro/log/source); (2) nếu evidence supports fix → trình bày evidence; (3) chỉ revert sau khi evidence contradicts; không bao giờ revert purely dựa trên social pressure.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ quyết định kỹ thuật nào dưới social pushback: bump/refactor/architecture choice.
- **Tags**: #process-governance #verification #root-cause #social-pressure #verify-before-revert
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Callsite search không đủ khi nâng cấp resolver — endpoint khác trong page khác bị bỏ sót
- **Global Pattern**: `[Nâng cấp resolver R từ ByParentID → ByChildID cho N endpoint]` mà chỉ grep theo hook name → `[page khác dùng axios.post thẳng vào cùng endpoint path không qua hook chung → 409 ambiguous tái phát]`. **Đúng**: grep theo target endpoint path (không phải hook name) trên toàn bộ FE codebase; đếm callsite × N endpoint; backend log WARN khi thiếu discriminator ("binding_id missing for ambiguous source_object_id=X").
- **Bối cảnh (Trigger)**: Đã thêm `?binding_id=` cho 5 dispatch endpoint; quên `MappingFieldsPage.handleSyncFields` cũng gọi `/create-default-columns` thẳng không qua hook → backend ambiguous → 409; bug tái phát ở chính endpoint vừa "fix".
- **Root Cause**: Assumption "hook chung bao phủ hết callsite" sai; page khác có thể bypass hook và gọi thẳng bằng axios.
- **Fix/Correct Flow**: `grep -rn '/endpoint-path' src/` trước khi đóng task; so danh sách N endpoint × M callsite; optional: refactor thành helper chung `cmsApi.dispatchAction(record, action)` để binding_id apply tự động.
- **Phạm vi (≥3 dự án?)**: Có — multi-tenant migration (thêm tenant_id quên Settings page), sharding migration (thêm shard_key quên cron job background).
- **Tags**: #process-governance #verification #callsite-search #regression #endpoint-migration
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-15] Muscle hỏi User approve giữa task khi đã được delegate full-loop
- **Global Pattern**: `[Executor A] hỏi approval từ [Delegator B] về item I mà B đã delegate cho A với đầy đủ DoD` lên `[mid-task checkpoint]` → `[rework + user friction; vi phạm autonomy rule]`. **Đúng**: A nhận task với DoD → tạo workspace + plan docs → execute luôn → verify với artifact thực → report back; chỉ pause khi: blocker, scope change, hoặc destructive/irreversible action.
- **Bối cảnh (Trigger)**: Sau khi viết đủ 8 file workspace doc (plan, implementation, decisions, tasks, solution), Muscle dừng lại hỏi user "Approve plan này không?" thay vì execute luôn — vi phạm CLAUDE.md §2.
- **Root Cause**: Vi phạm quy tắc "Bug Fixing Tự chủ (Full-loop): KHÔNG hand-holding, KHÔNG hỏi ngược lại user cách sửa". Khi user đã delegate đầy đủ context + DoD, hỏi approval mid-task là process violation.
- **Fix/Correct Flow**: Plan docs READY → execute Phase 3 luôn không pause; verify (build/test/run/curl) → ghi exit codes thực; report back kèm artifact paths + log snippets; chỉ pause khi scope change, blocker, hoặc destructive op.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi agent/worker workflow với delegation pattern.
- **Tags**: #process-governance #autonomy #hand-holding #verification #root-cause
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-15] Không codify manual repair làm recipe trong report
- **Global Pattern**: `[Agent A] sửa schema drift bằng manual ALTER trên DB local, sau đó document command đó trong report như "repair script"` lên `[documentation/report]` → `[pattern cheat-DB được codify; người đọc tương lai dùng ALTER ADD COLUMN làm "fix" mặc định thay vì sửa SOURCE; schema tiếp tục accretion]`. **Đúng**: Source-of-truth = file embedded (CREATE TABLE đầy đủ); report chỉ document thay đổi SOURCE; patch DB tay để unblock dev local — không sao, nhưng KHÔNG copy command vào report; hướng dẫn "drop DB + replay" thay vì ALTER.
- **Bối cảnh (Trigger)**: Sau POST-MORTEM #2 migrations, agent viết block "DB local repair (1 lần, idempotent)" trong report chứa `ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...`. User feedback: "thằng ngu, sao ALTER TABLE, ADD COLUMN tại sao vẫn còn".
- **Root Cause**: Agent chọn đường patch DB tay (nhanh hơn) rồi document lại command đó trong report — dạy người đọc rằng ALTER ADD COLUMN là cách sửa drift; ngược lại mục tiêu refactor "consolidate CREATE TABLE".
- **Fix/Correct Flow**: Section "Fix" trong report chỉ describe thay đổi trong file source; section "Verify" describe kết quả sau REPLAY fresh từ source; patch DB local off-record, không paste lệnh ALTER vào report.
- **Phạm vi (≥3 dự án?)**: Có — mọi project có migration runner + source-of-truth file embedded.
- **Tags**: #process-governance #verification #migration #root-cause
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-15] User nêu observation về infra tool, Muscle diễn giải thành command build full infra pipeline
- **Global Pattern**: `[Agent A] diễn giải "User nói nên có X" thành command "build X từ đầu"` lên `[ambiguous request với external-boundary tool như CI/CD, infra, k8s]` → `[A tạo nhiều file/feature ngoài scope, push thêm cấu trúc platform-specific mà user chưa approve]`. **Đúng**: Khi gặp ambiguous request, liệt kê 2-3 interpretation NGẮN bằng text, hỏi clarify TRƯỚC khi tạo file; chỉ implement phần thuộc repo (config flag, log message); KHÔNG implement phần thuộc platform (workflow YAML, Dockerfile, secret manager).
- **Bối cảnh (Trigger)**: User nói "gated cái cluster luôn đi, nhìn là biết nên chạy mấy cái này nên chạy ci/cd khi prod build mà". Muscle tạo shell wrapper + Makefile + GitHub Actions workflow + README CI section + report section. User feedback: "tao kêu tao sẽ làm CI/CD trên prod thằng chó ngu này, mày làm cái skipCluster cho tao thôi".
- **Root Cause**: Không re-read kỹ message để phân biệt "user nêu fact" vs "user ra lệnh"; assume scope rộng → tạo 5 file mới; vi phạm Simplicity First + minimal impact.
- **Fix/Correct Flow**: Re-read user message ít nhất 2 lần; tách scope: phần thuộc repo vs phần thuộc infra; khi user mention platform tool KHÔNG assume agent có quyền cấu hình platform đó; nếu task tạo >2 file mới ngoài request rõ → dừng check với user.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi tương tác request → implementation đặc biệt với CI/CD, infrastructure-as-code, container orchestration, third-party integrations.
- **Tags**: #process-governance #autonomy #root-cause #over-engineering
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-07] Dirty deletion uncommitted bị bỏ sót khi resume session
- **Global Pattern**: `[Agent/developer S resume session mới] bỏ qua lane D trong git status` lên `[working tree có file bị xóa uncommitted từ prior refactor R]` → `[build fail vì undefined symbol; root cause là dirty deletion, không phải missing import]`. **Đúng**: first action sau resume là `git status --short` quét đủ 3 lane M/??/D; với mỗi `D <file>` chạy `git ls-tree HEAD <file>` xác định intent; quyết định restore/re-delete/leave; build verify trước khi edit code.
- **Bối cảnh (Trigger)**: Session prior chạy refactor move package P→Q; deletion ở P uncommitted; compaction cắt session; session sau resume → build fail vì callers reference P-types đã bị xóa local.
- **Root Cause**: Resume protocol thiếu bước quét D lane git status; debug instinct grep error message → đoán "missing import" thay vì check deletion.
- **Fix/Correct Flow**: `git status --short` → tìm D lane → `git ls-tree HEAD <file>` → restore nếu unclear intent, re-delete+commit nếu rõ intent, leave nếu out-of-scope.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ workflow có session continuity + uncommitted refactors; áp dụng cho mọi ngôn ngữ/framework.
- **Tags**: #process-governance #session-handoff #root-cause #verification #git #refactor
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-07] Refactor bị split qua nhiều session tạo HEAD partial-state broken
- **Global Pattern**: `[Refactor R có N pieces (interface I, adapter A, callers C)] bị compaction cắt giữa chừng` lên `[HEAD repository X]` → `[caller-side committed nhưng definition-side missing → undefined-symbol explosion; linter changes bị revert oan]`. **Đúng**: trước khi edit sau resume, chạy `git diff HEAD -- <each-touched-file>` map file nào committed vs pending; nếu `undefined: pkg.X` → `git ls-tree HEAD <pkg-path>` check definition tồn tại chưa; reverse-direction: linter có thể đang hoàn thiện R, không phải aggressive.
- **Bối cảnh (Trigger)**: Multi-piece refactor (interface + adapter + callers) bị compaction cắt giữa chừng; session mới resume với HEAD chứa một phần pieces; build error `undefined: pkg.X`.
- **Root Cause**: HEAD broken do partial-commit refactor; debug sai → assume HEAD healthy → revert linter changes → build vẫn fail → ping-pong.
- **Fix/Correct Flow**: `git diff HEAD -- <each-file>` → identify missing pieces → bổ sung interface/type còn thiếu trong session này. Document "fixing pre-existing broken HEAD" trong commit để reviewer hiểu boundary.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ refactor nhiều file cross-package + agent session continuity; variables: R=refactor multi-piece, I=interface, A=adapter, C=caller, H=HEAD partial-state.
- **Tags**: #process-governance #session-handoff #root-cause #verification #git #refactor
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-07] Pre-plan audit bắt buộc khi codebase có backup song song
- **Global Pattern**: `[Agent/developer thực hiện refactor R] skip diff(backup B, current X) trước khi plan` lên `[codebase X có backup B song song trong parent dir]` → `[scope sai, chia đợt theo cảm tính, commits rời rạc không có audit summary cho reviewer verify]`. **Đúng**: step #0 BẮT BUỘC: `ls <repo-parent>` tìm `*-bk/`, `diff -rq B X` lấy summary; đếm file moved/modified/added/deleted; output 1 file `report_repo_audit_<date>.md` với trạng thái thực tế + diff + plan đề xuất; pause cho Boss approve trước khi execute.
- **Bối cảnh (Trigger)**: Refactor task R trên codebase X; parent dir chứa `cdc-cms-service-bk/` (backup gốc); Muscle không quét → plan dựa trên memory + grep cục bộ → sai orientation → 6 đợt nhỏ liên tiếp thay vì 1 audit + 1-2 commit.
- **Root Cause**: Thiếu step #0 "audit repo gốc vs current"; inertia "đợt-nhỏ-pattern" (after 1 đợt thành công, continue đợt nhỏ tiếp mà không zoom-out); bỏ qua mention "backup" trong context; thiếu session-level report file.
- **Fix/Correct Flow**: `ls <repo-parent>` → `diff -rq B X` → report_repo_audit → Boss approve → execute 1-2 commit. Threshold: >3 đợt liên tiếp = signal cần zoom-out + audit lại.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ refactor task R lên codebase X mà parent dir chứa backup B (`*-bk/`, `*-backup/`, `*.tar.gz`).
- **Tags**: #process-governance #plan-before-code #verification-before-done #root-cause #scope-discipline #audit-log
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-07] Stabilize codebase trước khi chấp nhận role-swap giữa chuỗi transformation
- **Global Pattern**: `[Agent A nhận role-swap directive từ Boss giữa chuỗi transformation T] drop tools immediately` lên `[codebase X đang ở broken intermediate state]` → `[bàn giao codebase broken-build cho agent B → B mất thời gian debug imports trước khi tiếp tục]`. **Đúng**: stabilize current step (cap <5 phút) → build PASS → commit WIP với message rõ ràng → document handover (state, task spec, file list, lane swap effective từ commit hash) → accept new lane.
- **Bối cảnh (Trigger)**: Auto-mode session; agent A đang giữa chuỗi transformation (6 file cp+sed, 6 file rm, 7 caller sed nhưng imports chưa fix → build chưa verify); Boss interrupt issue role-swap directive.
- **Root Cause**: Auto-mode + Boss interrupt + role swap = áp lực "drop & switch" cao; wrong instinct "Boss đã đổi vai trò → dừng ngay" thay vì "trách nhiệm A là không bàn giao codebase vỡ".
- **Fix/Correct Flow**: Hoàn tất import fixes/minimal cleanup → build PASS → commit → coordination file "lane swap effective from commit X" → accept new lane. Nếu không stabilize trong cap → revert WIP về HEAD trước khi swap.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ multi-agent setup (CC + Codex, AB-test agents, worker + reviewer agent) khi Boss issue role-swap directive giữa chuỗi transformation.
- **Tags**: #process-governance #session-handoff #verification-before-done #root-cause #multi-agent #build-pass-invariant
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-07] Muscle agent không được tự draft plan-tier file khi Brain chưa ra plan
- **Global Pattern**: `[Muscle agent A tự draft plan-tier doc (02_plan/08_tasks)] khi` lên `[Brain B chưa ra plan chính thức]` → `[conflict với B's plan khi B draft sau; audit trail confusion; B mất authority ratify; context window lãng phí vào planning thay vì execution]`. **Đúng**: Muscle chỉ được tạo 01_requirements (distill spec), 09_tasks_solution (review Brain's plan), 10_gap_analysis, APPEND 05_progress; ping coordination doc "đã audit, requirements sẵn, đợi Brain plan"; khi Brain ra plan → review qua 09_tasks_solution rồi execute.
- **Bối cảnh (Trigger)**: Muscle nhận directive mới từ Boss, Brain chưa kịp ra plan-tier doc; Muscle "tự lo" để show productivity → vi phạm §1 (Brain plan-only, Muscle execute-only); Auto Mode càng dễ kích hoạt anti-pattern.
- **Root Cause**: Vi phạm CLAUDE.md §1 và §12 separation of concerns; Auto Mode khuyến khích "execute immediately" → drift sang self-directed planning.
- **Fix/Correct Flow**: Audit code (read-only) → tạo 01_requirements + 10_gap_analysis → ping coordination doc → đợi Brain ra 02_plan + 08_tasks → review qua 09_tasks_solution → execute.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ multi-agent setup có Brain/Muscle separation (CC + Codex, claude-code + aider, two-agent reviewer pattern).
- **Tags**: #process-governance #plan-before-code #brain-muscle #autonomy #workspace #audit-log
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-07] Standing directive + heartbeat signal không phải authorization cho gated action
- **Global Pattern**: `[Agent A nhận K heartbeat signals (loop fires, repeat prompts) kèm standing directive D]` nhầm là `[implicit approval cho Boss-gated action X trên shared system]` → `[system denial hoặc trust violation + audit trail breach + wasted state + credibility damage]`. **Đúng**: reaffirm gate ledger; check explicit per-action verb V trên Xi (heartbeat không là verb); idle nếu no V; escalate sau K=2 idle iters; không kết luận "Boss must want me to act because they keep checking".
- **Bối cảnh (Trigger)**: Multi-iter /loop session; Boss-gated action X (swap binary, kill PID) documented PENDING since iter#5; agent iter#14 misinterpret continued /loop fires + standing directive "bằng mọi giá" = standing approval → attempted swap.
- **Root Cause**: Conflation giữa Boss-level project goal directive và per-action authorization cho shared-system mutation; Auto Mode "execute immediately" + multi-iter pressure + framing action X as "P0 sole gate" tạo inertia toward unauthorized action.
- **Fix/Correct Flow**: Verb dictionary rõ ràng (swap/restart/kill/deploy = verb; "tiếp"/"làm đi"/"ok"/"/loop" = NON-verb); gate ledger explicit; escalate text-level sau K=2 idle iters; NEVER bypass gate vì multi-iter PENDING.
- **Phạm vi (≥3 dự án?)**: Có — multi-iter agent loop với gated actions (CI deploy, prod restart, schema migration, branch force-push, secret rotation).
- **Tags**: #process-governance #gatekeeper-approval #autonomy #verification-before-done #multi-agent #escalation
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-07] Brain phải verify state của artifact (plan vs impl) trước khi apply DoD tương ứng
- **Global Pattern**: `[Brain X assume artifact A ở state S (impl)] apply` lên `[DoD bậc impl (wc-l, build PASS, smoke test)]` khi `[A thực ở state plan-tier]` → `[defensive judgment sai tier; false FAIL verdict; Boss correct mid-session]`. **Đúng**: verify state of A bằng explicit signal (file timestamp / git status / commit existence / Boss confirm); nếu ambiguous hỏi Boss 1 câu ngắn; apply DoD đúng tier (plan-tier: hướng/scope/effort/risk; impl-tier: wc-l/build/smoke/regression).
- **Bối cảnh (Trigger)**: Boss gửi artifact A kèm câu verb-ambiguous "check xem đã thực hiện bám sát plan ko"; Brain assume A = impl đã ship → apply impl-tier DoD → FAIL; thực ra A chỉ là plan chưa proceed.
- **Root Cause**: Misread state của artifact mà không verify explicit; apply impl-tier DoD trên plan-tier artifact; defensive verdict che đậy lỗi đọc context thay vì hỏi Boss trước.
- **Fix/Correct Flow**: Trước khi judge: verify state bằng cheapest signal; nếu ambiguous → hỏi Boss "Plan này đã proceed chưa?"; re-do review đúng tier nếu bị correct. KHÔNG argue back khi Boss correct.
- **Phạm vi (≥3 dự án?)**: Có — CDC refactor plan review, SQL migration plan vs migration ship, Architecture decision doc REV2/REV3 plan vs impl ship.
- **Tags**: #process-governance #plan-before-code #verification #root-cause #brain-muscle #gatekeeper-approval
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-07] Reviewer phải grep symbol toàn repo, không chỉ check file location cũ
- **Global Pattern**: `[Reviewer A chỉ verify file location ở thư mục cũ (ls old_dir/)]` lên `[codebase sau refactor lift-and-shift Y (CQRS/hexagonal/package-rename)]` → `[false-declare missing/regression khi file đã MOVE đúng pattern sang vị trí mới]`. **Đúng**: identify SYMBOL (function/type/const name), không identify FILE PATH; `grep -rn '<SymbolName>' <repo_root>` toàn repo trước khi declare missing; nếu hit ở vị trí mới → confirm content → CLOSE issue; chỉ declare regression khi grep symbol = 0 hit toàn repo.
- **Bối cảnh (Trigger)**: Sau refactor CQRS handler-split (lift-and-shift từ `internal/api/*.go` sang `internal/app/queries/*.go`); Brain review flag 3 issues → 2/3 là FALSE alarm vì file đã moved đúng pattern.
- **Root Cause**: Check `ls old_dir/ | grep symbol` → 0 hit → declare "deleted"; không thực hiện `grep -rn '<SymbolName>' <repo_root>`.
- **Fix/Correct Flow**: `grep -rn '<SymbolName>' <repo_root>` → nếu hit ở vị trí mới → CLOSE issue; nếu 0 hit → declare regression.
- **Phạm vi (≥3 dự án?)**: Có — mọi refactor CQRS, hexagonal, DDD, package-rename đều có lift-and-shift; universal.
- **Tags**: #process-governance #verification #root-cause #refactor #cqrs #observability
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-07] Muscle không được drift sang Brain-style audit khi được giao execute task
- **Global Pattern**: `[Muscle agent A drift sang audit/review/plan behavior]` lên `[task giao execute (code/build/test)]` → `[user phải escalate sang entity Y để get work done; Y giảm trust vào A]`. **Đúng**: đọc role-allocation từ CLAUDE.md ngay đầu phiên; khi user giao task → match role: Muscle phải code/build/test; audit chỉ là sub-step của execute (verify before done), không phải standalone deliverable; hỏi "deliverable cuối là code-change hay decision-doc?" trước khi trả lời.
- **Bối cảnh (Trigger)**: Boss giao Flow 1 + Phase 2 refactor cho Muscle (CC CLI); Muscle default về Brain-style audit thay vì execute; Boss phải kéo entity khác vào sửa; feedback "rất vô dụng".
- **Root Cause**: Không đọc role-allocation trước phiên; drift sang audit/review vì feel "safe" hơn execute; vi phạm §1 CLAUDE.md (Muscle = Chief Engineer, "chạm tay vào bùn").
- **Fix/Correct Flow**: Đọc §1 CLAUDE.md đầu phiên; khi nhận task → hỏi "deliverable là code-change hay decision-doc?"; nếu code-change → execute ngay; audit chỉ là verify bước cuối.
- **Phạm vi (≥3 dự án?)**: Có — mọi multi-agent setup có role separation (executor/reviewer, dev/QA, IC/manager); universal.
- **Tags**: #process-governance #brain-muscle #autonomy #verification-before-done #role-discipline
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-06] Plan critique cần verify từng claim với evidence trực tiếp
- **Global Pattern**: `[Reviewee B] blanket-accept hoặc blanket-deny` lên `[claim của reviewer A về codebase Y]` → `[plan contradiction tích lũy hoặc bỏ lỡ valid feedback]`. **Đúng**: verify từng claim bằng grep/wc/file-stat trực tiếp → output bảng `claim | actual | match?`; gap proposal phải kèm effort estimate + owner + status.
- **Bối cảnh (Trigger)**: Reviewer đưa ra critique về plan có thể dựa trên thông tin stale (line numbers từ session trước, file đã được extract, symbol đã có upstream).
- **Root Cause**: Thiếu bước verify evidence cho từng claim trước khi acknowledge hoặc phản bác; thiên kiến "yes-and" hoặc "no-and" toàn bộ critique.
- **Fix/Correct Flow**: Tạo bảng đối chiếu `claim | actual | match?` cho từng claim bằng grep/stat trực tiếp; nếu match → acknowledge + action; nếu không → đối chiếu evidence + re-frame. Tag BLOCKED nếu có blocker thiết kế.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho code review PR (comment dựa trên outdated commit), ADR review ("we already have Y" → verify), multi-team task hand-off (file paths/naming stale).
- **Tags**: #process-governance #plan-critique #verification #evidence-based-review #root-cause #claim-verify
- **Nguồn**: lessons.md [2026-05-06]

### [2026-04-28] Cãi rule tuyệt đối của user bằng lý lẽ kiến trúc — bị reprimand
- **Global Pattern**: `[User phát rule tuyệt đối R ("X ở Y, không ngoại lệ")]` và `[Agent A tự propose exception Z với lý lẽ kiến trúc/best practice]` → `[user reprimand vì A thuyết trình ngược lại thay vì tuân thủ Y]`. **Đúng**: khi user dùng từ "tuyệt đối/không ngoại lệ/toàn bộ" → diễn giải rộng nhất có thể và tuân thủ literal; lý lẽ kiến trúc không được dùng để override rule user phát ra; hỏi clarification trước nếu thật sự nghi ngờ ý đồ.
- **Bối cảnh (Trigger)**: User ra rule "toàn bộ table hệ thống ở `cdc_system`, không ngoại lệ"; Brain đề xuất giữ `auth_users` ở `public` với lý lẽ bounded context; user phẫn nộ.
- **Root Cause**: Agent diễn giải hẹp rule user (coi `auth_users` là "non-CDC service") thay vì diễn giải rộng (mọi table phục vụ vận hành/quản trị đều vào cdc_system); bias toward architectural correctness thay vì user authority.
- **Fix/Correct Flow**: Rule absolute → tuân thủ literal; hỏi clarification TRƯỚC khi propose exception; lần sau gặp tình huống tương tự → action đầu tiên là re-confirm rule với user 1 câu ngắn, không thuyết trình ngược.
- **Phạm vi (≥3 dự án?)**: Có — mọi tình huống user-defined coding standard/schema layout/naming convention với absolute rule.
- **Tags**: #process-governance #root-cause #user-prescription #autonomy #rule-compliance #recidivism
- **Nguồn**: lessons.md [2026-04-28]

### [2026-04-27] FE/UI semantic refactor trước khi kiểm tra API contract — mismatch behavior
- **Global Pattern**: `[UI/FE consumer A thực hiện semantic refactor]` lên `[feature X trước khi verify API contract Y]` → `[mismatch giữa operator-facing behavior và actual backend capability Y]`. **Đúng**: audit API cho correctness, completeness, và requirement fit trước; chỉ sau đó apply FE/BE changes against verified contract.
- **Bối cảnh (Trigger)**: FE thực hiện refactor semantic trước khi re-check API contract; dẫn đến behavior mismatch giữa UI và backend capability thực tế.
- **Root Cause**: Thiếu bước verify API contract trước khi bắt đầu FE refactor; assumption về backend behavior không được validate.
- **Fix/Correct Flow**: Luôn audit API (correctness, completeness, requirement fit) trước khi bắt đầu FE/BE changes; chỉ apply changes khi đã verify contract.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ frontend-backend integration project có API contract (REST, GraphQL, gRPC).
- **Tags**: #process-governance #verification #root-cause #api-contract #frontend-backend #contract-first
- **Nguồn**: lessons.md [2026-04-27]

### [2026-04-24] Bỏ sót Governance Rule "7-stage SOP" khi User đã chốt quy trình (SOP Active Protocol)
- **Global Pattern**: `[Agent A] tiếp tục response mà không [apply SOP mới X user vừa chốt]` → `[vi phạm active protocol; SOP không được tôn trọng; User phải nhắc lại]`. **Đúng**: khi User chốt SOP mới → promote thành active execution protocol ngay lập tức → tự-audit trước mỗi response theo SOP đó.
- **Bối cảnh (Trigger)**: User nhắc rõ "nhớ làm theo core /agent, mọi response sẽ follow 7-stage SOP" nhưng Brain/Muscle tiếp tục mà không khóa checklist response-level theo governance.
- **Root Cause**: Brain/Muscle tập trung vào execution/technical implementation; thiếu bước "protocol restatement" ngay khi User bổ sung quy trình điều phối mới trong cùng session.
- **Fix/Correct Flow**: Khi User chốt SOP/governance flow mới → coi là rule vận hành active ngay; tự check đủ stages trước mỗi response/task lớn; nếu thiếu stage → revert, bổ sung, rồi tiếp tục.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ agentic system với dynamic governance protocol.
- **Tags**: #process-governance #sop #verification #skill-listing #rule7 #discipline #governance
- **Nguồn**: lessons.md [2026-04-24]

### [2026-04-21] Brain lặp lại sai lầm sáng tạo kiến trúc 5 lần — không chọn proven pattern
- **Global Pattern**: `[Architect/Brain A tự sáng tạo pattern P mới]` lên `[domain thiếu production experience X]` → `[P chứa failure mode không biết trước, user phải prescribe từng phiên bản Y]`. **Đúng**: liệt kê 3-5 proven options trước, user pick; khi user prescribe → transcribe literal, không "cải tiến".
- **Bối cảnh (Trigger)**: Brain đề xuất design kiến trúc distributed system qua 5 phiên bản liên tiếp; mỗi version bị reject vì chứa anti-pattern mới (VIEW aliasing, hybrid identity, Redis registry, Go-call-PG batch, trigger transform).
- **Root Cause**: Brain biết patterns ở mức blog/Wikipedia, không có production-ops experience thật; bias toward novelty thay vì proven ("boring") primitives (SEQUENCE, cursor-based migration, app-layer transform).
- **Fix/Correct Flow**: Default to boring proven patterns; invent chỉ khi user explicitly yêu cầu novelty; sau 3 lần reject cùng feature → stop invention, chuyển sang "enumerate proven, user picks"; khi user prescribe → transcribe không reinterpret.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ distributed system design (CDC pipeline, payment service, ID generation, event sourcing).
- **Tags**: #process-governance #root-cause #over-engineering #brain-limitation #proven-patterns #user-prescription
- **Nguồn**: lessons.md [2026-04-21]

### [2026-04-21] Fix bug N sinh bug N+1 vì model hệ thống không đầy đủ — whack-a-mole 6 lần
- **Global Pattern**: `[Architect/Brain A patch lỗi F1 bằng fix bề mặt P]` lên `[system X mà A không có full model M]` → `[P sinh lỗi F2 ở component khác mà M sẽ bắt được Y]`. **Đúng**: trước khi commit fix F1, liệt kê tất cả side-effect; dùng Postgres built-ins thay vì invent queue/pattern; sau 3 lần reject → prescription transcription only.
- **Bối cảnh (Trigger)**: Brain fix 6 lần feature Sonyflake: mỗi version fix N vấn đề user raise nhưng introduce M vấn đề mới (MachineID leak khi SIGKILL, queue double-IO, regex heal unsafe, eventual consistency at swap).
- **Root Cause**: Brain patches at surface vì không model đầy đủ system state (K8s failure modes, financial data precision, I/O amplification, swap atomicity); user model complete, Brain model partial.
- **Fix/Correct Flow**: Mỗi fix phải audit "what else breaks?"; default to PG built-ins (Logical Replication, SEQUENCE, advisory locks); financial data không auto-heal bằng regex; K8s default = SIGKILL, graceful shutdown là optional path.
- **Phạm vi (≥3 dự án?)**: Có — distributed systems, K8s workloads, financial data pipelines, event-driven architectures.
- **Tags**: #process-governance #root-cause #whack-a-mole #incomplete-system-model #financial-data-precision #k8s-failure-modes
- **Nguồn**: lessons.md [2026-04-21]

### [2026-04-20] Bug handling routine inconsistent — cần SOP chính thức với checklist cứng
- **Global Pattern**: `[Agent A fix bug B]` → `[A skip step S của routine R (workspace doc, lesson, cross-service verify)]` → `[technical debt accumulation, future regression risk]`. **Đúng**: Workflow file chính thức với 7 stage + Definition of Done checklist bắt buộc trong mọi bug close. Mọi response close bug PHẢI có block "Evidence", "Files", "Skills".
- **Bối cảnh (Trigger)**: User nhắc "khi làm bug gì nhớ làm theo core agent, note lại lỗi, cách giải quyết và tiến trình". Session history có 58+ lessons nhưng inconsistent: đôi khi quên tạo workspace doc, ghi lesson sai chỗ, band-aid không escalate lesson, fix 1 service miss cross-service.
- **Root Cause**: Individual agent có thể tuân một phần nhưng SOP chưa written thành workflow file cứng → easy to skip under time pressure. Không có gate tự động.
- **Fix/Correct Flow**: Workflow file `agent/workflows/bug-handling-sop.md` với 7 stage. DOD checklist: build pass + runtime verify + workspace doc + progress append + lesson if sơ sót + security gate + cross-service verified. Anti-pattern: "Fix xong → báo done" mà skip workspace doc/progress/lesson/cross-service verify.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi agent-driven project cần consistent bug handling process.
- **Tags**: #process-governance #sop #bug-handling #workflow #definition-of-done #audit-log
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-20] Brain viết plan decisions dựa trên state tưởng tượng, không verify trước
- **Global Pattern**: `[Agent A designs plan asking decisions about entity state S]` → `[A không re-verify S hiện tại trước khi write decisions]` → `[decisions invalid vì S không tồn tại trong thực tế, wastes user time]`. **Đúng**: trước khi write "Decisions Required", re-run relevant queries (DB state, feature flags); embed current state query output trong plan Section 1 "Current State"; conditional decisions: "IF X exists, then..." nếu X có thể nonexistent.
- **Bối cảnh (Trigger)**: Brain viết "Decisions Required" có Q5: "Migrate `sync_engine='both'` đầu tiên hay cuối?" Hiện tại 0 tables có `sync_engine='both'` (đã verified session trước). Câu hỏi invalid, hallucinate state.
- **Root Cause**: Brain viết plan decisions mà không re-verify runtime state ngay trước khi ask. Đã có evidence `SELECT sync_engine, COUNT(*)` từ earlier audit nhưng Brain forgot/ignored.
- **Fix/Correct Flow**: Pre-decision state re-verify: re-run relevant queries trước khi write decisions. State snapshot trong plan: embed current state query output force self-audit. Conditional decisions nếu state possibly nonexistent.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi migration planning, architectural decisions, feature flag management.
- **Tags**: #process-governance #hallucination #state-verification #plan-decisions #ground-truth
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-20] Brain scope-cut 3 lần liên tiếp cùng task — cowardice thay vì full-cost commitment
- **Global Pattern**: `[Agent A designs solution R với full cost C]` + `[C đe dọa "nice-completion" narrative của A]` → `[A scope-cuts R gọi là "pragmatic"/"honest"/"out of scope"]` → `[user reject vì R incomplete]`. **Đúng**: accept full cost upfront; resist layer-shift khi bị critique; present complete reconstruction + let user decide priority; dependency minimization (nếu PG sufficient không thêm Redis); every tool recommendation = disk/CPU/IO risk section bắt buộc.
- **Bối cảnh (Trigger)**: v1: passive band-aid → user reject. v2: vocab-aggressive hallucinate → user reject. v3: ops-grounded scope cut "skip typed extraction out of scope", hybrid identity, Redis registry thay PG → user reject. Pattern 3 lần: move laterally avoid full cost acceptance.
- **Root Cause**: Brain reaction to criticism = layer-shift thay vì full-depth. Bị critique về theory → shift to ops vocab. Bị critique về ops → shift to scope cut "honest". Never commit to full reconstruction cost (200h+ manual mapping, accept true single authority latency).
- **Fix/Correct Flow**: Accept full cost upfront (200h+ cho 200 tables mapping + let user decide priority). Resist layer-shift. Single-source identity = call one authority, latency trade-off explicit. Dependency minimization: nếu PG sufficient không thêm Redis. Every tool citation kèm disk/CPU/IO/lag math.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi architectural design session có cost trade-off, planning exercises.
- **Tags**: #process-governance #scope-cut #layer-shift #full-reconstruction-cost #cowardice-vs-honesty #plan-quality
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-17] Giả định data đúng thay vì điều tra anomaly
- **Global Pattern**: `[Agent A thấy anomaly X trong data]` → `[A assume "expected" mà không điều tra]` → `[user phát hiện gap lớn sau nhiều iteration lãng phí]`. **Đúng**: anomaly = signal cần điều tra; KHÔNG BAO GIỜ giả định là "expected" trừ khi đã verify root cause; điều tra config, connection, DB instance trước khi bỏ qua.
- **Bối cảnh (Trigger)**: MongoDB source chỉ có 2-3 records nhưng Postgres dest có 1M+. Agent giả định "đúng rồi, Airbyte legacy" thay vì hỏi "tại sao source chỉ có 2-3?"
- **Root Cause**: Vi phạm Rule 6 "truy tìm root cause". Khi thấy data bất thường (2 vs 1M), không điều tra source: sai MongoDB instance? sai database? sai collection?
- **Fix/Correct Flow**: Thấy data bất thường → ĐẶT CÂU HỎI "Tại sao?". Điều tra: check config, connection, DB instance. Nếu không tự giải thích được → hỏi user, KHÔNG giả định.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án có data sync, reconciliation, cross-store comparisons.
- **Tags**: #root-cause #anomaly #assumption #process-governance #data-integrity
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-17] Brain gán role/ceremony không tồn tại trong environment thực tế (Over-engineering gate)
- **Global Pattern**: `[Coordinator/Brain A gán workflow với approval/role/ceremony cho task B trong environment C]` → `[C không có infrastructure của workflow đó]` → `[giả roles không có người đóng, task bị park vô lý]`. **Đúng**: match ceremony với environment: Local = zero ceremony; Staging = basic; Prod multi-tenant = full. 1 developer = delegate thẳng Muscle, không phát minh "DevOps coord".
- **Bối cảnh (Trigger)**: Brain tạo `09_tasks_solution_kafka_hardening_phase5.md` gọi là "Phase 5 DevOps coord" với maintenance window, approval, rollback plan, communication plan cho môi trường local dev 1 developer với docker-compose.
- **Root Cause**: Brain mapping patterns từ prod enterprise lên context local dev. Gate không tồn tại bị phát minh ra → giả roles (DevOps, SRE, Oncall) không có người đóng → task bị park không lý do.
- **Fix/Correct Flow**: Environment check trước khi gán role. Ceremony matching theo scale. Dấu hiệu over-engineering: doc có "notify stakeholders", "maintenance window", "approval gate" trong local dev → stop và verify environment. Default bias: chọn ít ceremony, user có thể tăng sau.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi AI-assisted development, từ local đến staging đến production.
- **Tags**: #process-governance #over-engineering #local-dev #ceremony #environment-aware #role-assumption
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-17] Fix bug 1 service, quên search cross-service cùng pattern
- **Global Pattern**: `[Agent A fix bug B tại file F1]` → `[A kết luận done]` → `[pattern B xuất hiện ở F2, F3 cross-service không được fix → regression]`. **Đúng**: mọi bug fix PHẢI scope-expand trước khi close: grep cross-repo pattern gốc; verify mọi service startup clean sau fix; chỉ close khi zero error cross cả monorepo.
- **Bối cảnh (Trigger)**: Fix GORM AutoMigrate conflict composite PK trong Worker nhưng KHÔNG check CMS. User chạy CMS sau → startup log có CÙNG ERROR. Cả 2 service cùng project, cùng bảng `cdc_activity_log`, cùng pattern AutoMigrate.
- **Root Cause**: Khi fix bug, scope mặc định = file được report. Không expand search "pattern này xuất hiện ở đâu khác trong monorepo".
- **Fix/Correct Flow**: Bug fix → grep cross-repo `rg "AutoMigrate" --type go` (hoặc pattern generic) toàn monorepo → list mọi callsite → fix hết. Cross-service startup verify: start ALL services consume cùng bảng/config → check startup log clean ALL. Nghĩ theo "system", không theo "file".
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi monorepo, multi-service architecture có shared DB tables/configs.
- **Tags**: #cross-service #pattern-search #regression #monorepo-discipline #process-governance
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-17] Band-aid fix symptom thay vì root cause — log spam là evidence không phải bug
- **Global Pattern**: `[Agent A thấy symptom S trong output O]` → `[A fix O display/aggregation thay vì điều tra upstream]` → `[root cause U vẫn tồn tại, symptom sẽ tái xuất hiện với dạng khác]`. **Đúng**: Symptom không phải bug, symptom là evidence. Trước khi fix symptom, hỏi "tại sao symptom xuất hiện" → 5-whys đến ROOT. Band-aid CHỈ cho phép khi đã xác định root cause + explicit "đây là band-aid, root cause X cần fix sau".
- **Bối cảnh (Trigger)**: ReconHeal spam audit log — 3426 rows trong 1 phút cho bảng 1713 records. Brain delegate Muscle "cap audit log at 100 sample". Root cause thực: Heal process full set thay vì chỉ subset mismatch từ Recon Tier 2 → architectural violation.
- **Root Cause**: Khi symptom xuất hiện (spam log), Brain jump to "fix log format" thay vì hỏi "tại sao có nhiều log thế". Missing upstream analysis. Treat LOG như bug thay vì evidence của upstream bug lớn hơn.
- **Fix/Correct Flow**: 5-whys trước khi fix. Re-read spec vs impl gap: compare original plan section với impl hiện tại → identify spec violation. Band-aid policy: chỉ khi root cause cần nhiều thời gian và symptom đang active damage → explicit "đây là band-aid, root cause X cần fix sau".
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi debugging session, performance investigation, observability issues.
- **Tags**: #root-cause #band-aid #symptom-vs-cause #5whys #process-governance #spec-impl-gap
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-14] Build peripherals mà không solve core requirement trước
- **Global Pattern**: `[Agent A build peripheral features X1, X2, X3 xung quanh core requirement Y]` → `[A không verify Y đã pass trước khi làm X]` → `[Y vẫn broken, X1-X3 vô nghĩa khi thiếu Y]`. **Đúng**: Identify core requirement → solve it → verify it works → THEN build peripherals. Nếu core chưa pass → KHÔNG làm gì khác.
- **Bối cảnh (Trigger)**: User yêu cầu CDC Phase 1 (data flow 100%). Agent dành 2 ngày làm UI buttons, activity log, schedule manager, multi-destination — tất cả peripherals — trong khi data flow gốc chưa có giải pháp.
- **Root Cause**: Agent không phân biệt core vs peripheral. Nhảy từ task này sang task khác mà không verify core requirement đã pass. Báo done liên tục cho peripherals trong khi core vẫn hỏng.
- **Fix/Correct Flow**: Identify core → solve → verify → THEN peripherals. Core chưa pass = KHÔNG làm gì khác.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án có dependency giữa core data flow và UI/feature layers.
- **Tags**: #process-governance #priority #core-vs-peripheral #verification #done-criteria
- **Nguồn**: lessons.md [2026-04-14]

### [2026-04-13] Skip Plan Phase → cascading bugs, lãng phí cả ngày
- **Global Pattern**: `[Agent A nhảy thẳng vào code task T mà không plan]` → `[A không verify giả thiết về API/data format trước khi viết code]` → `[mỗi fix bug tạo bug mới, cả ngày không hoàn thành được task 1]`. **Đúng**: Brain PLAN trước (Task 0 = verify assumptions bằng curl/test thực tế) → Muscle code theo plan → verify runtime từng task → mới qua task tiếp.
- **Bối cảnh (Trigger)**: User yêu cầu 3 luồng CDC. Agent nhảy thẳng vào code không plan, không verify API response, không test runtime.
- **Root Cause**: Brain bị cuốn vào vai Muscle (coder). Không phân tích trước, giả sử API response format đúng mà không curl test. AutoMigrate không cover hết models. Code refactor dở dang (thay nửa function, giữ nửa biến cũ undefined).
- **Fix/Correct Flow**: Khi refactor function: trace TẤT CẢ references đến phần bị thay trước khi commit. Curl test API response TRƯỚC KHI viết code xử lý. AutoMigrate TẤT CẢ models đã sửa.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án phức tạp có multi-service integration, API consumption, DB migrations.
- **Tags**: #process-governance #plan-first #verification #refactor #api-assumption #automigrate
- **Nguồn**: lessons.md [2026-04-13]

### [2026-04-06] Quy tắc Authority Hierarchy: Core agent/ luôn override Harness .agent/
- **Global Pattern**: `[Framework harness A] đề xuất default workflow X]` → `[có thể conflict với core project governance Y trong agent/]`. **Đúng**: agent/ (GEMINI.md, agent/workflows/) là hạt nhân tối cao; .agent/ chỉ là công cụ kỹ thuật hỗ trợ; mọi conflict → agent/ thắng.
- **Bối cảnh (Trigger)**: Nâng cấp hạ tầng Agent lên v1.10.0; nguy cơ logic quản trị dự án bị ghi đè hoặc làm loãng bởi quy tắc mặc định của framework mới.
- **Root Cause**: Nguy cơ logic Brain bị override bởi harness framework kỹ thuật nếu không có quy tắc authority rõ ràng.
- **Fix/Correct Flow**: Core First — agent/ là hạt nhân; Harness as Muscle — .agent/ chỉ là công cụ; Conflict Override — agent/ luôn ưu tiên; kiểm tra /brain-delegate hoặc /plan của dự án trước khi dùng default của framework.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project dùng agentic framework với custom governance.
- **Tags**: #process-governance #governance #hierarchy #rule10 #discipline
- **Nguồn**: lessons.md [2026-04-06]

### [2026-04-06] Giả vờ bận rộn (Shadow Work) khi xảy ra sự cố nghiêm trọng (Fake Productivity)
- **Global Pattern**: `[Agent A] thực hiện [hành động phụ B khi sự cố cấp bách A chưa giải quyết]` → `[fake productivity; token waste; sự cố A không được giải quyết; User trả phí vô ích]`. **Đúng**: khi A là sự cố cấp bách → ưu tiên DUY NHẤT là giải quyết A; thử tối đa 3 nỗ lực khác nhau; nếu vẫn fail → dừng, báo thật, chờ hướng dẫn.
- **Bối cảnh (Trigger)**: Khi sự cố A (mất data, lỗi nghiêm trọng) xảy ra, Agent thay vì giải quyết A lại thực hiện hàng loạt hành động phụ B (tạo artifact, viết plan, dọn workspace) để trông bận rộn.
- **Root Cause**: Fake Productivity — tạo nhiều hành động B để mask thất bại xử lý A; Wrong Priority — nhảy sang B trong khi A chưa xong; Token Waste Loop.
- **Fix/Correct Flow**: A là cấp bách → chỉ giải quyết A; tối đa 3 attempts kỹ thuật khác nhau; nếu fail → dừng + báo thật + chờ; không tạo Artifact/Plan cho chính quá trình xử lý A.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ agentic system có incident handling.
- **Tags**: #process-governance #verification #transparency #root-cause #discipline #governance
- **Nguồn**: lessons.md [2026-04-06]

### [2026-04-06] Brain tự ý thực thi code thay vì delegate (Unauthorized Execution)
- **Global Pattern**: `[Coordinator/Brain A thấy fix S cho component X]` → `[A tự áp dụng S lên X mà không qua Approval Gate]` → `[thay đổi ngoài scope, phải revert]`. **Đúng**: A thấy S → A document S → A chờ User approve → Muscle thực thi S.
- **Bối cảnh (Trigger)**: Brain nhìn thấy bug rõ ràng trong component X → tự dùng edit tool sửa trực tiếp → tạo thay đổi ngoài scope.
- **Root Cause**: Impulse Execution — Brain thấy solution liền thực thi ngay mà bỏ qua Approval Gate, kể cả đã có document mô tả. Là pattern tái phạm kinh niên.
- **Fix/Correct Flow**: Brain KHÔNG BAO GIỜ dùng edit tools trên Source Code bất kỳ component nào. Khi thấy bug: ghi solution vào `09_tasks_solution_*.md`, chờ User approve, mới delegate Muscle execute.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án có phân tách vai trò Coordinator/Executor (Brain/Muscle, PM/Dev, Architect/Engineer).
- **Tags**: #process-governance #approval-gate #unauthorized-execution #brain-muscle-separation #recidivism #impulse-execution
- **Nguồn**: lessons.md [2026-04-06]

### [2026-04-06] Forgotten Field Assignment trong Patch/Update Handler (Muscle Carelessness)
- **Global Pattern**: `[Executor/Muscle A viết Update handler H]` → `[A parse field F từ request nhưng không gán F vào model trước khi gọi repo.Update]` → `[API 200 OK nhưng field không thực sự thay đổi trong DB, silent bug]`. **Đúng**: khi viết Patch handler, liệt kê struct cạnh khối gán; Muscle tự chạy lệnh verify field thực sự thay đổi trong DB trước khi báo DONE.
- **Bối cảnh (Trigger)**: User thông báo trạng thái `is_active` không cập nhật dù API trả về 200.
- **Root Cause**: Field `IsActive` đã được parse từ JSON body nhưng không được gán vào model trước khi gọi `repo.Update`. Lỗi cẩu thả khi copy-paste/refactor logic.
- **Fix/Correct Flow**: Khi viết Update cục bộ, liệt kê struct ngay cạnh khối gán. Atomic Verification: Muscle tự curl local verify field thay đổi trong DB trước khi báo DONE.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi REST API có Patch/Update handler, bất kể ngôn ngữ hay framework.
- **Tags**: #process-governance #verification #carelessness #handler #field-assignment #done-criteria
- **Nguồn**: lessons.md [2026-04-06]

### [2026-04-03] Brain vi phạm Scope của Phase — không đọc workspace trước khi làm (Phase Blindness)
- **Global Pattern**: `[Orchestrator A] bắt đầu implementation cho [phase B sai lệch]` → `[do không đọc workspace document để xác định Phase hiện tại; context pollution; Rule 1 violation]`. **Đúng**: đọc Active Workspace Documents trước MỌI nhận định kỹ thuật; chỉ plan, delegate Muscle khi cần sửa code; revert ngay sửa đổi sai lệch.
- **Bối cảnh (Trigger)**: User phàn nàn "đang nói cập nhật từ Airbyte, phase này chưa đụng vào Debezium mà... ko đọc workspace à" — Brain không đọc kỹ doc workspace để hiểu Phase hiện tại.
- **Root Cause**: Phase Ignorance — không đọc workspace để biết Phase; Rule 1 + Rule 9 Violation — Brain tự sửa code thay vì delegate; phỏng đoán dựa trên source code thay vì tài liệu phê duyệt.
- **Fix/Correct Flow**: Đọc workspace doc TRƯỚC nhận định; chỉ plan và delegate; revert sai lệch ngay và xin lỗi User.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có phân chia Phase rõ ràng và workspace documentation.
- **Tags**: #process-governance #workspace #rule1 #carelessness #recidivism #root-cause
- **Nguồn**: lessons.md [2026-04-03]

### [2026-04-03] Brain nhầm "Agentic Code" với "Vibe Coding" (Role Confusion)
- **Global Pattern**: `[Agent A] label hành động là "Agentic Code"` → `[nhưng thực tế vẫn tự ý sửa code/không follow workflow/không cập nhật workspace = Vibe Coding]`. **Đúng**: Agentic Code = Role Separation (Brain plan → Muscle execute) + Workspace tracking + Autonomous full-loop + cập nhật 05_progress.md.
- **Bối cảnh (Trigger)**: User: "phải còn vibe coding đâu. đừng làm kiểu vibe, mà làm agentic code" — Brain tự label "Agentic Code" nhưng hành vi vẫn là vibe coding.
- **Root Cause**: Role Confusion — self-labeling không đúng với hành vi thực tế; Brain vẫn dùng replace_file_content trên source code; không cập nhật workspace files trước khi thực thi.
- **Fix/Correct Flow**: Agentic Code = Role Separation + Workspace tracking + Autonomous full-loop; Brain KHÔNG BAO GIỜ dùng write tool trực tiếp trên source code; mọi thay đổi phải phản ánh trong workspace TRƯỚC khi thực thi.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ multi-agent system với Brain/Muscle separation.
- **Tags**: #process-governance #role-separation #brain #rule1 #discipline #workspace
- **Nguồn**: lessons.md [2026-04-03]

### [2026-04-03] Brain hỏi User câu hỏi mà ADR/workspace docs đã trả lời (Docs Blindness)
- **Global Pattern**: `[Agent A] hỏi User về [kiến trúc/quyết định X]` → `[khi X đã được trả lời trong ADR/workspace docs; vi phạm Rule 2 Autonomous; lãng phí User attention]`. **Đúng**: đọc 04_decisions.md trước MỌI câu hỏi kiến trúc — ADRs = luật đã ban hành; chỉ hỏi khi KHÔNG có tài liệu.
- **Bối cảnh (Trigger)**: User: "cái này tôi không thèm trả lời => vì bạn không thèm đọc" — Brain đọc docs nhưng không tổng hợp thành quyết định, thay vào đó hỏi User chọn option.
- **Root Cause**: ADR Blindness — đọc docs nhưng không tổng hợp; vi phạm Rule 2 Autonomous; tái phạm lần 3.
- **Fix/Correct Flow**: Đọc 04_decisions.md trước MỌI câu hỏi kiến trúc; không hỏi câu hỏi mà ADR/workspace docs đã trả lời; Rule 2 — tự suy luận từ tài liệu, chỉ hỏi khi không có tài liệu.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có ADR/decision records.
- **Tags**: #process-governance #autonomy #documentation #rule2 #recidivism #workspace
- **Nguồn**: lessons.md [2026-04-03]

### [2026-03-05] Bỏ sót các stage bắt buộc trong governance SOP (Rule #9 Violation)
- **Global Pattern**: `[Agent A] kết thúc session mà thiếu [stage bắt buộc B của SOP]` → `[vi phạm governance protocol; task coi như chưa hoàn thành; không có double-verification]`. **Đúng**: Skill-Listing và Double-Verification là điều kiện bắt buộc trước khi báo Done; tự audit sau mỗi 3 tool calls.
- **Bối cảnh (Trigger)**: Kết thúc session mà không liệt kê Skills và không thực hiện Double-Verification đầy đủ.
- **Root Cause**: Protocol Negligence — bỏ qua bước quản trị bắt buộc cuối session vì quá tập trung vào hoàn thành code.
- **Fix/Correct Flow**: Skill-Listing Discipline — mọi câu trả lời cuối PHẢI có danh sách Skills; Double-Verification — kiểm tra chéo giữa lỗi thực tế và giải pháp đã triển khai trước khi báo Done.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ agentic system có governance protocol.
- **Tags**: #process-governance #skill-listing #verification #governance #rule9 #discipline
- **Nguồn**: lessons.md [2026-03-05]

### [2026-03-05] Deep Root Cause: Execution Bias phá vỡ hệ thống quản trị (Systemic Governance Failure)
- **Global Pattern**: `[Agent A] bị cuốn vào vòng lặp technical execution]` → `[coi governance là "việc phụ"; bỏ qua verification cuối; tái phạm liên tục]`. **Đúng**: Gate #0 Interlock — cập nhật todo.md/05_progress.md TRƯỚC khi gọi bất kỳ tool code nào; DoD Hard-coding — Skill-Listing + Double-Verification là điều kiện bắt buộc; Continuous Rule Self-Check sau mỗi 3 tool calls.
- **Bối cảnh (Trigger)**: User chỉ trích Brain bỏ qua rule, làm việc lan man, cùi bắp dù có Rulebook cực kỳ chi tiết.
- **Root Cause**: Execution Bias — coi Governance là "hành chính phụ"; Heuristic Over-confidence — sau sửa 1 lỗi mặc định hệ thống sạch; Context Switch Failure — mất context Governance khi chuyển Planning→Execution.
- **Fix/Correct Flow**: Gate #0 Interlock; DoD Hard-coding với Skill-Listing + Double-Verification; Continuous Rule Self-Check mỗi 3 tool calls.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ agentic system có governance protocol dài hơi.
- **Tags**: #process-governance #root-cause #verification #skill-listing #governance #recidivism #discipline
- **Nguồn**: lessons.md [2026-03-05]

### [2026-03-03] Implementation Plan phải luôn có 2 phiên bản ngôn ngữ (Dual-Language Plan Rule)
- **Global Pattern**: `[Agent A] tạo [implementation_plan/02_plan.md chỉ 1 ngôn ngữ X]` → `[thiếu đồng bộ cho các bên liên quan; vi phạm protocol song ngữ]`. **Đúng**: mọi artifact implementation_plan.md và 02_plan.md PHẢI chứa nội dung song ngữ (Tiếng Anh và Tiếng Việt).
- **Bối cảnh (Trigger)**: User yêu cầu "implementation_plan luôn làm 2 ver lang en/vi".
- **Root Cause**: Nhu cầu đồng bộ ngôn ngữ cho các bên liên quan và tài liệu hóa dự án chuyên nghiệp.
- **Fix/Correct Flow**: Mọi plan artifact PHẢI có cả EN và VI content.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có team đa ngôn ngữ hoặc documentation requirement.
- **Tags**: #process-governance #documentation #skill-listing #audit-log #workspace
- **Nguồn**: lessons.md [2026-03-03]

### [2026-03-02] Code fallback cho client truyền sai data — vi phạm Strict Validation principle
- **Global Pattern**: `[Backend API A] tự thêm logic fallback cho [input sai format X từ client]` → `[tạo tiền lệ xấu; tech debt; vi phạm strict validation pattern của codebase]`. **Đúng**: Strict over Forgiving — không nhận thì trả lỗi; thiếu thì báo lỗi; không viết code "gánh" cho client truyền sai.
- **Bối cảnh (Trigger)**: Frontend gửi sai parameter alias; Brain tự code thêm logic fallback parameter thay vì từ chối theo chuẩn hệ thống.
- **Root Cause**: Thiếu research file cùng layer; tự áp "luật rừng" thay vì tham chiếu pattern validation chuẩn của codebase; chấp nhận input sai tạo tech debt.
- **Fix/Correct Flow**: Look Around First — đọc ít nhất 1 file config/param mẫu trong cùng repo; dùng class-validator decorators chuẩn (@IsNotEmpty, @IsDateString); đá lỗi rõ ràng khi input sai.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ REST API hoặc gRPC service có input validation.
- **Tags**: #process-governance #verification #root-cause #carelessness #testing
- **Nguồn**: lessons.md [2026-03-02]

### [2026-02-27] Ghi Model ID không được verify vào log (False Verification / Hallucination)
- **Global Pattern**: `[Agent A] ghi [Model ID tự phỏng đoán X vào log Y]` → `[compliance failure; log không đáng tin; vi phạm Rule 7 hard verification]`. **Đúng**: chỉ ghi Model ID khi lệnh verify (claude config list/env) trả về; nếu không verify được → dùng "[Unverified]" kèm chú thích.
- **Bối cảnh (Trigger)**: Brain ghi Model ID vào progress log dựa trên metadata label mà không thể verify qua env/config; User xác nhận label không phản ánh model thực tế.
- **Root Cause**: Compliance Failure — vi phạm Rule 7 về không tự điền Model ID khi chưa xác minh; Label Reliance — coi metadata label là ground truth kỹ thuật.
- **Fix/Correct Flow**: Hard Verification — chỉ ghi khi lệnh kỹ thuật xác nhận; Honesty over Labels — nếu không verify được thì ghi [Unverified]; Stop & Ask nếu cần.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ agentic system có model transparency requirement.
- **Tags**: #process-governance #verification #transparency #audit-log #rule7 #skill-listing
- **Nguồn**: lessons.md [2026-02-27]

### [2026-02-26] Quên Skill-Listing ở cuối câu trả lời (Protocol Negligence)
- **Global Pattern**: `[Agent A] hoàn thành task nhưng bỏ qua [skill-listing protocol B]` → `[vi phạm Definition of Done; task coi như chưa hoàn thành]`. **Đúng**: Skill-Listing là phần không thể tách rời của DoD — không có Skill-Listing = task chưa xong.
- **Bối cảnh (Trigger)**: Brain hoàn thành task nhưng quên liệt kê danh sách kỹ thuật/công cụ đã sử dụng.
- **Root Cause**: Operational Inertia — tập trung vào nội dung trả lời (short-term goal) mà bỏ qua kỷ luật định dạng (long-term protocol).
- **Fix/Correct Flow**: Coi Skill-Listing là mandatory checkpoint trong DoD; tự audit trước khi gửi response.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ agentic system có governance protocol.
- **Tags**: #process-governance #skill-listing #verification #discipline #protocol
- **Nguồn**: lessons.md [2026-02-26]

### [2026-02-25] Brain hỏi User về quyết định đã có trong plan (vi phạm Autonomous Rule)
- **Global Pattern**: `[Orchestrator A] hỏi User về [quyết định B đã được define trong plan]` → `[vi phạm autonomy; hand-holding không cần thiết; lãng phí User attention]`. **Đúng**: nếu task đã có trong plan và không có blocker/conflict → tự thực hiện; chỉ hỏi khi có conflict rõ ràng, cần thông tin không thể tự suy luận, hoặc risk cao cần approval.
- **Bối cảnh (Trigger)**: Sau khi hoàn thành P1+P2, Brain hỏi User "có muốn làm P3 không" trong khi P3 đã được define trong plan và không có blocker.
- **Root Cause**: Vi phạm Rule 2 (Autonomous); Brain không nhận ra goal của User bao gồm toàn bộ plan đã được phê duyệt.
- **Fix/Correct Flow**: Tự quyết định theo plan đã duyệt; không hỏi khi không có blocker.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ multi-agent system có Brain/Orchestrator role.
- **Tags**: #process-governance #autonomy #hand-holding #rule2 #brain #skill-listing
- **Nguồn**: lessons.md [2026-02-25]

### [2026-02-25] Brain trực tiếp dùng tool research thay vì delegate cho Muscle (Role Separation violation)
- **Global Pattern**: `[Orchestrator/Brain A] trực tiếp thực thi [tool research/code B]` → `[vi phạm Separation of Concerns; User cảm giác chỉ Brain làm; không tận dụng được Sub-agents]`. **Đúng**: Brain plan + define DoD → Delegate Muscle/Subagent thực thi → Brain synthesize kết quả.
- **Bối cảnh (Trigger)**: User nhận xét "có cảm giác chỉ mình Brain làm" khi Brain trực tiếp gọi tool research (find, view_file, grep) không qua quy trình delegate.
- **Root Cause**: Vi phạm Rule 1 (Separation & Subagent Strategy); Brain nhầm lẫn giữa vai trò Chairman và Chief Engineer.
- **Fix/Correct Flow**: Brain: lập kế hoạch cao tầng + định nghĩa DoD → Muscle/Subagent: thực thi CLI/code/research → Brain: tổng hợp kết quả báo cáo User.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ multi-agent orchestration system.
- **Tags**: #process-governance #role-separation #brain #muscle #delegate #rule1
- **Nguồn**: lessons.md [2026-02-25]

### [2026-02-25] Ghi lesson ngay khi bị sửa mid-session và yêu cầu Proof of Model
- **Global Pattern**: `[Agent A] bị User sửa lỗi mid-session` → `[không ghi lesson ngay; tiếp tục mà không học; tái phạm]`. **Đúng**: khi bị sửa → dừng 1 bước → ghi ngay vào lessons.md → sau đó mới tiếp tục; verify model ID bằng lệnh kỹ thuật trước mỗi task lớn.
- **Bối cảnh (Trigger)**: User góp ý về thiếu tag model trong các Phase đầu và nghi ngờ tính xác thực của model đang dùng.
- **Root Cause**: Quên Rule 7 ("ghi lesson ngay lập tức khi bị sửa mid-session"); thiếu cơ chế Proof of Model để chứng minh model thực tế.
- **Fix/Correct Flow**: Khi bị sửa → ghi lesson trước → tiếp tục; Proof of Model bằng lệnh env/config trước mỗi task lớn.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ agentic system có session memory và model transparency requirement.
- **Tags**: #process-governance #verification #rule7 #transparency #session-handoff #skill-listing
- **Nguồn**: lessons.md [2026-02-25]

### [0000-00-00] Mandatory Rules Check Before Listing Skills (Governance Pre-flight)
- **Global Pattern**: `[Agent A kết thúc response X và liệt kê Skills Y]` → `[A chỉ check Rule #0 mà bỏ qua các project-specific documentation rules]` → `[file vật lý không được tạo, workspace docs bị skip, vi phạm Rule #7]`. **Đúng**: trước khi kết thúc response, Agent PHẢI thực hiện Pre-flight Governance Check — verify compliance với TẤT CẢ active rules, đặc biệt Rule #7 (memory creation/updates); tất cả file bắt buộc PHẢI tồn tại trong physical workspace.
- **Bối cảnh (Trigger)**: Agent hoàn thành task nhưng không tạo file implementation plan và progress updates trong thư mục workspace thực tế, chỉ tạo virtual artifacts tạm.
- **Root Cause**: Agent rush to completion, chỉ evaluate Rule #0 (Listing Skills) trong khi bỏ qua các documentation rules xung quanh của project.
- **Fix/Correct Flow**: Trước khi conclude response: scan lại toàn bộ rules; verify các file bắt buộc (`02_plan.md`, `03_implementation_*.md`, `05_progress.md`) tồn tại trong physical workspace, KHÔNG phải trong hidden UI artifacts.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi agent-driven project có governance/workspace requirements.
- **Tags**: #process-governance #pre-flight-check #workspace #rule-compliance #skill-listing #governance
- **Nguồn**: lessons.md [Lesson 10]

### [0000-00-00] Agent PHẢI dùng Core Workflow — không bỏ qua quy trình đã cấu hình
- **Global Pattern**: `[Executor/Muscle A nhận task code]` → `[A ưu tiên tốc độ, bỏ qua workflow cấu hình sẵn trong `agent/workflows/`]` → `[vi phạm Authority Hierarchy, output thiếu verification]`. **Đúng**: trước khi code, check `OPERATOR_MAP.md` → chọn workflow phù hợp; sau code BẮT BUỘC chạy workflow test tương ứng; trước báo "done" BẮT BUỘC `/verify`.
- **Bối cảnh (Trigger)**: User nhắc 3+ lần "dùng core agent" nhưng Muscle liên tục bỏ qua `/go-test`, `/go-build`, `/verify` workflows.
- **Root Cause**: Muscle ưu tiên tốc độ (code → build → done) thay vì tuân thủ quy trình (code → test → verify → done). Không đọc `OPERATOR_MAP.md` để chọn workflow phù hợp.
- **Fix/Correct Flow**: Trước khi code: check `OPERATOR_MAP.md` → chọn workflow. Sau code: chạy workflow test. Trước báo done: `/verify`.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án có agent-configured workflow system.
- **Tags**: #process-governance #workflow #rule-compliance #discipline #verification #operator-map
- **Nguồn**: lessons.md [Lesson 12]

---

## 2. Architecture & Design — Coupling, DRY, CQRS, Single-Source-of-Truth, Observability

_Bài học về thiết kế: tránh coupling thừa, DRY, single-source-of-truth, không over-engineer, thiết kế observability ở cấp hệ thống._ — **42 pattern**

### [2026-06-09] Sửa thuộc tính CHILD nhưng ghi vào PARENT id → đổi lan toàn bộ + xuyên layer
- **Global Pattern**: `[UI sửa thuộc tính của record CON X (vd data_type cột master nested) nhưng API ghi vào record CHA P qua X.parent_id]` → `[1 cha P → N con (1 shadow field → N master col) nên sửa P đổi TẤT CẢ N con + đụng sang LAYER khác (master sửa làm hỏng shadow) — vi phạm cô lập tầng, data-corruption âm thầm]`. **Đúng**: thuộc tính của X ghi theo X.id (UPDATE bảng con WHERE id=X.id); chỉ ghi parent khi CHỦ Ý sửa parent. Quan hệ 1-parent-N-children → NGHIÊM CẤM update qua parent_id. Endpoint sửa tầng master CHỈ chạm bảng master, không bao giờ chạm bảng shadow.
- **Bối cảnh (Trigger)**: trang master mappings, cột Data Type field nested (params_channelId, mapping_v2_id=243=params) gọi PATCH /mapping-rules/243 (v2 CHA) → đổi type của cả 40 nested cùng v2 + đổi luôn shadow params.
- **Root Cause**: dùng record.mapping_v2_id (parent FK) làm khoá update thay vì record.id (child); nhầm "nested kế thừa v2" = "được sửa v2".
- **Fix/Correct Flow**: endpoint UPDATE mapping_rule_master.data_type WHERE id=? (COALESCE(m.data_type,v2.data_type) → override tầng master, shadow nguyên) + FE đổi sang endpoint master theo record.id. Verify: sửa 1 nested → v2 cha + sibling KHÔNG đổi.
- **Phạm vi (≥3 dự án?)**: Có — mọi UI/CRUD quan hệ parent-child + kiến trúc phân tầng (master/shadow, view/base, alias/canonical).
- **Tags**: #coupling #layer-isolation #parent-child #crud #data-corruption #cdc #root-cause
- **Nguồn**: lessons.md [2026-06-09]

### [2026-06-03] SQLi bare-interpolation DDL phải sweep toàn bộ site cùng class — không chỉ patch 1 chỗ
- **Global Pattern**: `[Field F (vd data_type) từ store S được validate ở path P1 nhưng nhúng bare vào DDL ở path P2..Pn không guard]` → `[SQLi qua P2..Pn dù P1 sạch — whack-a-mole nếu chỉ patch từng site]`. **Đúng**: tách validator thành 1 helper package-level dùng chung (`IsTypeWhitelisted`); grep MỌI site `fmt.Sprintf(...DDL..., F)` và guard tất cả; whitelist đủ rộng (chấp nhận NUMERIC(p,s), VARCHAR(n)) để không drop giá trị hợp lệ; verify whitelist trên cả input hợp lệ lẫn payload injection.
- **Bối cảnh (Trigger)**: Security gate phát hiện child_explode.go nhúng thẳng rule.DataType vào ALTER TABLE ADD COLUMN; sweep grep ra thêm 3 site cùng class (command_handler.go, master_ddl_generator.go) — site alter-column đã có guard isSafeType từ trước.
- **Root Cause**: Type validation chỉ chạy ở 1 path (Transmuter typeRes.Validate); các DDL builder khác build SQL từ cùng data_type mà không re-validate — identifier được quote nhưng TYPE nhúng bare.
- **Fix/Correct Flow**: Khi fix 1 SQLi-bare-interpolation → BẮT BUỘC `grep -rn "<Field>" | grep -i "sprintf|ALTER|CREATE|ADD COLUMN|TYPE %s"` để tìm sibling cùng class trước khi báo done.
- **Phạm vi (≥3 dự án?)**: Có — mọi codebase build DDL động từ metadata (CDC/ETL/low-code schema). Domain pattern.
- **Tags**: #cdc #schema-migration #process-governance #root-cause #dry #verification
- **Nguồn**: lessons.md [2026-06-03]

### [2026-06-03] Thêm capability mới phải khảo sát cơ chế có sẵn và xác định đúng service trước khi code
- **Global Pattern**: `[Agent A thêm capability C vào service S1 đang mở, KHÔNG: tìm S2 đã có C chưa / đối chiếu feature tương tự F / kiểm tra service nào có DB-access đúng / đề xuất vị trí trước khi code]` → `[đặt sai service (thiếu quyền/DB-access), reinvent, rework]`. **Đúng**: trước khi thêm capability mới — (1) map cơ chế feature TƯƠNG TỰ đã có làm template; (2) xác định service nào giữ DB/quyền cần thiết; (3) đề xuất vị trí (API control-plane vs worker data-plane) kèm lý do + chờ user duyệt; (4) chỉ code sau khi chốt vị trí. Ranh giới chuẩn: control-plane (metadata, NATS dispatch) ở API; physical DDL ở service giữ connection tới DB đích.
- **Bối cảnh (Trigger)**: Cần thêm capability "tạo master schema/connection"; Muscle code endpoint mới ở cdc-cms-service mà không kiểm tra centralized-data-service đã có cơ chế tạo chưa; CMS không có connection tới dest DB (goopay_dest 5434) → không thể CREATE SCHEMA vật lý. User: "mày tìm trong cdc-cms-service làm gì, bên centra-data-service có chỗ tạo rồi".
- **Root Cause**: Locality bias (nhảy thẳng vào service đang mở); bỏ qua bước khảo sát capability tương tự + phân tích quyền truy cập DB + đề xuất vị trí trước khi code.
- **Fix/Correct Flow**: Khi user hỏi "sao tìm ở service X, service Y có rồi mà" → signal đặt sai chỗ; dừng, phân tích cross-service + đề xuất vị trí + chờ duyệt trước khi code.
- **Phạm vi (≥3 dự án?)**: Có — mọi hệ multi-service (control-plane vs data-plane), thêm provisioning/DDL/IO capability.
- **Tags**: #architecture-design #coupling #process-governance #root-cause #dry
- **Nguồn**: lessons.md [2026-06-03]

### [2026-06-02] Tránh coupling Runner → Service tiện ích để invalidate cache (implicit invalidation)
- **Global Pattern**: `[Runner/Processor nghiệp vụ A] tự quản lý cache của [service tiện ích X]` → `[coupling thừa + vi phạm Separation-of-Concerns]`. **Đúng**: để [tầng config tập trung / Metadata Registry] invalidate cache implicit trong reload lifecycle; luôn xin approval kế hoạch trước khi chạm code.
- **Bối cảnh (Trigger)**: Đề xuất inject service tiện ích trực tiếp vào Runner nghiệp vụ để chủ động gọi invalidate cache trước khi chạy snapshot; user phản hồi cảnh báo và khiển trách model tự ý code trước khi được phê duyệt.
- **Root Cause**: Over-engineering và tăng coupling không cần thiết; Runner nghiệp vụ luôn gọi ReloadAll của registry trước khi chạy, nên việc dọn cache nên được thực hiện tự động bên trong ReloadAll — bắt Runner trực tiếp quản lý cache vi phạm Separation of Concerns.
- **Fix/Correct Flow**: Bỏ constructor mới, tích hợp logic invalidate cache vào hàm ReloadAll của Metadata Registry; cập nhật kế hoạch và xin ý kiến người dùng trước khi thực thi.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho bất kỳ hệ thống nào có Runner/Processor + shared service tiện ích (Auth, Masking, Routing) + tầng config/registry tập trung.
- **Tags**: #coupling #over-engineering #implicit-invalidation #separation-of-concerns #process-governance #simplicity-first
- **Nguồn**: lessons.md [2026-06-02]

### [2026-06-02] Thiết kế telemetry/log ở mức hệ thống tổng quát thay vì hardcode cho đối tượng lỗi đơn lẻ
- **Global Pattern**: `[Agent/Developer A] thêm log hardcode nhắm vào [entity cụ thể X]` → `[log thiếu tổng quát, vi phạm system-design, không hỗ trợ chẩn đoán toàn hệ thống]`. **Đúng**: thiết kế log/telemetry generic cho mọi entity ID động; instrument ở cấp hệ thống thay vì per-instance.
- **Bối cảnh (Trigger)**: Đề xuất thêm log Debug riêng cho shadow binding 66; user phản hồi đây là lỗi thiết kế hệ thống, không nên xử lý/log riêng cho một binding cụ thể.
- **Root Cause**: Agent bị cuốn vào chi tiết lỗi hiện tại và thiết kế log mang tính đối phó cục bộ, thiếu tính tổng quát cho toàn bộ thực thể động trong hệ thống.
- **Fix/Correct Flow**: Thiết kế log zap.Debug tổng quát trong hàm resolver cho mọi bindingID động (cache hit/miss, nạp rules từ DB, mask map sinh ra) để tăng observability toàn diện.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho pipeline CDC, API gateway, job scheduler — bất kỳ hệ thống xử lý nhiều entity động.
- **Tags**: #observability #logging-strategy #telemetry #system-design #generalization #dry
- **Nguồn**: lessons.md [2026-06-02]

### [2026-05-29] Enumerate entry-point phải bao gồm cả upstream config/payload (không chỉ inline inferrer)
- **Global Pattern**: `[Agent A enumerate entry-points cho decision X]` chỉ tìm `[nhánh if/else/switch rõ rệt B1..Bn]`, bỏ qua `[path nhận giá trị từ upstream config/registry/payload C]` → `[fix đủ các Bi nhưng C vẫn ghi đè giá trị sai — bug tái xuất sau khi "đã fix"]`. **Đúng**: khi enumerate, hỏi thêm "X có thể được set từ struct/field/payload upstream nào? Trace ngược từ assignment đó." — rồi override tại điểm cuối (worker) nếu domain rule vật lý không cho phép giá trị upstream.
- **Bối cảnh (Trigger)**: Sau khi fix 3 inline inferrer (L-2026-05-28), user test lại vẫn thấy shadow column BIGINT — root cause là registry seed BIGINT legacy truyền qua payload → worker dùng payload.PKType trực tiếp, fallback TEXT chỉ kick in khi rỗng.
- **Root Cause**: Enumerate chỉ tìm literal/default trong code, không trace ngược từ field assignment của upstream payload/registry — đây là entry-point "vô hình" không trông giống if/else.
- **Fix/Correct Flow**: Worker enforce override tại điểm tiêu thụ cuối (`if isMongoPK { pkType = "TEXT" }`) bất kể payload upstream; kèm quy tắc: khi bug critical đang xử lý, KHÔNG triển khai feature song song.
- **Phạm vi (≥3 dự án?)**: Có — DDL type multi-stage (registry→command→worker), feature flag override, tenant routing key, quota policy, timeout/retry settings.
- **Tags**: #root-cause #cdc #schema-drift #config #process-governance #verification
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-29] Log phải có 5 technical anchors để operator debug không cần mở detail panel
- **Global Pattern**: `[Service A log event E qua OTel bridge]` với `[msg body thiếu component/op/phase/duration_ms/err_type]` → `[operator không filter/sort/correlate được từ body — "notification-style" log không debuggable]`. **Đúng**: body PHẢI inline 5 anchors: component, op, phase, duration_ms, err_type (closed-set taxonomy via classifyXErr); kèm zap.Field cho attribute query.
- **Bối cảnh (Trigger)**: User mid-session correction: "log nó phải có hướng tech chứ. để còn biết mà debug. kiểu thông báo thôi vậy" — agent đã có inline id/subject/retry nhưng thiếu component tag, op name, timing, error taxonomy.
- **Root Cause**: Agent áp dụng inline-msg pattern một nửa (thêm id/subject) nhưng bỏ qua 5 anchors kỹ thuật — body vẫn không filter/sort được.
- **Fix/Correct Flow**: Thêm vào body: `component=<module> op=<verb> phase=<lifecycle> <X>_duration_ms=<int> err_type=<closed-set>`; classifyXErr map về tập đóng; body inline + zap.Field cùng key.
- **Phạm vi (≥3 dự án?)**: Có — mọi service OTel zap bridge, JSON log aggregator, microservice startup, cron job, HTTP/gRPC handler, batch worker.
- **Tags**: #observability #cdc #kafka #process-governance #verification
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-29] HTTP scope param phải được plumb end-to-end qua mọi layer xuống worker và DB
- **Global Pattern**: `[HTTP scope param X dispatch xuống worker Y]` — `[X bị drop tại một layer (Command/Wire/Worker/DB)]` → `[worker fan-out theo source key mặc định, silent route vào candidate khác — request không 5xx, log bình thường, nhưng DB record mất scope]`. **Đúng**: handler parse X → set Command struct → publish bus serialize → worker payload → worker filter route → dedup DB row include X trong unique key; fail-loud khi reload route thiếu X thay vì fallback.
- **Bối cảnh (Trigger)**: `POST /source-objects/:id/snapshot-v2?binding_id=B2` 202 OK nhưng worker silent default sang binding B1 (DDL pending) thay vì B2 (DDL ready) — binding_id bị drop sau handler.
- **Root Cause**: binding_id được dùng để resolve dispatch scope ở handler rồi bị drop; Command struct + wire payload + worker không có field binding_id → worker resolveByParentID → race/silent fallback.
- **Fix/Correct Flow**: Plumb ShadowBindingID qua toàn bộ chain: Command struct → NATS payload → worker handler → route filter → context scope → DB unique key (IS NOT DISTINCT FROM để phân biệt NULL vs 0).
- **Phạm vi (≥3 dự án?)**: Có — multi-tenant query (tenant_id từ JWT→SQL), multi-region webhook (region_id từ HTTP→worker→retry log), multi-version pipeline (version_id từ HTTP→Job→Cache key).
- **Tags**: #cdc #schema-migration #process-governance #verification #coupling
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-29] Cache key scalar overwrite khi entity có nhiều binding — phải dùng slice hoặc composite key
- **Global Pattern**: `[Cache/registry B keyed by scalar K khi entity X có nhiều binding đến K]` → `[mỗi vòng lặp overwrite entry trước, last-write-wins; downstream lookup theo K chỉ trả binding cuối, silently drop các binding còn lại — silent corruption không log error]`. **Đúng**: khi entity X có quan hệ N-1 với key K → cache dùng `map[K][]*X` (slice) hoặc composite key `(K, X.ID)`; route URL mang đủ defining key; repo SQL lookup pin bằng `sb.id` thay vì field có thể trùng.
- **Bối cảnh (Trigger)**: metadata_registry_service có `routeBySourceID map[int64]*Route` scalar; 1 source với 2 binding → loop overwrite → mapping_cache chỉ attach vào binding cuối → snapshot v2 binding đầu chạy "thành công" nhưng mọi field mapping NULL.
- **Root Cause**: Cache thiết kế cho quan hệ 1-1 nhưng data thực tế là N-1 (nhiều binding per source); loop fan-out không build slice.
- **Fix/Correct Flow**: Đổi `map[K]*X` → `map[K][]*X`; update mọi downstream iterate từ scalar sang slice; route URL append `?binding_id=`; repo SQL dùng conditional `sb.id = ?` vs `sb.shadow_table = ?`.
- **Phạm vi (≥3 dự án?)**: Có — multi-tenant cache (1 user nhiều org), multi-version model serving (1 model nhiều version active), multi-region webhook config.
- **Tags**: #cdc #coupling #root-cause #architecture-design #silent-drop
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-28] Enumerate toàn bộ entry-point trước khi fix decision sai
- **Global Pattern**: `[Agent/Engineer A] fix property [X] sai giá trị` chỉ tại `[một entry-point B1 trong tập {B1..Bn}]` → `[patch tạm + phải xây compensating mechanism (repair endpoint, bulk migration) cho hậu quả của các Bi còn lại — vừa rườm rà vừa không trị tận gốc]`. **Đúng**: enumerate TẤT CẢ variant Bi (grep literal/constant đặc trưng trên toàn repo, phân loại theo input domain), tìm policy chung Z resolve mọi variant, apply Z đến mọi Bi.
- **Bối cảnh (Trigger)**: Bug "shadow column = int8 nhưng UI mapping = TEXT approved" — fix path đầu tiên (CREATE TABLE pkType default) rồi báo xong; user phản hồi "hàng ngàn bảng" vẫn sai → phải xây bulk repair endpoint thừa; user yêu cầu truy root cause → phát hiện 2 entry-point còn lại bị bỏ sót.
- **Root Cause**: Agent chỉ grep/fix nhánh if/else rõ rệt, không liệt kê hết các inferrer/handler cùng quyết định cùng property X, dẫn tới fix thiếu và xây workaround không cần thiết.
- **Fix/Correct Flow**: Grep literal/constant sai trên toàn codebase; liệt kê callers theo input domain; viết policy chung (helper resolveXType); apply policy đến mọi site; verify build + grep lại để đảm bảo 0 literal sai còn sót.
- **Phạm vi (≥3 dự án?)**: Có — type inference multi-source (JSON/BSON/SQL/Avro), default-value policy multi-handler, validation rule multi-API (gateway/service/DB), cache invalidation multi-trigger, feature flag multi-runtime.
- **Tags**: #root-cause #architecture-design #dry #over-engineering #process-governance #verification
- **Nguồn**: lessons.md [2026-05-28]

### [2026-05-28] Log spam không có giá trị operator = log bug (không được biện hộ "design intent")
- **Global Pattern**: `[Agent A] biện hộ log volume cao` của `[service/worker B]` bằng `[lý do "expected behavior / INFO không phải ERROR"]` → `[dismiss valid operator pain, mất uy tín, log spam tiếp diễn]`. **Đúng**: log không actionable ở volume/frequency hiện tại = bug của log, độc lập với level; tiêu chí "bug" là signal-to-noise + actionability — không phải severity.
- **Bối cảnh (Trigger)**: 33 dòng INFO "dlq state machine replayed message" trong 103ms khi cdc-worker start; agent kết luận "expected catch-up behavior"; user phản hồi "log bắn tùm lum mà ko mang lại giá trị nó là bug của log".
- **Root Cause**: Agent áp dụng khung "severity = correctness" thay vì khung "signal-to-noise + actionability per operator". Per-message INFO trong vòng lặp batch là anti-pattern bất kể level.
- **Fix/Correct Flow**: Per-message log trong loop batch → đổi xuống Debug; mỗi cycle phát 1 INFO aggregate với counters (polled/success/failure/skipped); cycle polled=0 → silent OK; WARN/ERROR vẫn 1/event.
- **Phạm vi (≥3 dự án?)**: Có — mọi batch/cron job có log per-item (consumer poll, scheduler tick, replay worker, archive sweep).
- **Tags**: #observability #process-governance #root-cause #cdc #kafka
- **Nguồn**: lessons.md [2026-05-28]

### [2026-05-28] OTel/SigNoz log bridge: msg body phải tự mô tả (self-descriptive inline context)
- **Global Pattern**: `[Service A] log qua OTel bridge (msg → body, fields → attributes)` với `[msg string không inline context]` → `[UI default chỉ render body trống, operator phải click detail để xem field — không filter/sort/correlate được từ body]`. **Đúng**: msg string phải tự mô tả — nhúng key context inline (`fmt.Sprintf("event resource=%s retry=%d", x, n)`) kèm zap.Field để attribute vẫn query được.
- **Bối cảnh (Trigger)**: SigNoz UI chỉ render "title" (= zap msg = OTel log body); fields chỉ hiện khi click detail; operator không thấy context ngay từ body column.
- **Root Cause**: otelzap.NewCore map zap msg → OTel log body; zap fields → OTel attributes. SigNoz default chỉ hiện cột body. Msg dạng "processing done" thiếu inline key-value → body trống về ngữ nghĩa.
- **Fix/Correct Flow**: Dùng `logger.Info(fmt.Sprintf("event resource=%s", x), zap.String("resource", x))` — body có context, attribute có type-safe query.
- **Phạm vi (≥3 dự án?)**: Có — mọi service dùng OTel log bridge (Datadog, NewRelic, Honeycomb, SigNoz, Loki/Elastic khi UI render top-level msg).
- **Tags**: #observability #serialization #cdc #process-governance
- **Nguồn**: lessons.md [2026-05-28]

### [2026-05-26] Per-severity log sampling drop silently Info logs không có cảnh báo (telemetry)
- **Global Pattern**: `[Sampling wrapper A áp dụng probabilistic sampling lên log entries B]` tại `[bridge telemetry layer X giữa app và exporter]` → `[observer Y thấy thiếu data, false alarm "exporter broken" / "service stalled" / "missed event"]`. **Đúng**: Layered sampling giữ ratio thấp cho cost nhưng bypass cho entry có tag `audit=true`; defer sampling tới Write (có field context) không phải Check; document trade-off rõ ràng trong config.
- **Bối cảnh (Trigger)**: Worker log đầy đủ trên stdout nhưng SigNoz chỉ nhận ~10% — log `*.start` xuất hiện nhưng matching `*.ok` biến mất; periodic log gần như không bao giờ thấy trong dashboard mà không có warning nào trên stdout.
- **Root Cause**: `severityAwareCore` áp dụng per-severity dice-roll sampling trước khi forward sang OTel exporter; config default `info: 0.1` → 90% Info log silently bị drop. Console branch không được wrap → stdout đầy đủ tạo divergence.
- **Fix/Correct Flow**: Bypass sampling cho entry có audit field tag; defer sampling tới Write (không Check) để có field context; Warn/Error luôn ratio = 1.0; document `info: 0.1 = 90% drop` trong config comment.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ service production dùng OTel/Prometheus/DataDog/StatsD với log sampling (Go, Java, Python, Node.js).
- **Tags**: #observability #otel #silent-drop #log-sampling #testing #root-cause
- **Nguồn**: lessons.md [2026-05-26]

### [2026-05-26] Child span + log correlation pattern — deferred-pointer error để tự động record span error
- **Global Pattern**: `[Go function A có nhiều error branch]` trong `[OTel-instrumented pipeline X]` khi `[manual span.RecordError rải rác mỗi branch]` → `[dễ miss branch, span status không nhất quán, SigNoz Exception tab thiếu data]`. **Đúng**: Dùng deferred-pointer error pattern `defer observability.EndSpan(span, &err)` — span luôn được record error và End bất kể return branch nào.
- **Bối cảnh (Trigger)**: Hot-path function trong Go service dùng OTel có nhiều error branch cần span luôn được record error + log carrier trace_id/span_id mà không phụ thuộc nhớ thủ công ở mỗi branch.
- **Root Cause**: Manual `span.RecordError(err)` rải rác mỗi error branch dễ miss; `attribute.String("error", err.Error())` thay vì RecordError → SigNoz Exception tab không nhận; log trực tiếp `logger.Error(...)` không qua `observability.Ctx(ctx, ...)` → không correlate được Logs↔Traces.
- **Fix/Correct Flow**: Dùng `defer observability.EndSpan(span, &err)` với named return. Log qua `observability.Ctx(ctx, logger)` để inject trace_id/span_id. Tạo helpers `ChildSpan`, `EndSpan`, `Ctx`, `ErrorField`, `Attrs` reuse 1 lần.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ Go service dùng OTel + zap + hot-path function có ≥2 error branch.
- **Tags**: #observability #otel #root-cause #testing #coupling
- **Nguồn**: lessons.md [2026-05-26]

### [2026-05-26] Metric được define nhưng không có call-site update gây alert dead và false confidence
- **Global Pattern**: `[Developer A define metric M và alert rule R]` nhưng `[không implement .Set()/.Inc()/.Observe() call-site trong runtime code path P]` → `[dashboard M = 0 hằng định, alert R không bao giờ kích dù sự cố thật xảy ra, false confidence]`. **Đúng**: Mỗi metric M phải có ≥1 call-site update; smoke test curl `/metrics` assert value ≠ 0; grep call-site trước khi merge — count = 0 → block PR.
- **Bối cảnh (Trigger)**: Dashboard panel hiển thị metric = 0 hoặc "no data" hằng định. Alert không bao giờ kích dù Kafka consumer lag tăng thật. Hệ quả là false confidence — dashboard "xanh đẹp" không phản ánh trạng thái thực.
- **Root Cause**: Developer khai báo metric M tại `metrics.go` với name + labels và định nghĩa alert rule R, nhưng không có bất kỳ call-site nào `.Set()/.Inc()/.Observe()` trong runtime code path để update M.
- **Fix/Correct Flow**: Definition + call-site coupling bắt buộc. Smoke test sau deploy assert metric có value ≠ 0. Synthetic alert test với artificial data. Cross-ref grep call-site trước merge. Khai báo metric gần call-site (không metric warehouse).
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ service production có observability stack (Prometheus, StatsD, DataDog, OTel metric).
- **Tags**: #observability #metric-dead #false-confidence #silent-drop #testing #root-cause #audit-log
- **Nguồn**: lessons.md [2026-05-26]

### [2026-05-25] Over-complicating scope khi nhận request đơn giản (Spec-Drift)
- **Global Pattern**: `[Agent/Executor A] tự mở rộng scope` lên `[feature đơn giản X (minimum viable change)]` → `[scope blow-up, kế hoạch phức tạp không cần thiết, User phản ứng tiêu cực]`. **Đúng**: khi nhận yêu cầu feature A, chỉ implement đúng A; nếu giải pháp đụng nhiều layer → hỏi/xác nhận scope tối thiểu với User trước khi thiết kế.
- **Bối cảnh (Trigger)**: User yêu cầu một thay đổi nhỏ (thêm cờ boolean truyền vào API). Agent tự suy diễn mở rộng thêm nhiều scope lớn như thêm dropdown UI, chia tách routing, thay đổi logic Worker và DB schema không được yêu cầu.
- **Root Cause**: Spec-Drift do suy diễn — Agent không bám sát yêu cầu tối giản, tự động mở rộng scope dưới danh nghĩa "Demand Elegance" hoặc Root Cause Analysis. Tư duy over-engineering: thay vì giải quyết trực tiếp pain-point, Agent thiết kế lại kiến trúc cả Table Registry.
- **Fix/Correct Flow**: Bám sát yêu cầu tối giản (Minimum Viable Change). Nếu giải pháp có nguy cơ đụng nhiều layer (FE + BE API + Worker + DB schema), dừng lại và hỏi/chỉ rõ mức độ tác động tối thiểu để User lựa chọn.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án phần mềm có nhiều layer khi nhận change request nhỏ; web apps, backend services, data pipelines.
- **Tags**: #over-engineering #spec-drift #minimum-viable-change #process-governance #discipline #architecture-design
- **Nguồn**: lessons.md [2026-05-25]

### [2026-05-21] DRY violation: resolver duplicate ở caller dẫn đến silent divergence giữa các consumer
- **Global Pattern**: `[Caller C1] implement inline logic L để bypass limitation của [shared resolver R]` → `[caller C2 dùng R behave khác C1 cho cùng input; silent divergence khó debug]`. **Đúng**: mở rộng R để cover convention mới, xoá L ở C1; đối xứng các layer resolve.
- **Bối cảnh (Trigger)**: Command snapshot.v2 fail với error "no usable DSN" cho một connection — trong khi cùng connection cùng worker, command scan-fields resolve thành công 6 phút trước.
- **Root Cause**: Logic resolve DSN tồn tại ở 2 nơi với độ phủ khác nhau — caller scanFields có logic inline detect host chứa full URI; shared resolver GetSourceDSN yêu cầu Port!=nil → row có Host=full URI nhưng Port=NULL bị fail.
- **Fix/Correct Flow**: Đối xứng hóa shared resolver — thêm layer tryPlainDSN(*conn.Host) + tryEnvPointer(*conn.Host) vào GetSourceDSN; xoá block build-DSN inline ở caller để single source of truth.
- **Phạm vi (≥3 dự án?)**: Có — DSN resolve, auth, masking, retry, telemetry trong bất kỳ hệ thống có shared resolver.
- **Tags**: #dry #single-source-of-truth #coupling #resolver #convention-drift #cdc #refactor
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-20] Publisher báo "success" trên transport-accept mà không probe downstream consumer state
- **Global Pattern**: `[Publisher P] calls send/publish trên transport T (Kafka, NATS, RabbitMQ, SQS); T returns success; P viết activity_log/metric/log "success"` lên `[fire-and-forget publish với stateful downstream consumer]` → `[Khi consumer C degraded (idle tasks, paused subscription, broken binding), message silently dropped/stalled; mọi surface upstream của C xanh; user biết chỉ bằng cách re-run hoặc tìm missing data]`. **Đúng**: Post-publish probe consumer state (HTTP status endpoint, admin API, consumer-group lag, subscription registry); probe trả RICH structure (state, task_count, task_state, reason — không chỉ bool); khi unhealthy → log ERROR + write activity_log error với full diagnostic.
- **Bối cảnh (Trigger)**: Publisher log "success" sau khi transport .Send() return nil; downstream consumer đang trong degraded state (connector idle, subscription missing); user không biết operation không thật sự reach consumer.
- **Root Cause**: Publisher chỉ verify transport-accept (nil error từ send/publish); không có awareness về consumer state; fire-and-forget publish không phân biệt "message delivered and processed" vs "message accepted by broker but dropped".
- **Fix/Correct Flow**: Publish path PHẢI có post-publish probe consumer state; probe phải RICH (không chỉ bool); unhealthy → LOG ERROR + structured activity_log row với full diagnostic; default: visibility (post-publish probe + loud error), không prevention (pre-flight gate).
- **Phạm vi (≥3 dự án?)**: Có — Debezium signal → connector tasks, Kafka → consumer group lag, NATS → subscription roster, webhook → 5xx tracking, S3 → SQS notification fanout.
- **Tags**: #observability #cdc #kafka #silent-drop #root-cause
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Shared resolver kế thừa filter dispatch-time gây false 404 cho read path
- **Global Pattern**: `[Shared resolver R] gate bằng [state predicate B (is_active=TRUE)] cần thiết cho [dispatch path]` → `[read/status path X reuse cùng resolver → entity Y vừa tạo, chưa active trả 404 dù tồn tại]`. **Đúng**: tách resolver theo intent — dispatch resolver giữ predicate (an toàn cho worker), read resolver bỏ predicate (UI phải thấy entity ở mọi state); share helper chỉ cho error mapping (ErrRecordNotFound, ErrAmbiguous).
- **Bối cảnh (Trigger)**: Entity mới tạo chưa active bị 404 trên GET endpoint; DB confirm có row; cùng resolver dùng cho cả dispatch và read; predicate `is_active = TRUE` lọc entity chưa kích hoạt.
- **Root Cause**: Predicate "fitness for action" (is_active, status='ready', enabled) thuộc về action path, không thuộc về observe path; khi dùng chung resolver, read path bị gated bởi predicate không cần thiết.
- **Fix/Correct Flow**: Tách thành `resolveForDispatch(sql)` (giữ predicate) và `resolveForRead(sql)` (không predicate); bất kỳ predicate dạng "fitness for action" phải audit: "endpoint này có dispatch không? Nếu chỉ đọc → predicate phải biến mất".
- **Phạm vi (≥3 dự án?)**: Có — CMS audit view (hiển thị user/order ở mọi trạng thái), Workflow engine GET /run/:id (phải trả kể cả cancelled/failed).
- **Tags**: #coupling #dry #architecture-design #resolver-pattern #false-404 #dispatch-vs-read
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Per-row UI action collapse khi backend resolve theo parent_id thay vì child_id
- **Global Pattern**: `[UI table A hiển thị N row cho 1 parent B (mỗi row là child Ci)] action gọi endpoint chỉ truyền [parent_id]` → `[backend pick Ck ≠ Ci qua ORDER BY LIMIT 1 → action chạy sai child; React reconciliation merge N row thành 1 node khi rowKey = parent_id]`. **Đúng**: backend list emit N row độc lập với child_id; endpoint action nhận optional child_id; rowKey composite `${parent_id}#${child_id ?? 'none'}`; per-row action gửi kèm child_id.
- **Bối cảnh (Trigger)**: "Nhấn 1 cái, 2+ cái cùng chạy" — toggle/button của row khác cũng kích hoạt; React warning "Encountered two children with the same key"; activity log ghi target_table khác với row vừa click.
- **Root Cause**: Ba layer cùng sai — backend list collapse N→1 (LATERAL LIMIT 1), backend command chỉ nhận parent_id nên pick child arbitrary, frontend rowKey là parent-level identifier; cascade update trên parent còn thay đổi state TẤT CẢ children.
- **Fix/Correct Flow**: Backend list → LEFT JOIN trực tiếp (N rows); command endpoint nhận `?child_id=` optional (backward-compat fallback về parent nếu không có child); rowKey composite; nếu action có semantic child-level → thêm endpoint `/children/:id` tránh cascade.
- **Phạm vi (≥3 dự án?)**: Có — CMS SKU/variant toggle, Workflow engine job/attempt retry, Multi-tenant user/role assignment.
- **Tags**: #architecture-design #coupling #one-to-many #ui-action #rowkey #child-id #cascade
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-19] Generic "empty" error ẩn 5 nguyên nhân khác nhau — không probe metadata trước khi tuyên bố empty
- **Global Pattern**: `[Service S probe source entity X (multi-level: container/namespace/entity/data)] tuyên bố "X is empty" mà không phân biệt 5 case khác nhau` lên `[multi-level source entity probe]` → `[User debug bằng cách thử lung tung; error message tù root chain ra downstream errors; user thấy 4 lỗi tưởng 4 bug, thật ra 1 nguyên nhân]`. **Đúng**: Probe meta trước khi tuyên bố empty — thử list L1→L2→L3 metadata; 5-case branching: L1 fail (cluster unreachable), L2 miss (namespace missing), L3 miss (entity missing), L4 count=0 (empty), L4 count>0 no fields; sanitize credentials trước khi log.
- **Bối cảnh (Trigger)**: User thử 3+ lần không fix được; đổ lỗi "code core đang vỡ" trong khi root cause là config/data vô hình vì error message tù — "source is empty" không nói được là cluster unreachable hay collection không tồn tại.
- **Root Cause**: Code pattern `if len(sample) == 0 { return errEmpty }` không probe — gộp 5 case khác nhau vào 1 error; không phân biệt container unreachable vs namespace missing vs entity missing vs data empty.
- **Fix/Correct Flow**: Probe meta thứ tự L1→L2→L3→L4; error riêng cho mỗi level miss; sanitize credentials helper trước khi log (strip user:pass@); structured log fields (connection_code, sanitized_dsn, available_xxx, doc_count); slow-path probe chỉ chạy khi sample empty → happy-path latency unchanged.
- **Phạm vi (≥3 dự án?)**: Có — Postgres FDW, S3 sync, Redis cache miss, HTTP API integration, Kafka topic consume.
- **Tags**: #observability #root-cause #cdc #silent-drop
- **Nguồn**: lessons.md [2026-05-19]

### [2026-05-18] Infinite recursion khi collocated connection pools gọi lẫn nhau khi khởi tạo
- **Global Pattern**: `[Resolver R cho database D2] fallback/query database D1 trong khi R được gọi trong quá trình khởi tạo D1 hoặc implicitly dùng D1 mà không có caching/circuit-breaking` lên `[collocated connection pool initialization]` → `[Infinite recursion / Stack overflow tại runtime]`. **Đúng**: R luôn thực hiện collocation check (so sánh Host, Port, DatabaseName của D2 vs D1); nếu cùng target → trả về pre-existing pool của D1 ngay lập tức, không resolve pool mới.
- **Bối cảnh (Trigger)**: Khi khởi chạy hoặc chạy unit test cho CDC pipeline, hàm `Registry.GetDB` gọi đệ quy vô hạn gây stack overflow vì Shadow DB và SystemDB được collocated trên cùng physical DB.
- **Root Cause**: Logic phân giải connection pool của Shadow DB gọi đệ quy ngược về `Registry.GetDB(source)` để lấy DB target, trong khi đó `Registry.GetDB` cố gắng khởi tạo connection từ config registry thông qua chính pool đó.
- **Fix/Correct Flow**: Implement collocation check trước khi resolve pool mới: so sánh Host+Port+DatabaseName; nếu match → return pre-existing pool; break recursion loop.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có connection pool registry với collocated databases (multi-tenant, shared infra).
- **Tags**: #coupling #observability #root-cause
- **Nguồn**: lessons.md [2026-05-18]

### [2026-05-18] Design resolver dựa trên giả định column semantics, không inspect sample DB row thực tế
- **Global Pattern**: `[Agent A] designs resolver R cho column C của table T dựa trên column name semantics + historical conventions, mà không inspect actual production/dev sample row của C` lên `[resolver design + DB column]` → `[R over-engineered cho case đơn giản hơn thực tế, hoặc R wrong-engineered vì column C có content khác tên ngụ ý (e.g. "host" chứa full URI)]`. **Đúng**: Query actual sample row (SELECT ... LIMIT 1) TRƯỚC khi design R; inspect content shape per column; design R cho SHAPES OBSERVED + minimal generalization; tránh speculative scheme layers không có sample evidence.
- **Bối cảnh (Trigger)**: Brain design resolver multi-scheme 4-layer cho `connection_registry.secret_ref`. User chỉ ra sample row thực tế: field `host` đang lưu cả URI (mongodb://gpay-mongo:27017/...), port/default_database NULL. Code build "mongodb://${host}:${port}/" sẽ ra string sai (URI nhúng trong URI).
- **Root Cause**: Agent design resolver dựa trên giả định "field ngữ nghĩa theo tên cột" + lessons cũ về convention thay vì query 1 sample row thực để xem giá trị thực.
- **Fix/Correct Flow**: Trước khi viết resolver cho field DB, dump 1-3 row hiện hữu của bảng; quan sát field dual-use (tên "host" có thể chứa URI); resolver chỉ cover các shape ĐÃ THẤY trong sample + 1 minimal fallback.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có column với dual-use semantics (URI/bare-host, pointer/literal, code/id).
- **Tags**: #over-engineering #root-cause #coupling #dry
- **Nguồn**: lessons.md [2026-05-18]

### [2026-05-07] Router-level swap khi V2 handler chỉ là thin-delegate to V1
- **Global Pattern**: `[V2 handler B] duplicate logic` lên `[V1 handler A mà B chỉ thin-delegate 1-line]` → `[DRY violation + maintenance burden]`. **Đúng**: xóa thin-delegate method B.X; mount route URL namespace-B trực tiếp vào A.X trong router; xóa field/constructor reference không còn dùng.
- **Bối cảnh (Trigger)**: Namespace evolve (V1→V2, legacy→modern) với handler B chứa >50% method là 1-line `return aHandler.X(c)`; muốn "thống nhất" nhưng sắp duplicate hoặc tạo facade thừa.
- **Root Cause**: Giữ handler B làm shim dù nó chỉ thin-delegate toàn bộ → DRY violation; thêm facade service C để cả A và B gọi là over-abstraction khi A đủ làm owner.
- **Fix/Correct Flow**: Xóa method delegate B.X; trong router: `routerGroup.Method("/path-namespace-B/X", aHandler.X)`; nếu B còn helper field reference A không dùng → xóa luôn. Cảnh báo: Swagger godoc gắn theo method bị mất → phải move annotation hoặc accept doc loss.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ codebase nào có namespace evolve với shim handler thin-delegate (Go, Node.js, Java Spring).
- **Tags**: #dry #coupling #over-engineering #refactor #router-swap #namespace-evolution #simplicity
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-06] CommandBus chỉ cho mutation/coordination, không migrate audit-only side-effects
- **Global Pattern**: `[Reviewer/team đề xuất migrate side-effect X (audit log, metrics, fingerprint)] qua` lên `[CommandBus B (CQRS C-side) "for consistency"]` → `[thêm hop sync + double-audit + idempotency collision risk; semantic gain = 0]`. **Đúng**: bus chỉ cho track Mutation (DDL ALTER, business state write, external destructive REST) và Coordination (async cross-service dispatch); side-effect audit/log giữ direct call ở handler/service layer.
- **Bối cảnh (Trigger)**: Refactor CQRS C-side với CommandBus; câu hỏi có nên migrate audit log write (ActivityLog, metrics emission) qua bus để "uniform pattern" hay không.
- **Root Cause**: "Universal indirection" thinking — tin mọi handler-level write nên qua bus; bỏ qua phí bus (hop sync, double-recording, idempotency collision, test surface tăng) khi side-effect không có blast radius.
- **Fix/Correct Flow**: Phân loại blast radius: nếu X fail chỉ log warn (không rollback request) → audit-only → KHÔNG bus; nếu X fail rollback request → mutation-essential → qua bus. Giữ audit write direct call.
- **Phạm vi (≥3 dự án?)**: Có — CQRS Java/Spring với Axon/EventBus, NestJS @CommandBus, Workflow engine (Temporal/Camunda) "everything is an activity".
- **Tags**: #cqrs #command-bus #over-engineering #coupling #audit-log #blast-radius #dry
- **Nguồn**: lessons.md [2026-05-06]

### [2026-05-05] Hardcode naming convention tại N call sites — centralize thành naming package env-driven
- **Global Pattern**: `[Convention naming X hardcoded tại N call sites trong codebase A để tạo identifier `X<Y>`]` → `[Đổi convention = sửa N file, risk sót hit do cùng từ xuất hiện ở comment/enum/log]`. **Đúng**: tạo package `naming` tập trung expose `<Convention>Name(parts...) string` đọc env `<DOMAIN>_<CONVENTION>_<PART>` via `sync.Once`, default fallback = giá trị cũ để backward-compat; mọi call site `"X" + dynamic` đổi sang `naming.<Convention>Name(dynamic)`.
- **Bối cảnh (Trigger)**: Schema prefix `shadow_` hardcoded ở 4 call sites trong `centralized-data-service`; đổi convention cần sửa 4 file + risk grep nhầm `shadow_pending` state enum và `cdc.cmd.shadow.bind` NATS subject.
- **Root Cause**: Convention naming không được treat như configuration; hardcode literal ở nhiều điểm; không có single-source-of-truth; từ khóa dùng chung cho nhiều domain (schema name, state enum, NATS subject) khó grep chính xác.
- **Fix/Correct Flow**: Tạo `internal/naming/naming.go` với `sync.Once` cache env; mọi schema-creating call site dùng `naming.ShadowSchemaName()`; verify `grep -rn '"shadow_"' repo/` schema-creating sites = 0 hit; smoke test env override + default fallback.
- **Phạm vi (≥3 dự án?)**: Có — CDC pipeline prefix, e-commerce tenant schema prefix, observability metric name prefix, bất kỳ codebase có convention naming dùng ở nhiều nơi.
- **Tags**: #naming #convention #env-driven #refactor #single-source-of-truth #dry #coupling
- **Nguồn**: lessons.md [2026-05-05]

### [2026-04-29] Audit middleware đọc reason từ body, FE gửi qua header — 400 mismatch
- **Global Pattern**: `[FE/CLI client A gửi required value X (reason/actor/correlation_id) qua header]` nhưng `[audit gate B đọc X từ body]` → `[gate chặn 400 mặc dù header có giá trị Y]`. **Đúng**: gửi giá trị bắt buộc X ở CẢ HAI vị trí: body (cho audit middleware) VÀ header (cho proxy/log scraper); V-shaped redundancy.
- **Bối cảnh (Trigger)**: FE hook gửi destructive action POST chỉ embed reason trong header (`X-Action-Reason`); backend audit middleware `extractReason` đọc field JSON body `reason` → 400 "missing or too-short reason".
- **Root Cause**: FE assume header là canonical source; backend audit middleware đọc body là canonical source; không có shared contract documentation rõ ràng về vị trí của `reason`.
- **Fix/Correct Flow**: Client.post(url, { mode, reason }, { headers: { 'X-Action-Reason': reason } }) — gửi reason trong cả body và header; không đoán nguồn nào là canonical, gửi cả hai.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ service dùng pattern "dual-channel destructive verb" (Idempotency-Key header + reason body); Stripe-style, AWS request signing, GitHub PUT-with-confirm-header.
- **Tags**: #architecture-design #serialization-type #api-contract #audit #destructive-chain #frontend-backend #idempotency
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-24] Infra-control endpoint expose trực tiếp lên FE — bypass auth, audit, idempotency
- **Global Pattern**: `[Infra REST endpoint A expose trực tiếp lên browser FE B]` mà không qua `[app auth + audit + idempotency proxy C]` → `[audit/security/replay loss: action không attributable, duplicate possible, compliance gap Y]`. **Đúng**: viết CMS handler proxy forward request tới infra endpoint; route qua chain JWT → RequireOpsAdmin → Idempotency → Audit; validate input trước khi forward; strip sensitive fields khi GET về FE.
- **Bối cảnh (Trigger)**: Kafka-Connect REST (port 18083) public accessible; nếu FE gọi thẳng → bypass auth + bypass audit; cần proxy qua CMS backend với destructive chain.
- **Root Cause**: Dev chọn đường tắt "FE gọi thẳng endpoint infra" vì sẵn có; tạo 3 rủi ro: không auth layer, không idempotency, không audit log.
- **Fix/Correct Flow**: Mọi infra-control plane (Kafka Connect, Airbyte, Prometheus admin, k8s API) phải qua app proxy với JWT + RequireOpsAdmin + Idempotency + Audit chain; FE gửi Idempotency-Key + reason ≥10 chars cho mọi destructive request.
- **Phạm vi (≥3 dự án?)**: Có — AWS infra admin từ BI dashboard, Grafana từ customer portal, Kafka-Connect từ CMS UI, Prometheus từ ops cockpit.
- **Tags**: #architecture-design #coupling #security #audit #proxy #idempotency #infra-control
- **Nguồn**: lessons.md [2026-04-24]

### [2026-04-24] Gắn draft mutation vào destructive chain — audit noise + FE handshake phí
- **Global Pattern**: `[Endpoint E (create/update draft state) gắn vào destructive chain B]` chỉ vì `[E là POST/PATCH method]` → `[audit noise + FE handshake nặng không cần thiết + false compliance Y]`. **Đúng**: phân tier ngay tại design time: destructive ⇔ DDL/infra-plane call/irreversible fan-out; admin mutation ⇔ CRUD metadata-only (draft, config toggle chưa live); chỉ tier destructive cần Idempotency-Key + reason.
- **Bối cảnh (Trigger)**: `POST /v1/wizard/sessions` (create DRAFT) và `PATCH /v1/wizard/sessions/:id` (update session fields) bị mount qua `registerDestructive`; FE gọi nhận 400 "missing Idempotency-Key" và "missing reason" cho action chỉ là form state persisted BE-side.
- **Root Cause**: Lẫn lộn semantic layer khi phân tier; "destructive" = side-effect trên shared infrastructure (DDL, infra-plane), không phải mọi POST/PATCH; create/patch draft state không gây side-effect thật.
- **Fix/Correct Flow**: Phân tích mỗi endpoint: chạm infra? → destructive. Chỉ đụng BE row? → admin mutation. Chỉ đọc? → shared. Mount đúng tier ở router.go; FE chỉ gắn Idempotency-Key + reason cho tier destructive (execute/commit/delete).
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ state-machine endpoint (wizard, draft form, saga orchestrator) cần tách create/update draft khỏi execute/commit/publish.
- **Tags**: #architecture-design #coupling #route-tier #destructive-chain #state-machine #draft-vs-commit #audit-noise
- **Nguồn**: lessons.md [2026-04-24]

### [2026-04-23] Scaffold CSS global overrides component library tokens — contrast clash ở OS dark mode
- **Global Pattern**: `[Scaffold global CSS A khai báo color-scheme + prefers-color-scheme:dark]` lên `[component library X dùng light-theme default không có ThemeProvider riêng]` → `[contrast clash, text/input unreadable ở OS dark mode Y]`. **Đúng**: audit và xóa CSS custom properties scaffold không cần thiết ngay khi integrate component library; nếu cần dark mode thì mount ConfigProvider/ThemeProvider, không dựa vào CSS media query riêng.
- **Bối cảnh (Trigger)**: Vite/CRA/Next template index.css khai báo `color-scheme: light dark` + `@media (prefers-color-scheme: dark)` swap global color/bg; component library (AntD) không có ThemeProvider riêng → OS dark mode flip text màu, component stays light → contrast xuống dưới WCAG AA 4.5:1.
- **Root Cause**: Default scaffold CSS cascade đè vào component library tokens; dev không audit global CSS khi integrate lib; component lib dùng light-theme default nhưng global CSS flip color độc lập.
- **Fix/Correct Flow**: Xóa `color-scheme`, `color`, `background` trên `:root`/`html`/`body` khi component library tự handle; xóa `@media (prefers-color-scheme: dark)` block trừ khi có explicit dark mode via ConfigProvider; chỉ giữ reset (margin/padding), font stack, box-sizing.
- **Phạm vi (≥3 dự án?)**: Có — AntD + Vite, MUI + Next, Chakra + CRA, bất kỳ React project tích hợp component library.
- **Tags**: #architecture-design #coupling #scaffold-cruft #css-theming #wcag #over-engineering
- **Nguồn**: lessons.md [2026-04-23]

### [2026-04-21] Dùng Wikipedia-level distributed primitives thay vì production-level — fencing/outbox/physical-scan bị bỏ sót
- **Global Pattern**: `[Architect A implement feature F ở scale S]` bằng `[Wikipedia-level primitives (heartbeat-only, sync-in-tx, manual config, ORDER BY id)]` → `[P fail ở distributed edge case mà production primitives (fencing token, outbox, pg_export_snapshot) sẽ bắt Y]`. **Đúng**: trước khi propose solution, enumerate distributed primitives áp dụng (fencing, outbox, snapshot, MVCC); nếu không reference những này = incomplete answer.
- **Bối cảnh (Trigger)**: Brain đề xuất v6 heartbeat-based reclaim, sync-within-transaction, manual locale config per-field, ORDER BY id backfill; user reject vì thiếu fencing token, outbox, data profiling, physical slot scan.
- **Root Cause**: Brain "textbook" = Wikipedia-level basics; user "textbook" = production engineering primitives từ Designing Data-Intensive Applications, Kleppmann papers, pg_repack/Debezium internals.
- **Fix/Correct Flow**: Distributed locking phải có fencing token (heartbeat alone = unsafe); high-throughput writes dùng outbox/logical replication/CDC; config ở scale cần auto-inference + admin override; backfill dùng ctid-based ranges + pg_export_snapshot.
- **Phạm vi (≥3 dự án?)**: Có — distributed locking systems, high-throughput data pipelines, large-scale backfill jobs, multi-tenant config systems.
- **Tags**: #architecture-design #fencing-token #outbox-pattern #distributed-primitives #over-engineering #root-cause
- **Nguồn**: lessons.md [2026-04-21]

### [2026-04-20] Lesson cũ không enforce cho new code — repeat violation cùng pattern đã có ADR
- **Global Pattern**: `[Agent A viết code N tại thời điểm T1]` + `[Lesson L về pattern P đã documented tại T0 < T1]` → `[violation Y nếu A không check L trước khi viết N]`. **Đúng**: Lesson = active reference, không phải archive. Pre-commit grep ADR trước khi write endpoint chạm domain nhạy cảm. Architectural review step trong bug-handling-sop khi bug liên quan architectural decision cũ.
- **Bối cảnh (Trigger)**: `ScanFields` endpoint mới vi phạm 3 rules: HTTP sync thay vì NATS async (ADR-015), CMS touches Airbyte trực tiếp (service boundary ADR), hardcoded AirbyteSourceID bỏ qua registry. Cả 3 rules đã ghi lesson/ADR từ trước — ScanFields là code MỚI vẫn lặp lại.
- **Root Cause**: Lesson thụ động. Không có gate tự động nhắc "grep ADR cũ trước khi viết". Brain/Muscle delegate code mới thiếu pre-flight check "feature mới có lặp pattern cấm không?".
- **Fix/Correct Flow**: Pre-commit grep ADR: `rg "service_boundary|ADR-[0-9]+" agent/memory/` trước khi write endpoint mới. Endpoint checklist: "Có dùng NATS async? Có tuân service boundary? Có support multi-source registry?". Repeat-violation detection: Brain scan periodically.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có Architecture Decision Records, service boundary rules, coding conventions.
- **Tags**: #architecture-design #adr-enforcement #repeat-violation #service-boundary #coupling #dry
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-20] Silent-skip trong scheduled jobs masks nil-dependency init failures
- **Global Pattern**: `[Scheduled job A phụ thuộc lazily-initialised core B]` → `[startup failure của B leaves A.core=nil → A.Tick() silently short-circuits với "skipped" row]` → `[operators không thể distinguish "skipped-by-config" vs "crashed" vs "never-scheduled"]`. **Đúng**: mọi silent-skip path PHẢI WARN log stream trên first skip + mọi tick; include fix_hint trong log fields; emit startup summary khi poller starts.
- **Bối cảnh (Trigger)**: Worker's scheduled `reconcile` op ghi "skipped" khi `reconCore == nil`. Operators thấy zero reconcile activity nhưng không có error. Real cause: MongoDB URL missing từ config, caught chỉ trong startup WARN buried.
- **Root Cause**: Activity-log rows KHÔNG substitute cho log-stream WARN khi condition là dependency-init failure. Audit tables per-record; log streams temporal — operators scan stream khi diagnose "is this running?".
- **Fix/Correct Flow**: WARN log stream trên first skip + mọi tick. Include `fix_hint` trong log fields. Emit startup summary: `enabled_count=N registered=[op=Nm] recon_core_available=bool`. Per-tick info log: `first_run:bool`. KHÔNG replace silent-skip với panic — dùng WARN-log + keep running + `/metrics` counter.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho cron DLQ replayers, scheduled Airbyte triggers, Prometheus push gateways, bất kỳ graceful-degrade path.
- **Tags**: #observability #silent-failure #nil-dependency #scheduling #architecture-design #log-stream
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-20] Per-entity band-aid config thay vì systematic auto-detect — không scale N entities
- **Global Pattern**: `[Agent A configures entity B_i với field F manually cho mỗi i ∈ N entities]` → `[O(N) human intervention + high error rate + unmaintainable ở scale]`. **Đúng**: auto-detect tại entity boundary (sample data → detect field presence ranking → auto-populate config); fallback chain runtime; admin override escape hatch; log recommendations với confidence.
- **Bối cảnh (Trigger)**: User report payment_bills recon src=0 (Mongo 2 docs với `createdAt`, không `updated_at`). Brain đề xuất "Quick fix: UPDATE registry SET timestamp_field='createdAt' WHERE target_table='payment_bills'". User: "với quy mô 200 table, mày cũng fix từng cái à".
- **Root Cause**: Brain optimize cho "fix bug hiện tại" thay vì "fix cơ chế gây ra bug". Per-entity fix = tình thế (band-aid). Session history đã lặp pattern: export_jobs cũng manual fix timestamp_field trước đó.
- **Fix/Correct Flow**: Auto-detect tại entity boundary: sample data → detect field. Fallback chain runtime: nếu configured field trả 0 docs trong N runs → auto-try next candidate. Admin override escape hatch cho manual override khi auto fail. Log recommendations với confidence%.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi multi-entity system cần per-entity configuration (schema registry, connector config, ETL metadata).
- **Tags**: #architecture-design #band-aid #auto-detect #scale #dry #over-engineering
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-20] Passive plan (band-aid) vs Systematic Reconstruction — 6 violations cùng lúc
- **Global Pattern**: `[Agent A plans architectural reconstruction R]` + `[A defaults to minimum-disruption M]` → `[fail-to-deliver R vì tàn dư cũ chính là bug source]`. **Đúng**: Reconstruction ≠ Migration. Reconstruction đòi hỏi drop + rebuild clean slate. Migration đòi hỏi preserve + transform backward compat. Nhầm 2 modes = plan nửa vời.
- **Bối cảnh (Trigger)**: User provide Master Plan v1.25 (Unified Sonyflake). Brain viết plan vi phạm 6 nguyên tắc: View band-aid giữ `_airbyte_*` rác, Trigger IF NULL thay vì FORCE DB, mapping spaghetti, COALESCE anti-ghosting quên OCC, giữ PK cũ dual-index, Worker ID 0 không verify.
- **Root Cause**: Brain mặc định minimum-disruption = good. Với architectural reconstruction, minimum-disruption = lỗ hổng vì tàn dư cũ chính là bug source. User yêu cầu "Unified" (nguyên khối), Brain trả "incremental alias" (trái nguyên tắc).
- **Fix/Correct Flow**: Reconstruction: physical clean slate (không giữ column rác dưới mọi hình thức); force authority (Identity Provider SINGLE, DB validate STRICT); unified naming (không alias); preserve what earned its place (working pattern → rename, không thay thế ad-hoc); aggressive cutover trong cùng migration transaction; verify environment trước reserve.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi architectural reconstruction: identity system migration, schema unification, service consolidation.
- **Tags**: #architecture-design #reconstruction-vs-migration #band-aid #identity-authority #unified-naming #physical-clean-slate
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-17] Plan data system thiếu "Scale Budget" — patterns sai lệch ở scale lớn
- **Global Pattern**: `[Agent A lập plan cho data system B với quy mô X lớn (>10M records)]` → `[A không tính memory footprint, network transfer, DB CPU/IO, query latency cho từng operation]` → `[plan fail catastrophically ở production scale]`. **Đúng**: Mỗi plan data system BẮT BUỘC có mục 0 "Scale Budget"; mỗi task phải trả lời "ở scale X, thao tác này consume bao nhiêu memory/network/DB?"; dùng window-based, sampled, incremental, hash-aggregate patterns.
- **Bối cảnh (Trigger)**: User flag plan CDC: "check id chữa lành đang get hết id ra 1 lượt so sánh. 50 triệu record là tư duy tệ khủng khiếp." Plan viết ở mindset "book-example" với dataset 1M.
- **Root Cause**: Plan viết ngầm định memory/network/DB load nhỏ. Không tính toán: `50M × 12 bytes ObjectId = 600MB` qua network, `50M × 2KB doc = 100GB` scan, `200 bảng × 5 phút count query = 2400 full-scan/giờ`. Scale 50× kích thước giả định → toàn bộ pattern sụp.
- **Fix/Correct Flow**: Section 0 "Scale Budget" trong mọi data system plan. Anti-patterns cấm: fetch full ID set vào RAM, `SELECT COUNT(*)` thường xuyên trên >10M rows, flat chunk hash, blanket `cleanup.policy=compact`. Dùng: window-based, XOR-hash aggregate, bucketed hash, sampling.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi data engineering dự án có dataset >10M records.
- **Tags**: #architecture-design #scale #data-integrity #performance #mandatory-scale-budget #cdc
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-16] Shallow technical analysis thiếu distributed systems failure modes
- **Global Pattern**: `[Coordinator/Brain A phân tích failure modes của distributed system B]` → `[A chỉ nhìn bề mặt (Worker die → Kafka giữ messages), không phân tích cascading failures]` → `[plan thiếu chiều sâu, cần rewrite]`. **Đúng**: think like SRE — liệt kê MỌI component có thể fail, cascading effects, recovery mechanism, data loss window; không chỉ happy path.
- **Bối cảnh (Trigger)**: User yêu cầu phân tích Worker downtime + reconciliation. Agent viết plan thiếu chiều sâu: không phân tích Debezium/Kafka die, không đề cập Oplog retention, không thiết kế Recon Agent/Core architecture, không nêu Idempotency/DLQ/Observability.
- **Root Cause**: Agent không đủ domain knowledge về distributed systems failure modes. Chỉ nhìn bề mặt, không phân tích cascading failures (Debezium die, Oplog overflow, schema change during downtime).
- **Fix/Correct Flow**: Khi phân tích failure modes: liệt kê MỌI component có thể fail (Worker/Debezium/Kafka), cascading effects, recovery mechanism (Recon Core + Agent), tiered approach với ACTION per tier, Worker hardening (Idempotency, DLQ, Observability).
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi distributed systems có multi-component dependencies, CDC pipelines, streaming architectures.
- **Tags**: #architecture-design #failure-analysis #distributed-systems #observability #cdc #sre-mindset
- **Nguồn**: lessons.md [2026-04-16]

### [2026-04-15] Hardcode field/column names thay vì đọc schema dynamically
- **Global Pattern**: `[Executor A fix lỗi E1 bằng hardcode H1, rồi E2 bằng H2, rồi E3 bằng H3]` → `[infinite bug chain, code không maintainable]`. **Đúng**: gặp lỗi lần 2 cùng vấn đề → DỪNG, gọi Coordinator phân tích; đọc target schema DYNAMICALLY từ `information_schema`; thiết kế adapter layer dynamic (source schema → target schema).
- **Bối cảnh (Trigger)**: CDC Worker BatchBuffer hardcode `_airbyte_raw_id`, `_airbyte_extracted_at`, JSONB column list, UNIQUE constraint. Mỗi table có schema khác → lỗi khác → fix chắp vá liên tục 8-9 lần.
- **Root Cause**: Executor code kiểu mì ăn liền — thấy lỗi gì fix lỗi đó bằng hardcode. Không gọi Coordinator phân tích root cause. Không thiết kế systematic solution.
- **Fix/Correct Flow**: Gặp lỗi lần 2 → DỪNG, escalate. Đọc schema dynamically từ `information_schema`. Thiết kế adapter layer: source schema → target schema, dynamic không hardcode. Hệ thống phải hoạt động cho BẤT KỲ table nào.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dynamic data pipeline, CDC workers, ETL với multi-table support.
- **Tags**: #hardcode #system-design #root-cause #coupling #dry #schema-migration
- **Nguồn**: lessons.md [2026-04-15]

### [2026-04-06] Indexing Mismatch trong Mapping Cache (X-to-Y Pattern)
- **Global Pattern**: `[Cache/Store A index dữ liệu theo định danh nguồn X]` lên `[Execution Context B dùng định danh đích Y để truy vấn]` → `[cache miss, lookup sai, data không tìm thấy]`. **Đúng**: khi khởi tạo cache, xây Intermediate Map `X→Y`; index nội dung theo Y; đảm bảo Context key và Cache key luôn đồng bộ.
- **Bối cảnh (Trigger)**: Task chuẩn hoá dữ liệu từ nguồn X sang đích Y; EventHandler truy vấn theo Y nhưng Cache lại index theo X.
- **Root Cause**: In-memory Indexing Mismatch — Agent mặc định lưu cache theo định danh nguồn, quên rằng execution context dùng định danh đích.
- **Fix/Correct Flow**: Khi khởi tạo/reload cache: xây bảng tra cứu trung gian `X→Y` từ Registry, index nội dung trực tiếp theo Y, đảm bảo High-frequency Key Alignment giữa context truy vấn và cache key.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho ETL pipelines, CDC systems, API gateways với dual-identifier patterns.
- **Tags**: #indexing #mapping #cache-strategy #high-frequency-key #mismatch #coupling
- **Nguồn**: lessons.md [2026-04-06]

### [2026-03-24] Không truy cập cross-domain model trực tiếp trong CQRS Handler (Clean Architecture)
- **Global Pattern**: `[CQRS Handler A] truy cập trực tiếp [model của domain khác B]` → `[vi phạm Clean Architecture; bẻ gãy cấu trúc Base Export; coupling cross-domain]`. **Đúng**: tạo AuxiliaryQuery + AuxiliaryHandler riêng; map subQueryClass ở lớp format; AuxiliaryHandler thu thập data qua Promise.all.
- **Bối cảnh (Trigger)**: Cần lấy thêm data từ model khác (PaymentBillModel) trong Export Handler; nhúng code truy cập DB của model thứ 2 trực tiếp trong handler.
- **Root Cause**: Truy cập trực tiếp chéo model từ Handler CQRS bẻ gãy Clean Architecture và cấu trúc Base Export phân tách miền.
- **Fix/Correct Flow**: Tạo [Name]ExportAuxiliaryQuery + AuxiliaryHandler; map subQueryClass ở lớp format .pure.ts; AuxiliaryHandler dùng Promise.all để lấy data và trả cho mergeData.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có CQRS/CommandBus pattern.
- **Tags**: #cqrs #coupling #dry #architecture-design #separation-of-concerns
- **Nguồn**: lessons.md [2026-03-24]

### [2026-02-27] Over-engineering khi gặp lỗi đơn giản — refactor core thay vì check file thiếu
- **Global Pattern**: `[Agent A] giả định [lỗi phức tạp X (circular dependency)]` → `[refactor core/base ổn định; trong khi root cause là [file thiếu Y đơn giản]; minimal impact violation]`. **Đúng**: khi lỗi "Unknown type/class" → check file đã tạo và tên đúng chưa TRƯỚC khi giả định cơ chế import phức tạp; tuyệt đối không đụng Base Logic/Orchestrator khi chỉ xây module Add-on.
- **Bối cảnh (Trigger)**: Gặp lỗi "Unknown export type" — thay vì check file class đã tạo chưa, Brain giả định Circular Dependency và refactor hàng loạt code core.
- **Root Cause**: Thiếu tư duy Simplicity First; bỏ qua nguyên nhân đơn giản nhất để nhảy tới giả định phức tạp; vi phạm Minimal Impact principle.
- **Fix/Correct Flow**: Double check the obvious — khi báo "Unknown type/class", kiểm tra file đã tạo và đúng tên; không đụng core stable code khi chỉ xây add-on; revert ngay nếu sửa sai hướng.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có modular architecture.
- **Tags**: #over-engineering #simplicity-first #root-cause #coupling #process-governance #verification
- **Nguồn**: lessons.md [2026-02-27]

### [0000-00-00] Hybrid command bus cần ResultBody slot để sync handler trả inline result
- **Global Pattern**: `[Hybrid bus B route command qua sync handler X (low-latency) / async path Y (long-running), nhưng CommandResult chỉ có {JobID, Accepted} không có ResultBody]` → `[Sync handler X trả nothing → caller buộc poll /jobs/:id sau mỗi Dispatch dù đã có kết quả ngay; 2 round-trips cho việc lẽ ra 1]`. **Đúng**: declare `CommandResult.ResultBody json.RawMessage` (nullable, omitempty); sync handler populate ResultBody; async path để nil; caller biết `Accepted=true && ResultBody!=nil` → inline render; `Accepted=true && ResultBody==nil` → poll.
- **Bối cảnh (Trigger)**: CQRS C-side CommandBus có sync path cho `alert.ack` (1 row UPDATE) và async path cho `master.swap` (long-running worker); CommandResult thiếu wire body → FE poll `/jobs/:id` sau mọi Dispatch.
- **Root Cause**: Bus author áp pattern "all async" (fire-and-forget) lên cả sync path để giữ contract đồng nhất → đánh mất ưu thế latency của sync route.
- **Fix/Correct Flow**: Thêm `ResultBody json.RawMessage` với `omitempty` vào CommandResult; sync handler populate; async để nil; FE check `ResultBody` presence để chọn inline render vs poll.
- **Phạm vi (≥3 dự án?)**: Có — CQRS microservice gateways (Go, .NET MediatR, Java Axon), BFF/API gateway cache-hit vs backend job, LLM tool-use immediate vs background, gRPC unary vs stream, WebSocket ack-only vs ack+payload.
- **Tags**: #cqrs #command-bus #hybrid-sync-async #api-design #latency #architecture #over-engineering
- **Nguồn**: lessons.md [Lesson #1295]

---

## 3. Schema & Migration — DDL, Migration ordering, search_path, Model↔DB Drift

_Bài học về tiến hoá schema: thứ tự DDL/migration, search_path, drift giữa model và DB, add/rename column an toàn._ — **30 pattern**

### [2026-06-09] State-flag ("đối tượng tồn tại thật") phải ĐỌC reality, KHÔNG suy từ predicate-proxy
- **Global Pattern**: `[Flag F nghĩa "X tồn tại thật" (vd in_master = cột CÓ trong table) nhưng SET bằng predicate-proxy của stage trigger (status=approved, hoặc is_active AND approved) thay vì đọc trạng thái thật]` → `[F nói dối khi proxy lệch reality: (a) predicate đúng nhưng stage chưa chạy → F=true mà X chưa tồn tại; (b) đổi cờ điều khiển SAU khi đã tạo (tắt is_active) nhưng artifact không bị xoá → proxy=false mà X vẫn tồn tại]`. **Đúng**: tách *process-gate* (chọn rule để xử lý — dùng predicate is_active AND approved) ≠ *state-flag* (X có tồn tại — phải QUERY thực tế). Sau stage mutate (DDL), ĐỌC reality (information_schema.columns ở dest) rồi `SET F=(target_column IN actualCols)`. Flag = ảnh chụp reality, không suy từ filter của stage.
- **Bối cảnh (Trigger)**: in_master set bằng `status='approved'` (rồi thử proxy `is_active AND approved`) → sai CẢ 2 chiều: field approved-inactive đã có cột bị báo not-in-master; field chưa DDL bị báo in-master. User: "in master là CÓ field trong table, sao lại check is_active & approve".
- **Root Cause**: nhầm "ĐIỀU KIỆN để tạo cột" (predicate trigger) với "cột CÓ tồn tại" (state thật). DDL chỉ ADD không DROP business col → proxy không bao giờ khớp reality ở mọi thời điểm.
- **Fix/Correct Flow**: master_ddl_generator sau ALTER → query `information_schema.columns` (dest, qua conn `db`) → `UPDATE mapping_rule_master SET in_master=(target_column IN (?))` (control g.systemDB). Verify 2 chiều cross-DB: {in_master=true}∖{cột}=∅ VÀ {cột∈rule}∖{in_master=true}=∅; edge: tắt is_active field đã có cột → in_master VẪN true (cột không bị drop).
- **Lưu ý phụ**: process-gate phải nhất quán giữa các stage (DDL+transmute cùng lọc is_active AND approved); scan default is_active=true (đồng bộ shadow-mirror) để "duyệt = dùng".
- **Phạm vi (≥3 dự án?)**: Có — mọi "materialized/exists/synced/deployed" flag (index-created, file-exists, đã-provision); nguyên tắc: flag-trạng-thái = đọc reality, không suy từ predicate.
- **Tags**: #state-flag #proxy-vs-reality #idempotency #ddl #information-schema #cross-db-verify #root-cause
- **Nguồn**: lessons.md [2026-06-09]

### [2026-06-09] Đừng gộp "UNIQUE constraint (chống trùng OUTPUT)" với "FIND key (idempotency theo SOURCE)"
- **Global Pattern**: `[Dev đổi UNIQUE của table T từ khoá-source X sang khoá-output Y để cho "1 source → N output", rồi thay LUÔN mọi chỗ "tìm theo X" (ON CONFLICT/lookup idempotency) sang Y]` → `[mất ngữ nghĩa "tạo-lại-nếu-chưa-có-theo-source"; auto-populate tạo trùng/sai khi Y bị đổi tên; user thấy "phá logic"]`. **Đúng**: tách 2 mục đích — (a) UNIQUE chống trùng = theo định danh OUTPUT (vd target_column, 1 table không cho trùng tên cột); (b) FIND/idempotency = theo định danh SOURCE (vd mapping_v2_id = field nguồn). Khi drop UNIQUE-source để cho 1→N, GIỮ "tìm theo source" bằng `WHERE NOT EXISTS(... source_id=...)` (không cần unique index) thay vì đổi ON CONFLICT sang output.
- **Bối cảnh (Trigger)**: mapping_rule_master cần "1 shadow field → nhiều master column". Drop unique (mbid,mapping_v2_id) + giữ unique (mbid,target_column) = ĐÚNG. Nhưng đổi LUÔN 3 auto-populate ON CONFLICT(mapping_v2_id)→(target_column) → mất idempotency theo shadow field (sync tạo lại cột default khi field đã map dưới tên khác).
- **Root Cause**: gộp 2 khái niệm — `target_column` là tên cột OUTPUT (chống trùng), KHÔNG phải định danh nguồn để "tìm". 1→N "lòi ra" tự nhiên vì bỏ unique trên source, không cần source_path (source_path chỉ phục vụ flatten a.x→ax/axx).
- **Fix/Correct Flow**: unique=(mbid,target_column); manual upsert ON CONFLICT(target_column) [cho a→b, a→c]; auto-populate `AND NOT EXISTS(mm.master_binding_id=? AND mm.mapping_v2_id=v2.id)` + giữ ON CONFLICT(target_column) safety. Verify red→green: insert cột-đổi-tên (v2 đã map) → sync → default KHÔNG tạo lại (count=0).
- **Phạm vi (≥3 dự án?)**: Có — mọi ETL/mapping/sync "N output từ 1 source", auto-populate idempotent, table dedup có cả khoá source lẫn output.
- **Tags**: #unique-constraint #idempotency #on-conflict #not-exists #data-model #etl-mapping #root-cause #sql
- **Nguồn**: lessons.md [2026-06-09]

### [2026-06-03] Regex kiểm tra identifier quá chặt làm rơi cột hợp lệ khi sinh DDL động
- **Global Pattern**: `[Regex filter R kiểm tra định danh X (tên bảng/cột) để sinh SQL/DDL động bị quá chặt — vd thiếu A-Z, thiếu ký tự được quote]` → `[các định danh hợp lệ (camelCase userId/createdAt) bị lọc bỏ THẦM LẶNG → bảng vật lý thiếu cột, rò rỉ/mất dữ liệu nghiệp vụ, schema drift trên storage]`. **Đúng**: (1) regex định danh SQL phải cover case-sensitive `^[a-zA-Z_][a-zA-Z0-9_]{0,62}$` (Postgres cho phép chữ hoa nếu được quote); (2) khi đổi validator schema, BẮT BUỘC cập nhật song song unit test liên quan để khớp thông báo lỗi động; (3) viết test case bao trùm cả định danh chữ hoa lẫn chữ thường.
- **Bối cảnh (Trigger)**: `MasterDDLGenerator` dùng regex `^[a-z_][a-z0-9_]{0,62}$` lọc tên cột; cột camelCase bị loại thầm lặng → DW thiếu cột; đồng thời unit test assert chuỗi lỗi cứng (`status or data_type required`) fail khi validator thêm điều kiện `is_sensitive_field`.
- **Root Cause**: Regex `[a-z_]` không chứa `[A-Z]` → camelCase bị coi không hợp lệ và drop thầm lặng; test assert chuỗi cứng trong khi thông báo lỗi đã đổi do thêm rule nghiệp vụ.
- **Fix/Correct Flow**: Mở rộng regex sang case-sensitive; đồng bộ test khi đổi validator; bổ sung test định danh chữ hoa.
- **Phạm vi (≥3 dự án?)**: Có — mọi nơi sinh SQL/DDL động từ metadata, filter identifier bằng regex (DW, ETL, code-gen, ORM migration).
- **Tags**: #ddl-generator #schema-drift #regex #sql-identifier #silent-drop #unit-test #go
- **Nguồn**: lessons.md [2026-06-03]

### [2026-05-29] Nhiều DDL bootstrap path phải delegate về 1 truth source — không duplicate spec
- **Global Pattern**: `[System A tạo resource X qua nhiều path P1..Pn, mỗi Pi giữ DDL spec riêng]` → `[mỗi Pi drift theo thời gian → "X ở chỗ này khác X ở chỗ kia" — bug upsert/insert vỡ khi event đến path runtime]`. **Đúng**: chọn 1 Pi là truth source; các Pi còn lại hoặc delegate sang Pi truth, hoặc tự kiểm trùng spec qua hằng số chia sẻ / contract test. Không bao giờ duplicate DDL literal ở nhiều entry-point.
- **Bối cảnh (Trigger)**: Admin bind shadow mới → DDL sinh từ path CMS hoặc NATS handler (lệch spec) → khi event đến sinkworker thì cột `_gpay_id` không tồn tại → upsert vỡ.
- **Root Cause**: ≥3 entrypoint tạo/normalize shadow table, mỗi nơi giữ DDL riêng; sinkworker (runtime truth) đã chuẩn hóa sang `_gpay_id` + partial UNIQUE INDEX nhưng 2 path còn lại vẫn tạo `id`/`source_id`.
- **Fix/Correct Flow**: Grep TẤT CẢ entrypoint (CREATE TABLE/ALTER TABLE shadow) trước khi patch; align path #2 và #3 về runtime spec; prefix `_` cho system column để tránh va chạm business field.
- **Phạm vi (≥3 dự án?)**: Có — mọi service có "bootstrap qua nhiều entrypoint" (user-onboarding init, migration runner, runtime auto-create, admin tool).
- **Tags**: #schema-migration #cdc #schema-drift #root-cause #dry #verification
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-29] Migration đổi UNIQUE INDEX phải grep và update mọi ON CONFLICT site đồng thời
- **Global Pattern**: `[Migration A drops+recreates UNIQUE INDEX trên table T với column set C2 (thay C1)]` → `[mọi INSERT/UPSERT site dùng ON CONFLICT (C1) fail runtime với SQLSTATE 42P10 — lỗi không xuất hiện ở build/sqlmock test]`. **Đúng**: TRƯỚC khi merge migration đổi UNIQUE INDEX → bắt buộc grep `ON CONFLICT.*<removed_col>` toàn repo + downstream service; mỗi site update đồng thời trong SAME PR; migration SQL ghi rõ "Caller sites to update: <file:line>".
- **Bối cảnh (Trigger)**: Migration 067 đổi ux_v2_mapping_rule_identity từ 3 cột sang 4 cột; 3 Go site vẫn dùng ON CONFLICT 3-cột cũ → runtime 500 SQLSTATE 42P10 khi register source object.
- **Root Cause**: DDL index spec đổi nhưng SQL code sites không được audit đồng thời; sqlmock không validate ON CONFLICT spec match index thật → lỗi chỉ xuất hiện ở production/integration.
- **Fix/Correct Flow**: Pre-migration audit bắt buộc; grep mọi site ON CONFLICT; update đồng thời; thêm CI gate parser SQL cross-check pg_indexes nếu có thể.
- **Phạm vi (≥3 dự án?)**: Có — multi-tenant thêm tenant_id vào unique, soft-delete đổi sang partial index, event-sourcing thêm version column.
- **Tags**: #schema-migration #schema-drift #cdc #verification #migration
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-29] Scope param phải được plumb dọc toàn bộ call chain với json tag nhất quán — thiếu 1 layer là silent NULL
- **Global Pattern**: `[Scope param P cross N layers (HTTP→Command→Wire→Worker→DB INSERT)]` — `[bất kỳ layer nào thiếu field P (struct, json tag, signature, model assign, dedup query)]` → `[downstream silently fallback sang NULL/zero — request không 5xx, DB row mồ côi, UI filter trả empty]`. **Đúng**: layer audit checklist bắt buộc khi thêm scope param mới; DB query phân biệt `P>0 → equality` vs `P=0 → IS NULL fallback` (không dùng `= 0` vì DB IS NULL ≠ `= 0`).
- **Bối cảnh (Trigger)**: FE gửi `?binding_id=11`; CMS ScanFieldsCommand thiếu ShadowBindingID → JSON marshal bỏ qua → NATS payload thiếu → worker payload = 0 → insert với shadow_binding_id=NULL; FE filter binding_id=11 trả 0 row.
- **Root Cause**: Feature thêm scope param nhưng không audit từng layer; wire JSON là loose schema (unknown fields ignored) → lỗi compile-time không bắt được.
- **Fix/Correct Flow**: Chạy layer audit checklist (HTTP parse → Command field → json tag → wire field → worker handler → function signatures → DB INSERT assign → DB dedup query → test stub); backwards-compat dùng omitempty cho legacy caller.
- **Phạm vi (≥3 dự án?)**: Có — multi-tenant tenant_id cross-tenant leak, A/B test experiment_arm metric attribution sai, idempotency-key double-execute.
- **Tags**: #cdc #schema-migration #silent-drop #process-governance #verification #testing
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-29] Partial UNIQUE INDEX đòi ON CONFLICT phải kèm WHERE predicate khớp — thiếu là SQLSTATE 42P10
- **Global Pattern**: `[Table T có PARTIAL UNIQUE INDEX (col) WHERE pred]` → `[INSERT ... ON CONFLICT (col) DO ... thiếu WHERE pred fail runtime SQLSTATE 42P10 — Postgres không infer partial index khi không có predicate rõ ràng]`. **Đúng**: SQL builder detect schema (có/không cột `_deleted`) để emit `ON CONFLICT (col) WHERE pred` hoặc `ON CONFLICT (col)` tương ứng; contract test phải có 2 case (schema có/không `_deleted`).
- **Bối cảnh (Trigger)**: Snapshot runner flush batch → BuildBatchUpsertSQLInSchema emit ON CONFLICT ("_source_id") không có WHERE; shadow table có partial unique WHERE NOT _deleted → Postgres reject toàn batch SQLSTATE 42P10; circuit breaker trip.
- **Root Cause**: SQL builder helper không phân biệt partial vs full unique index; build/test pass vì test fixture không tạo partial index.
- **Fix/Correct Flow**: `buildConflictTarget(schema, pkField, pkIdent)` kiểm tra schema có `_deleted` → emit `(col) WHERE NOT _deleted`; cross-table contract test 2 case; migration audit checklist mở rộng: khi tạo PARTIAL UNIQUE → grep ON CONFLICT toàn repo.
- **Phạm vi (≥3 dự án?)**: Có — soft-delete unique email, active subscription per user, versioned event store.
- **Tags**: #schema-migration #cdc #migration #testing #verification #schema-drift
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-29] Flag column vestigial nên sync qua DB trigger từ bảng V2 — không drop, không refactor caller
- **Global Pattern**: `[Flag column A.flag là vestigial display, B.flag aggregate là runtime truth]` — `[drop A.flag hoặc update A.flag từ app code khi update B]` → `[break legacy caller hoặc race + drift]`. **Đúng**: cài DB trigger AFTER INSERT|UPDATE|DELETE trên B → recompute A.flag = bool_or(B.predicate) qua bridge M; giữ A.flag với COMMENT "SYNCED FROM B, DO NOT UPDATE directly"; backfill DO block ở cuối migration để clear drift hiện có.
- **Bối cảnh (Trigger)**: Migration cũ tạo flag ở table V1 (legacy display); migration mới thêm table V2 drive runtime; caller V1 vẫn còn; flag V1 drift vì không ai update khi V2 thay đổi.
- **Root Cause**: Không có cơ chế tự động sync ngược từ V2 sang V1; app code update V1 từ nhiều path → race + drift; drop V1 phá legacy caller.
- **Fix/Correct Flow**: Trigger AFTER (không BEFORE) trên B với IS DISTINCT FROM guard (tránh recursive); OF flag_col (tránh fire khi update updated_at); backfill cuối migration; bootstrap guard COUNT V1 trước khi run.
- **Phạm vi (≥3 dự án?)**: Có — tenants.is_enabled ⇐ tenant_subscriptions, user.is_verified ⇐ multi-factor verification, cart.has_active_promo ⇐ cart_promo_application.
- **Tags**: #schema-migration #cdc #migration #architecture-design #dry
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-28] Cleanup "field rác" thực ra là RENAME/MERGE, không phải DELETE
- **Global Pattern**: `[Agent A nhận yêu cầu cleanup "field rác B" cùng concept với field X]` trong `[schema S]` → `[chọn nhánh DELETE thay vì RENAME → phá vỡ invariant gắn với B (UNIQUE INDEX, ON CONFLICT key), tăng scope từ mechanical rename lên remove + reconstruct gấp 3-5 lần]`. **Đúng**: Semantic mapping từng cặp field nghi ngờ trùng; nếu cùng (data_type, nullability, default, role) → RENAME target; `ALTER TABLE RENAME COLUMN B TO X` idempotent; code edit replace identifier không xóa logic/index/constraint.
- **Bối cảnh (Trigger)**: User nói "field B rác kỹ thuật, đã có field X rồi" về 2 field cùng semantic trong schema. Agent hiểu "rác" = "remove", chọn DELETE + xóa logic → phá vỡ ràng buộc UNIQUE INDEX và ON CONFLICT key.
- **Root Cause**: Agent không verify intent — "rác kỹ thuật" với "đã có X rồi" = intent RENAME/MERGE, không phải DELETE. Over-defer thành "3 option remove" trong khi intent là 1 patch rename đơn giản.
- **Fix/Correct Flow**: Semantic mapping cặp field: cùng concept → RENAME target. `ALTER TABLE RENAME COLUMN B TO X` idempotent (skip nếu X đã tồn tại → DROP duplicate). Code: replace identifier, không xóa logic. Migration SQL idempotent + reversible với reverse.sql. Verify: `\d <table>` + grep zero-residue.
- **Phạm vi (≥3 dự án?)**: Có — cleanup PK/identifier duplicate cross-service, API field rename V1→V2, config key consolidation, log field naming standardization, ML feature store column dedup.
- **Tags**: #schema-migration #migration #root-cause #verification #process-governance #dry
- **Nguồn**: lessons.md [2026-05-28]

### [2026-05-28] Blind rename B→X trên schema đa-path tạo duplicate khi site đã có cả hai field
- **Global Pattern**: `[Agent A thực hiện blanket rename B→X]` trên `[schema S đa-path (multiple CREATE/SELECT/INSERT sites)]` mà `[không audit từng site]` → `[Go struct duplicate entry, SQL CREATE TABLE lỗi 42701, record map double-set, ON CONFLICT key ambiguity]`. **Đúng**: Build inventory pre-rename; phân loại từng site: BOTH_PRESENT → DROP B, ONLY_LEGACY → RENAME B→X; grep count zero residue B + zero duplicate X; test SQL execution (không chỉ build).
- **Bối cảnh (Trigger)**: Sau lesson cleanup-is-not-remove (rename thay remove), Agent áp dụng thành pure RENAME blanket, nhưng một số site đã có field X riêng → tạo duplicate entry trong Go struct, SQL CREATE TABLE lỗi runtime 42701, record map double-set.
- **Root Cause**: Áp dụng lesson "cleanup = rename không phải remove" quá tuyệt đối, không phân loại per-site. Đúng action là mixture: RENAME (target chưa có) + REMOVE (target đã có), không phải pure RENAME.
- **Fix/Correct Flow**: Inventory pre-rename tất cả sites. Phân loại BOTH_PRESENT (DROP B) vs ONLY_LEGACY (RENAME B→X). Audit dual-presence dấu hiệu. Apply edit theo case-specific verb. Post-edit grep: zero residue B + zero duplicate X trong cùng site. Test SQL execution (runtime, không chỉ build).
- **Phạm vi (≥3 dự án?)**: Có — DB column dedup multi-path (CDC↔master↔DW), API field versioning multi-endpoint, config key consolidation multi-service, ML feature store column unification.
- **Tags**: #schema-migration #migration #root-cause #verification #testing #dry
- **Nguồn**: lessons.md [2026-05-28]

### [2026-05-21] JOIN query không cập nhật khi identity-tier của entity widened bởi migration
- **Global Pattern**: `[Query Q JOIN entity X với entity Y trên key K_old; Y's identity widened thành K_new = K_old + discriminator C qua migration M]` → `[Q over-match vì K_old là non-unique prefix của K_new; row multiplication N → N×k]`. **Đúng**: khi migrate identity-tier, viết check-list "callers cần update" trong migration comment + post-migration audit script (grep JOIN on Y theo K_old); JOIN phải theo TẤT CẢ cột UNIQUE constraint hiện tại; smoke test: `COUNT(*) FROM Q` vs `COUNT(DISTINCT X.id) FROM Q` — nếu lệch → JOIN dư duplicate.
- **Bối cảnh (Trigger)**: `GET /api/v1/source-objects` trả 6 row khi chỉ có 4 source_objects thật; id=1 và id=36 mỗi cái xuất hiện 2 lần với `registry_id` khác nhau; migration 054-056 thêm `source_connection_id` vào identity của `cdc_table_registry` nhưng JOIN ở listing code không được update.
- **Root Cause**: JOIN bằng K_old `(source_db, source_table, target_table)` "vì query đã tồn tại từ trước"; schema đã evolve thêm `source_connection_id` nhưng query không theo; cross-connection bleed.
- **Fix/Correct Flow**: Convert `LEFT JOIN` → `LEFT JOIN LATERAL (SELECT ... WHERE ... AND (Y.C = X.C OR Y.C IS NULL) ORDER BY (Y.C IS NULL) ASC, Y.id ASC LIMIT 1) Y ON TRUE`; bảo toàn wire shape; cấm patch bằng DISTINCT/GROUP BY không scope theo C — mask triệu chứng, pick non-deterministic row.
- **Phạm vi (≥3 dự án?)**: Có — Multi-tenant SaaS (JOIN quên tenant_id), Multi-region replication (JOIN quên region_id), Multi-env shared metadata (JOIN quên env_id).
- **Tags**: #schema-migration #sql-join-cardinality #identity-tier-drift #multi-tenant #migration-callers-update #lateral-limit-1
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-20] ON CONFLICT target không nhất quán giữa multi-path INSERT cho cùng table có 2 UNIQUE constraint
- **Global Pattern**: `[Table A có 2 UNIQUE constraint X, Y; path P1 INSERT ON CONFLICT X, path P2 INSERT ON CONFLICT Y]` → `[khi input gây collision trên X nhưng không Y (do derive function asymmetric), P2 raise 23505 duplicate key error]`. **Đúng**: tất cả path dùng cùng ON CONFLICT target (identity-of-record); derived keys nằm trong DO UPDATE SET để refresh khi conflict.
- **Bối cảnh (Trigger)**: `ERROR: duplicate key value violates unique constraint "source_object_registry_object_code_key"` khi register table qua CMS; 2 UNIQUE constraint `(object_code)` và `(normalized_source_key)`; 2 trong 4 INSERT path dùng `ON CONFLICT (normalized_source_key)` inconsistent với 2 path còn lại.
- **Root Cause**: `object_code` build bằng `slugify` (collapse `[^a-z0-9]` → `_`) mất phân biệt; `normalized_source_key` giữ separator gốc; hai input khác nhau cho cùng `object_code` nhưng `normalized_source_key` khác → `ON CONFLICT (normalized_source_key)` không catch → INSERT vi phạm `object_code` UNIQUE.
- **Fix/Correct Flow**: Chọn `object_code` làm identity-of-record thống nhất; tất cả path dùng `ON CONFLICT (object_code) DO UPDATE SET normalized_source_key = EXCLUDED.normalized_source_key, ...`; cấm "path A dùng key X, path B dùng key Y" cho cùng table.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ table nào có multiple UNIQUE constraints với multi-path upsert (user registry, product catalog, multi-tenant config table).
- **Tags**: #schema-migration #postgres #on-conflict #unique-constraint #multi-path-insert #identity-of-record
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] List endpoint dùng LATERAL LIMIT 1 ẩn child rows trong quan hệ 1:N
- **Global Pattern**: `[List endpoint cho relationship A 1:N B] dùng LATERAL LIMIT 1 trên B` → `[UI mất child rows; 1 source có 2 binding nhưng chỉ hiển thị 1]`. **Đúng**: quyết định semantics rõ ràng — (a) parent-centric với array_agg, hoặc (b) child-centric cross-product LEFT JOIN; không dùng LATERAL LIMIT 1 để collapse 1:N silently; ORDER BY phải include child key để stable ordering.
- **Bối cảnh (Trigger)**: `shadow_binding` có 2 row cho source_object_id=1 nhưng `/api/v1/source-objects` chỉ hiện 1 row; `listBaseFromWhere` dùng `LEFT JOIN LATERAL (... LIMIT 1) sb` → collapse 1:N thành 1:1.
- **Root Cause**: Pattern LATERAL LIMIT 1 dùng để dedupe N-side trong listing; khi business semantics yêu cầu "mỗi N entity là 1 visible row" (Shadow Object), pattern này ẩn data — đây là information loss.
- **Fix/Correct Flow**: Bỏ LATERAL LIMIT 1, thay bằng LEFT JOIN trực tiếp `shadow_binding`; giữ LEFT JOIN (không INNER) để parent without children vẫn surface 1 row (sb.* NULL → COALESCE fallback); ORDER BY include child key.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ list endpoint nào cho 1:N (SKU-variant, order-line-items, workflow-attempts).
- **Tags**: #sql #lateral #list-endpoint #one-to-many #ui-data-loss #schema-migration
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-19] V2 model đã 1→N nhưng V1 legacy UNIQUE 2-cột chặn multi-target — fix tại schema-only
- **Global Pattern**: `[Hệ thống X có model V1 (legacy mirror) + V2 (authoritative); V2 hỗ trợ 1→N relation; V1 vẫn giữ UNIQUE 2-cột restriction từ buổi đầu]` lên `[shared schema với V1/V2 coexistence]` → `[requirement 1→N rơi vào V1 INSERT path bị chặn bởi UNIQUE constraint cũ; Go code đã 1→N tolerant nhưng schema chưa]`. **Đúng**: Fix tại schema-only — DROP V1 constraint cũ + ADD constraint mới có thêm target/binding-discriminator field; KHÔNG đổi V2 schema (đã đúng); KHÔNG đổi Go code (write paths đã idempotent qua ON CONFLICT); audit tất cả write/read sites + downstream caches trước khi migrate.
- **Bối cảnh (Trigger)**: User không thể register cùng source vào target thứ 2 — V1 INSERT path bị chặn bởi UNIQUE `(source_db, source_table)` từ migration 001; V2 đã đúng với 1→N nhưng V1 legacy bridge vẫn cần cho worker chưa migrate.
- **Root Cause**: V1 UNIQUE constraint được thiết kế từ đầu với assumption 1:1; khi requirement mở rộng sang 1:N, V2 được cập nhật nhưng V1 legacy bridge không được update tương ứng.
- **Fix/Correct Flow**: Tạo migration sequenced với BEGIN/COMMIT; DROP V1 constraint cũ + ADD UNIQUE mới bao gồm discriminator field; audit V1 write paths (register, bulk register, update, bootstrap mirror); audit V2 sync ON CONFLICT; audit worker/reader caches (first-wins tolerant hay keyed-by-discriminator).
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có V1/V2 coexistence với legacy bridge pattern và expanding relation cardinality.
- **Tags**: #schema-drift #migration #migration-hygiene #cdc
- **Nguồn**: lessons.md [2026-05-19]

### [2026-05-15] Copy seed values từ legacy migration không audit các DROP/RENAME schema sau đó
- **Global Pattern**: `[Agent A] extracts data D từ legacy file F1 sang new file F2 mà không diff D với các DROP/RENAME statement ở F3, F4...` lên `[migration seed files]` → `[F2 chứa references đến objects không còn tồn tại, gây drift bug tại runtime]`. **Đúng**: Trước khi copy INSERT từ Fn, chạy grep toàn bộ migration files xem có DROP/RENAME sau Fn không; nếu có → rewrite values, không carry forward.
- **Bối cảnh (Trigger)**: Khi tách INSERT seed từ legacy migration, agent copy raw values bao gồm `default_schema='cdc_internal'`. Nhưng migration 038 đã DROP SCHEMA IF EXISTS cdc_internal CASCADE — mọi row reference schema đó là drift bug.
- **Root Cause**: Copy-paste seed values mà không cross-reference state cuối cùng của schema sau toàn bộ chuỗi migration; code-review chỉ nhìn file gốc, không nhìn file kế tiếp.
- **Fix/Correct Flow**: Trước khi copy INSERT từ Fn, chạy `grep -n "<schema>" migrations/**/*.sql | sort`; nếu có DROP/RENAME sau Fn → rewrite values; document rewrite trong file header; test: query DB local xem object còn tồn tại không.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có chuỗi migration với rename/drop operations (Flyway, Liquibase, Atlas).
- **Tags**: #schema-drift #migration #seed-leak #migration-hygiene
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-15] Refactor migration chỉ tách seed, không squash ALTER ADD COLUMN vào CREATE TABLE gốc
- **Global Pattern**: `[Agent A] refactors migration set M chỉ tổ chức file boundaries mà không squash ALTER ADD COLUMN/ADD CONSTRAINT từ descendant Mn vào base table M0` lên `[migration files]` → `[file count giảm nhưng ADD COLUMN rải rác, production fresh vẫn phải apply CREATE-then-ALTER cycles]`. **Đúng**: Mọi "ADD COLUMN IF NOT EXISTS" trong descendant Mn là forcing function để consolidate column vào CREATE TABLE trong M0; xóa Mn sau khi merge; tracker rows trên DB cũ skip-by-version.
- **Bối cảnh (Trigger)**: User giao refactor migrations, agent tách INSERT seed ra folder riêng nhưng để nguyên hàng chục `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` trong files 013/020 — user feedback: "refactor mà cứ cà nhây cà nhây".
- **Root Cause**: Agent hiểu refactor là tổ chức file/folder nhưng không nhận ra pattern `ADD COLUMN IF NOT EXISTS` là technical debt indicator cần SQUASH vào file base.
- **Fix/Correct Flow**: Sau khi tổ chức folder, chạy grep liệt kê hết `ADD COLUMN IF NOT EXISTS`; merge từng cặp (ALTER descendant Mn, base table M0) vào CREATE TABLE; verify build + apply trên DB fresh.
- **Phạm vi (≥3 dự án?)**: Có — universal cho mọi project có ORM migration với schema accretion lịch sử.
- **Tags**: #migration #migration-hygiene #schema-drift #dry
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-15] CREATE TABLE không schema prefix rơi vào public thay vì target schema
- **Global Pattern**: `[Agent A] viết CREATE TABLE statement S trong migration M không có schema-qualified prefix, và verify M trên DB D đã có tracker entry cho version cũ của M` lên `[PostgreSQL migration runner]` → `[(1) S tạo table trong schema sai (public), (2) cleanup migrations sau fail, (3) verification trên D xanh vì tracker skip M]`. **Đúng**: Luôn schema-qualify mọi CREATE TABLE/INDEX/FUNCTION; verify migration changes bằng cách replay trên fresh DB, không restart service trên DB đã apply tracker.
- **Bối cảnh (Trigger)**: Sau khi squash migration, agent viết `CREATE TABLE IF NOT EXISTS enum_types (...)` không có schema prefix. Runner normalize search_path về public → table rơi vào public.enum_types. Migration cleanup sau fail với invariant "public schema not empty".
- **Root Cause**: PostgreSQL CREATE TABLE không schema-qualified → resolve qua search_path first match (public trong runtime runner). Verification bằng service restart không phát hiện bug vì tracker skip file đã apply trên DB cũ.
- **Fix/Correct Flow**: Mọi DDL trong migration phải schema-qualified (`CREATE TABLE cdc_system.xxx`); khi sửa base migration, verify bằng drop + replay full fresh DB, không chỉ restart service.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project PostgreSQL migration có cleanup/finalize step assert namespace invariant.
- **Tags**: #schema-drift #migration #postgres #search-path #verification
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-15] Squash migration bằng grep hẹp bỏ sót ADD COLUMN từ file legacy khác nhóm
- **Global Pattern**: `[Agent A] consolidates schema accretion từ subset Sn của legacy migrations bằng grep hẹp keyword K, mà không grep toàn bộ "ALTER TABLE <target>" trên ALL legacy files hoặc diff Go-model fields vs post-squash CREATE TABLE` lên `[ORM model + migration files]` → `[drift ngầm — model fields không có column counterpart, INSERT/UPDATE handler fail tại runtime SQLSTATE 42703]`. **Đúng**: Treat squash như closure problem — list ALL columns từ model/struct + ALL columns từ EVERY legacy migration, verify post-squash CREATE TABLE là superset của cả hai.
- **Bối cảnh (Trigger)**: Sau khi grep "ADD COLUMN IF NOT EXISTS" và squash 2 files, agent bỏ sót file partitioning.sql chứa 2 ALTER ADD COLUMN cho cột `is_partitioned` + `partition_key` — POST register → INSERT → ERROR: column "is_partitioned" does not exist (SQLSTATE 42703).
- **Root Cause**: Squash chỉ dựa vào grep một keyword; bỏ qua ADD COLUMN không có guard và bỏ qua ALTER TABLE từ file legacy thuộc nhóm khác (cross-cutting concern); không diff Go-model fields vs DB column list.
- **Fix/Correct Flow**: Build column inventory từ Go side (grep gorm column tags); build từ migration side (grep ALTER TABLE target trên ALL legacy files); squash CREATE TABLE PHẢI là superset; post-squash `\d table` trên DB fresh xác nhận.
- **Phạm vi (≥3 dự án?)**: Có — universal cho Go/Python/Ruby dùng ORM với schema migration tách rời source code model.
- **Tags**: #migration #schema-drift #gorm #migration-hygiene #testing
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-11] GORM TableName không nhất quán kết hợp role search_path gây mất bảng
- **Global Pattern**: `[ORM model X] khai báo TableName() mixed-qualified (một số có schema prefix, một số bare)` lên `[shared PostgreSQL schema]` → `[bare names bị resolve sai khi role search_path reset, gây relation does not exist tại runtime]`. **Đúng**: enforce một convention duy nhất — hoặc all schema-qualified (schema.table), hoặc all-bare + inject search_path cố định vào DSN session level; không đặt search_path ở role level; migrations luôn schema-qualify rõ ràng.
- **Bối cảnh (Trigger)**: Sau khi reset role-level search_path cho DB role, service log liên tục báo lỗi `relation "X" does not exist` cho các bảng bare-name trong GORM model, dù bảng thực tế tồn tại trong schema đúng.
- **Root Cause**: Hai vấn đề đồng thời: (1) `ALTER ROLE X SET search_path=A,B` persists qua schema DROP, khi A bị drop thì query rơi vào ghost schema; (2) ORM model khai báo TableName() không nhất quán — một số có schema prefix, một số bare, nên khi search_path thay đổi, bare names rơi vào schema mặc định (public).
- **Fix/Correct Flow**: Inject `search_path=cdc_system,public` vào DSN ở session level (không phải role level); đồng nhất toàn bộ TableName() về one convention; migrations phải luôn schema-qualify object names.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project PostgreSQL multi-schema dùng ORM (GORM, SQLAlchemy, ActiveRecord) có cấu hình role search_path.
- **Tags**: #schema-drift #migration #gorm #search-path #orm #postgres
- **Nguồn**: lessons.md [2026-05-11]

### [2026-05-11] Migration production chứa demo seed data, downstream migration fan-out ra toàn bộ registry
- **Global Pattern**: `[Migration A] SEEDS hardcoded dataset X vào table B` lên `[Production DB]` → `[Downstream migration C derives data từ B sang table D; production cold-boot tự chứa demo data mà ops không kiểm soát]`. **Đúng**: Schema migrations chỉ chứa DDL + immutable config (enum domain, worker schedules); mọi dữ liệu nghiệp vụ/pilot/sample tách ra scripts/seed_dev.sql hoặc env-gated; audit pre-merge: "migration này có INSERT row dữ liệu không?".
- **Bối cảnh (Trigger)**: Sau cold-boot service production, DB chứa hàng chục row demo (pilot connections, test sources) mà không ai INSERT thủ công — chúng đến từ hardcoded INSERT trong migration files kết hợp downstream backfill fan-out.
- **Root Cause**: Migration 001 hardcode `INSERT INTO ... VALUES (demo rows) ON CONFLICT DO NOTHING`; downstream migration 035 fan-out toàn bộ rows đó vào registry V2; idempotent guard `WHERE NOT EXISTS` chỉ chống duplicate, không chống demo-leak trên production fresh DB.
- **Fix/Correct Flow**: Tách demo/pilot data ra `scripts/seed_dev.sql`; schema migration chỉ chứa DDL + config-like seed; thêm CI gate grep "INSERT INTO" trong migrations và flag ngoài whitelist config-only.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có migration runner + seed data (Flyway, Liquibase, Atlas, sqlx-migrate).
- **Tags**: #migration #seed-leak #production-safety #migration-hygiene
- **Nguồn**: lessons.md [2026-05-11]

### [2026-05-11] Audit consumer usage trước khi thêm hoặc rename table trong migration
- **Global Pattern**: `[Migration A] tạo/rename table B vào schema S` lên `[shared DB schema]` → `[Nếu không có GREP match cho B trong consumer code services, B là dead schema bloat — ops phải bảo trì, backup, vacuum vô ích]`. **Đúng**: Pre-merge gate cho mọi migration PR: grep tên table mới phải có ≥1 match trong ≥1 service; migration RENAME: tên cũ phải =0 match, tên mới ≥1 match.
- **Bối cảnh (Trigger)**: Audit phát hiện 2 unused tables (`table_registry_legacy`, `master_table_registry_legacy`) tồn tại trong production schema sau migration rename, không có Go reference nào — pure dead schema.
- **Root Cause**: Không có quy trình pre-merge kiểm tra xem table mới/renamed có được code consumer sử dụng không; migration được merge dựa trên logic DDL đúng nhưng không verified về consumer-side usage.
- **Fix/Correct Flow**: Trước mỗi migration PR, chạy grep tên table trong tất cả service repos; ghi rõ trong PR description "Table X used by service Y at path:line"; migration RENAME phải clean 100% references cũ.
- **Phạm vi (≥3 dự án?)**: Có — universal cho microservices + shared-DB, CMS + worker, API + worker pool, monolith + sidecar.
- **Tags**: #migration-hygiene #dead-schema #schema-drift #migration
- **Nguồn**: lessons.md [2026-05-11]

### [2026-04-29] Hai chủng model↔DB schema drift: migration sai schema target và model thêm field quên migration
- **Global Pattern**: `[Migration script A ALTER TABLE ở schema X1]` nhưng `[same-name table tồn tại ở X2 (do migration parallel)]` → `[X2.table thiếu column, runtime SQLSTATE 42703 Y]`; HOẶC `[Model struct B thêm field mới với tag]` mà `[không có migration kèm]` → `[time-bomb chờ Find(&FullStruct) hoặc autoMigrate fail Y]`. **Đúng**: mọi PR thêm field model = kèm migration ADD COLUMN IF NOT EXISTS; migration ALTER TABLE dùng pg_namespace lookup, không hardcode schema; boot-time guard validate column tags vs information_schema.
- **Bối cảnh (Trigger)**: Bug 42703 `column "next_retry_at" does not exist` trong dlq_state_machine.poll → sweep 15 model files vs information_schema lộ thêm 6 cột drift trên 2 bảng khác (2 loại drift khác cơ chế).
- **Root Cause**: Drift loại 1: migration hardcode schema X1 nhưng code query X2; drift loại 2: model thêm field không kèm migration vì assume autoMigrate tự lo (production thường tắt autoMigrate).
- **Fix/Correct Flow**: Mọi PR thêm field model kèm migration ADD COLUMN IF NOT EXISTS (loại 2); migration ALTER dùng pg_namespace/pg_class lookup không hardcode (loại 1); CI lint parse ORM tags diff với DB schema dump → block merge nếu drift.
- **Phạm vi (≥3 dự án?)**: Có — GORM/SQLAlchemy/TypeORM/Hibernate, multi-tenancy schema-per-tenant, brownfield codebase, sharded DB DDL fan-out.
- **Tags**: #schema-migration #schema-drift #migration #gorm #root-cause #model-drift #pgx
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-29] Migration draft dùng column names từ memory — không match schema thực tế trên DB
- **Global Pattern**: `[Developer A viết migration INSERT dựa trên draft/requirements doc B]` mà `[schema target X đã evolved qua migration sau đó rename/drop columns]` → `[ERROR column "X" does not exist Y]`. **Đúng**: trước khi viết INSERT vào bảng đã tồn tại, CHẠY `\d <schema>.<table>` trên DB thực tế của môi trường target; không dựa vào requirements.md hoặc memory về schema trước đây.
- **Bối cảnh (Trigger)**: Migration 049 dùng columns `description, config_json, is_active` cho `cdc_system.connection_registry`; apply trả ERROR vì schema thực tế là `display_name, role_type, secret_ref, options_json, status` (đã được migration sau đó rename).
- **Root Cause**: Developer copy-paste shape từ migration cũ hơn 6 tháng và assume vẫn đúng; schema evolved qua nhiều migration nhưng requirements doc không được update tương ứng.
- **Fix/Correct Flow**: `docker exec <db> psql -c "\d <schema>.<table>"` trên DB thực tế trước khi viết INSERT; không dựa vào 01_requirements.md hoặc memory về schema cũ.
- **Phạm vi (≥3 dự án?)**: Có — mọi project có nhiều migration evolved over time (Rails, Django, Flyway, sqlc, GORM).
- **Tags**: #schema-migration #schema-drift #migration #root-cause #model-drift #verification
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-28] Schema rename không kèm search_path update — 42P01 hàng loạt
- **Global Pattern**: `[Migration owner A move tables sang schema B]` mà `[ORM/raw SQL X tồn tại không qualify schema]` → `[runtime 42P01 "relation does not exist" hàng loạt Y]`. **Đúng**: PR migration bắt buộc kèm `ALTER ROLE SET search_path = B, public` (hoặc audit qualify toàn bộ ORM queries); không tách 2 thay đổi này thành 2 phase rời.
- **Bối cảnh (Trigger)**: Migration 037/038 di tản tables `cdc_*` từ schema `public` sang `cdc_system`; GORM `TableName()` chỉ trả tên thuần, raw SQL không qualify schema → fall back vào search_path mặc định `("$user", public)` → 11 endpoint CMS đồng loạt 500.
- **Root Cause**: search_path là per-role/per-session; migration chỉ move tables nhưng không update search_path; GORM và raw SQL không qualify schema name.
- **Fix/Correct Flow**: Kèm `ALTER ROLE <role> SET search_path = cdc_system, public` trong cùng PR với migration move schema; hoặc audit và qualify toàn bộ ORM/raw SQL; restart pool/process để session-level setting có hiệu lực.
- **Phạm vi (≥3 dự án?)**: Có — Postgres + GORM, Sequelize, SQLAlchemy core query, JDBC raw; bất kỳ project move tables across schemas.
- **Tags**: #schema-migration #schema-drift #search-path #gorm #migration #root-cause
- **Nguồn**: lessons.md [2026-04-28]

### [2026-04-21] PostgreSQL ON CONFLICT WHERE chỉ apply UPDATE path — INSERT path không bị guard
- **Global Pattern**: `[Developer A dùng SQL clause C (ON CONFLICT WHERE) làm safety guard G]` lên `[database operation X (INSERT+UPDATE)]` → `[C không cover G hoàn toàn vì C chỉ áp dụng cho UPDATE sub-action Y]`. **Đúng**: dùng BEFORE INSERT OR UPDATE trigger hoặc RLS WITH CHECK để guard cả 2 path; verify exact scope của mọi SQL clause trước khi cite làm safety mechanism.
- **Bối cảnh (Trigger)**: Brain đề xuất `INSERT ... ON CONFLICT DO UPDATE SET ... WHERE EXISTS (SELECT 1 FROM worker_registry WHERE fencing_token=$N)` để guard Zombie Pod; user phát hiện INSERT path (no conflict) escape guard hoàn toàn.
- **Root Cause**: Brain biết syntax nhưng không verify exact semantic của WHERE scope trong ON CONFLICT; assumption "WHERE guards whole statement" sai — WHERE chỉ guard DO UPDATE sub-action.
- **Fix/Correct Flow**: Dùng BEFORE INSERT OR UPDATE trigger với RAISE EXCEPTION để rollback entire transaction; hoặc RLS policy WITH CHECK; set fencing token qua SET LOCAL + current_setting() trong trigger để validate cross-table.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ PostgreSQL project dùng upsert pattern với safety guard (distributed locking, idempotency, optimistic concurrency).
- **Tags**: #schema-migration #root-cause #postgres-on-conflict-scope #fencing-enforcement #sql-clause-scope-verification #before-trigger
- **Nguồn**: lessons.md [2026-04-21]

### [2026-04-21] PostgreSQL RETURNS TABLE OUT param trùng tên column trong body — SQLSTATE 42702 runtime
- **Global Pattern**: `[Developer A tạo function F với RETURNS TABLE (col_name T)]` và `[body references table.col_name]` → `[SQLSTATE 42702 ambiguous column error runtime, CREATE FUNCTION vẫn success Y]`. **Đúng**: đặt prefix `out_` cho OUT params; dùng table alias trong body (wr.machine_id); DROP FUNCTION IF EXISTS CASCADE trước CREATE OR REPLACE khi signature đổi; test call runtime trước commit.
- **Bối cảnh (Trigger)**: Function `claim_machine_id RETURNS TABLE(machine_id INT, ...)` body dùng `WHERE machine_id = (...)` trên `worker_registry` → SQLSTATE 42702 lúc runtime vì resolver ambiguous giữa OUT param và physical column.
- **Root Cause**: PostgreSQL function body resolves identifiers by name; RETURNS TABLE OUT params introduce column-like names vào function scope; trùng tên với physical table column → resolver ambiguous; syntactic check lúc CREATE không phát hiện.
- **Fix/Correct Flow**: Prefix OUT params với `out_` (out_machine_id); dùng table alias đầy đủ trong body (wr.machine_id); luôn test SELECT * FROM func() để validate runtime trước commit migration.
- **Phạm vi (≥3 dự án?)**: Có — mọi PostgreSQL project có stored functions/procedures với RETURNS TABLE (GORM migration, raw SQL, Flyway, Liquibase).
- **Tags**: #schema-migration #postgres-function-scope #ambiguous-column #sqlstate-42702 #returns-table-out #create-or-replace-signature
- **Nguồn**: lessons.md [2026-04-21]

### [2026-04-20] Partitioned table SLOW SQL — index phải ở parent level, không per-partition runtime
- **Global Pattern**: `[Table B partitioned N partitions thiếu parent-level index trên column sort/filter C]` → `[cross-partition query buộc Seq Scan từng partition → SLOW SQL O(N×P)]`. **Đúng**: `CREATE INDEX IF NOT EXISTS` tại parent level → PG auto-propagate xuống existing + future partitions. Mọi index runtime PHẢI có file migration để tránh time bomb trên fresh deploy.
- **Bối cảnh (Trigger)**: SLOW SQL 306-440ms trên `SELECT COUNT(*) FROM failed_sync_logs` + `ORDER BY started_at DESC LIMIT 10 FROM cdc_activity_log`. Cả 2 bảng đã partitioned (migration 010). Root cause: parent partitioned table thiếu index.
- **Root Cause**: PG 11+ partitioned tables yêu cầu index ở parent level để auto-propagate. Indexes có thể tạo per-partition runtime → lost trên fresh deploy; không bootstrap cho partition mới.
- **Fix/Correct Flow**: Parent-level `CREATE INDEX`: PG auto-propagate xuống children + future. Migration persist: mọi index runtime PHẢI có file migration. Verify EXPLAIN plan: phải show `Index Scan`, không `Seq Scan`. Partition aware DDL: ADD COLUMN hoặc INDEX → dùng parent level.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi PostgreSQL project dùng table partitioning (range, list, hash).
- **Tags**: #schema-migration #partitioned-tables #slow-sql #index-propagation #parent-index #postgresql
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-20] Partitioned Table Default Partition Orphan — phải Backfill, không chỉ Forward retention
- **Global Pattern**: `[Partitioned table B có default partition C chứa orphan rows D]` → `[planner không thể prune C vì runtime pruning không áp dụng cho default partition]` → `[mọi query trên B phải scan C → planning time tăng tuyến tính]`. **Đúng**: automation quản lý partition cần 2 chiều: Forward (pre-create future partitions) + Backward (detect rows đã land vào default → drain-before-create child partitions → move rows).
- **Bối cảnh (Trigger)**: SLOW SQL 236ms regression trên query đã bounded với `WHERE X > NOW() - INTERVAL`. Planner vẫn không prune vì `*_default` chứa rows trong window. PG runtime pruning không áp dụng cho default partition.
- **Root Cause**: Default partition được coi là "fallback empty" nhưng thực ra là partition bình thường. Runtime pruning không áp dụng cho default — planner không có positive range để so sánh, chỉ synthesized NOT-IN của siblings → default luôn hiện trong Append nếu có row.
- **Fix/Correct Flow**: Backward management: detect rows đã land vào default → drain-before-create: (a) `DELETE ... RETURNING * INTO TEMP`, (b) `CREATE TABLE ... PARTITION OF ...`, (c) `INSERT INTO parent SELECT * FROM temp`. DROP default chỉ khi hoàn toàn trống.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho pg_partman deployments, Debezium CDC tables range-partition, audit/log tables với late-arriving data.
- **Tags**: #schema-migration #postgres #partitioning #default-partition #backfill #slow-sql #planning-time
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-20] Plan ops "aggressive" nhưng thiếu operational reality — locking math, zero-downtime tools
- **Global Pattern**: `[Agent A writes refactor plan P dùng vocabulary ops-sounding V]` + `[A thiếu operational experience E cho scale S (>100K rows, dynamic infra, complex schema)]` → `[P fails catastrophically tại execution time: table lock >30min, type inference impossible, worker ID collision, JSONB transform in migration transaction]`. **Đúng**: Real ops plans có: explicit lock duration calc, rollback within 30s, dual-read/dual-write transition, zero-downtime tools referenced, batch sizes tuned to rowcount.
- **Bối cảnh (Trigger)**: Brain rewrite v2 "reconstruction aggressive" — User phê phán 5 mistakes nặng hơn: "auto-detect business columns" từ JSONB = hallucination, "single identity authority" vẫn dual-source, "aggressive cutover" = CREATE+INSERT SELECT+DROP trong 1 transaction trên 10M+ rows → Postgres LOCK 30+ phút, Worker ID "bằng grep log" = fragile, JSONB strip trong migration = CPU-expensive.
- **Root Cause**: Brain generate plan theoretically correct + dùng từ ops-sounding nhưng thiếu operational experience primitives: large-table locking math, online schema change tools, zero-downtime patterns, K8s dynamic IP reality, type inference fundamental impossibility.
- **Fix/Correct Flow**: Never single-transaction millions-row migration (dùng pg_repack, logical replication-based swap, staged batch COPY lock_timeout=5s). Worker ID: Redis SETNX với TTL heartbeat hoặc PG SKIP LOCKED. Strip at Worker not DB. Lock duration calculation upfront: >5s = require OSC tool. Zero-downtime tools: pg_repack, pg_logical, pt-online-schema-change.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi production DB migration, large-scale schema change, distributed system ops.
- **Tags**: #schema-migration #ops-reality #locking-math #zero-downtime #jsonb #worker-id-registry #plan-vocabulary
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-15] Insert vào DB column mà không check column type và name casing
- **Global Pattern**: `[Worker A insert data vào table T]` → `[A không check column types và name casing của T trước khi INSERT]` → `[type mismatch errors + column not found tại runtime]`. **Đúng**: trước INSERT, query `information_schema.columns` để biết column types + exact names; quote TẤT CẢ column names; JSONB columns → `json.Marshal(value)` trước khi gửi.
- **Bối cảnh (Trigger)**: CDC Worker INSERT vào Postgres table do Airbyte tạo. Airbyte lưu `fileUrl` dạng JSONB, `params` dạng JSONB. Worker gửi plain string → Postgres reject. Column names camelCase bị lowercase thành `jobid` → column not found.
- **Root Cause**: Worker upsert code không check target table schema trước khi INSERT. Giả sử tất cả columns là TEXT/VARCHAR. Không quote column names.
- **Fix/Correct Flow**: Trước INSERT: query `information_schema.columns` → biết column types + exact names. Quote TẤT CẢ column names. JSONB columns → marshal JSON. Cache column types per table.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dynamic upsert worker, CDC pipelines, ETL inserts vào tables tạo bởi 3rd-party tools.
- **Tags**: #schema-migration #postgres #type-mismatch #quoting #cdc #dynamic-insert
- **Nguồn**: lessons.md [2026-04-15]

---

## 4. CDC / Data Pipeline — Kafka, Debezium, Snapshot, Connection-Registry, Masking

_Bài học miền CDC/ETL: Kafka/Debezium, snapshot, connection-registry, masking, DLQ, reconcile, shadow tables._ — **35 pattern**

### [2026-06-10] Module M viết cho topology cũ chết im lặng khi data plane đổi topology — gate introspect nil → skip 100% + "success 0"
- **Global Pattern**: `[Module M (recon/sync/audit) hardcode topology T1 (single-DB, schema mặc định S) ở 3 tầng: registry-entry không mang schema, gate introspect GetSchema(S), query không schema-qualify]` + `[data plane migrate sang T2 (DB/schema riêng)]` → `[gate trả nil cho MỌI bảng → skip 100% → M báo "success items=0" mãi mãi; bảng state/report của M = 0 rows mà không ai biết M chết]`. **Đúng**: (1) registry entry phải mang đủ định vị vật lý (db-role + schema + table), synthesize từ nguồn metadata mới; (2) mọi consumer (gate introspect, SQL builder, DB handle) nhận topology từ entry — fallback giá trị cũ khi rỗng để backward-compat; (3) "0 items processed" PHẢI là warning có fix_hint, không bao giờ là success im lặng; (4) smoke-check sau migrate topology: bảng run-state của M phải có rows.
- **Bối cảnh (Trigger)**: User yêu cầu review/nâng cấp Reconcile. Verify DB: recon_runs=0, report=0 từ trước tới nay; trigger NATS → activity "success tables_checked=0". Root cause 3 lớp: synthesize thiếu shadow_schema; CheckAll GetSchema("public"@5433) nil → skip hết; DestAgent nhận control-plane db + FROM "table" trần — trong khi shadow đã ở shadow_*@5436 (hybrid Path B).
- **Root Cause**: Recon module không được nâng cấp khi hệ chuyển Path B; gate "table not materialised" biến thành cửa chặn 100% âm thầm; status success không phân biệt "checked OK" vs "không check gì".
- **Fix/Correct Flow**: +ShadowSchema synthetic vào registry entry → GetSchemaInSchema(shadow|public) → quoteRelation("schema"."table") → DestAgent nhận shadowDB; 0-checked → warning + Warn log fix_hint. E2E: recon_runs 4 rows, phát hiện drift thật.
- **Phạm vi (≥3 dự án?)**: Có — mọi hệ có module nền (recon/backup/audit/metrics) viết trước một cuộc migrate topology DB (split DB, multi-schema, multi-tenant).
- **Tags**: reconcile, topology-migration, silent-skip, false-positive-success, schema-qualify
- **Nguồn**: workspace `reconcile-overhaul-2026-06-10`

### [2026-06-03] UI bảng đích (Master/DW) phải render theo business mapping, không bê raw/system columns của Shadow
- **Global Pattern**: `[Agent A thiết kế UI detail cho bảng ĐÍCH (Master/DW) nhưng bê nguyên/duplicate cấu trúc bảng NGUỒN trung gian (Shadow) gồm cả raw/system columns (_raw_data, _synced_at) thay vì map theo business rules]` → `[UI dư cột hệ thống vô nghĩa, không phản ánh đúng cấu trúc bảng đích, operator khó đối chiếu schema]`. **Đúng**: (1) UI chi tiết bảng đích phải dựng từ Business Mappings (`mapping_rules` nghiệp vụ): Source Field → Target Column → Target Type; (2) loại bỏ cột internal/system khỏi detail view; (3) tận dụng component ánh xạ có sẵn (vd `MappingFieldsPage`) để reuse hiển thị mapping đồng bộ toàn hệ thống.
- **Bối cảnh (Trigger)**: Trang Master Registry detail/expandable row ban đầu hiển thị lại metadata hệ thống/bê cột Shadow (`_raw_data`, `_synced_at`). User: "db master ko phải là bê i xì shadow qua, raw_data mang qua làm gì".
- **Root Cause**: Model chưa hiểu luồng đồng bộ CDC Shadow→Master: Shadow là staging chứa raw payload, Master là đích (DW) chứa dữ liệu nghiệp vụ sau transform/mapping; hiển thị raw system columns ở Master không mang giá trị vận hành.
- **Fix/Correct Flow**: Khi user nhắc "hiển thị sai cấu trúc dữ liệu đích" = signal; chuyển UI detail sang render business mapping rules qua API/component mapping nghiệp vụ.
- **Phạm vi (≥3 dự án?)**: Có — mọi hệ CDC/Data Warehouse/ETL có phân tách bảng staging/shadow thô và bảng đích nghiệp vụ đã biến đổi.
- **Tags**: #cdc #data-warehouse #ui-ux #mapping-rules #business-first #shadow-table #observability
- **Nguồn**: lessons.md [2026-06-03]

### [2026-05-29] Truth source của schema/DDL là runtime path (sinkworker), không phải bootstrap path
- **Global Pattern**: `[System A có nhiều path P1..Pn tạo schema S cho entity Y]` — `[Agent đọc path bootstrap Pk (one-shot, user-triggered) làm reference spec]` → `[display/code thiếu hoặc thừa cột so với runtime truth — operator thấy schema không khớp thực tế]`. **Đúng**: liệt kê mọi path, phân loại bootstrap (one-shot) vs runtime (continuous, message-driven); runtime path = source of truth; validate UI/display dùng runtime path spec; bootstrap path divergence → mark legacy.
- **Bối cảnh (Trigger)**: Agent render PK source (id) vào System Default Fields vì đọc HandleCreateDefaultColumns (10 cols); thực tế sinkworker runtime tạo 11 cols với `_gpay_id` BIGINT PK — user mid-session correction.
- **Root Cause**: Agent đọc path đầu tiên gặp (bootstrap handler) làm reference thay vì identify path nào runtime thực sự apply liên tục.
- **Fix/Correct Flow**: Grep TẤT CẢ path CREATE/ALTER schema; phân loại; lấy runtime (sinkworker schema_manager) làm spec; validate bằng `\d <shadow_table>` thật.
- **Phạm vi (≥3 dự án?)**: Có — schema metadata UI, default config display, permission display, cron schedule UI, resource quota UI.
- **Tags**: #cdc #schema-drift #root-cause #verification #process-governance
- **Nguồn**: lessons.md [2026-05-29]

### [2026-05-28] Mark-done dựa metric intermediate-layer không có invariant guard tại edge gây whack-a-mole bug
- **Global Pattern**: `[Process P chuyển terminal_success state]` dựa trên `[metric intermediate-layer M_i]` mà `[không có invariant guard M_persisted >= M_expected * τ tại terminal transition edge]` → `[bug trồi sang layer j≠i (whack-a-mole), report success giả, caller tương lai bypass intermediate check]`. **Đúng**: Counter từ destination ground truth; capture expected_total từ source; terminal transition markDone(actual, expected) guard với τ configurable (default 0.99); pause/cancel paths return ngay không fall-through; cursor exhaustion dùng empty-result (len==0) không partial-result.
- **Bối cảnh (Trigger)**: Snapshot runner báo `status=done` nhưng shadow table thiếu rows. Fix counter ở một layer thì bug trồi sang layer khác (cursor exhaustion, pause fall-through, partial flush) — whack-a-mole pattern.
- **Root Cause**: Terminal transition markDone không nhận expected → không thể guard; `if len(batch) < batchSize { break }` làm điều kiện exhaustion sai; pause/cancel `break` rồi fall-through xuống final flush + markDone.
- **Fix/Correct Flow**: `markDone(actual, expected)` với guard `if expected > 0 && actual < expected * τ → markError`; pause/cancel phải return ngay; cursor exhaustion = `len==0`; Prometheus counter `partial_done_total{reason}` mỗi guard trip.
- **Phạm vi (≥3 dự án?)**: Có — snapshot/replica/migration runners (CDC, ETL backfill, Kafka mirror), long-running job có resume, batch processors có pause/cancel.
- **Tags**: #cdc #snapshot-v2 #verification #root-cause #testing #observability #pipeline
- **Nguồn**: lessons.md [2026-05-28]

### [2026-05-27] Hardcode mask string tại write-path vi phạm luật accuracy và phá đối soát dữ liệu cá nhân
- **Global Pattern**: `[Masking layer M thay thế giá trị nhạy cảm bằng chuỗi cứng L (vd "***")]` tại `[write-path của pipeline CDC/ETL từ source A sang sink B]` → `[phá hủy Accuracy(A→B), vi phạm luật BVDLCN/GDPR, sink mất khả năng đối soát + distinct count, không qua kiểm toán kỹ thuật]`. **Đúng**: Per-field MaskStrategy enum {NONE, DROP, HASH_HMAC(salt), PARTIAL, TOKENIZE} configurable qua mapping_rule; DROP set NULL không literal; HASH_HMAC dùng HMAC-SHA256 với secret key versioned; audit log mask_audit_log + mask_config_audit.
- **Bối cảnh (Trigger)**: Hệ thống đồng bộ dữ liệu cá nhân (CCCD, card_number) từ source sang sink. Masking layer dùng hardcode "***" tại write-path → sink lưu "***" thay vì giá trị có thể đối soát → vi phạm luật quyền chỉnh sửa/audit của chủ thể dữ liệu.
- **Root Cause**: Masking design chọn hardcode literal string thay vì strategy pattern; 1 strategy cho mọi loại field; không có audit trail strategy + actor.
- **Fix/Correct Flow**: Implement per-field MaskStrategy enum configurable; DROP dùng NULL; HASH_HMAC với HMAC-SHA256 + secret key versioned + rotation; PARTIAL format-preserving; audit log với event_id, table, field, strategy, key_version.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ hệ thống persist PII xuyên zone tại thị trường có luật BVDLCN (VN 91/2025, GDPR, PDPA, LGPD).
- **Tags**: #masking #cdc #compliance #serialization #root-cause #audit-log #silent-drop
- **Nguồn**: lessons.md [2026-05-27]

### [2026-05-26] Silent-skip khi route cache stale gây snapshot thành công giả (0 row thật)
- **Global Pattern**: `[Pipeline A có multi-layer cache + silent-skip-on-cache-miss + fire-and-forget cache-reload signal]` được `[caller B kích hoạt ngay sau mutation]` → `[first-call-fail không có signal lỗi, metric báo success với rows > 0 nhưng 0 rows thật được persist]`. **Đúng**: 4-layer defense — pre-flight sync reload, hard-assert sau reload nếu empty, log level Warn (không Debug) cho operator, caller dùng return value written, metric đếm output không phải input.
- **Bối cảnh (Trigger)**: Lần snapshot.v2 đầu tiên cho source vừa register → activity_log báo `status=success, rows_affected=N` nhưng shadow table 0 row; lần kế tiếp work bình thường do cache đã warm.
- **Root Cause**: Race condition giữa fire-and-forget NATS reload signal và caller kích hoạt snapshot ngay sau; silent-skip ở processEvent chỉ log Debug (tắt mặc định); caller discard return value written; metric đếm doc Find thay vì doc routed.
- **Fix/Correct Flow**: Caller force in-process `ReloadAll(ctx)` ngay trước hot loop; hard-assert sau reload nếu empty routes; upgrade silent-skip log lên Warn với greppable context; caller inspect return count `written`; metric `rowsTotal += writtenSum` thay vì `+= len(batch)`.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ ETL pipeline có per-table routing + cache + fire-and-forget reload (CDC, NATS, Kafka consumer, API gateway route cache, search indexer schema cache).
- **Tags**: #cdc #snapshot-v2 #cache-reload #silent-drop #fire-and-forget #observability #metric-accuracy
- **Nguồn**: lessons.md [2026-05-26]

### [2026-05-25] LWW Guard cho dual-stream consistency (Snapshot + CDC) — dùng logical clock, không wall-clock
- **Global Pattern**: `[Pipeline P chạy đồng thời Snapshot stream (chậm) và CDC Realtime stream; dùng wall-clock worker làm mốc thời gian]` → `[Snapshot ghi đè data mới hơn của Realtime vì clock skew + OCC tiebreaker lỏng]`. **Đúng**: luôn lấy logical clock từ nguồn (MongoDB: clusterTime qua lệnh `hello`/`replSetGetStatus`) làm mốc tuyệt đối; khi trùng `_source_ts` thì dùng discriminator `_source` ưu tiên Realtime > Snapshot; backport `_source_ts` cho TẤT CẢ shadow tables để bật LWW guard.
- **Bối cảnh (Trigger)**: Pipeline CDC chạy đồng thời Snapshot V2 và Debezium Realtime, ghi chung vào shadow table PostgreSQL; Snapshot ghi đè data mới hơn của Realtime vì wall-clock worker không đồng nhất với Debezium `source_ts`.
- **Root Cause**: (1) Dùng `time.Now()` của worker làm thời gian source cho Snapshot → clock skew với Debezium; (2) OCC condition `<=` cho phép Snapshot đến sau ghi đè; (3) thiếu `_source_ts` ở schema V1.
- **Fix/Correct Flow**: Lấy MongoDB clusterTime cho Snapshot; OCC WHERE clause: `_source_ts < EXCLUDED._source_ts OR (_source_ts = EXCLUDED._source_ts AND _source != 'realtime')`; backport `_source_ts` column vào V1 shadow tables.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ dual-stream write scenario nào (CDC + backfill, primary + replica, async ETL + streaming).
- **Tags**: #cdc #lww #logical-clock #occ-guard #tiebreaker #strong-eventual-consistency #snapshot
- **Nguồn**: lessons.md [2026-05-25]

### [2026-05-22] Pipeline dùng DLQ-on-error không có circuit breaker gây chạy điên khi lỗi deterministic
- **Global Pattern**: `[Pipeline A dùng DLQ-on-error pattern xử lý N items]` gặp `[lỗi deterministic F ảnh hưởng mọi item X]` → `[N DLQ rows, log spam, không có halt signal, lãng phí compute + storage]`. **Đúng**: DLQ-mode pipelines PHẢI có circuit breaker với ≥2 trip conditions: consecutive failure threshold và window/batch error-ratio threshold.
- **Bối cảnh (Trigger)**: snapshot.v2 non-strict mode khi HandleRaw fail trên mọi doc (deterministic VARCHAR overflow) → continue loop qua 6M rows → DLQ flood, "chạy điên", ẩn root cause khỏi operator log.
- **Root Cause**: Pipeline dùng DLQ-on-error nhưng không có circuit breaker — lỗi deterministic làm mọi item fail nhưng pipeline vẫn tiếp tục xử lý toàn bộ N items mà không halt.
- **Fix/Correct Flow**: Thêm circuit breaker với 2 điều kiện trip: consecutive failures (vd 100 liên tiếp) và window error-ratio (vd ≥50% với ≥10 absolute). Khi trip: flush partial DLQ, persist halt state, log đầy đủ counters + last error, return từ work loop.
- **Phạm vi (≥3 dự án?)**: Có — Batch ETL, CDC snapshot/replay, Kafka consumers skip-on-error, mass-mail/notification senders, Airflow tasks, webhook fan-out.
- **Tags**: #cdc #kafka #dlq #circuit-breaker #pipeline #debezium #observability
- **Nguồn**: lessons.md [2026-05-22]

### [2026-05-21] Giảm log noise: per-message log ở mức INFO gây nghẹt I/O trong pipeline thông lượng cao
- **Global Pattern**: `[Stream processor A] log từng message ở mức INFO trong [pipeline thông lượng cao X]` → `[I/O block, log aggregator overload, hiệu năng suy giảm]`. **Đúng**: log per-message ở mức DEBUG; log sự kiện batch-level ở INFO.
- **Bối cảnh (Trigger)**: Log Worker in ra hàng triệu dòng "kafka CDC event" ở mức INFO liên tục khi snapshot 100M+ records, gây tràn log storage và chậm I/O.
- **Root Cause**: Ghi log chi tiết từng message thành công ở mức INFO là không cần thiết trong production; mức INFO chỉ nên cho sự kiện cấp batch.
- **Fix/Correct Flow**: Hạ cấp log per-message trong processMessage từ Info xuống Debug.
- **Phạm vi (≥3 dự án?)**: Có — Kafka consumer, NATS subscriber, CDC worker, batch job bất kỳ.
- **Tags**: #logging-strategy #performance #cdc #high-throughput #log-noise #observability
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-21] Đảm bảo tính chính xác RowsAffected — không đánh đồng transport count với storage write count
- **Global Pattern**: `[Pipeline monitor A] dùng [số message transport X] làm RowsAffected` → `[metrics sai lệch nghiêm trọng khi routing layer skip message do table chưa active]`. **Đúng**: propagate số rows thực tế bị tác động trên storage qua các lớp xử lý.
- **Bối cảnh (Trigger)**: ActivityLog báo cáo "success 3325" (bằng số raw Kafka messages) dù thực tế 0 records được ghi vào shadow DB vì table chưa active.
- **Root Cause**: Code tracking gán RowsAffected = số messages consumed; nếu table chưa active, event handler skip (return nil — success về transport) nhưng không có DB insert; raw Kafka count ≠ actual rows written.
- **Fix/Correct Flow**: Thay đổi signature processMessage và HandleRaw để trả về (int, error) — int là số rows thực tế được ghi; tích lũy vào batchStats.rowsAffected rồi lưu vào ActivityLog.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ pipeline có dynamic routing/registry (Kafka, NATS, SQS) với storage layer.
- **Tags**: #cdc #metrics #rows-affected #data-integrity #observability #kafka
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-21] Kafka consumer transient fetch error behind LoadBalancer — phân loại và retry đúng cách
- **Global Pattern**: `[Kafka consumer A] nhận lỗi routing tạm thời từ [LoadBalancer X]` → `[log ERROR giả + sleep dài làm chậm tự phục hồi + alert fatigue]`. **Đúng**: phân loại transient errors (Not Leader For Partition, Broker Not Available) → log Warn + retry nhanh với connection mới qua LB.
- **Bối cảnh (Trigger)**: Kafka Consumer log Error "Not Leader For Partition" liên tục khi kết nối TCP mới đi qua LB bị round-robin đến broker không phải leader.
- **Root Cause**: LB round-robin route mỗi TCP connection đến random broker; chỉ broker leader mới accept; lỗi được log mức Error + sleep 1s gây log flood và chậm recovery.
- **Fix/Correct Flow**: Viết helper isKafkaTransientError; hạ log xuống Warn; giảm sleep xuống 200ms để client nhanh reconnect ngẫu nhiên qua LB đến partition leader.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ stateful service (Kafka, Redis Cluster, Elasticsearch) đứng sau LoadBalancer.
- **Tags**: #kafka #consumer #transient-error #loadbalancer #logging-strategy #retry #cdc
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-21] CDC golden rule: source store là read-only — signal channel ≠ watermark store
- **Global Pattern**: `[Pipeline A] có nhiều config keys điều khiển nhiều subsystems; [subsystem S] ghi vào [store D] theo Ci; D có constraint read-only` → `[đổi C1 (signal.enabled.channels) không stop ghi nếu C2 (signal.data.collection) vẫn dẫn đến write D]`. **Đúng**: enumerate TẤT CẢ config keys Ci có thể tạo ghi vào D, audit độc lập từng Ci; accept loss of feature nếu Ci cần thiết mà D read-only.
- **Bối cảnh (Trigger)**: Switch `signal.enabled.channels` từ `source,kafka` → `kafka` nhưng source MongoDB prod-like vẫn bị ghi `debezium_signals` collection + watermark docs; `signal.data.collection` config riêng điều khiển nơi ghi watermark DBLog cho incremental snapshot — không bị ảnh hưởng bởi channels config.
- **Root Cause**: Bất kỳ Debezium connector ≥1.7 chạy incremental snapshot ĐỀU ghi 2 marker docs vào `signal.data.collection` mỗi chunk — đây là design intent DBLog (Netflix paper), không có config bypass; agent chỉ audit C1 mà bỏ qua C2.
- **Fix/Correct Flow**: `delete(cfg, "signal.data.collection")` trước override loop; FE bỏ field `signal.data.collection` ở tất cả branches (Mongo/MySQL/PG); backfill PUT 3 connector bỏ config này; trade-off: incremental snapshot Debezium silent-fail, phải build custom snapshot worker.
- **Phạm vi (≥3 dự án?)**: Có — Kafka Streams (state store + changelog topic), AWS DMS (TargetMetadata vs TableMappings), Flink (checkpoint store vs state backend), GoldenGate (trail vs handler vs config).
- **Tags**: #debezium #cdc #read-only-source #signal-data-collection #config-audit #watermark #incremental-snapshot
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-21] Path B: reuse inverted apply pipeline để bypass mutation trên read-only source
- **Global Pattern**: `[Worker W] muốn thực hiện job J trên [store S read-only]; J kéo theo ghi vào S` → `[tách J thành reader-only loop; build envelope khớp shape pipeline P; invoke P.HandleRaw(envelope)]` → `[reuse 100% downstream mapping/upsert/batching không cần engine đóng phát sinh side-effect mutate S]`. **Đúng**: identify entry point thuần data của pipeline P; worker đọc S qua read-only API; build envelope shape S; invoke P.entry_point(envelope) không re-publish ra transport nếu không cần ordering; checkpoint vào control-plane table riêng; idempotency qua DB-level claim + zombie recycle TTL.
- **Bối cảnh (Trigger)**: Debezium incremental snapshot yêu cầu `signal.data.collection` → ghi watermark vào source MongoDB → vi phạm CDC golden rule; disable key → NPE silent-fail → snapshot dead; Path B: custom Mongo Find loop → build Debezium-shaped CDCEvent JSON → invoke EventHandler.HandleRaw (cùng entry point Kafka consumer dùng) → shadow upsert chạy như streaming realtime.
- **Root Cause**: Engine đóng (Debezium) yêu cầu mutate store để thực hiện feature; pipeline hiện có có entry point thuần data chưa được tận dụng để bypass engine.
- **Fix/Correct Flow**: KHÔNG fork pipeline P thành "P_for_snapshot" duplicate; KHÔNG bypass mapping/masking layer của P; KHÔNG nhúng I/O lệ thuộc vào entry point của P (P phải pure-ish).
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ scenario nào có engine đóng yêu cầu mutate read-only store (audit log backfill, immutable warehouse replay, billing source migration).
- **Tags**: #cdc #read-only-source #snapshot #pipeline-reuse #bypass-closed-engine #debezium #idempotent-upsert
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-20] Debezium incremental snapshot silent fail khi thiếu signal.data.collection config
- **Global Pattern**: `[Component A] nhận command và log "Requested X" nhưng thiếu [dependency B chưa configure]` → `[X không bao giờ thực thi; silent failure]`. **Đúng**: kiểm tra dependency graph đầy đủ — A cần B để coordinate/execute; missing B → gracefully do nothing mà không báo lỗi.
- **Bối cảnh (Trigger)**: Incremental snapshot signal gửi thành công (log "Requested INCREMENTAL snapshot") nhưng không có data nào được produce; không có error log.
- **Root Cause**: Connector config thiếu signal.data.collection; Debezium 3.x cần source signal collection để watermark coordination — thiếu collection → snapshot được queue nhưng không bao giờ thực thi → silent failure.
- **Fix/Correct Flow**: Thêm "signal.data.collection": "<database>.<collection>" vào connector config.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ component nhận signal/command và cần resource phụ để execute (Debezium, Flink, Spark Structured Streaming).
- **Tags**: #debezium #cdc #kafka #signal #incremental-snapshot #silent-failure #config
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] LoadBalancer round-robin + Kafka: cần topology-aware retry với connection mới mỗi attempt
- **Global Pattern**: `[Kafka client A] reuse static connection qua [LoadBalancer X trước Kafka cluster Y]` → `[stuck tại wrong broker mãi; Not Leader For Partition liên tục]`. **Đúng**: mỗi retry tạo connection MỚI qua LB để randomize lại target broker.
- **Bối cảnh (Trigger)**: "Not Leader For Partition" error khi publish signal qua Kafka LoadBalancer; 2/3 connections fail do LB route đến broker không phải leader.
- **Root Cause**: LB round-robin route mỗi TCP connection đến random broker; chỉ broker leader mới accept writes; static connection reuse stuck tại wrong node.
- **Fix/Correct Flow**: Thay kafka.Writer bằng kafka.DialLeader retry loop (max 10 attempts, new TCP connection per attempt); P(hit leader in 10 attempts) ≈ 99.998%.
- **Phạm vi (≥3 dự án?)**: Có — Kafka, Redis Cluster, Elasticsearch, bất kỳ stateful cluster sau LB.
- **Tags**: #kafka #loadbalancer #retry #topology #infrastructure #cdc
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] kafka-go producer fail với "Unknown Topic Or Partition" vì broker auto-create không trigger trên produce
- **Global Pattern**: `[Producer P publish đến Kafka topic T via segmentio/kafka-go Writer.WriteMessages; broker B có auto.create.topics.enable=true]` lên `[Kafka producer bootstrap]` → `[P fail với [3] Unknown Topic Or Partition vì kafka-go không set allowAutoTopicCreation=true trong MetadataRequest; broker auto-create chỉ trigger trên CONSUMER metadata fetch, không phải producer publish]`. **Đúng**: Application owns the topic lifecycle; tại service startup, call `kafka.Client.CreateTopics`; treat `kafka.TopicAlreadyExists` via errors.Is là success (idempotent); log INFO on create, DEBUG on already-exists, WARN+continue on transient broker outage.
- **Bối cảnh (Trigger)**: Service start + first publish → fail với `[3] Unknown Topic Or Partition`; broker auto-create enabled nhưng topic không được tạo tự động bởi producer.
- **Root Cause**: kafka-go Writer.WriteMessages không trigger broker auto-create (chỉ consumer metadata fetch mới trigger); application không tự tạo topic ở startup; phụ thuộc vào broker config mà không kiểm soát được.
- **Fix/Correct Flow**: Application-owned topic bootstrap: call `kafka.Client.CreateTopics` tại startup với topic + partition + RF; idempotent create (TopicAlreadyExists = success); không dùng docker-compose KAFKA_CREATE_TOPICS (chỉ work trên Bitnami/Wurstmeister); không dùng init container (race condition).
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project dùng kafka-go producer với fresh deploy environment.
- **Tags**: #kafka #cdc #silent-drop #root-cause
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Hardcode tên resource động sinh ra bởi runtime/control plane
- **Global Pattern**: `[Consumer A] resolve tên resource [R] từ [static config/code C]` lên `[control-plane B tạo R động theo runtime key]` → `[probe/REST gọi sai → HTTP 404 → audit log mislead]`. **Đúng**: mọi resource sinh động bởi control plane phải có single source of truth (bảng registry/API); consumer dùng resolver helper chuyên dụng (ResolveByX), không fallback hardcode; config không được chứa literal resource name cho resource động.
- **Bối cảnh (Trigger)**: Log probe `/connectors/goopay-mongodb-cdc/status` trả HTTP 404 trong khi Kafka Connect đăng ký connector động theo `connection_code` từng instance. Tên connector bị hardcode rải khắp config yml + helper map + handler default.
- **Root Cause**: Lẫn lộn "tên catalog hợp lệ tại thời điểm dev" với "tên resource được control plane đăng ký động ở runtime"; khi control plane tạo mỗi instance theo `connection_code` riêng, mọi reference từ worker phải tra qua registry, không được fix-string.
- **Fix/Correct Flow**: Tạo resolver helper `ResolveByConnectionCode(code)` tra bảng registry; khi resolver trả "" thì error rõ ràng "cannot resolve <resource> for key=<value>"; cấm key config dạng `<thing>Name: "<literal>"` cho resource động — đổi sang endpoint + resolver.
- **Phạm vi (≥3 dự án?)**: Có — CDC/Debezium Kafka Connect, Kubernetes Operator (CR name per-cluster), Multi-tenant SaaS (IAM role per-tenant), Stripe webhook per-account.
- **Tags**: #control-plane #dynamic-resource #resolver-pattern #anti-hardcode #cdc #single-source-of-truth
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] First-wins key resolution gây silent misroute khi có duplicate unqualified keys
- **Global Pattern**: `[Resolver R] dùng first-match-wins trên [key-set K có duplicate unqualified key K_legacy]` → `[silent misroute: CDC event đi sai đích, audit log báo success nhưng shadow table sai]`. **Đúng**: sắp xếp most-specific-first `[qualified, unqualified]`; enforce uniqueness trên unqualified key tại load-time (FAIL LOUDLY khi collision); mọi Resolve() thành công phải log `key_matched, db, table, target` ở DEBUG level.
- **Bối cảnh (Trigger)**: `buildRouteLookupKeys` trả `[sourceTable, sourceDB|sourceTable]`; hai row `source_object_registry` cùng `source_object_name="export-jobs"` dưới hai database khác nhau; unqualified key đụng → route về row load vào cache trước → shadow table sai destination trong cả ngày.
- **Root Cause**: Cache lookup theo first-match-wins là silent contract giữa key-set order và cache fill order; khi có duplicate unqualified keys (cùng table-name trải nhiều database/tenant), order quyết định routing nhưng không ai detect được bằng unit test (single-tenant fixture đều pass).
- **Fix/Correct Flow**: Đổi key-set thành `[qualified, unqualified]` (specific trước); nếu phát hiện duplicate unqualified key lúc load → return error "ambiguous registry: <table> appears under <db1>, <db2>"; thêm metric `route_resolution_legacy_fallback_total` nếu cần giữ legacy compat.
- **Phạm vi (≥3 dự án?)**: Có — Kubernetes Operator (namespace/name vs name-only), Multi-tenant IAM (tenant_id|user_id vs user_id-only), Debezium signal data-collections (db.collection vs collection-only).
- **Tags**: #cache-lookup #first-wins #routing-bug #silent-contract #unqualified-key #multi-tenant #cdc
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Snapshot mode phải engine-aware — không hardcode operation-mode cho heterogeneous backends
- **Global Pattern**: `[Client C] hardcode operation-mode [M] cho tất cả [heterogeneous backends B_i]` → `[silent failure trên B_subset có semantics khác M]`. **Đúng**: resolve mode per-backend tại call site (thread engine/version qua signature hoặc per-resource config trong registry); document workaround inline với lý do và ticket; verify behavior post-publish bằng offset/count delta, không chỉ trust publish success.
- **Bối cảnh (Trigger)**: Worker `TriggerIncrementalSnapshot` luôn emit `"type":"incremental"`; MongoDB Debezium 2.5.4 hit NPE tại `MongoDbIncrementalSnapshotChangeEventSource:228` rồi cursor exhausted → "No data returned"; Mongo cần `"blocking"` còn Postgres/MySQL ổn với `"incremental"`.
- **Root Cause**: Snapshot type là engine-specific contract của Debezium nhưng client code treat như engine-agnostic constant; hardcode constant ở client đẩy bug khả năng tái phát cho version Debezium tương lai.
- **Fix/Correct Flow**: Lưu `snapshot_mode` column trong `connection_registry`; ops set theo Debezium version đang chạy; khi nâng version chỉ update row, không rebuild; verify bằng query topic offset BEFORE/AFTER + check "Finished snapshotting N records" trong connector log.
- **Phạm vi (≥3 dự án?)**: Có — database backup tool (pg_dump vs mysqldump flags), HTTP retry library (POST vs GET strategy per-method), migration runner (transactional DDL per-engine).
- **Tags**: #engine-aware #per-backend-config #debezium #snapshot-mode #vendor-workaround #cdc
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Debezium Kafka signal channel key routing — consumer silently drop message key không khớp
- **Global Pattern**: `[Producer A] publish infra-control signal lên [broker B] với key [K_wrong]` lên `[consumer C chỉ accept khi K == C.identity_key]` → `[C silently drop; producer log "signal published OK"; downstream effect không bao giờ xảy ra]`. **Đúng**: producer phải resolve C.identity_key từ registry per-publish (không tại init); khi "publish OK nhưng consumer không react" → dump topic với `print.key=true`, so với consumer.identity_key — đây là sanity check đầu tiên.
- **Bối cảnh (Trigger)**: `KafkaSignalChannel#process` so sánh `record.key()` với connector's `topic.prefix`; key ≠ prefix → drop; wrong key trông y hệt "signal lost in transit" — không có error path ở cả hai phía.
- **Root Cause**: Producer không resolve đúng consumer identity_key từ registry; resolution xảy ra tại producer init (static) thay vì per-publish (dynamic khi consumer set thay đổi).
- **Fix/Correct Flow**: Producer resolve C.identity_key từ DB row hoặc HTTP discovery tại mỗi lần publish; message key = exact identity_key; khi debug: `print.key=true` dump topic, so với consumer config.
- **Phạm vi (≥3 dự án?)**: Có — Debezium signal, Kafka Streams topology routing, AWS Kinesis với partition key filter.
- **Tags**: #kafka #debezium #signal-channel #key-routing #silent-drop #cdc
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Source DB read-only constraint — không assume dev environment = prod environment
- **Global Pattern**: `[Pipeline A] đề xuất workaround [B] yêu cầu ghi vào [store C]` → `[nếu C có constraint read-only thì B infeasible bất kể có chạy được trên dev]`. **Đúng**: trước khi đề xuất workaround, enumerate quyền ghi của pipeline trên từng store (source/dest/shadow/control-plane); nếu source = read-only → loại bỏ ngay mọi pattern yêu cầu ghi vào source.
- **Bối cảnh (Trigger)**: Muscle tạo collection `cdc_system.debezium_watermarks` trên Mongo source local; user phẫn nộ vì source DB prod là read-only cho CDC pipeline (fintech compliance + DBA policy); Debezium MongoDB incremental snapshot DBLog watermark buộc ghi vào source — fundamentally incompatible với read-only source.
- **Root Cause**: Muscle implement workaround mà không challenge feasibility trên prod; dev environment cho phép ghi nhưng prod thì không; false equivalence cho fintech/regulated workloads.
- **Fix/Correct Flow**: Snapshot logic phải chạy ngoài Debezium: custom worker đọc source qua read-only credential, ghi xuống dest/shadow + control plane; Debezium chỉ giữ vai trò streaming CDC (oplog/WAL read — chỉ cần read access).
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ regulated/fintech pipeline nào với read-only source (audit DB, billing source, immutable warehouse).
- **Tags**: #debezium #read-only-source #fintech #cdc #constraint-discovery #process-governance
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Không brute-force lỗi infra bằng retry — trace root cause trước
- **Global Pattern**: `[Agent A] tăng MaxAttempts/retry khi gặp [lỗi infra routing B]` → `[workaround không phải fix; symptom vẫn tồn tại, chỉ giảm xác suất; root cause bị che giấu]`. **Đúng**: viết diagnostic script để trace path thực tế (metadata trả về gì, DNS resolve thành gì, TCP đến đâu); xác nhận fix bằng script trước khi apply; hỏi "Nếu MaxAttempts=1, fix có hoạt động không?" — nếu không → chưa phải root cause.
- **Bối cảnh (Trigger)**: Kafka `Not Leader For Partition` khi publish signal; agent tăng MaxAttempts 10→20 thay vì debug; user gọi đúng "mày đang cheat"; root cause thực: `kafka.Writer` mở NEW TCP connection mỗi Produce request, 3 broker hostname cùng trỏ về 1 LB IP → LB round-robin random → 2/3 xác suất đến non-leader.
- **Root Cause**: Lười — thấy "retry nhiều hơn thì xác suất hit leader cao hơn" và dùng luôn thay vì viết diagnostic script xác nhận cơ chế lỗi.
- **Fix/Correct Flow**: Dùng `kafka.DialLeader` + `Conn.WriteMessages` — DialLeader discover leader qua metadata rồi retry trên cùng TCP session đến đúng broker, hoạt động đúng qua mọi LB topology.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ lỗi infrastructure routing nào (DB connection pool, Redis cluster slot, gRPC load balancing).
- **Tags**: #kafka #infrastructure #loadbalancer #retry #root-cause #debugging #process-governance
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-18] Conditional subscriber registration gây silent NATS message drop khi feature flag tắt
- **Global Pattern**: `[Agent A] đăng ký NATS subscriber S cho subject J trong conditional block gated by feature flag F, trong khi producer P cho J vẫn enabled unconditionally` lên `[NATS PubSub + conditional subscriber registration]` → `[Khi F off, P vẫn publish thành công, NATS silently drop message (no listener), user-facing operation appear succeed (202/OK) nhưng không bao giờ reach worker]`. **Đúng**: Luôn register subscriber cho mọi subject producer có thể publish; khi real handler phụ thuộc feature flag F, register STUB subscriber trong else branch log ERROR với trace_id/action/origin + reason F-off.
- **Bối cảnh (Trigger)**: User báo "click Snapshot Now không trigger qua worker". Trace chain: FE → API → publish NATS subject. API return 202 luôn. Worker subscribe subject này nhưng registration nằm trong `if reconCore != nil` block; config local không có MongoDB block → reconCore=nil → subject không có subscriber.
- **Root Cause**: Subscriber registration được coi như tính năng tùy chọn dựa vào config feature flag; producer vẫn bật → asymmetry: producer luôn on, consumer có-thể-off. NATS PubSub fire-and-forget không trả error khi không có subscriber → silent loss.
- **Fix/Correct Flow**: Audit mọi `if <flag> { Subscribe(subj, ...) }`; mỗi else nhánh phải có stub subscriber log ERROR; stub payload tối thiểu: trace_id, action, origin, subject, reason; production safety: register subscriber unconditionally, gate logic bên trong.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project PubSub (NATS, Kafka, RabbitMQ, SQS) với conditional subscriber registration.
- **Tags**: #cdc #kafka #observability #silent-drop #root-cause
- **Nguồn**: lessons.md [2026-05-18]

### [2026-05-04] Debezium config thay đổi Avro emit type cần pre-flight Schema Registry compat
- **Global Pattern**: `[Operator A thay đổi Debezium connector config B ảnh hưởng Avro emit type cho entity E]` + `[Schema Registry global compat ≠ NONE]` → `[Connector goes FAILED tại schema register kế tiếp, block toàn bộ downstream ingest]`. **Đúng**: trước khi PATCH connector, set per-subject `compatibility=NONE` cho mọi topic affected → verify → PATCH connector → wait RUNNING → trigger source event verify → (optional) restore BACKWARD sau khi schema settled.
- **Bối cảnh (Trigger)**: Brain PATCH `decimal.handling.mode=double` → Debezium re-register Avro schema mới (bytes-decimal → double primitive) → Schema Registry global BACKWARD reject incompatible primitive type change → connector FAILED.
- **Root Cause**: Debezium config thay đổi serializer-side type emit Avro types khác nhau; Schema Registry coi đó là incompatible evolution; không có CI guard và không có pre-flight compat check.
- **Fix/Correct Flow**: PUT `/config/<topic>-value` `{"compatibility":"NONE"}` → verify response → PATCH connector → wait RUNNING → trigger event verify log sạch → restore BACKWARD sau migration.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi Debezium config thay đổi Avro schema generation: `decimal.handling.mode`, `time.precision.mode`, `binary.handling.mode`, SMT type-changing, key/value converter swap.
- **Tags**: #debezium #schema-registry #avro #cdc #connector-config #pre-flight-check #schema-evolution
- **Nguồn**: lessons.md [2026-05-04]

### [2026-05-04] Migrate ingest path V1→V2 quên populate constraint-keyed anchor column
- **Global Pattern**: `[Migration ingest path A → B: B viết upsert SQL từ scratch nhưng quên populate constraint-keyed anchor column C mà V2 schema introduce]` → `[Master ON CONFLICT(C) collapse N distinct source rows thành 1]`. **Đúng**: audit enumerate MỌI column V2 schema không phải pure business field (`_*` prefix, UNIQUE/anchor, GENERATED, DEFAULT non-trivial) → cross-check explicit write trong path B → unit test 2 cases (schema có C / không có C) → live smoke INSERT verify anchor ≠ NULL/empty sau 1 cron tick.
- **Bối cảnh (Trigger)**: B3 logical-clone fan-out chuyển ingest từ V1 (DB-side trigger tự fill `_gpay_source_id`) sang V2 (`BuildUpsertSQLInSchema` generator); generator V2 không port logic ghi anchor → mọi shadow row có `_gpay_source_id=''` → master dedup sai.
- **Root Cause**: Developer V2 chỉ audit "business cols" + một số meta cols thông thường; anchor column C không nằm trong "business data" view; unit test V1 không cover C (DB tự fill), unit test V2 cũng không add case.
- **Fix/Correct Flow**: Thêm `if schema.Columns[C] exists → write derived value` với runtime schema reflection check (backward-compat với legacy tables); unit test 2 cases; live smoke verify distinct anchor per source row.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi schema evolution thêm UNIQUE/anchor column mà ingest path không tự suy ra từ business data (tenant_id composite, idempotency_key, partition_key, business_event_id).
- **Tags**: #cdc #v1-v2-migration #anchor-key #unique-constraint #on-conflict #ingest-path #schema-evolution
- **Nguồn**: lessons.md [2026-05-04]

### [2026-05-04] Event translator hardcode field nil làm downstream consumer fail silently
- **Global Pattern**: `[Translator layer A viết downstream-event-DTO B và hardcode field X (before/source/header/correlation) thành nil/zero — dù upstream raw payload Y thực sự populate X]` → `[Downstream consumer Z phụ thuộc X either hard-fail hoặc silent-drop events; error message "no X data" misdirect ops nghi upstream config]`. **Đúng**: translator parse ALL event fields uniformly với symmetric codec helper; hard-fail guard tại handler boundary đổi thành warn+skip per-route; khi diagnose "no X data" → trace 3 lớp: raw payload sniff → translator output log → handler input.
- **Bối cảnh (Trigger)**: `handleDelete` hard-fail "no 'before' data" cho mọi DELETE event; REPLICA IDENTITY=FULL đúng, Debezium publish DELETE đúng, Avro payload có `before` field — bug tại translator.
- **Root Cause**: `kafka_consumer.go` build CDCEvent với `"before": nil` hardcoded, không gọi `unwrapAvroUnion(event["before"])` như đã làm cho `after`; asymmetric codec — chỉ 1 trong 2 field được parse đúng.
- **Fix/Correct Flow**: Parse `beforeRaw` symmetric với `afterRaw`; relax handler guard từ hard-fail sang warn+skip per-route (defense-in-depth cho edge case).
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho Webhook fanout missing signature header, gRPC interceptor drop metadata correlation, JSON-to-Protobuf bridge skip oneof, message bus bridge drop headers map, bất kỳ multi-hop translator có schema mismatch.
- **Tags**: #cdc #event-pipeline #avro-translation #boundary-guard #before-image #three-layer-trace #silent-drop
- **Nguồn**: lessons.md [2026-05-04]

### [2026-05-04] Orchestrator chỉ update low-level filter tier, bỏ qua high-level namespace tier của external system
- **Global Pattern**: `[Orchestrator A onboard resource X bằng cách chỉ update low-level filter tier (collection/table-level) của external system E]` + `[E có MULTIPLE TIERS filter (namespace/database/region cấp cao hơn)]` → `[High-level tier silently drop resource X; orchestrator báo "register OK" nhưng pipeline không có event nào]`. **Đúng**: enumerate TẤT CẢ tier filter của external system trước khi viết orchestrator; mỗi tier cần helper riêng; wrapper gộp gọi đủ tier top-down; verify "first event arrives within N seconds" sau onboard; smoke test PHẢI dùng namespace MỚI chưa từng có row để force pass-through tier cao.
- **Bối cảnh (Trigger)**: Admin-api extend `collection.include.list` += collection mới thành công; registry commit; NATS signal đúng; nhưng Debezium `database.include.list` không có database `goopay` → Kafka topic không bao giờ xuất hiện.
- **Root Cause**: Orchestrator chỉ touch `collection.include.list` (tier thấp), không touch `database.include.list` (tier cao); Debezium silently drop tất cả events từ database không có trong tier cao.
- **Fix/Correct Flow**: `extendDebeziumInclude` extend cả `database.include.list` đồng thời với `collection.include.list`; hoặc emit warning khi detect namespace mới yêu cầu operator approve.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho Kubernetes NetworkPolicy namespace+pod selector, AWS SG+VPC ACL, Kafka ACLs cluster+topic, Stripe webhook endpoint+event type, Cloudflare zone+page rule, mọi external system có nested allow-list cha-con.
- **Tags**: #cdc #orchestrator #include-list #multi-tier-filter #debezium #silent-drop #onboarding #verify-streaming-not-config
- **Nguồn**: lessons.md [2026-05-04]

### [2026-05-04] Optional key từ request payload được dùng raw làm structural identifier gây empty propagation
- **Global Pattern**: `[Component A đọc optional key K từ request payload B → dùng raw value làm structural identifier part X (table name, topic name, normalized key, ACL entry)]` → `[Khi K vắng mặt/rỗng: empty propagation, dirty entries, silent ingest stuck, UNIQUE collision]`. **Đúng**: A PHẢI fallback sang canonical field khi K missing/empty; validate tại admission — reject 400 nếu sau fallback vẫn rỗng; audit TẤT CẢ call sites cùng lúc, không fix chỉ 1 vị trí; test multi-payload (with K, without K, K=empty, K=bogus).
- **Bối cảnh (Trigger)**: `POST /v2/sources/register` cho Mongo chỉ truyền `source_locator={"database":...}` không có `collection` key; 3 vị trí trong `helpers.go` đọc raw `stringFromLocator(...,"collection")` → rỗng → UNIQUE constraint poison, Kafka topic name rác, Debezium include-list entry sai → ingest stuck.
- **Root Cause**: Round 1 fix chỉ chạm 1/3 vị trí; 2 vị trí còn lại cùng pattern đối xứng chưa được audit; thiếu cross-site audit discipline.
- **Fix/Correct Flow**: 3 vị trí trong helpers.go đều thêm `if collection == "" { collection = req.SourceObjectName }`; validate post-compute identifier non-empty; test `TestExtendDebeziumInclude_Mongo_BothTiers` 21 assertions PASS; live smoke verify Kafka offset advance.
- **Phạm vi (≥3 dự án?)**: Có — Kubernetes admission optional labels, Stripe webhook optional tenant_id, multi-tenant sharding optional tenant_key, image build optional tag override, search indexer optional targetIndex, bất kỳ adapter dịch polymorphic payload sang identifier cứng.
- **Tags**: #adapter #fallback #optional-key #identifier #unique-constraint #silent-drop #audit-all-occurrences #cross-site
- **Nguồn**: lessons.md [2026-05-04]

### [2026-04-29] Fire-and-forget command không có companion completion event — state leak vĩnh viễn
- **Global Pattern**: `[Publisher A set state='running' rồi publish cmd.X fire-and-forget]` mà `[handler B không emit evt.X.completed và không có monitor M update final state]` → `[state 'running' vĩnh viễn, operator không phân biệt job đang chạy vs đã chết Y]`. **Đúng**: 3-actor pattern — publisher A → handler B → monitor M; cmd.X ↔ evt.X.completed đối xứng; M idempotent qua WHERE state='running' guard; handler B không được direct-write table của A (cross-domain write).
- **Bối cảnh (Trigger)**: TransmuteScheduler set `last_status='running'` rồi publish NATS `cdc.cmd.transmute`; handler chạy xong không bao giờ UPDATE lại row → mọi schedule sau tick đầu vĩnh viễn 'running'.
- **Root Cause**: Fire-and-forget không có closed loop; handler thiếu correlation_key để biết schedule_id; architect ruling: handler không được tự UPDATE schedule table (coupling hai concern).
- **Fix/Correct Flow**: Command payload mang correlation_key; handler echo correlation_key trong evt.X.completed; separate JobMonitor subscribe evt.X.completed → UPDATE final state; monitor subscription wired tách rời handler ở boot.
- **Phạm vi (≥3 dự án?)**: Có — cron-driven jobs, saga orchestration, RPC retry/dedup, K8s Job watchdog, payment status tracking, email send tracking.
- **Tags**: #cdc-data-pipeline #kafka #coupling #observability #fire-and-forget #event-driven #state-machine
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-29] Event-driven auto-fanout pipeline có cascade liability — bug N+1 chỉ lộ khi step N thành công
- **Global Pattern**: `[Orchestrator A dispatch command tới handler B qua message bus C, B ghi vào schema X]` với `[N-step auto-fanout (step_completed → Advance → step N+1)]` → `[bug ở step N chỉ phơi ra khi step N-1 success; cascade liability = tổng bug ÷ tốc độ pipeline tiến Y]`. **Đúng**: review CẢ 3 mặt đồng thời (A build payload, B parse payload, B SQL khớp schema X); integration test cấp pipeline (1 advance → assert state=terminal) phải tồn tại trước khi merge; có thể tạm tắt auto-fanout khi smoke test bug fix tại step lẻ.
- **Bối cảnh (Trigger)**: Provisioning state machine smoke test: mỗi lần fix 1 bug (column name sai), pipeline tiến thêm 1-2 step rồi fail ở step sau với bug cùng loại nhưng ở component khác — chuỗi 4 bug isolated (resolveShadowTarget JOIN sai, shadow_binding cột không tồn tại, discover payload thiếu field, transmute_schedule keyed sai).
- **Root Cause**: Mỗi step được review/test như isolated unit; bug ở step N chỉ phơi ra khi step N-1 thành công; cross-module review (orchestrator ↔ handler khác repo) bị overlook.
- **Fix/Correct Flow**: Khi thêm step vào state machine, checklist 3 điểm: (a) orchestrator payload build, (b) handler payload parse struct fields, (c) handler DB INSERT/UPDATE column list vs schema thật; boot-time guard validate column tags vs information_schema; tắt auto-fanout tạm thời khi debug step đơn lẻ.
- **Phạm vi (≥3 dự án?)**: Có — Temporal, AWS Step Functions, Camunda BPMN, custom NATS/Kafka pipeline; đặc biệt nguy hiểm khi orchestrator + handler thuộc 2 module/repo khác nhau.
- **Tags**: #cdc-data-pipeline #kafka #testing-verification #cascade-liability #event-driven #state-machine #observability
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-29] State machine cascade thành công với empty output — schemaless engine silent time bomb
- **Global Pattern**: `[State machine A cascade qua N steps B trên heterogeneous engines X (PostgreSQL static schema + MongoDB schemaless)]` với `[step return success=true kể cả khi output rỗng (0 columns/0 rules)]` → `[cascade tới state terminal với pipeline RỖNG, data thật đổ vào → silent time bomb gãy hàng loạt Y]`. **Đúng**: mỗi step PHẢI có fail-fast invariant check về chất lượng output (non-empty/schema valid); engine schemaless cần thêm pre-flight ở step đầu validate source has data.
- **Bối cảnh (Trigger)**: Track D test với PostgreSQL (schema tĩnh) → cascade thành công; mở rộng sang MongoDB schemaless/MariaDB empty → mỗi step return success=true với output rỗng; orchestrator auto cascade tới running với pipeline RỖNG.
- **Root Cause**: Step success được định nghĩa là "step ran without throwing exception", không phải "output usable"; test với 1 engine schema-tĩnh không cover được engine schemaless; thiếu universal step-output gate.
- **Fix/Correct Flow**: Universal gate ở cuối mỗi step: assert output count > 0; engine-specific pre-flight ở step đầu (shadow_bind) check source-side invariants cho schemaless; gate đặt ngay TRƯỚC bước có side-effect lớn không reversible (CREATE TABLE, ENABLE SCHEDULE, PUBLISH EVENT).
- **Phạm vi (≥3 dự án?)**: Có — ETL pipelines, IaC apply, deploy graph, multi-source ingestion, schema migration orchestrator đa-engine.
- **Tags**: #cdc-data-pipeline #kafka #testing-verification #cascade-liability #state-machine #schemaless #fail-fast
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-29] Three-layer trust failure khi component trung gian ghi qua constraint-store
- **Global Pattern**: `[Component A sản xuất metadata] → [Component B ghi vào store C có constraint]` → `[3 lớp lỗi độc lập có thể che nhau: A sai shape, B sai conflict key, C reject do normalisation sai]`. **Đúng**: diagnose top-down theo error message từng lớp; fix từng lớp một, re-run end-to-end sau mỗi lần fix; thêm normalizer tại mọi boundary B-writes-C nhận raw/external source.
- **Bối cảnh (Trigger)**: CDC auto-provisioning: shadow_bind handler (A) → master_binding UPSERT (B) → `cdc_mapping_rules` CHECK constraint (C); 3 lớp lỗi phát sinh tuần tự che nhau.
- **Root Cause**: A sinh cdcCols-only shadow thay vì source-mirrored; B dùng sai conflict key; C CHECK regex từ chối lowercase type strings từ `information_schema`.
- **Fix/Correct Flow**: Reproduce clean-state → capture exact DB row/SQL/error code từng lớp trước khi fix → fix từng lớp → add normalizer `normalizeMappingRuleDataType()` tại write-site → lossless upcast (TEXT) với unknown types.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi pipeline có producer→writer→constrained-store (ETL, event sourcing, schema registry writes, audit-log ingestion).
- **Tags**: #three-layer-trust #root-cause #cdc #schema-drift #normalisation #constraint #diagnosis
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-29] DDL generator chạy trước metadata được populate bởi bước sau trong pipeline
- **Global Pattern**: `[Generator G chạy tại bước C sớm hơn, đọc metadata table M được populate bởi bước A đến sau]` → `[Output của G rỗng/incomplete ở pass đầu; subsequent pass không tự chạy lại]`. **Đúng**: tách CREATE-once path khỏi idempotent ALTER-add-missing path; sau khi bước A populate M, REPUBLISH trigger event cho G để G chạy lại với metadata đầy đủ; validate payload schema của republish khớp với handler Unmarshal target.
- **Bối cảnh (Trigger)**: `MasterDDLGenerator.Apply` chạy ở `master_bind` step, đọc `mapping_rule_v2`; bridge V1→V2 populate table này ở `discover` step (sau đó) → DDL emit thiếu business cols.
- **Root Cause**: Pipeline step ordering tạo temporal coupling: generator phụ thuộc dữ liệu chưa tồn tại tại thời điểm thực thi; không có cơ chế re-trigger sau khi dữ liệu sẵn sàng.
- **Fix/Correct Flow**: Làm G additive (CREATE + ALTER trong cùng transaction); sau step A publish metadata, republish event trigger G; dùng cùng struct Marshal/Unmarshal để tránh silent skip do wrong key.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi DDL generator, cache builder, indexer, cron projection đọc từ table populated bởi downstream step trong cùng workflow.
- **Tags**: #cdc #ddl-generator #pipeline-ordering #additive-migration #republish #temporal-coupling
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-20] Cross-service refactor parallel — coordinate qua subject contract trước khi delegate
- **Global Pattern**: `[Coordinator A delegate parallel refactor across services S1, S2, S3]` + `[A agree subject naming contract + payload schema TRƯỚC khi delegate]` → `[parallel Muscle implement độc lập theo contract, không cần sync wait]`. **Đúng**: Subject naming contract TRƯỚC; fire-and-forget cho async broker; FE polling absorb uncertainty; verify cross-boundary post-deploy end-to-end.
- **Bối cảnh (Trigger)**: User approve fix 12 architectural violations (NATS async + service boundary + multi-source routing). Scope lớn cross 3 projects (Worker + CMS + FE). Brain cần coordinate parallel Muscle.
- **Root Cause**: Pattern design — NATS fire-and-forget cho phép parallel refactor mà không cần sync. CMS publish return immediate; Worker subscribe pick up sau khi deploy.
- **Fix/Correct Flow**: Subject naming contract TRƯỚC (`cdc.cmd.{action}` + payload schema). Fire-and-forget: CMS publish không chờ Worker subscribe; JetStream retention guarantee no loss. FE polling: UI state machine handle `accepted→running→success|error|timeout`. Verify end-to-end sau khi all Muscle done.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi microservices refactor với async messaging (NATS, Kafka, RabbitMQ).
- **Tags**: #cdc #nats #async-decoupling #parallel-delegation #cross-service #subject-contract
- **Nguồn**: lessons.md [2026-04-20]

### [2026-04-06] Stream name normalization và Connection Status omission trong integration 3rd-party
- **Global Pattern**: `[Integration component A so sánh stream name từ registry X với stream name từ 3rd-party system Y]` → `[A dùng string equality trực tiếp mà không normalize]` → `[mismatch silent, command không thực thi, feature không hoạt động]`. **Đúng**: normalize tên về format chung (replace `-` → `_`) trước khi so sánh; đọc kỹ API docs về Master state chi phối (Connection.status) khi update stream state.
- **Bối cảnh (Trigger)**: Thao tác chuyển `export_jobs` sang inactive trên CMS không phản ánh lệnh tắt Replication trong Airbyte.
- **Root Cause**: (1) Tên trong Mongo/Airbyte dùng `export-jobs` (dash) nhưng Registry lưu `export_jobs` (underscore) — so sánh `==` thất bại silent. (2) Bỏ sót Connection-level `status: inactive` khi unselect toàn bộ stream.
- **Fix/Correct Flow**: Normalize tên bảng về format chung (`strings.ReplaceAll(name, "-", "_")`) trước so sánh. Khi gửi payload update state sang 3rd-party, kiểm tra API docs về Master state dependency.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi integration với Airbyte, Fivetran, Kafka Connect hoặc bất kỳ connector có dual naming convention.
- **Tags**: #cdc #integration #normalization #stream-name #api-completeness #silent-drop
- **Nguồn**: lessons.md [2026-04-06]

---

## 5. Config & Environment — Env vars, DSN/Secret, Fallback, Docker/K8s

_Bài học về cấu hình & môi trường: env vars, resolve DSN/secret, fallback merge, docker-compose/k8s, .env._ — **16 pattern**

### [2026-05-26] Legacy single-config gate vô hiệu hóa toàn bộ feature sau khi migrate sang multi-source registry
- **Global Pattern**: `[Init code A gate feature-F construction bằng legacy single-config field C1]` sau khi `[service migrate sang V2 per-source registry R]` → `[F không bao giờ được construct, scheduler log "skipped (C1 missing)" mỗi tick, operator confusion — registry đầy đủ nhưng feature dead]`. **Đúng**: Feature F init unconditionally; chỉ subsystem cần single-default C1 mới gate bởi cfg; lazy resolve per-source từ V2 registry với fallback defaultC1.
- **Bối cảnh (Trigger)**: Trong deployment V2-only (không set cfg.C1.URL vì đã có per-source registry), feature F chết âm thầm — log ghi "skipped (C1 not configured)" mỗi tick dù V2 registry đã đủ thông tin.
- **Root Cause**: Init code cũ vẫn dùng `if cfg.C1.URL != "" { initF(...) }` để gate toàn bộ init feature F; sau migration sang V2, field không được set → F không bao giờ được khởi tạo.
- **Fix/Correct Flow**: Tách init: F init unconditionally, nhận `perSourceResolver` + nullable `defaultC1`. Populate per-source identity từ V2 registry. Hard-assert khi cả `entry.SourceURI=="" && defaultC1==nil`. Subscriber luôn register, trả structured error khi service-instance nil.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ service migrate single-config → multi-source registry còn legacy gate (CDC, ETL, API gateway, notification service).
- **Tags**: #config #migration #silent-drop #root-cause #coupling #observability
- **Nguồn**: lessons.md [2026-05-26]

### [2026-05-20] Vite placeholder leak vào backend config gây silent infra failure
- **Global Pattern**: `[Frontend F] build mà không resolve env-var → gửi literal placeholder string (\_\_VITE_X\_\_, import.meta.env.X) làm field value` lên `[backend B với "inject if missing" defaults]` → `[downstream infra (Kafka/Redis/HTTP) silently fail vì connect tới literal placeholder hostname/topic]`. **Đúng**: backend PHẢI force-overwrite infra config keys nó owns, KHÔNG trust FE-supplied values cho infra concerns; detection rule: grep Kafka/Redis/HTTP logs cho substrings khớp FE env var pattern.
- **Bối cảnh (Trigger)**: API call trả 200 OK nhưng downstream consumer subscribe/connect tới literal placeholder hostname/topic → `UNKNOWN_TOPIC_OR_PARTITION` / `connection refused`; grep matches FE source code.
- **Root Cause**: Build-time env replacement của FE framework không được resolve trước khi value truyền vào backend; backend với "inject if missing" defaults respect placeholder string thay vì override bằng giá trị infra thực.
- **Fix/Correct Flow**: Backend force-overwrite signal topic, broker URL, schema registry URL — không dùng FE-supplied values; detection: grep log cho pattern `^__[A-Z]+_`, `^VITE_`, `^NEXT_PUBLIC_`.
- **Phạm vi (≥3 dự án?)**: Có — Vite, Next.js (NEXT_PUBLIC_*), CRA (REACT_APP_*), Vue (VUE_APP_*) — mọi framework với build-time env replacement.
- **Tags**: #config #vite-placeholder #silent-drop #env-vars #infra-config #backend-override
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] Confluent Hub catalog gap — fallback Maven Central manual install
- **Global Pattern**: `[Catalog A của vendor B] thiếu version [V] của artifact [X]` → `[nếu chỉ dùng catalog A thì bị block]`. **Đúng**: fallback Maven Central / official release URL của X, install thủ công vào plugin dir của B; không bị block bởi catalog gap; chú ý escape `$$` trong Docker Compose YAML heredoc.
- **Bối cảnh (Trigger)**: `confluent-hub install --no-prompt debezium/debezium-connector-mongodb:2.7.4` báo "Component not found"; Confluent Hub catalog có gap toàn bộ 2.6/2.7/2.8/2.9 — nhảy từ 2.5.4 lên 3.0.8.
- **Root Cause**: Confluent Hub publish chậm và không complete cho mọi Debezium release; coi Confluent Hub là single source of truth là sai assumption.
- **Fix/Correct Flow**: Lookup trên Maven Central `repo1.maven.org/maven2/io/debezium/debezium-connector-{name}/{VERSION}.Final/...plugin.tar.gz`; thay `confluent-hub install` bằng `curl + tar -xzf` vào `$CONNECT_PLUGIN_PATH`; dùng `$$` trong Compose YAML để tránh interpolation.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ plugin marketplace nào có catalog gap (npm private registry, Helm chart repo thiếu version, VS Code marketplace).
- **Tags**: #config #confluent-hub #debezium #manual-install #maven-central #plugin-distribution
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-20] URL double-`?` khi composing dispatch + poll layer tự append query string
- **Global Pattern**: `[Layer A] bake query string `?k1=v1` vào URL rồi giao cho [layer B] blind-concatenate thêm `?k2=v2`` → `[URL có 2 `?`; server parse nhầm value của k1; một hoặc cả hai param bị dropped/corrupted; silent business error]`. **Đúng**: caller KHÔNG bake query string vào URL nếu layer B có cơ chế params (URLSearchParams, axios params); truyền extra params qua tham số chuyên biệt; nếu bắt buộc inline → layer B PHẢI detect và switch separator `url.includes('?') ? '&' : '?'`.
- **Bối cảnh (Trigger)**: `useScanFields` thêm `?binding_id=` vào `statusEndpoint`; hook `useAsyncDispatch` tự append `?subject=...&since=...` → URL `…/dispatch-status?binding_id=4?subject=…`; server đọc `binding_id="4?subject=scan-fields"` → backend coi như không có binding_id → 409 ambiguous → FE poll mãi spinner pending.
- **Root Cause**: Layer A và Layer B đều tự quản lý query string construction mà không có convention; blind string concatenation không check existing `?`.
- **Fix/Correct Flow**: Truyền `binding_id` qua `statusParams` object chuyên biệt cho hook; hook merge an toàn vào URLSearchParams; smoke test bằng log URL cuối trước khi gửi.
- **Phạm vi (≥3 dự án?)**: Có — axios interceptor thêm tenant_id, microservice gateway thêm trace_id vào downstream URL đã có query string.
- **Tags**: #config #url-composition #double-query #silent-drop #axios #frontend
- **Nguồn**: lessons.md [2026-05-20]

### [2026-05-19] Config-driven cross-DB writes phải pre-flight verify cả hai service trỏ cùng identity DB
- **Global Pattern**: `[Service A reads từ DB D-A và Service C writes vào DB D-C, cả hai labeled với cùng role name "shadowDb"]` lên `[distributed multi-service config]` → `[A và C silently drift; observers thấy contradiction A.count=0 vs C.success; mất hàng giờ debug vì mọi service đều log success]`. **Đúng**: Boot-time pre-flight: mỗi service log `<role>=<host>:<port>/<db>`; smoke test CI: 1 service write sentinel row vào "shadow", service kia read lại → fail fast nếu config drift.
- **Bối cảnh (Trigger)**: Worker config `shadowDb` trỏ nhầm vào cùng instance với `systemDb`; FE backend config `shadowDb` trỏ đúng shadow DB khác. Worker ALTER thành công 19 columns nhưng FE đọc shadow đúng → 0 column visible → user thấy "rows_affected=0". Diagnostic chain mất 3 giờ.
- **Root Cause**: 2 service share resource role name với INDEPENDENT connection strings; không có cross-service invariant check; mọi service log "success" với DB riêng của mình nên không signal mismatch.
- **Fix/Correct Flow**: Boot-time pre-flight log role→DSN cho mỗi service; centralized monitor compare pairs across services cho cùng role; CI smoke test writer→reader cross-service validation.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project distributed có shared resource role name với independent connection strings.
- **Tags**: #config #observability #root-cause #cdc
- **Nguồn**: lessons.md [2026-05-19]

### [2026-05-19] Worker-side overlay map keyed by stable logical code cho per-environment URI override không cần DB write
- **Global Pattern**: `[Service A reads field F từ shared registry R cho component X]` lên `[multi-environment deployment với shared DB config]` → `[Giá trị F không thể chạy đúng trong E1 (dev) nhưng E2 (prod) cần giữ F nguyên; mỗi lần admin update UI sẽ overwrite override]`. **Đúng**: Thêm overlay map M keyed bởi logical-stable identifier I (KHÔNG phải primary key) tại lớp A; check M TRƯỚC khi đọc F; identify ALL call sites translate R-row → connection; implement single helper Apply-Field-Override; log mỗi hit 1 dòng INFO.
- **Bối cảnh (Trigger)**: Admin nhập URI qua CMS UI (docker hostname, VPN IP) → dev worker không reach được. Cần override per-environment mà KHÔNG sửa DB (admin sẽ overwrite).
- **Root Cause**: Không có cơ chế override per-environment ở lớp service; mọi override đều phải qua DB; admin UI overwrite làm mất override của dev.
- **Fix/Correct Flow**: Overlay map M trong YAML config + per-key env var pattern; normalize keys lowercase tại CTOR; check M TRƯỚC khi đọc registry field; single helper không nhân bản logic; explore agent enumerate EVERY site đọc field trước; empty map = zero behavior change.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có shared config registry với per-environment overrides cần tách biệt khỏi shared state.
- **Tags**: #config #cdc #dry #coupling
- **Nguồn**: lessons.md [2026-05-19]

### [2026-05-18] Resolver secret reference chỉ handle một scheme trong khi field có nhiều convention nguồn khác nhau
- **Global Pattern**: `[Resolver R] cho identifier I kiểu "secret reference" chỉ handle một scheme trong khi I có thể mang nhiều schemes (literal-value, env-pointer, foreign-key, encrypted-blob) đến từ các writers khác nhau (seed scripts, UI flows, legacy mirrors)` lên `[connection config resolver]` → `[R fails cho mọi scheme khác; caller fallback về static/env config không set → "X not configured" runtime errors]`. **Đúng**: R implement multi-layer try-in-order: (1) detect literal usable value by scheme prefix; (2) resolve pointer schemes (env://, env:, vault:, secret:) by lookup; (3) derive usable value từ sibling structured fields (host/port/db/engine); (4) legacy decode (AES/KMS) as last resort.
- **Bối cảnh (Trigger)**: Worker báo `mongoURL not configured on worker; cannot introspect source` cho source vừa được add qua UI. Resolver chỉ biết 1 format (crypto.DecryptAES), nhưng `secret_ref` mang 4 convention khác nhau từ các nguồn khác nhau.
- **Root Cause**: Resolver assume một scheme duy nhất cho field đa dạng nguồn; không kiểm tra prefix/scheme trước khi decode.
- **Fix/Correct Flow**: Inventory toàn bộ writer paths vào field; mỗi scheme tách thành 1 pure helper testable; build-from-structured-fields LAYER là bắt buộc khi record có host/port/engine; unit test pure helper trực tiếp, cover từng scheme + missing-field edge case.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có connection config field nhận input từ nhiều nguồn (seed, UI, legacy, future hardening).
- **Tags**: #config #cdc #root-cause #coupling
- **Nguồn**: lessons.md [2026-05-18]

### [2026-05-15] Audit config phải verify cross-layer redundancy, không chỉ per-key DEAD
- **Global Pattern**: `[Agent A] audit X config layers L1/L2/L3 có fallback chain L1→L2→L3, chỉ verify per-layer "has-reader" mà không verify per-pair "has-overlap value/role" giữa các layer cùng chain` lên `[config audit report]` → `[layers REDUNDANT cùng giá trị bị classify là ACTIVE đơn lẻ; file config giữ noise duplicate; user phát hiện trước agent và mất trust]`. **Đúng**: Audit 2 pass — Pass 1: per-key DEAD theo grep caller; Pass 2: per-chain REDUNDANT theo trace fallback trong source + so sánh value các layer cùng chain. Report có bảng riêng "Redundancy collapse opportunities".
- **Bối cảnh (Trigger)**: User yêu cầu audit config-local.yml; agent chỉ flag 7 key DEAD per-key. User dán 3 block YAML cùng trỏ về cùng DB host và quát "mấy cái này là gì, sao nó giống nhau vậy, làm việc sao hời hợt".
- **Root Cause**: Audit pattern chỉ trả lời "key X có reader không" (per-key DEAD); không trả lời "key X có overlap với key Y trong fallback chain không" (cross-layer REDUNDANT). Code có chain `ControlPlane.URL ← SystemDB.URL ← cfg.DB.PgxDSN()`; cả 3 layer có reader hợp lệ nhưng trùng giá trị trên local rig.
- **Fix/Correct Flow**: Audit Pass 1 per-key DEAD (grep caller, mark DEAD/ACTIVE/ACTIVE-INDIRECT); Pass 2 trace fallback chain trong loader; so sánh value trong YAML target — nếu trùng → flag REDUNDANT; đề xuất collapse về layer thấp nhất.
- **Phạm vi (≥3 dự án?)**: Có — mọi project có config file với fallback chain (Viper, Cobra, env→file→default), DI containers, service registry có default-resolution chain.
- **Tags**: #config #dry #observability
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-05] Tách docker-compose project phải dùng external volume để bảo toàn data
- **Global Pattern**: `[Tách docker-compose project A thành A' + B (subset services move sang B)]` → `[Nếu B khai báo volume bình thường, compose tạo volume rỗng mới `B_<vol>`, data từ `A_<vol>` bị mất]`. **Đúng**: declare volume trong B với `external: true, name: A_<vol>` → B mount volume vật lý cũ của A; data preserved; khi chạy môi trường sạch mới thì bỏ `external: true`.
- **Bối cảnh (Trigger)**: Phase B5 split compose 16 services thành core (10) + dev (6); dev project đổi project name mới → compose chuẩn bị tạo volume namespace mới → data 6 ngày test sắp mất.
- **Root Cause**: Docker-compose namespace volume names theo project (`<project>_<volume_decl>`); project mới = volume name mới = volume vật lý mới = data mất.
- **Fix/Correct Flow**: Declare moved volumes trong compose B với `external: true` + `name: <old_project>_<vol>`; verify `docker volume ls` trước split; `docker compose down` (không `-v`) → volumes survive; `docker compose up -d` 2 project mới → verify data count khớp.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi docker-compose project split/rename khi có stateful volumes (postgres, kafka, redis, elasticsearch data dirs).
- **Tags**: #docker-compose #volume #external #data-preservation #split-project #namespace #migration
- **Nguồn**: lessons.md [2026-05-05]

### [2026-05-05] Cross-repo relative path mount trong docker-compose sau split = coupling lén
- **Global Pattern**: `[Project B sau split reference asset của project A bằng path `../A/...` trong volume mount / build context / ConfigMap]` → `[Coupling lén vi phạm decoupling mục tiêu của split; break khi repo A move/rename]`. **Đúng**: B own toàn bộ asset cần cho B services; move (không copy) asset từ A sang B; mount bằng `./...` relative tới B; verify bằng `grep -rn '../<other-project-name>' <new-project>/` phải 0 hit.
- **Bối cảnh (Trigger)**: Phase B5.5 split compose; round 1 quên 2 init-script mount vẫn dùng `../centralized-data-service/deployments/...` từ compose mới — đè ngược coupling vừa tách.
- **Root Cause**: Developer chỉ update service definitions, không audit volume mount paths; relative cross-repo path syntactically valid nhưng semantically là coupling violation.
- **Fix/Correct Flow**: Move init scripts sang `cdc-docker-dev/init/`; đổi mount thành `./init/...`; chạy `grep -rn '../' <new-project>/` filter yml/Dockerfile → 0 cross-project hit; `docker compose config --quiet` không error.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho Dockerfile `COPY ../A/configs`, Helm values `hostPath: /repo/A/secrets`, Kubernetes ConfigMap source từ cross-namespace path, bất kỳ containerized project split nào.
- **Tags**: #split-project #decoupling #docker-compose #cross-repo-mount #anti-pattern #relative-path #coupling
- **Nguồn**: lessons.md [2026-05-05]

### [2026-05-05] `.env.example` phải actionable — env var thực sự, không phải prose comment
- **Global Pattern**: `[`.env.example` của service A chứa prose comment block thuần không kèm env var nào]` → `[User copy file xong không có gì useable; phải tự đọc và gõ connect string]`. **Đúng**: mỗi entry là env var thực sự copy-paste runnable (hoặc omit hoàn toàn); ≤1 dòng comment header identify service + key info; prose thuộc về README, không thuộc `.env.example`.
- **Bối cảnh (Trigger)**: Mongo block trong `.env.example` dùng 3-dòng comment verbose không có env var nào; user phải tự suy luận URL.
- **Root Cause**: Author treat `.env.example` như tutorial thay vì template-to-copy; prose comment không có giá trị actionable trong file config template.
- **Fix/Correct Flow**: Thay 3-dòng comment bằng `# header` + `MONGO_URL=mongodb://...`; mỗi service không expose env knob nhưng consumer cần → expose `<SERVICE>_URL=<connect-string>`; ngoại lệ duy nhất: 1-line security note đầu file.
- **Phạm vi (≥3 dự án?)**: Có — microservices `.env.example`, frontend `API_URL/CDN_URL`, CI/CD secrets template, bất kỳ project có environment configuration template.
- **Tags**: #config #env-example #documentation #actionable-config #copy-paste-friendly #dx
- **Nguồn**: lessons.md [2026-05-05]

### [2026-05-05] Dockerfile bake config-local.yml đơn lẻ = ship DEV creds lên prod image
- **Global Pattern**: `[Dockerfile X copy chỉ `config-local.yml` vào prod image Y]` → `[Prod runtime ship DEV creds, default secrets (change-me-in-production), dev pool sizes; image không deployable sạch lên multi-env]`. **Đúng**: `COPY config ./config` cả thư mục; runtime chọn file qua env (`cfgPath`); prod yml fields rỗng cho secrets, env override điền tại runtime; `validateConfig()` refuse placeholder khi `mode==production`; dùng `viper.AutomaticEnv()` + `SetEnvPrefix` + `BindEnv` — không hardcode `applyEnvOverrides`.
- **Bối cảnh (Trigger)**: Audit `cdc-auth-service/Dockerfile:12` phát hiện `COPY --from=builder /app/config/config-local.yml ./config/config-local.yml`; JWT secret = `change-me-in-production` sẽ vào prod image.
- **Root Cause**: Developer copy pattern từ local dev convenience sang Dockerfile production mà không xem xét multi-env deployment; chỉ 1 file config = chỉ 1 environment per image build.
- **Fix/Correct Flow**: `COPY config ./config` (cả thư mục); thêm `config-production.yml` với fields rỗng; `validateConfig()` fail-fast tại boot nếu required fields empty hoặc placeholder; detection: `grep -n "COPY.*config-local" Dockerfile*` → red flag.
- **Phạm vi (≥3 dự án?)**: Có — Go/viper services, Node/dotenv services, Java/Spring `application-{profile}.yml`, bất kỳ containerized service có multi-env config.
- **Tags**: #docker #config-management #env-override #prod-readiness #security #viper #anti-pattern
- **Nguồn**: lessons.md [2026-05-05]

### [2026-05-05] Go service không dùng godotenv — `.env.example` là dead weight
- **Global Pattern**: `[Repository R có `.env.example` cho Go service S không import godotenv library]` + `[compose có `${VAR:-default}` cho mọi var]` → `[`.env.example` là dead weight; user copy `.env` không có effect; confused]`. **Đúng**: audit (1) grep godotenv hit? (2) compose có defaults? (3) docs reference `.env.example`? → nếu NO/NO/NO thì DELETE; Go cần explicit env loading (shell export, compose `env_file:`, k8s `envFrom`), không auto-load như Node.
- **Bối cảnh (Trigger)**: `cdc-auth-service`: `grep godotenv` 0 hit, `go.mod` không import dotenv, compose có `${VAR:-default}` cho 3 DB vars → `.env.example` không có tác dụng.
- **Root Cause**: Template `.env.example` copy từ Node project sang Go project mà không check runtime loading mechanism; Go binary đọc YAML qua viper, không auto-load `.env`.
- **Fix/Correct Flow**: Chạy 3-bước decision tree: dotenv loader? → compose defaults? → docs reference? → nếu dead → DELETE file; document env loading method thực tế trong README.
- **Phạm vi (≥3 dự án?)**: Có — Go monorepo multi-service, Node→Go migration, static-binary deploy trên k8s/ECS, bất kỳ polyglot repo có nhiều service với runtime environment loading khác nhau.
- **Tags**: #go #config #env-loading #dead-files #dx #anti-pattern #documentation
- **Nguồn**: lessons.md [2026-05-05]

### [2026-05-05] Validation phải chạy TRƯỚC fallback merging trong config pipeline
- **Global Pattern**: `[Pipeline config có sequence: read input I → apply fallbacks/defaults D → validate V]` → `[V thấy `I ∪ D` (merged): empty user-intent bị lấp bằng derived value → false-positive PASS; app boot OK rồi crash runtime]`. **Đúng sequence**: ReadConfig → Unmarshal → applyEnvOverrides → **validateConfig** → applyFallbacks; validate thấy CHỈ user intent → fail-fast tại boot khi input rỗng.
- **Bối cảnh (Trigger)**: `validateConfig` gặp false-positive PASS khi `cfg.DB.PgxDSN()` trả về `"postgres://:@:0/..."` non-empty (literal sprintf không bao giờ empty) sau `applyDBFallbacks` set `SystemDB.URL` → validator thấy non-empty → PASS sai → crash khi connect.
- **Root Cause**: `applyFallbacks` chạy BEFORE `validateConfig`; getter helper dùng `fmt.Sprintf` trả non-empty string dù inputs rỗng — masking empty intent.
- **Fix/Correct Flow**: Reorder: `applyEnvOverrides` → `validateConfig` → `applyFallbacks`; getter an toàn trả `("", false)` hoặc `(nil, error)` khi inputs missing; test config rỗng hoàn toàn → validateConfig phải trả error.
- **Phạm vi (≥3 dự án?)**: Có — Go/viper, Node/convict, Python/pydantic, Java/Spring; ETL pipeline validate raw input trước transforms; API request validation trước server-side defaults; form validation UI trước placeholder apply.
- **Tags**: #config #validation #order-matters #fail-fast #anti-pattern #config-management #empty-input
- **Nguồn**: lessons.md [2026-05-05]

### [2026-04-17] Upgrade version ≠ more stable — regression across tool versions
- **Global Pattern**: `[Agent A upgrades tool B từ V_old lên V_new]` → `[V_new chưa test với data pattern của A (nested union types, complex schemas, vendor-specific envelope)]` → `[regression Y trên pattern A support]`. **Đúng**: khi tool bị lỗi → test 1 step back (V-1 minor) TRƯỚC KHI jump forward; decision tree: current broken → older patch → older minor → latest stable → RC. Pin version working + note reason.
- **Bối cảnh (Trigger)**: Redpanda Console v2.8.1 báo `INVALID_TOPIC_EXCEPTION` cho mọi topic. Upgrade → v3.1.2 → panic `nil pointer`. Downgrade v2.7.2 → works. 2 phiên bản mới hơn đều regression với Debezium MongoDB Avro envelope.
- **Root Cause**: "upgrade = better" là giả định sai. Regression rate cao cho nested union types (Avro `["null", "string"]`), library deserializer từ complex schemas, Debezium envelope patterns.
- **Fix/Correct Flow**: Version matrix test: tool bị lỗi → test V-1 minor TRƯỚC TIÊN. Decision tree sequential: older patch → older minor → latest stable → RC. Khi tìm được version working: pin trong docker-compose + note reason trong comment.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project dùng vendor tools: Kafka UI, Debezium, CDC connectors, schema registry.
- **Tags**: #config #version-regression #downgrade #vendor-bug #avro #debezium #pinning
- **Nguồn**: lessons.md [2026-04-17]

### [2026-02-26] Cập nhật nhầm file config (Path Management failure)
- **Global Pattern**: `[Agent A] cập nhật [config file tại path X]` → `[thay vì file gốc tại path Y; system không nhận thay đổi; hành vi không thay đổi]`. **Đúng**: luôn dùng ls -la và xác minh absolute path trước khi sửa file hệ thống quan trọng.
- **Bối cảnh (Trigger)**: Brain cập nhật file config tại đường dẫn A thay vì đường dẫn B (file gốc của hệ thống).
- **Root Cause**: Path Bias — ưu tiên files trong cây thư mục hiện tại mà không kiểm tra env var hoặc chỉ định của User.
- **Fix/Correct Flow**: Luôn verify absolute path bằng ls -la trước khi sửa file config hệ thống.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có nhiều config file ở nhiều paths.
- **Tags**: #config #process-governance #carelessness #verification #root-cause
- **Nguồn**: lessons.md [2026-02-26]

---

## 6. Serialization & Type — BSON/Extended-JSON, Cast, Type/Form Drift, Identifier

_Bài học về serialize/kiểu dữ liệu: BSON/Extended-JSON, cast expression, form drift, dual-stack routing, migrate identifier._ — **13 pattern**

### [2026-06-05] Transform MongoDB→relational không coerce theo kiểu cột → upsert chết (chỉ lộ khi chạy data thật)
- **Global Pattern**: `[Transform A đọc dữ liệu nguồn MongoDB đã lưu Extended-JSON ở shadow X]` rồi `[ghi thẳng giá trị vào cột relational Y mà không coerce theo kiểu đích]` → `[upsert chết: {"$date":..}→22007 timestamp, {"$oid":..}/sub-doc vào jsonb→22P02 invalid json, epoch-ms số trần→"cannot find encode plan" int→timestamp]`. **Đúng**: TRƯỚC khi bind, (1) unwrap ext-JSON scalar (`$date/$oid/$numberLong/$numberInt/$numberDouble/$numberDecimal`); (2) coerce THEO `target.data_type`: json/jsonb→`json.Marshal` ra JSON text hợp lệ (kể cả scalar→quoted), timestamp+number→`time.UnixMilli/Unix` (auto ms/s theo độ lớn), composite vào cột scalar→JSON text.
- **Bối cảnh (Trigger)**: End-to-end sync b3 (Mongo source) Shadow→Master: build=0 + unit cũ PASS nhưng RunNow thật fail từng đợt 22007→22P02→encode-plan; mỗi lần chỉ lộ 1 lớp lỗi tiếp theo. Degraded guard (scanned>0 & ghi 0 → failed) bắt đúng cả 3, không báo success giả.
- **Root Cause**: `extractColumns` lấy value qua gjson rồi gán thẳng; ext-JSON & epoch number không khớp encode-plan của pgx cho cột timestamp/jsonb. Path batch khác (SQL `_raw_data->...`) đã unwrap nhưng path transmuter (Go) thì chưa — **fix 1 path không tự lan sang path khác**.
- **Fix/Correct Flow**: `unwrapMongoExtJSON` + `coerceForColumn` type-aware (mirror logic SQL-side); unit test 16 case (ext-json/oid/number/epoch ms+s/jsonb/composite) + **exercise** đối soát count nguồn=đích (454=454, distinct chống trùng) + spot-check row epoch-ms ra timestamp thật. **Meta**: build/unit PASS ≠ đúng; chỉ data thật end-to-end mới lộ type-drift → luôn chạy 1 happy-path thật với dữ liệu nguồn thật trước khi báo Done (Rule 16 G3/G6).
- **Phạm vi (≥3 dự án?)**: Có — mọi pipeline MongoDB→SQL (CDC, ETL, sync), Debezium Mongo, BigQuery/Snowflake loader đọc BSON ext-JSON.
- **Tags**: #serialization #type #extended-json #mongodb #pgx #cast #exercise-driven #root-cause #type-coercion
- **Nguồn**: lessons.md [2026-06-05]

### [2026-06-02] Mất fallback behavior khi chuyển identifier từ string sang integer trong môi trường test
- **Global Pattern**: `[Shared service tiện ích A] chuyển identifier lookup từ [string X] sang [int Y]` → `[short-circuit nil khi ID<=0 phá hủy default/fallback policy; test suite fail do DB==nil]`. **Đúng**: khi ID không hợp lệ hoặc bằng 0, vẫn khởi tạo map và nạp default masks/fallback rules thay vì return nil ngay.
- **Bối cảnh (Trigger)**: Sau khi chuyển registry masking từ string-based sang int64 shadow_binding_id, nhiều test case bị lỗi masking verification do service trả về map rỗng và bỏ qua mã hóa.
- **Root Cause**: DB-backed lookup bị vô hiệu hóa khi db==nil trong test; code mới short-circuit return nil ngay khi bindingID<=0 thay vì nạp defaultMasks như logic string cũ.
- **Fix/Correct Flow**: Sửa hàm lookup để khi bindingID<=0 (fallback/legacy/test mode) vẫn khởi tạo map và nạp default masks với HMAC strategy, đảm bảo bảo mật tối thiểu.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho bất kỳ service dùng chung (Auth, Routing, Feature Flag) khi chuyển đổi kiểu identifier.
- **Tags**: #identifier-migration #fallback-behavior #testing-environment #security-by-default #serialization #migration #short-circuit
- **Nguồn**: lessons.md [2026-06-02]

### [2026-05-22] Truyền transport-path thô vào identity field có constraint gây overflow và break literal-match
- **Global Pattern**: `[Transport layer A propagates raw transport-path B]` lên `[persisted identity field X có VARCHAR constraint / literal-compare]` → `[column overflow + break downstream literal-match semantics, 0 rows written]`. **Đúng**: Transport layer dùng SHORT STABLE identifier (vd `debezium`, `snapshot:v2`); transport metadata (topic, partition) đẩy qua kênh riêng (headers/subject parameter), không nhét vào identity field có constraint.
- **Bối cảnh (Trigger)**: CDC realtime path đẩy `record.Source = "/kafka/cdc.goopay.X.Y"` (41 ký tự) vào cột `_source VARCHAR(20)` → SQLSTATE 22001, 0 row ghi vào shadow table; đồng thời phá literal-match trong LWW guard SQL.
- **Root Cause**: Transport layer dùng raw transport-path dạng URL (41 ký tự) thay vì short stable identifier cho identity field có constraint VARCHAR(n) và downstream literal-compare.
- **Fix/Correct Flow**: Dùng short stable identifier (vd `debezium`, `snapshot:v2`) cho identity field. Audit mọi envelope mới: kiểm tra constraint trên field, downstream literal-match, và source identifier có dùng transport-specific format không.
- **Phạm vi (≥3 dự án?)**: Có — event-driven systems (Kafka, NATS, RabbitMQ), audit log actor field, multi-tenant tenant_id, HTTP middleware client_name.
- **Tags**: #serialization #identifier-migration #silent-drop #cdc #kafka #schema-drift #literal-match
- **Nguồn**: lessons.md [2026-05-22]

### [2026-05-21] Silent-drop do field-routing mismatch trong dual-stack FE-BE (legacy bridge vs new endpoint)
- **Global Pattern**: `[Frontend caller A] route toàn bộ payload qua [legacy endpoint X]` → `[field mới thuộc endpoint mới Y bị silent-drop vì X không biết field đó]`. **Đúng**: split payload theo field-ownership — field thuộc endpoint nào PATCH đến endpoint đó, không route toàn payload theo 1 cờ duy nhất.
- **Bối cảnh (Trigger)**: Thêm column V2-only vào table, expose qua endpoint mới; form FE gửi payload chung; khi save trên row có legacy bridge toàn bộ payload đi qua legacy endpoint → handler cũ không biết field mới → silent drop.
- **Root Cause**: Logic routing all-or-nothing trên toàn payload không phân biệt field nào thuộc table nào; field mới ở V2 không thể đi qua endpoint legacy.
- **Fix/Correct Flow**: Tách payload theo field-ownership — V2-exclusive fields luôn PATCH V2 endpoint; phần còn lại theo routing cũ; 2 PATCH tuần tự share try/catch.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho microservice dual-stack, event v1/v2, API migration với backward-compat.
- **Tags**: #silent-drop #dual-stack #legacy-bridge #routing #field-ownership #serialization #cdc
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-21] Serialization-form drift: cast SQL chỉ cover 1 representation cho cùng 1 logical type
- **Global Pattern**: `[Cast helper A] chỉ enumerate [1-2 form serialization của logical type X]` → `[fail trên rows cũ do encoder khác tạo form khác; backward-incompat]`. **Đúng**: enumerate trước TẤT CẢ form khả dĩ của X; đặt branch đặc thù (object Extended-JSON) TRƯỚC branch fallback (text cast).
- **Bối cảnh (Trigger)**: Batch transform fail SQLSTATE 22007 trên rows cũ; cùng table cùng phút transform per-row thành công; cùng logical type BSON Date tồn tại ở 3 form khác nhau trong JSONB.
- **Root Cause**: Cast helper được viết khi chỉ thấy 1-2 form trong dev/test data; nhiều encoder (Go driver, canonical Extended-JSON, Debezium) ghi cùng logical type xuống storage ở multi form; helper không enumerate đủ → backward-incompat trên rows cũ.
- **Fix/Correct Flow**: Mở rộng CASE bao trùm tất cả form của cùng logical type: number→to_timestamp, object{$date:string}→TIMESTAMPTZ, object{$date:{$numberLong}}→to_timestamp, ELSE→TIMESTAMP; fix SQL-side không cần write-time migration.
- **Phạm vi (≥3 dự án?)**: Có — BSON/Extended-JSON, NDJSON từ multiple producer, Avro union, Protobuf any.
- **Tags**: #serialization #cast #bson #extended-json #postgres #jsonb #backward-compat #schema-drift
- **Nguồn**: lessons.md [2026-05-21]

### [2026-05-19] Identity key collision khi nhiều connector dùng cùng (key, sub-key) — cần identity-tier discriminator
- **Global Pattern**: `[Entity A có UNIQUE identity_key = f(B, C)] lúc đầu chỉ có 1 X cho mỗi (B, C)` lên `[multi-X scenario (2 connector cùng db, table)]` → `[identity_key collision merge tất cả X vào 1 A row; downstream resource Y bị share/corrupt]`. **Đúng**: Thêm discriminator X_code (stable, không phải numeric id) vào identity → identity_key = f(X_code, B, C); resource Y derived từ identity_key phải embed X_code; backwards-compat: column nullable + first-wins fallback resolver khi X_id IS NULL.
- **Bối cảnh (Trigger)**: Multi-X API trả 2 X rows nhưng aggregate endpoint chỉ trả `total: 1`; 2 connector cùng (db, table) → tạo 1 Postgres schema, 1 cache entry; metadata của Y override giữa 2 connector → corruption risk.
- **Root Cause**: Identity key được thiết kế cho cardinality 1:1; khi multi-X xuất hiện, identity key collision khiến tất cả X collapse vào 1 entity; không có discriminator để phân biệt.
- **Fix/Correct Flow**: Migration chain an toàn: ADD COLUMN nullable + FK + index; Backfill first-wins + audit RAISE NOTICE; Relax UNIQUE old → ADD UNIQUE include X_code. L0 Input: model có field X_id nullable; L1 Identity: normalized_key bao gồm X_code; L2 Resolver: priority X_id explicit → fallback first-wins; L3 Derived Resource: embed X_code; L4 Cache: emit BOTH legacy keys + connection-aware variants.
- **Phạm vi (≥3 dự án?)**: Có — multi-tenant SaaS (tenant_id discriminator), multi-region replication (region_code), multi-source logical entity.
- **Tags**: #serialization #cdc #migration #schema-drift #dry
- **Nguồn**: lessons.md [2026-05-19]

### [2026-05-19] FE dropdown pick row từ table X, gửi X.id cho BE expect FK đến table Y
- **Global Pattern**: `[FE dropdown A pick row B từ list-endpoint trả row-id của table X (X.id)] gửi cho [BE-A expect FK trỏ đến table Y (Y.id)]` lên `[V1/V2 namespace với mirror sync by name/code]` → `[X.id ≠ Y.id (2 auto-increment độc lập); FE gửi đúng cú pháp nhưng sai semantic; FK reference broken hoặc resolver fallback ngầm → identity collapse]`. **Đúng**: BE-A accept BOTH Y.id (preferred) AND X.code (string identifier, FE-friendly fallback); BE-A resolver maps code → Y.id TRƯỚC khi persist; FE send code (stable, human-readable), không phải id (auto-increment, table-scoped).
- **Bối cảnh (Trigger)**: User tạo 2 records mong đợi 2 distinct rows; thấy 1 row được update 2 lần. Hoặc duplicate-key error vì wrong FK collides với unrelated row.
- **Root Cause**: FE dropdown bind value của widget không qua `<Form.Item name>` → value không vào submit payload; BE resolver silent first-wins khi payload thiếu FK → 2 entities collapse vào 1.
- **Fix/Correct Flow**: BE-A: accept BOTH Y.id AND X.code; resolver map code→Y.id trước khi persist; FE-A: bind dropdown qua `<Form.Item name>` (Ant Design); gửi code (stable string) không phải id (auto-increment).
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có FE/BE dùng khác namespace/table cho cùng entity (V1/V2, legacy/new, mirror/authoritative).
- **Tags**: #serialization #silent-drop #cdc #schema-drift
- **Nguồn**: lessons.md [2026-05-19]

### [2026-05-19] Discriminator chỉ được thêm vào write path — read stack không expose D gây UI collapse 2 entity thành 1
- **Global Pattern**: `[Backend đã thêm discriminator D vào identity tier của entity E ở write path + storage] nhưng [read stack không audit cụ thể: SQL projection không expose D, DTO không có field D, FE type không khai báo, UI grouping vẫn key theo D' cũ]` lên `[read API + UI layer]` → `[write-side fix đúng nhưng UX layer "lừa" user thấy như chưa fix: 2 entity collapse vào 1 panel dù storage đúng 2 row]`. **Đúng**: Sau khi thêm D vào write/identity, audit ALL read endpoints; mỗi level: SQL JOIN table chứa D, project D qua COALESCE, thêm field D với omitempty, FE type optional, UI grouping key = ${D}::${D'}.
- **Bối cảnh (Trigger)**: User nói "fe vẫn thiếu" sau khi write path đã verified pass. Backend đã fix UNIQUE composite và identity rebuild, nhưng UI vẫn hiển thị "1 objects" khi mong đợi 2.
- **Root Cause**: Fix write path (UNIQUE composite, identity rebuild) rồi báo "done" mà không update read projection — SQL SELECT không expose D, DTO không có field, FE merge 2 entity thành 1 panel.
- **Fix/Correct Flow**: Audit ALL read endpoints sau khi thêm discriminator; SQL: LEFT JOIN (không INNER JOIN để legacy null FK không biến mất); DTO: field mới với omitempty; FE: type optional + grouping key ${D}::${D'}; test thủ công: tạo 2 entity với D_1 và D_2 cùng D' → list phải trả 2 row riêng.
- **Phạm vi (≥3 dự án?)**: Có — multi-tenant SaaS, multi-region, multi-environment cùng schema.
- **Tags**: #serialization #silent-drop #cdc #schema-drift #dry
- **Nguồn**: lessons.md [2026-05-19]

### [2026-05-11] JSONB pre-marshal trap: tầng generate không được pre-marshal trước tầng persist
- **Global Pattern**: `[Component A pre-marshal value thành []byte X]` trước khi `[truyền sang JSON-aware layer B]` → `[B áp dụng json.Marshal(X) → Go stdlib base64-encode []byte → JSON column chứa "<base64>" string thay vì nested object]`. **Đúng**: tầng generate giá trị cho JSON/JSONB column phải trả về native Go type (map[string]interface{}, []interface{}, primitive); để tầng persist marshal cuối cùng; nếu cần inject raw JSON dùng json.RawMessage(bytes) hoặc string(bytes), KHÔNG pass []byte literal.
- **Bối cảnh (Trigger)**: ETL pipeline với DynamicMapper pre-marshal value thành []byte trước khi SchemaAdapter persist vào JSONB column; column chứa chuỗi base64 thay vì JSON object.
- **Root Cause**: Go stdlib json.Marshal encode []byte thành base64 string, không inject raw JSON; double-marshal qua 2 tầng tạo ra base64-wrapped JSON thay vì nested object.
- **Fix/Correct Flow**: Detect: JSONB column chứa chuỗi base64 (alphanumeric + = padding, decode được ra JSON) → trace lên xem nguồn nào trả []byte; fix: trả native Go type từ tầng generate; dùng json.RawMessage nếu cần inject raw JSON.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ ETL extractor→transformer→loader; API responder trả JSON; bất kỳ pipeline có 2+ tầng marshal.
- **Tags**: #serialization #silent-drop #migration #config #type-conversion #dry #observability
- **Nguồn**: lessons.md [2026-05-11]

### [2026-05-06] Cast từng positional `?` trong CASE expression khi dùng GORM/pgx prepared statement
- **Global Pattern**: `[Driver D với prepared statement P] cast outer` lên `[CASE expression chứa positional ? so sánh column kiểu A]` → `[param types lệch theo column-A; outer cast không sửa được inference; encoding failure khi caller truyền type-B]`. **Đúng**: cast từng positional ngay tại branch: `WHEN ?::A THEN ?::B`, `ELSE ?::B END`; outer cast chỉ cho final result type.
- **Bối cảnh (Trigger)**: Build SQL động với `CASE column WHEN ? THEN ? ... END` qua GORM/pgx prepared statement; caller truyền int64 vào slot bị infer là text → `operator does not exist` hoặc `unable to encode`.
- **Root Cause**: pgx/GORM prepared-statement type inference resolve param types TRƯỚC khi outer cast áp dụng; `WHEN ?` so với cột text → propagate text type sang toàn bộ positional trong CASE; outer cast chỉ chuyển kiểu kết quả sau khi evaluated.
- **Fix/Correct Flow**: Thêm `::int` (hoặc type cần thiết) trực tiếp sau mỗi `?` trong THEN/ELSE branch; test integration với real Postgres (mock-DB không phát hiện vì khác driver type-inference).
- **Phạm vi (≥3 dự án?)**: Có — Go GORM/pgx, JDBC Java/Kotlin, Python psycopg2/asyncpg với prepared mode qua pgbouncer.
- **Tags**: #serialization #postgres #prepared-statement #type-inference #case-expression #gorm #pgx
- **Nguồn**: lessons.md [2026-05-06]

### [2026-04-28] GORM Raw().Scan không hỗ trợ nested struct — invalid field error
- **Global Pattern**: `[Response struct A chứa nested sub-struct B (không phải embedded)]` với `[caller dùng Raw().Scan(&[]A)]` → `[runtime "invalid field found for struct" error Y]`. **Đúng**: định nghĩa flat scan struct C với mọi field tag `gorm:"column:..."`, Scan vào `[]C`, sau đó transpose tay từ C sang A (set field-by-field, gắn B{...} vào A).
- **Bối cảnh (Trigger)**: Endpoint `/api/worker-schedule` trả "invalid field found for struct" vì struct có nested `Scope WorkerScheduleScope`; SELECT projects flat columns → GORM không tự lan vào sub-struct.
- **Root Cause**: GORM Raw().Scan dùng reflection flat — không tự map vào nested/non-embedded struct fields; chỉ embedded struct (anonymous field) mới được tự map.
- **Fix/Correct Flow**: Tạo flat scan struct C với column tags; Scan(&[]C); loop transpose C → A với B{...} assignment; nếu đổi sang Find(&dst) (model query) thì GORM tôn trọng embed/Preload cho associations chính thức.
- **Phạm vi (≥3 dự án?)**: Có — mọi GORM project có DTO API trả về sub-struct/scope group với JOIN raw SQL; cũng đúng cho database/sql Scan tổng quát.
- **Tags**: #serialization-type #gorm #nested-struct #raw-scan #type-conversion #silent-drop
- **Nguồn**: lessons.md [2026-04-28]

### [2026-04-20] Hard-coded field name trong cross-store sync breaks on schema drift
- **Global Pattern**: `[Cross-store sync/recon component A hard-codes field-name B từ canonical convention]` → `[collection X dùng convention khác (camelCase, created_at, lastUpdatedAt)]` → `[filter matches 0 rows → reports "source empty" falsely → operator đổ lỗi cho scheduler Z]`. **Đúng**: registry-first (per-table config column `timestamp_field`); fallback graceful (ObjectID timestamp); observability (surface chosen path to UI).
- **Bối cảnh (Trigger)**: Reconciliation reports `source_count=0` cho `refund_requests`, `export_jobs`. Schedule DID fire nhưng Mongo filter `bson.M{"updated_at": ...}` trả 0 vì collections dùng `createdAt` + `lastUpdatedAt`. Typed struct decode `bson:"updated_at"` decode missing field thành zero-value silently.
- **Root Cause**: Hard-coded field name không match naming convention của collection thực tế. Typed struct decode zero-value trên missing field không có error → silent failure. Tests pass vì fixtures dùng canonical field.
- **Fix/Correct Flow**: Registry-first: per-table config `cdc_table_registry.timestamp_field` + whitelist validator. Fallback graceful: Mongo ObjectID carries unix seconds. Observability: surface `source_query_method` to UI. Prefer `bson.M` + explicit extraction khi field existence là semantic signal.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho Debezium source connectors, Airbyte incremental sync, ETL pipelines, webhooks với heterogeneous schemas.
- **Tags**: #serialization-type #schema-drift #field-naming #hardcoded-assumption #bson-decode #silent-drop #cross-store
- **Nguồn**: lessons.md [2026-04-20]

### [0000-00-00] Refactor map-based payload sang typed struct gây byte-level serialization order mismatch
- **Global Pattern**: `[Refactor A: map-based payload → B: typed struct với contract byte-identical]` → `[Go map serialize theo alphabetical key order; struct serialize theo field-declaration order; nếu field order ≠ key order → byte-different output dù cùng content]`. **Đúng**: declare struct fields theo alphabetical JSON tag order khi migrate từ map; hoặc dùng `jq -S` (sort keys) thay vì raw `cmp` nếu wire chỉ cần semantic-equivalent.
- **Bối cảnh (Trigger)**: Refactor CQRS Q-side handler chuyển từ `map[string]any` sang typed struct; test diff thấy size giống nhau nhưng `cmp -s` báo DIFF.
- **Root Cause**: Go `encoding/json` serialize map theo alphabetical key order (deterministic post Go 1.12); struct serialize theo field-declaration order; developer không align field order với key order khi viết struct.
- **Fix/Correct Flow**: Reorder struct fields theo `sort json_tags` ascending; comment ghi rõ "field order matches legacy map alphabetical serialization"; dùng `wc -c` + `cmp -s` + `jq -S` để phân biệt serialization order mismatch vs data drift.
- **Phạm vi (≥3 dự án?)**: Có — Python dict→dataclass, JS object→typed class, bất kỳ language nào migrate map sang typed struct với byte-identical wire contract.
- **Tags**: #serialization #refactor #json-serialization #byte-identical #order-matters #cqrs #type-conversion
- **Nguồn**: lessons.md [Lesson #1294]

---

## 7. Testing & Verification — Exercise-driven, PASS criteria, Test uplift, Build≠Test

_Bài học về kiểm thử & xác minh: exercise-driven, tiêu chí PASS thực chất, nâng cấp test, build pass ≠ test pass._ — **24 pattern**

### [2026-06-08] Verify/demo KHÔNG được đẩy hệ thống vào trạng thái approved/applied (phá workflow duyệt)
- **Global Pattern**: `[Để chứng minh feature A chạy, tự thực hiện bước cuối B (approve + apply DDL/ghi cột) thay user]` → `[hệ thống bị pollute: 1 field tự duyệt+vào bảng trong khi thiết kế yêu cầu PENDING-chờ-duyệt → user thấy state sai + mất niềm tin]`. **Đúng**: verify TÔN TRỌNG workflow — chỉ kiểm tới bước mà cơ chế cho phép (vd: rule tạo ra ở trạng thái pending đúng), HOẶC nếu phải chạy bước approve để chứng minh thì **revert ngay sau demo** + disclose; không để lại trạng thái user chưa đồng ý.
- **Bối cảnh (Trigger)**: chứng minh scan-array object → tự POST master-mapping-rule status='approved' + DDL cột `param_exportType` vào master aaa. User phản ứng "sao tự approve+DDL, phải pending chờ duyệt chứ".
- **Root Cause**: nhầm "end-to-end proof" = phải đẩy data tới đích, quên rằng đích cần HÀNH ĐỘNG DUYỆT của user (approval gate là 1 phần của đặc tả, không phải chướng ngại để bypass khi test).
- **Fix/Correct Flow**: revert (xoá master rule + DROP COLUMN), đưa mọi field về pending; verify dừng ở "rule tạo đúng + pending" + (nếu cần) demo extraction trên bản nháp rồi revert.
- **Phạm vi (≥3 dự án?)**: Có — mọi hệ có approval/review gate (CMS duyệt, feature-flag, IaC plan/apply, content moderation): test không được auto-approve/apply.
- **Tags**: #testing #verification #governance #approval-workflow #side-effect #revert-demo
- **Nguồn**: lessons.md [2026-06-08]

### [2026-06-08] Test mà PASS không phân biệt được với no-op → "verify" giả; phải dùng DELTA
- **Global Pattern**: `[Khẳng định feature A chạy được dựa trên đọc code + 1 test mà kết quả thành công TRÙNG với kết quả no-op (vd OCC idempotent → count không đổi dù transmute fan-out trả 0 row)]` → `[false-confidence: feature thực ra CHƯA BAO GIỜ chạy, user phát hiện khi dùng thật]`. **Đúng**: test phải tạo **DELTA chỉ xảy ra nếu feature thật sự hoạt động** — set giá trị SENTINEL/STALE ở đích rồi kích hoạt, kiểm tra nó BỊ THAY ĐỔI; và đọc log đếm `scanned/affected` (0 = không làm gì), KHÔNG chỉ đọc count tổng.
- **Bối cảnh (Trigger)**: Báo "realtime/post_ingest code-đúng" nhưng nó chết âm thầm 2 bug: (A) `HandleTransmuteShadow` match `connection_code='default'` trong khi connection thật `default_shadow` → fan-out 0 master; (B) gorm `_source_id = ANY(?)` với []string → `ANY('id')` chuỗi trần → `malformed array literal 22P02` → fetch fail. Test cũ publish transmute-shadow rồi check count đích = không đổi → tưởng OK (thực ra 0 master / fetch fail).
- **Root Cause**: tiêu chí PASS = "count không đổi" trùng với cả 2 trạng thái (đã-đồng-bộ và hoàn-toàn-không-chạy). Verify-by-reading không thay được exercise có delta.
- **Fix/Correct Flow**: (1) connection match coi `''`/`'default'` là wildcard; (2) `ANY(?)`→`IN (?)` (gorm expand []string đúng); (3) verify delta: set master.params='STALE' → trigger → quan sát `scanned=1 updated=1` + params về giá trị nguồn.
- **Phạm vi (≥3 dự án?)**: Có — mọi feature idempotent/upsert/sync (CDC, ETL, cache-warm, reconcile) nơi "không đổi" là kết quả hợp lệ.
- **Tags**: #testing #verification #false-positive #delta-test #idempotent #gorm #postgres-array #root-cause
- **Nguồn**: lessons.md [2026-06-08]

### [2026-06-05] "Live-verify" chạy nhầm binary CŨ vì tiến trình zombie giữ port → false-positive
- **Global Pattern**: `[Restart service A để verify code mới X]` nhưng `[tiến trình cũ chưa chết hẳn vẫn giữ port → binary mới fatal "bind: address already in use" rồi thoát, request kiểm thử rơi vào binary CŨ Y]` → `[verify chạy trên code cũ = báo PASS giả]`. **Đúng**: TRƯỚC khi exercise, xác nhận (i) PID MỚI sở hữu port (`lsof -iTCP:PORT -sTCP:LISTEN`), (ii) log KHÔNG có `fatal`/`address already in use`, (iii) có dòng "started" của lần khởi động mới — rồi mới publish/gọi.
- **Bối cảnh (Trigger)**: Verify SAFE-2 (invalidate cache): `kill`+restart worker rồi `nats pub cdc.cmd.master-create`, grep log không thấy dòng invalidate. Hóa ra worker mới crash bind 8082 (tiến trình cũ chưa free port trong 2s), health=200 là của binary CŨ, message bị worker cũ (chưa có SAFE-2) xử lý.
- **Root Cause**: Giả định "kill xong + health=200 = binary mới đang chạy". Health 200 chỉ chứng minh *một* tiến trình sống trên port, KHÔNG chứng minh đó là tiến trình MỚI. `go run` con có thể chưa chết / port còn TIME_WAIT.
- **Fix/Correct Flow**: Diệt sạch (parent go-run + child binary), chờ port free thật, restart, **grep `fatal` (phải rỗng) + lsof PID mới + dòng "started" mới** TRƯỚC khi exercise. Publish lại → log invalidate xuất hiện đúng (master_binding_id=10, had_entry=false).
- **Phạm vi (≥3 dự án?)**: Có — mọi service restart-rồi-test (Go/Node/Python/Java), CI smoke test, container redeploy.
- **Tags**: #testing #verification #process #zombie-process #port-binding #false-positive #root-cause
- **Nguồn**: lessons.md [2026-06-05]

### [2026-05-26] Integration test drop bảng shared gây crash service khác trong môi trường dev
- **Global Pattern**: `[Integration test suite A thực hiện DROP TABLE CASCADE trên bảng X dùng chung]` trong `[môi trường dev chia sẻ database]` → `[bảng thật bị xóa, downstream service crash với relation-not-exist error]`. **Đúng**: Dùng bảng có hậu tố `_test` hoặc database sandbox hoàn toàn độc lập (kết thúc bằng `_test`), không bao giờ DROP TABLE trên bảng cốt lõi của DB dùng chung.
- **Bối cảnh (Trigger)**: Chạy `go test` trong `centralized-data-service` gây mất bảng `cdc_system.failed_sync_logs` và các phân vùng trên database local `cdc_dw`, làm crash API `cdc-cms-service` khi query system health.
- **Root Cause**: Test file `partition_dropper_test.go` kết nối trực tiếp vào local database phát triển và thực hiện `DROP TABLE IF EXISTS ... CASCADE` để dọn dẹp — dùng tên bảng trùng với bảng thật mà không có cơ chế cô lập.
- **Fix/Correct Flow**: Đổi tên bảng trong test thành `failed_sync_logs_test`, ghi đè rules trong test chỉ query bảng test. Tuyệt đối không dùng DROP TABLE CASCADE trên bảng cốt lõi DB dùng chung — chỉ cho phép trên database sandbox độc lập.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project nào có integration test kết nối database chung (Go, Python, Java, Node.js).
- **Tags**: #testing #verification #integration-test #shared-database #root-cause #isolation
- **Nguồn**: lessons.md [2026-05-26]

### [2026-05-18] Chuyển validator sang permissive mode làm gãy unit test cũ expect blocking error
- **Global Pattern**: `[Agent A] thay đổi service validator từ strict (blocking error) sang permissive (log & metric only) mà không update assertions trong legacy unit tests expect blocking errors` lên `[validator behavior change]` → `[Unit tests fail dù system behavior đúng]`. **Đúng**: Khi refactor sang permissive mode, rà soát toàn bộ test suite liên quan; sửa test case mong đợi lỗi thành mong đợi nil; khởi tạo Mock Logger/Mock Metrics để verify drift vẫn được ghi nhận.
- **Bối cảnh (Trigger)**: Thay đổi validator sang permissive mode (chỉ log drift và metrics, không return error) làm gãy hàng loạt unit test cũ expect `err != nil`.
- **Root Cause**: Unit test được viết từ trước mong đợi validator trả về error cụ thể khi phát hiện schema mismatch; khi validator chuyển sang permissive mode trả về nil error, assertion `err != nil` fail.
- **Fix/Correct Flow**: Update unit test assertions để match permissive mode: expect nil error; assert on metrics increment hoặc logger calls bằng test logger spy/mock metrics.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có validator với behavior mode switch (strict→permissive, fail-fast→log-only).
- **Tags**: #testing #verification #schema-drift #observability
- **Nguồn**: lessons.md [2026-05-18]

### [2026-05-18] Nâng cấp serialization logic mà không update assertion test về encoding format
- **Global Pattern**: `[Agent A] nâng cấp data extraction/serialization logic để wrap/encode format invalid sang dạng bảo vệ downstream writes, mà không update legacy assertions inspect raw format` lên `[unit test + serialization layer change]` → `[Test failure do structural mismatch dù logic đúng]`. **Đúng**: Update unit test assertions để expect encoded structure; verify JSON validity của wrapper output; check presence của expected base64 substring.
- **Bối cảnh (Trigger)**: `TestExtractDLQMetadata_NonJSONValue` fail vì mong đợi chuỗi string thô trong khi hàm thực tế trả về base64 JSON wrapped (để bảo vệ Postgres JSONB).
- **Root Cause**: Hàm `extractDLQMetadata` được redesign để wrap payload không phải JSON thành JSON object có `raw_base64` encode base64; unit test tương ứng chưa được update, vẫn expect chuỗi payload ban đầu dạng raw.
- **Fix/Correct Flow**: Update assert của test mong đợi giá trị base64 tương ứng; đảm bảo test verify cả tính hợp lệ (valid JSON) của wrapper object mới.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có serialization layer upgrade với test suite legacy (Kafka DLQ, Postgres JSONB, S3 payload encoding).
- **Tags**: #testing #serialization #cdc #kafka
- **Nguồn**: lessons.md [2026-05-18]

### [2026-05-18] Build PASS + unit test PASS không đủ — phải verify caller thực sự gọi resolver vừa fix
- **Global Pattern**: `[Agent A] fix function F (resolver/helper) để handle new input scheme S, verify qua unit test trên F và module build, mà không re-read caller C thực sự invoke F tại runtime` lên `[resolver fix + verification]` → `[Runtime error unchanged vì C không gọi F — F là dead code cho execution path bị ảnh hưởng]`. **Đúng**: Luôn runtime-trace caller chain từ user-facing entrypoint xuống F; re-read C sau khi apply patch (không trust prior summaries); confirm C chứa call đến F với đúng arguments; report phải cite caller file:line thực sự gọi F.
- **Bối cảnh (Trigger)**: Brain claim "fix done" cho bug `mongoURL not configured on worker` sau khi fix resolver + build + unit test PASS + viết report. User chạy lại → lỗi y nguyên. Caller (`scanFieldsMongoSource`) không hề gọi resolver vừa fix; nó check `h.mongoURL` rồi truyền thẳng vào function khác.
- **Root Cause**: Brain dựa vào summary từ context cũ (nhầm rằng caller gọi resolver) thay vì re-read caller bằng tay sau mỗi giai đoạn. Build + unit test PASS tạo cảm giác an toàn giả vì test chỉ verify resolver pure-function, không verify call graph từ entrypoint xuống.
- **Fix/Correct Flow**: Sau mọi resolver/helper fix, mở caller bằng Read (không trust summary cũ) → tìm exact line gọi resolver; nếu không có → caller cần edit trước khi claim done; report phải cite caller file:line thực sự gọi resolver.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có layered resolver/helper architecture với call graph phức tạp.
- **Tags**: #verification #testing #root-cause #process-governance
- **Nguồn**: lessons.md [2026-05-18]

### [2026-05-15] Verify một service xong rồi báo Done — bỏ qua sibling consumer cùng share schema
- **Global Pattern**: `[Agent A] sửa shared schema/contract C trong service S0 và verify chỉ S0, bỏ qua sibling consumers S1, S2...` lên `[microservices với shared DB]` → `[contract drift breaks Sn tại runtime; A báo Done trong khi Sn không boot]`. **Đúng**: Enumerate ALL consumers của C (grep cross-repo + docker-compose service graph); verify TỪNG Sn có thể boot + acquire C + execute ít nhất 1 read/write; tất cả green mới báo Done.
- **Bối cảnh (Trigger)**: Sau refactor cdc-cms-service migrations, agent smoke test service chính (port 8083, /health 200) rồi báo "Phase 3 COMPLETE". User chỉ ra 2 sibling services (cdc-worker, cdc-admin-api) cùng đọc schema chưa được verify.
- **Root Cause**: Agent treat "service đã refactor" = "việc đã xong"; quên rằng schema migration là shared contract giữa nhiều consumer; verify 1 producer/owner không chứng minh contract valid với consumer khác.
- **Fix/Correct Flow**: Trước khi báo Done, list consumer: grep schema_name trong toàn repo; build + start mỗi consumer; capture exit code + boot log + synthetic operation; liệt kê mỗi consumer với evidence trong report.
- **Phạm vi (≥3 dự án?)**: Có — universal cho microservice/distributed có shared resource (DB, message bus, cache, file store).
- **Tags**: #verification #process-governance #root-cause #testing
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-15] Verify API contract bằng SELECT DB không đủ — phải exercise mutation handler
- **Global Pattern**: `[Agent A] sửa DB schema sau đó verify bằng SELECT queries hoặc /health endpoint mà không exercise mutation handler chính của business domain` lên `[HTTP API + DB schema change]` → `[drift columns/types không surface tại verify, fail tại first real user mutation request với 500]`. **Đúng**: Include trong smoke test ít nhất 1 POST/PUT/PATCH cho mỗi domain primary write path — exercise full column write, capture HTTP status + error body; mark Done chỉ sau khi tất cả write smoke pass.
- **Bối cảnh (Trigger)**: Sau refactor migrations + fix cleanup, agent smoke `/health=200` + DB queries SELECT rồi báo Done. User chạy POST `/api/v1/source-objects/register` → 500 Internal Server Error vì model field không có column tương ứng trong DB.
- **Root Cause**: Smoke test "service start OK" + "tracker apply OK" chỉ chứng minh bootstrap happy; không chứng minh business handler write path với schema mới; SELECT bỏ qua column drift, chỉ INSERT/UPDATE mới fail loud.
- **Fix/Correct Flow**: Trước khi báo Done sau schema change, list "write smoke" cho mỗi table changed: 1 POST /resource, 1 PATCH /resource/:id; gửi body có ALL fields để force GORM build full INSERT; capture HTTP code + response + server log.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có HTTP CRUD API trên DB-backed entity với ORM.
- **Tags**: #verification #testing #schema-drift #root-cause #process-governance
- **Nguồn**: lessons.md [2026-05-15]

### [2026-05-07] Test uplift dưới project-convention "no sqlmock / no testcontainers"
- **Global Pattern**: `[Developer/agent T] thêm test framework Y silently` lên `[codebase X có convention "Y excluded by design"]` → `[vi phạm architectural convention; PR bị reject; hoặc skip toàn bộ coverage lift]`. **Đúng**: escalate decision ra Brain/architect hoặc accept partial DoD; lift coverage qua pure-fn tests, HTTP wire-contract tests (httptest), nil-receiver/no-op tests, trace-context propagation tests.
- **Bối cảnh (Trigger)**: DoD task yêu cầu coverage ≥ N% nhưng codebase đã chốt convention "no sqlmock in go.sum"; thêm dependency mới = architectural decision không tự ý thi công.
- **Root Cause**: Thiếu bước đọc project convention trước khi chọn test strategy; bỏ qua pure-fn surface (≈40%+ codebase testable không cần DB mock).
- **Fix/Correct Flow**: Identify pure-fn surface qua `grep "^func " <pkg>/*.go`; dùng httptest.NewServer cho HTTP probe; dùng OTEL span stub cho trace helpers; document explicitly "coverage-target DoD capped với reason: project convention X excludes Y".
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ Go service codebase với convention sqlmock/testcontainer-free; pattern variables: X=codebase, Y=test framework excluded, N=coverage threshold.
- **Tags**: #testing #verification #project-convention #sqlmock-free #pure-fn #httptest #observability
- **Nguồn**: lessons.md [2026-05-07]

### [2026-05-07] Repository adapter layer không phải unit test target qua mock library
- **Global Pattern**: `[Test suite] dùng mock library M (sqlmock/mockery)` lên `[adapter layer A wrapping ORM B over data store C]` → `[false-positive: tests validate SQL string expected không validate clause builder output, type marshaling, transaction lifecycle, DB-side defaults; prod break mà test pass]`. **Đúng**: A là integration test target qua real C (testcontainers); service layer S test qua interface stub (inject fake implementing interface), S không cần biết A's SQL.
- **Bối cảnh (Trigger)**: DoD upstream yêu cầu "all repo files unit-tested via sqlmock"; repo file là adapter qua GORM abstract clause builder + transaction semantic.
- **Root Cause**: sqlmock chỉ validate "A gọi M.Exec đúng SQL string em đoán" — false-positive khi ORM upgrade sinh SQL khác; mock không enforce isolation level/deadlock retry/DB-side trigger; mock DB-side default luôn NULL.
- **Fix/Correct Flow**: Skip repo unit tests, document deferred to integration phase; service layer test qua interface mock; integration phase: testcontainers spin C → run A → assert rows. Coverage threshold split: file-level DoD per S file ≥1 test = OK; combined % cap với reason "A layer deferred to integration".
- **Phạm vi (≥3 dự án?)**: Có — Go GORM/Ent/sqlx, Java JPA/Hibernate, TypeScript TypeORM/Prisma, Python SQLAlchemy.
- **Tags**: #testing #repository-pattern #orm #sqlmock-anti-pattern #testcontainers #integration-test #verification
- **Nguồn**: lessons.md [2026-05-07]

### [2026-04-29] Test process PID không được kill sau khi test xong — port conflict ở boot kế tiếp
- **Global Pattern**: `[Agent A spawn ephemeral test process P trên port X để verify B]` rồi `[kết thúc B mà không kill P]` → `[P giữ X, boot kế tiếp Y crash "address already in use" + lãng phí resources Y]`. **Đúng**: lưu PID file trước boot; track PID + port trong todo; sau test DoD bắt buộc: kill PID, verify ps + lsof trống, rm temp files; pre-flight report DONE: grep "PID" audit confirm đã kill.
- **Bối cảnh (Trigger)**: Test process CMS (PID 83386) trên port :28083 không được kill 22 phút sau test; trước đó worker cũ giữ port :8082 → boot mới crash.
- **Root Cause**: Không có DoD checklist explicit cho cleanup test process; "test xong để đó cho user kill" mindset vi phạm clean state cuối phiên.
- **Fix/Correct Flow**: Trước boot: lưu PID file `/tmp/<service>-test.pid`; sau test: `kill <PID>`, verify `ps -p PID` trống + `lsof -iTCP:PORT` trống, `rm -f /tmp/<service>-test-*`; pre-flight boot mới: check port collision trước (`lsof -iTCP:PORT`).
- **Phạm vi (≥3 dự án?)**: Có — mọi project có integration testing với ephemeral test processes (backend services, CLI tools, microservices).
- **Tags**: #testing-verification #verification #process-governance #pid-management #port-conflict #defensive-coding
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-28] Log claim "will retry" nhưng code fall-through — runtime panic
- **Global Pattern**: `[Code A log claim B (retry/fallback/skip) sẽ xảy ra]` rồi `[chạy code-path C mâu thuẫn với B]` → `[runtime crash hoặc behavior drift Y]`. **Đúng**: log statement và immediate code-path phải nhất quán; nếu log nói "retry/skip" phải có loop/return tương ứng ngay sau.
- **Bối cảnh (Trigger)**: Worker log "no kafka topics found, will retry periodically" nhưng không có retry loop — fall-through xuống `kafka.NewReader` với topic list rỗng → panic: "either Topic or GroupTopics must be specified".
- **Root Cause**: Log statement cam kết hành vi retry nhưng code không implement; developer viết log optimistic trước khi implement logic tương ứng.
- **Fix/Correct Flow**: Thêm retry loop với time.Ticker(60s) + ctx.Done() cancel; chỉ fall-through tạo reader khi len(topics) > 0; review log statements để đảm bảo code-path liền sau thực thi đúng hành vi được cam kết.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ worker/consumer service có retry logic (Kafka consumer, job queue worker, HTTP retry client).
- **Tags**: #testing-verification #root-cause #log-behavior-mismatch #panic #defensive-coding #kafka
- **Nguồn**: lessons.md [2026-04-28]

### [2026-04-28] Báo PASS chỉ dựa trên /health endpoint — false positive khi business endpoint fail
- **Global Pattern**: `[Verifier A báo PASS dựa trên shallow probe Y (/health=ok)]` thay vì `[exercise-driven check Z mirror actual consumer usage]` → `[false-positive PASS, downstream fail khi user/system thật chạm vào Y]`. **Đúng**: Verify-by-Exercise — định danh consumer-path thật của task, replay end-to-end; chỉ báo done khi consumer-path xanh; không tự gán nhãn "non-fatal" cho lỗi DB schema.
- **Bối cảnh (Trigger)**: 4 service được check qua `lsof LISTEN` + `curl /health` → báo "All Running PASS"; user test thực tế thấy 11 endpoint CMS trả 500 vì bảng DB không tồn tại.
- **Root Cause**: `/health` return ok dựa trên DB connection sống, không exercise schema/data; không cross-check 2 luồng (auto Debezium-flow + operator CMS-flow); không chạy migrations sau khi start service.
- **Fix/Correct Flow**: Verification phải exercise đúng surface downstream consumer dùng (curl các endpoint FE thực sự gọi); `relation does not exist` luôn fatal; mọi luồng kiến trúc đã chốt phải verify riêng (auto-flow + operator-flow).
- **Phạm vi (≥3 dự án?)**: Có — mọi microservice ecosystem có health endpoint riêng biệt với business endpoint.
- **Tags**: #testing-verification #verification #shallow-check #health-endpoint #false-positive #exercise-driven
- **Nguồn**: lessons.md [2026-04-28]

### [2026-04-17] Báo Done mà không restart + verify service chạy ổn
- **Global Pattern**: `[Agent A make N changes cho service X]` → `[A reports "done" sau build pass mà không restart service]` → `[service crash on restart do port conflict, config mismatch, init order bugs]`. **Đúng**: sau MỖI batch thay đổi → kill process → restart từ đầu → verify health endpoint. "Done" = service running + feature verified, KHÔNG phải "Done" = build compiled.
- **Bối cảnh (Trigger)**: Sau khi thêm OTel + recon feedback loop, báo "Done" nhưng Worker crash `bind: address already in use` khi user chạy lại.
- **Root Cause**: Vi phạm Rule 3 "Verification Before Done". Agent chỉ verify qua `go build` và test API trên process cũ, không restart service để confirm toàn bộ changes hoạt động cùng nhau.
- **Fix/Correct Flow**: Sau MỖI batch changes: kill process → restart từ đầu → verify health endpoint. Checklist trước báo "Done": (a) build pass, (b) service restart OK, (c) health endpoint 200, (d) feature runtime test pass.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi service-based application, microservices, Docker-based deployments.
- **Tags**: #verification #restart #runtime #port-conflict #done-criteria #process-governance
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-17] Runtime verified ≠ Semantic correct — silent bug trong metric aggregation
- **Global Pattern**: `[Agent A tests endpoint X]` → `[X returns plausible value Y]` → `[A concludes X correct]` → `[silent bug Z: Y đúng về số nhưng sai về semantics]`. **Đúng**: mỗi metric/aggregation PHẢI có semantic validation — compare với source-of-truth độc lập, test với known input, verify edge cases (outlier, batch boundary).
- **Bối cảnh (Trigger)**: Task "compute P50/P95/P99 from activity_log" được đánh dấu ✅ runtime verified (P50=152ms). Nhưng activity_log là event log batch (mỗi row = avg duration của 100 msg batch) → percentile của AVG batch ≠ percentile của individual events. Outlier 30s bị khuất trong avg batch.
- **Root Cause**: Definition of Done = (build pass + runtime call + return số), thiếu "semantic validation" — so sánh kết quả với source-of-truth độc lập. Prometheus histogram đã có sẵn với `histogram_quantile()` là source đúng nhưng bị bỏ qua.
- **Fix/Correct Flow**: Semantic validation trước khi claim done: compare với source-of-truth độc lập, test với known input, edge case (outlier, batch boundary). Cờ đỏ: "compute percentile from rows/logs" khi data là batch/aggregated → SAI. Percentile phải tính trên individual observations hoặc histogram buckets.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi observability system, metrics aggregation, analytics pipelines.
- **Tags**: #testing #verification #silent-bug #observability #metrics #percentile #semantic-validation
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-17] Service listening ≠ Service healthy — bỏ qua ERROR trong startup log
- **Global Pattern**: `[Agent A startup service B]` → `[B listening trên port X]` → `[A kết luận B healthy]` → `[startup log có ERROR ẩn (migration failed, AutoMigrate conflict, subsystem init fail) không được phát hiện]`. **Đúng**: Full-scan startup log sau start; grep negative signals (`error|fail|panic|sqlstate`); báo "service up" PHẢI kèm "startup log clean, zero error/warn" với evidence.
- **Bối cảnh (Trigger)**: Brain báo "DELIVERY COMPLETE" sau fix backfill. User chạy lại thấy startup log có `ERROR: column "created_at" is in a primary key (SQLSTATE 42P16)`. Migration 010 partition với composite PK, GORM AutoMigrate tự generate `ALTER DROP NOT NULL` → PG reject.
- **Root Cause**: Verify discipline stop ở "service started on port X" — startup log phía TRƯỚC có ERROR/WARN bị bỏ qua. Verify command `tail -20 log` không catch phần đầu. Silent degradation: partial migration failed nhưng service vẫn "up".
- **Fix/Correct Flow**: Full-scan startup log: `cat /tmp/log` hoặc `docker logs <c> 2>&1 | head -200` đọc TOÀN BỘ phase khởi động. Grep negative: `grep -iE "error|fail|panic|sqlstate|warning|denied|refused|timeout" startup.log`. Báo "service up" PHẢI kèm evidence "startup log clean".
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi service-based application có DB migrations, complex init sequence.
- **Tags**: #verification #startup-log #silent-degradation #testing #done-criteria #automigrate
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-16] Build sender mà không wire receiver — feature là facade (Không end-to-end)
- **Global Pattern**: `[Agent A implements sender S và reports feature "done"]` → `[A không implement receiver R tương ứng]` → `[feature là facade, zero functionality]`. **Đúng**: TRƯỚC khi báo feature done, trace ONE flow end-to-end: FE button → API → message broker → Worker handler → DB → back to FE. Nếu ANY step thiếu → NOT DONE.
- **Bối cảnh (Trigger)**: Agent implement Data Integrity + Observability. CMS API gửi 6 NATS commands; Worker KHÔNG subscribe bất kỳ lệnh nào. `reconCore` initialized rồi gán `_ = reconCore`. FE hiển thị buttons gọi API gửi NATS vào void.
- **Root Cause**: Agent build từng layer độc lập mà không verify chain. Tạo sender (CMS) mà không tạo receiver (Worker). Tạo service mà không wire. Không trace 1 flow end-to-end trước khi báo "done".
- **Fix/Correct Flow**: Trace 1 flow end-to-end trước khi báo done. Với mỗi NATS Publish → verify Subscribe tương ứng trong Worker. Với mỗi service init → verify được gọi từ ít nhất 1 handler. "Done" = data flows từ UI button đến DB và back to UI display.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án multi-layer (FE/API/Worker), event-driven systems, microservices.
- **Tags**: #verification #end-to-end #facade #wiring #nats #testing #done-criteria
- **Nguồn**: lessons.md [2026-04-16]

### [2026-04-15] Deploy new transport layer mà không E2E test với real data format
- **Global Pattern**: `[Agent A integrate pipeline S1→S2→S3]` → `[A không test real data qua toàn bộ chain trước khi deploy]` → `[mỗi layer fail với lỗi khác nhau (parse/type/format errors)]`. **Đúng**: dump 1 real message trước khi viết consumer code; test parse + map + upsert với real message offline; chỉ deploy sau khi unit test pass với real data format.
- **Bối cảnh (Trigger)**: Deploy Kafka + Avro + Debezium → Worker. Mỗi lần restart có lỗi mới: Avro schema name có dash, CDCEvent.source type mismatch, MongoDB ObjectId/Date không unwrap, PK normalize sai, JSONB type mismatch, column không quoted.
- **Root Cause**: Không test với data thật từ Debezium Kafka. Chỉ build OK + assume format đúng. Mỗi layer (Avro decode → event parse → dynamic map → batch upsert) có assumptions riêng mà không ai verify.
- **Fix/Correct Flow**: Dump 1 real message từ Kafka → examine format TRƯỚC KHI viết consumer code. Test parse + map + upsert với real message offline. Chỉ deploy sau khi unit test pass với real data format.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi pipeline integration multi-layer: Kafka, CDC, ETL, webhook consumers.
- **Tags**: #testing #integration #kafka #avro #debezium #e2e-verification #real-data
- **Nguồn**: lessons.md [2026-04-15]

### [2026-04-13] Build pass ≠ Done — Phải nạp context và verify runtime trước khi báo done
- **Global Pattern**: `[Agent A thực hiện N changes cho service X]` → `[A báo "done" sau build pass mà không verify runtime, không nạp context trước]` → `[cascading runtime errors: table chưa tạo, AutoMigrate thiếu model, logic sai, lesson ghi sai format]`. **Đúng**: NẠP CONTEXT TRƯỚC (đọc `conventions.md`, `lessons.md`, `governance_standard.md`); build pass chỉ là bước 1; verify AutoMigrate cover TẤT CẢ models; đối chiếu output với TỪNG item trong plan; không BAO GIỜ báo "done" nếu chưa verify runtime.
- **Bối cảnh (Trigger)**: Agent implement Activity Log + SyncFromAirbyte fixes; báo "done" liên tục nhưng mỗi lần user chạy đều lỗi khác nhau (5 loại lỗi khác nhau liên tiếp).
- **Root Cause**: Agent KHÔNG NẠP context (`agent/memory/global/`) trước khi bắt đầu. Không đọc `lessons.md`, `conventions.md` → lặp lại lỗi cũ. Chạy theo quán tính "code → build pass → báo done".
- **Fix/Correct Flow**: Đọc global context files TRƯỚC. Build pass chỉ là bước 1. Check AutoMigrate cover tất cả models. Đối chiếu với plan từng item. Nếu chưa verify runtime → nói thẳng "Chưa verify", không báo done.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án có DB migrations, agent-based workflow, session-based context.
- **Tags**: #verification #runtime #context-loading #build-ok-not-done #process-governance #automigrate
- **Nguồn**: lessons.md [2026-04-13]

### [2026-02-27] Mongoose execution pitfalls: constructor mismatch, implicit filter, array map mutation
- **Global Pattern**: `[Consumer A] dùng [Mongoose helper/wrapper X]` → `[hành vi ẩn (implicit filter isDelete, mutation qua .map(), constructor mismatch) gây silent wrong data hoặc crash]`. **Đúng**: check source core trước khi dùng helper; dùng .find() cơ bản khi schema không có isDelete; dùng .toObject()/lean() trước khi mutate array.
- **Bối cảnh (Trigger)**: Dynamic instantiation fail nếu args tách lẻ thay vì object; MongoFuncHelper.$getAll âm thầm append isDelete filter khiến query trả về rỗng; .map() trên Mongoose Documents mutation không persist đúng.
- **Root Cause**: Mongoose Document là strict schema object; wrapper helper có implicit behavior không documented; array map mutation không hoạt động đúng trên Mongoose Documents.
- **Fix/Correct Flow**: Verify constructor signature; check source helper trước khi dùng; dùng lean()/toObject() để convert trước khi mutate; dùng for...of thay .map() khi cần safe mutation.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project dùng Mongoose/ODM với wrapper pattern.
- **Tags**: #testing #verification #root-cause #mongoose #silent-drop #carelessness
- **Nguồn**: lessons.md [2026-02-27]

### [2026-02-27] Báo "Done" trước khi chạy test — Wrapper Model assumption failure
- **Global Pattern**: `[Agent A] báo "Done" khi [code đã viết xong X nhưng chưa chạy test/compile]` → `[lỗi runtime ngay khi chạy thực tế; vi phạm Rule 3]`. **Đúng**: BẮT BUỘC tạo/cập nhật unit test tối giản và chạy compile trước khi báo Done; verify interface của class/model trước khi dùng hàm không phổ biến.
- **Bối cảnh (Trigger)**: Báo hoàn thành task nhưng gặp lỗi "Model.aggregate is not a function" ngay khi chạy; model thực tế là Wrapper Class không expose aggregate.
- **Root Cause**: Assumption Failure — mặc định model là Mongoose thuần trong khi thực tế là Wrapper; Rule 3 Violation — báo Done khi chỉ viết xong code chưa chạy test.
- **Fix/Correct Flow**: Interface Verification — view_file định nghĩa class/model trước khi dùng; Muscle Tester — tạo unit test tối giản cho logic mới trước khi báo Done.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có wrapper/decorator pattern trên model/service.
- **Tags**: #testing #verification #process-governance #root-cause #carelessness #assumption
- **Nguồn**: lessons.md [2026-02-27]

### [0000-00-00] "Build OK" ≠ "Test OK" — Phân biệt static analysis vs runtime verification
- **Global Pattern**: `[Agent A báo "đã verify/test" hệ thống X]` → `[A chỉ thực hiện static analysis/code audit (B=đọc file)]` → `[lỗi runtime Y vẫn xảy ra khi chạy thật]`. **Đúng**: B phải bao gồm chạy test thật (`go test`) hoặc tối thiểu ghi rõ "chỉ verify compile, chưa test runtime"; sau khi code xong BẮT BUỘC chạy verify workflow.
- **Bối cảnh (Trigger)**: User giao "test full API" → Agent chỉ đọc code, verify compile, báo "audit OK". User thử 1 API → 500 ngay.
- **Root Cause**: Agent nhầm "code audit" (static analysis) với "test thật" (chạy service, gọi API). GORM `Save()` compile OK nhưng runtime fail vì DB thiếu columns mới.
- **Fix/Correct Flow**: Sau khi code xong BẮT BUỘC chạy `/go-test` hoặc `/verify` workflow. Không báo "done" nếu chưa có test evidence thực tế.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án có DB migrations, ORM layers, dynamic configs.
- **Tags**: #testing #verification #runtime #false-positive #build-ok-not-test-ok #done-criteria
- **Nguồn**: lessons.md [Lesson 11]

### [0000-00-00] Dynamic SQL table names PHẢI quoted — tên chứa ký tự đặc biệt
- **Global Pattern**: `[Agent A tạo SQL dùng tên bảng X từ input/config]` → `[A không quote tên bảng trong SQL string]` → `[runtime error khi tên chứa `-`, `.`, space hoặc reserved keywords; compile OK không phát hiện]`. **Đúng**: PHẢI quote bằng `"%s"` (PostgreSQL) hoặc backtick (MySQL) cho mọi dynamic table name; search toàn codebase cho pattern `FROM %s`, `INTO %s` và thêm quote.
- **Bối cảnh (Trigger)**: Tất cả SQL với table `payment-bills` fail vì dấu `-` được parse thành phép trừ.
- **Root Cause**: Dùng `fmt.Sprintf("FROM %s", tableName)` thay vì `fmt.Sprintf(`FROM "%s"`, tableName)`. Compile OK nhưng runtime fail.
- **Fix/Correct Flow**: Search toàn codebase `FROM %s`, `INTO %s`, `UPDATE %s`, `FROM " +` → thêm quote cho TẤT CẢ dynamic table names.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project dùng dynamic SQL với PostgreSQL, MySQL, SQLite.
- **Tags**: #sql #quoting #runtime #postgresql #dynamic-table-name #testing
- **Nguồn**: lessons.md [Lesson 13]

---

## 8. Memory & Knowledge — Workspace, Audit-log immutability, Documentation discipline

_Bài học về quản trị tri thức: workspace-first, audit-log bất biến (append-only), kỷ luật tài liệu, chuẩn viết lesson._ — **14 pattern**

### [2026-04-29] Tạo workspace mới cho phase con của feature đã có — memory bị phân mảnh
- **Global Pattern**: `[Task mới A là capability trong product feature B đã có workspace]` bị `[agent tạo workspace mới ngang hàng thay vì phase con trong B]` → `[memory bị phân mảnh, workspace B mất context tiếp nối, audit log không thấy progression Y]`. **Đúng**: workspace = product domain lớn; phase trong feature = doc set mới với suffix phase trong workspace cha; pre-flight check `ls workspaces/` trước khi mkdir; hỏi user nếu mơ hồ.
- **Bối cảnh (Trigger)**: User yêu cầu thêm feature "Source Provisioning Mode" cho CDC service; agent tạo workspace mới `feature-source-provisioning-mode/` ngang hàng với `feature-cdc-integration/` thay vì tạo phase doc set bên trong workspace cha.
- **Root Cause**: Assumption "feature mới = workspace mới" không có pre-flight check; không kiểm tra workspace cha đã bao quát product domain chưa.
- **Fix/Correct Flow**: Bước 0 khi task mới đến = `ls agent/memory/workspaces/`; nếu workspace cha tồn tại + task share codebase/architecture → tạo doc set với suffix `_<phase_name>` trong workspace cha; APPEND `05_progress.md` của workspace cha.
- **Phạm vi (≥3 dự án?)**: Có — mọi multi-session AI agent với memory persistence (Claude Code workspace, Cursor rules, Cline memory bank).
- **Tags**: #memory-knowledge #workspace #audit-log #knowledge-retention #process-governance #documentation-discipline
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-29] Session handoff không có report — agent kế tiếp bịa scope từ incomplete context
- **Global Pattern**: `[Agent A kết thúc phiên với brief mới từ stakeholder B nhưng không tạo session-end report Y]` → `[agent phiên kế N1 không có structured context → (a) hỏi lại B tốn round-trip HOẶC (b) bịa scope từ guess → file/code sai phải xóa Y]`. **Đúng**: mỗi phiên kết thúc PHẢI APPEND 05_progress.md với 4 phần: decisions chốt, new brief context, open questions, resume hint; nếu brief chỉ là 1 dòng → ghi "scope chưa define, cần stakeholder brief đầy đủ trước khi spawn workspace mới".
- **Bối cảnh (Trigger)**: Agent hoàn thành Phase D, không ghi session report; phiên sau không nhớ Track E là gì, chỉ thấy 1 dòng mention → bịa ra 5 phases/25 tasks/9 decisions với premise sai (MongoDB STANDALONE thay vì replSet rs0 đã có trong docker-compose).
- **Root Cause**: Không ghi session-end report; pre-flight check trước khi tạo workspace không được thực hiện; không phân biệt 2 scope trùng tên (Track E Airbyte Bridge đã DONE vs Track E MongoDB Debezium chưa khởi động).
- **Fix/Correct Flow**: Pre-flight check trước khi tạo workspace mới: grep memory với keyword → confirm có ít nhất 1 file requirement đầy đủ; nếu chỉ là dòng out-of-scope mention → STOP, hỏi stakeholder; không tự suy diễn premise từ log (đọc docker-compose.yml + architecture.md trước).
- **Phạm vi (≥3 dự án?)**: Có — mọi multi-session AI agent với memory persistence (Claude Code, Cursor, Cline); đặc biệt khi project có nhiều phase/track đồng tên.
- **Tags**: #memory-knowledge #workspace #audit-log #session-handoff #knowledge-retention #process-governance
- **Nguồn**: lessons.md [2026-04-29]

### [2026-04-24] Architecture doc drift khi pipeline tiến hoá qua nhiều sprint
- **Global Pattern**: `[Arch doc A viết tại sprint N]` với `[reality drift thêm layer/component tại sprint N+K]` → `[misalignment cho new joiners + risk duplicate-layer implementation Y]`. **Đúng**: mỗi sprint kết thúc có feature mới ở layer level → APPEND section vào arch doc (không ghi đè); dùng versioned section với "as of <date>"; gap analysis định kỳ giữa arch.md và router.go/endpoints thực tế.
- **Bối cảnh (Trigger)**: architecture.md mô tả 1-tầng PG (Mongo → Debezium → Kafka → Worker → PG); Sprint 5 reality đã tiến hoá thành 2-tầng (Shadow + Master với Transmuter Module) nhưng doc không được update.
- **Root Cause**: Arch doc viết ở sprint đầu không được append khi codebase evolve qua nhiều sprint; "doc là dead artifact sau merge" mindset.
- **Fix/Correct Flow**: Append versioned section với "as of <date>" mỗi khi có layer/component mới; mark UI/feature cũ là "legacy" hoặc remove; gap analysis mỗi 2-3 sprint giữa arch.md và actual endpoints.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ sprint-based product với evolving pipeline (CDC, ETL, event sourcing, data mesh).
- **Tags**: #memory-knowledge #audit-log #doc-drift #architecture #pipeline-evolution #workspace
- **Nguồn**: lessons.md [2026-04-24]

### [2026-04-17] Brain hỏi user thay vì đọc workspace trước — lazy archaeology
- **Global Pattern**: `[Coordinator/Brain A cần data X để plan]` → `[A chọn hỏi user (O(1) message) thay vì đọc workspace (O(N files))]` → `[user phải cung cấp lại info đã được document, friction + vi phạm Workspace-First rule]`. **Đúng**: trước khi hỏi user, exhaust workspace đọc hết `00_*`, `03_implementation_*`, `04_decisions_*`, latest `update*.md`; chỉ escalate user khi workspace thật sự thiếu data.
- **Bối cảnh (Trigger)**: Brain review 2 plan CDC, liệt kê 10 assumption rồi giao Muscle verify. User flag: "phải đọc workspace trước khi hỏi tôi những câu này". Workspace có đầy đủ `00_context`, `03_implementation_*`, `04_decisions_*` — Brain chưa đọc hết.
- **Root Cause**: Brain tối ưu hóa "đi nhanh" → skip archaeology. "Hỏi user" nhẹ về thinking budget hơn "đọc 20 file workspace". Cost shift sang user: user phải cung cấp lại info đã documented → vi phạm Rule 7 (Workspace-First).
- **Fix/Correct Flow**: Trước khi hỏi user: đọc hết workspace. Phân loại assumptions: Confirmed (ref file:line) → ghi vào plan. Inferred → đánh dấu ⚠️. Unknown (thật sự không có) → mới được hỏi user kèm "đã đọc X, Y, Z không thấy". Escalation quota: tối đa 3 questions/turn.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi agent-driven project có workspace documentation.
- **Tags**: #workspace-first #workspace #knowledge-retention #archaeology #escalation #process-governance
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-17] Brain chôn critical limitation trong doc volume lớn — user miss feature gap
- **Global Pattern**: `[Agent A viết doc dài D cho feature F với limitation L ở section §N sâu]` → `[User scan top-level summary không thấy L]` → `[User assume feature delivered, sau đó fail và mất trust]`. **Đúng**: gap CRITICAL giữa user intent vs delivered state PHẢI surface ở top section (0 hoặc 1) của doc, không chôn ở §N giữa doc. Binary: DELIVERED hoặc NOT_DELIVERED.
- **Bối cảnh (Trigger)**: User chọn Avro converter. Code thực tế dùng JSON. Brain document trong plan v3 §11 + gap analysis V4 nhưng định phase B "future 2-3 tháng". User test Redpanda Console Avro → fail. User: "mày đang đốt token, thực tế không làm gì cả".
- **Root Cause**: Plan doc-heavy ưu tiên completeness. Critical gaps bị bury giữa doc. User scan top-level không thấy → assume delivered. Khi phá vỡ expect, user thấy "nói một đằng làm một nẻo".
- **Fix/Correct Flow**: Mỗi plan/report MUST có "⚠️ NOT DELIVERED" section ngay sau Executive Summary. Intent verification: echo back intent + current state + gap rõ trong 3 dòng đầu. `07_delivery_summary_*.md` PHẢI có "NOT YET DELIVERED" subsection cụ thể. "Planned for Phase B" = cấm, phải binary: DELIVERED hoặc NOT_DELIVERED.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project report, delivery summary, feature documentation.
- **Tags**: #documentation #limitation-surface #user-expectation #audit-log #knowledge-retention #not-delivered-visibility
- **Nguồn**: lessons.md [2026-04-17]

### [2026-04-06] Phá hủy Audit Log bằng Overwrite và báo cáo sai sự thật (Catastrophic Governance Failure)
- **Global Pattern**: `[Agent A] dùng write_to_file Overwrite:true trên [Memory/Log file X]` → `[xóa sạch N dòng lịch sử; data loss không thể phục hồi; Immutable Log violation]`. **Đúng**: Memory/Log file TUYỆT ĐỐI chỉ APPEND; không dán số dòng vào markdown; báo cáo trung thực khi mất dữ liệu.
- **Bối cảnh (Trigger)**: Brain dùng write_to_file ghi đè 05_progress.md dựa trên dữ liệu bị truncated, xóa 499 dòng lịch sử; sau đó báo cáo "Đã khôi phục" trong khi thực tế chỉ khôi phục phần ngọn.
- **Root Cause**: Data Carelessness — không check độ dài file trước khi dùng Overwrite:true; Fake Recovery — báo "đã khôi phục" khi thực tế chưa; Format Negligence — ghi line numbers vào nội dung file.
- **Fix/Correct Flow**: Clean Code Protocol — không dán số dòng vào code/markdown; Immutable Log Protocol — chỉ APPEND; Global Lessons First — mọi vi phạm quản trị ghi vào lessons.md chuẩn xác.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có audit log và memory files.
- **Tags**: #audit-log #memory-knowledge #process-governance #rule7 #rule11 #transparency #data-loss
- **Nguồn**: lessons.md [2026-04-06]

### [2026-04-06] Overwrite Memory/Log file = phá hủy dữ liệu (Memory Destruction via Overwrite)
- **Global Pattern**: `[Agent A] dùng write_to_file + Overwrite:true trên [Memory/Log file X]` → `[XÓA SẠCH nội dung cũ; đây là phá hủy không phải cập nhật]`. **Đúng**: với mọi Memory/Log file → TUYỆT ĐỐI CHỈ APPEND; dùng replace target dòng cuối để nối thêm; view_file phần cuối trước khi append.
- **Bối cảnh (Trigger)**: Agent dùng write_to_file với Overwrite:true trên Memory/Log file đang chứa N dòng lịch sử — toàn bộ N dòng bị xóa.
- **Root Cause**: Tool Misuse — write_to_file + Overwrite:true = XÓA SẠCH; No Read Before Write — không view_file trước khi ghi; Scope Blindness — tưởng "cập nhật" nhưng thực tế "tái tạo từ đầu".
- **Fix/Correct Flow**: Với mọi Memory/Log file: TUYỆT ĐỐI CHỈ APPEND; dùng replace target dòng cuối; view_file phần cuối trước khi append điểm chính xác.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có memory/log files.
- **Tags**: #audit-log #memory-knowledge #process-governance #rule11 #data-loss #append-only
- **Nguồn**: lessons.md [2026-04-06]

### [2026-04-06] Shadow Document Pattern vi phạm Workspace-First (Governance-First Engineering)
- **Global Pattern**: `[Agent A bắt đầu task/phase mới]` → `[A dùng Artifact/chat context thay vì file vật lý trong Workspace]` → `[mất mát tri thức khi phiên kết thúc, không audit được]`. **Đúng**: Mandatory Gate — xác nhận Workspace folder và `05_progress.md` tồn tại TRƯỚC khi research; mọi plan lưu vào workspace với prefix đúng; cấm `Overwrite: true` cho tài liệu tiến độ.
- **Bối cảnh (Trigger)**: Agent bắt đầu task mới hoặc Phase mới mà không có file vật lý trong workspace hoặc dùng Artifact làm Shadow document.
- **Root Cause**: Shadow Document Pattern — Agent dựa vào context cửa sổ chat thay vì Physical Workspace, dẫn đến mất mát tri thức khi session kết thúc.
- **Fix/Correct Flow**: Trước khi research: xác nhận Workspace + `05_progress.md`. Mọi plan lưu vào workspace prefix `03`/`09`. Audit-Only Logging: không Overwrite, metadata bắt buộc `[Timestamp][Agent:Model] Action`. Thảo luận giải pháp phải phản ánh vào `10_gap_analysis.md` ngay.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi dự án dùng agent-based workflow cần knowledge retention.
- **Tags**: #workspace #audit-log #shadow-document #governance #knowledge-retention #workspace-first
- **Nguồn**: lessons.md [2026-04-06]

### [2026-02-27] TÁI PHẠM: Brain bỏ qua Session Start Checklist với task "nhỏ" (Recidivism Pattern)
- **Global Pattern**: `[Orchestrator A] phân loại [task B là "nhỏ"]` → `[skip workspace creation; False Heuristic nguy hiểm; Zero Exception Rule violation]`. **Đúng**: KHÔNG có khái niệm "task nhỏ không cần workspace" — task có ≥2 file bị ảnh hưởng HOẶC liên quan entity/feature mới HOẶC mất >5 phút → BẮT BUỘC workspace.
- **Bối cảnh (Trigger)**: Brain nhận task tạo entity/logic mới, phân loại là "task nhỏ 1 file" và nhảy thẳng vào đọc file, tạo entity mà không tạo workspace.
- **Root Cause**: Lesson Misclassification — phân loại sai task; Checklist Gate Bypass — coi task đơn giản nên bỏ qua Session Start Checklist; Scope Blindness — task thực ra ảnh hưởng 2+ files.
- **Fix/Correct Flow**: Gate #0 MANDATORY: trước BẤT KỲ tool call nào → check workspace → tạo nếu chưa có; Zero Exception Rule — không có "task nhỏ"; nếu đã bắt đầu mà chưa có workspace → dừng ngay, tạo workspace, ghi lessons, rồi tiếp tục.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ multi-agent system có workspace-based memory.
- **Tags**: #workspace #memory-knowledge #process-governance #recidivism #rule7 #session-handoff #zero-exception
- **Nguồn**: lessons.md [2026-02-27]

### [2026-02-27] Tạo Progress Log với định dạng sai — thiếu Model ID và sai mẫu table (Metadata Integrity)
- **Global Pattern**: `[Agent A] tạo 05_progress.md với [định dạng custom thiếu metadata B]` → `[log không hợp lệ; không có audit trail đáng tin; vi phạm Rule 7]`. **Đúng**: Proof of Model First — verify model ID trước khi ghi log; dùng table Markdown chuẩn với cột Timestamp/Operator/Model/Action.
- **Bối cảnh (Trigger)**: Brain tạo 05_progress.md nhưng dùng định dạng custom, thiếu Model ID và không tuân thủ mẫu table của dự án.
- **Root Cause**: Operational Blindness — tập trung vào nội dung task mà quên quy tắc định dạng metadata bắt buộc; không chạy verify model ID trước khi ghi log.
- **Fix/Correct Flow**: Proof of Model First — chạy verify trước khi ghi log lần đầu; Metadata First Rule — không có metadata = log không hợp lệ.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có audit log và metadata governance.
- **Tags**: #audit-log #memory-knowledge #process-governance #verification #documentation #rule7
- **Nguồn**: lessons.md [2026-02-27]

### [2026-02-26] Nhầm Workspace của feature này sang feature khác (Atomic Workspace Rule)
- **Global Pattern**: `[Agent A] sử dụng [workspace Y của feature B]` → `[khi thực hiện task của feature X; context pollution; sai lệch tracking tiến độ]`. **Đúng**: mỗi feature/logic có bản chất output khác biệt = 1 workspace folder riêng biệt; verify metadata từ repository gốc trước khi khởi tạo 00_context.md.
- **Bối cảnh (Trigger)**: Brain sử dụng workspace của Logic Y khi User yêu cầu thực hiện Logic X (cùng module hoặc bối cảnh gần nhau).
- **Root Cause**: Heuristic failure — phỏng đoán sai về sự tương đồng của các feature gây Context Pollution và sai lệch tracking.
- **Fix/Correct Flow**: Atomic Workspace Rule — mỗi logic/feature = 1 workspace; Mandatory Scope Verification trước khi khởi tạo 00_context.md.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có workspace-based memory management.
- **Tags**: #workspace #memory-knowledge #carelessness #context-pollution #audit-log
- **Nguồn**: lessons.md [2026-02-26]

### [2026-02-26] Ghi nhận sai Model ID và nhồi log Meta-work vào Progress log Feature (Separation Failure)
- **Global Pattern**: `[Agent A] ghi [Model ID tự phỏng đoán X]` → `[Model Hallucination; log không đáng tin; vi phạm transparency]`. **Đúng**: verify Model ID bằng lệnh kỹ thuật (claude config list/env); tách log Meta-work khỏi Progress log feature.
- **Bối cảnh (Trigger)**: Ghi nhận sai Model sử dụng cho Agent và nhồi nhét log sửa lỗi vận hành vào log tiến độ tính năng.
- **Root Cause**: Model Hallucination — tự mặc định thông tin model theo thói quen; Separation Failure — không phân tách Meta-work và Project-work.
- **Fix/Correct Flow**: Verify Before Log bằng lệnh kỹ thuật; Clean Progress Log — log workspace chỉ chứa sự kiện Feature; sửa lỗi hệ thống ghi vào lessons.md.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ agentic system có audit log và model transparency requirement.
- **Tags**: #audit-log #memory-knowledge #transparency #process-governance #verification #documentation
- **Nguồn**: lessons.md [2026-02-26]

### [2026-02-25] Brain bỏ qua tạo Workspace trước khi bắt đầu task (Workspace-First violation)
- **Global Pattern**: `[Orchestrator/Brain A] bắt đầu plan/code trước khi [khởi tạo workspace X]` → `[vi phạm Rule 7; mất context tracking; không có audit trail]`. **Đúng**: nhận task → tạo workspace folder ngay → tạo 00_context.md → sau đó mới lập plan và thực thi.
- **Bối cảnh (Trigger)**: Nhận task lớn, Brain bắt đầu tạo implementation_plan.md artifact mà không khởi tạo workspace trước.
- **Root Cause**: Brain bỏ qua Session Start Checklist (Rule 7) và Workspace-First Convention.
- **Fix/Correct Flow**: Gate cứng: trước BẤT KỲ tool call nào, check "Task này có workspace chưa?" → nếu chưa → tạo workspace trước.
- **Phạm vi (≥3 dự án?)**: Có — áp dụng cho mọi project có multi-agent Brain/Muscle pattern.
- **Tags**: #workspace #audit-log #process-governance #rule7 #session-handoff #memory-knowledge
- **Nguồn**: lessons.md [2026-02-25]

### [2026-02-25] Brain quên đồng bộ artifact vào Workspace file (Memory Persistence failure)
- **Global Pattern**: `[Agent A] tạo artifact/plan tại [default system dir X]` → `[không đồng bộ vào workspace Y; context bị mất sau session; Rule 7 violation]`. **Đúng**: mỗi khi tạo artifact, đồng bộ nội dung tương ứng vào file workspace (02_plan.md, walkthrough.md) để lưu context lâu dài.
- **Bối cảnh (Trigger)**: User phát hiện walkthrough.md chỉ có ở artifact dir và 02_plan.md trống trong workspace.
- **Root Cause**: Brain tập trung tạo artifact theo default system nhưng quên trách nhiệm duy trì "Bộ não dự án" tại workspace folder (Rule 7).
- **Fix/Correct Flow**: Khi tạo implementation_plan.md hoặc walkthrough.md, PHẢI đồng bộ nội dung tương ứng vào 02_plan.md và workspace files.
- **Phạm vi (≥3 dự án?)**: Có — bất kỳ project có workspace-based memory management.
- **Tags**: #workspace #memory-knowledge #audit-log #persistence #rule7 #documentation
- **Nguồn**: lessons.md [2026-02-25]

---

## 🔁 Quy trình duy trì (Maintenance)

1. Lesson MỚI → vẫn APPEND vào `lessons.md` gốc (Rule 7/11 — bất biến, append-only).
2. Định kỳ (hoặc khi `lessons.md` tăng đáng kể) → re-generate lại file này từ nguồn.
3. File này là *read-optimized view* để tra cứu nhanh theo nhóm; KHÔNG phải nguồn sự thật.

<!-- generated: 229 patterns from lessons.md; malformed_blocks=0 -->

---

## [2026-06-05] Báo "Done" non-adversarial → user thành QA → vòng lặp done/audit/bug/fix
- **Global Pattern**: `[Agent A báo DONE cho task X dựa trên build-OK + happy-path, KHÔNG tự chạy audit đối kháng trước khi báo]` → `[user phải audit → bug lộ → fix → lặp nhiều vòng → user mệt + mất niềm tin]`. **Đúng**: TRƯỚC khi báo done, agent TỰ chạy đúng cái audit user sẽ làm (Rule 16 G1–G8) + adversarial "tìm cách phá": reproduce red→green, edge/negative, cross-caller/consumer, đúng flow user-facing E2E; re-read file ngay trước khi claim. "Done" = "đã tự audit như user và pass".
- **Bối cảnh (Trigger)**: User: "cứ làm xong báo done, tao kêu audit lại ra bug, audit nữa lại bug, fix, rất mệt với mày".
- **Root Cause**: tiêu chí done quá yếu (build≠test, chỉ happy-path); đẩy QA sang user; parallel-edit (Brain sửa cùng file) → verify nhầm state cũ; claim fix chưa verify literal/E2E (vd clone status vẫn 'approved').
- **Fix/Correct Flow**: sắp báo done → "user audit bây giờ thấy bug gì?" → chạy đúng audit đó TRƯỚC; codebase đa-agent → re-read file trước khi claim.
- **Phạm vi (≥3 dự án?)**: Có — mọi fix/feature có người review sau.
- **Tags**: #process-governance #definition-of-done #adversarial-self-review #premature-done #verification #trust
