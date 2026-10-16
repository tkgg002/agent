# 13 — Audit & Đề xuất Tối ưu Quy trình Agent (Antigravity)

> Ngày: 2026-10-15 | Vai trò: Brain (Propose-Only — Rule #1/#13) | Trạng thái: **CHỜ APPROVE**
> Phạm vi: `/Users/trainguyen/Documents/work/agent` (GEMINI.md, workflows/, hooks/, tooling/, memory/global/lessons.md) + global rules + global skills.

---

## 0. Tóm tắt

Hệ thống quy trình được viết rất đầy đủ trên giấy nhưng **phần lớn không có hiệu lực thực tế trong Antigravity**, vì 3 lý do kỹ thuật:

1. `GEMINI.md` vượt giới hạn 24KB/file → **Rule 15–22 bị cắt, model không bao giờ thấy**.
2. `agent/workflows/` và `agent/hooks/` nằm ở vị trí/định dạng mà Antigravity **không discover** → các lệnh `/brain-delegate`, `/muscle-execute`… và các hook bảo vệ memory/secret **không hoạt động**.
3. `lessons.md` 479KB — không thể "đọc trước mọi việc" như Rule #0 yêu cầu (tự mâu thuẫn với Rule #21).

Ngoài ra có nhiều **mâu thuẫn nội bộ** (format log 3 kiểu, đánh số rule lệch, workflow tự `git push` trái Rule #15, linter đòi file không có trong registry).

---

## 1. Phát hiện (có bằng chứng)

### P0 — Rule/Workflow không có hiệu lực

| # | Phát hiện | Bằng chứng | Hậu quả |
|---|---|---|---|
| F1 | `GEMINI.md` = **30,949 bytes** > cap **24,000 bytes**/rule file | `wc -c GEMINI.md`; system prompt hiển thị `<truncated 6953 bytes>` đúng tại "TRỤ CỘT IV"; docs `agy-customizations/docs/rules.md` | Rule 15 (Security/No push), 16, 17 (Memory Protection), 18, 19 (VCS), 20 (Secret), 21, 22 + Workflows Reference **không được inline** |
| F2 | Global user rule là **bản copy gần như nguyên văn** GEMINI.md (cũng bị cắt `<truncated 6395 bytes>`) | System prompt có cả `RULE[user_global]` và `RULE[.../agent/GEMINI.md]` cùng nội dung | Tốn gấp đôi ngân sách rules (20k tokens), 2 bản sẽ drift (bản global đã thiếu dòng "không đoán mò") |
| F3 | `agent/workflows/*.md` (96 file) **không được discover** | Antigravity chỉ quét `.agents/`, `.agent/`, `_agents/`, `_agent/` + `~/.gemini/config/`; thư mục tên `agent/` (không dấu chấm) bị bỏ qua. Menu slash hiện tại không có lệnh nào của anh. Định dạng workflows đã deprecated → Skills (`migrate-workflows/SKILL.md`) | `/brain-delegate`, `/muscle-execute`, `/qa-agent`… chỉ là tên gọi trong văn bản |
| F4 | `agent/hooks/*.sh` viết theo **định dạng Claude Code** (`SessionStart`, `hookSpecificOutput`, `permissionDecision`, `systemMessage`), không có `hooks.json` | `ls .agents` → không tồn tại; contract Antigravity khác (`PreInvocation/PreToolUse/PostToolUse/Stop`, `decision`, `injectSteps`) — `docs/hooks.md` | Bảo vệ memory (Rule 17) và quét secret (Rule 20) **chỉ nằm trên giấy** trong Antigravity |

### P1 — Mâu thuẫn nội bộ

| # | Phát hiện | Bằng chứng |
|---|---|---|
| F5 | **Đánh số rule lệch** giữa hooks và GEMINI.md | Hooks: No-Shadow=Rule 7, Memory=11, DoD=16, Restore=18, Secret=19. GEMINI.md: 4, 17, 14, 19, 20. Rule #8 yêu cầu sync `CLAUDE.md` nhưng `ls CLAUDE.md` → **không tồn tại** trong `agent/` |
| F6 | **3 format `05_progress` khác nhau** | GEMINI.md + linter: `- [ts] [Agent:Model] Action`; `brain-delegate.md`: bảng `\| [Timestamp] \| [Agent/Model] \| ...`; `muscle-execute.md`: `[Provider:Model-Name] [Time] Action` → 2 format sau sẽ FAIL linter |
| F7 | `muscle-execute.md` bước 10: `git add -A` + `git push origin <branch>` gắn `// turbo` (auto-run) | Trái trực tiếp Rule #15 "KHÔNG push bất kỳ nhánh nào" |
| F8 | Workflow lõi **lỗi thời so với dự án** | `npm run lint/test` cho stack Go; path `agent/memory/project_context.md`, `agent/memory/tech_stack.md` (sai — đúng là `memory/global/project/<name>/`); `feature-goopay-backend/active_plan.md`; phase GĐ0–4 GooPay refactor; K8s path `workload-sre-*` |
| F9 | `verify_governance.py` lệch registry & dễ sai mục tiêu | Đòi `implementation_plan.md` (registry dùng `12_implementation_plan_*.md`); tự chọn "workspace sửa gần nhất" → có thể audit nhầm workspace; `ROOT_DIR` hardcode; Rule #22 bắt chạy **mọi turn** kể cả hỏi đáp → luôn FAIL/nhiễu. Chạy thử hôm nay: `PermissionError: Operation not permitted: .../memory/workspaces` |
| F10 | Tên tool lỗi thời | Rule #11 `send_command_input`, Rule #12 `command_status`/`sudo purge` — Antigravity hiện dùng `manage_task send_input` + thông báo nền tự động; Rule #3 `/muscle-dispatch` "v1.10.0", hạ tầng `.agent/` không tồn tại |

### P1 — Chi phí context

| # | Phát hiện | Bằng chứng |
|---|---|---|
| F11 | `lessons.md` = **479,155 bytes / 1,861 dòng / 237 entry**, mỗi entry 1 dòng dài hàng nghìn ký tự | `view_file` 200 dòng đầu đã bị cắt ở 46KB. Rule #0 "đọc trước mọi việc" vs Rule #21 "cấm đọc file > ~256KB" → **tự mâu thuẫn**; thực tế agent chỉ đọc được ~10% đầu |
| F12 | Skills global quá tải, lệch domain | ~300 skill trong `~/.gemini/config/skills` (Laravel, Unity, Healthcare, Solidity, Pentest…), một phần đã bị loại "do context budget". 96 workflow gồm cpp/flutter/kotlin/rust/gradle/perl không liên quan stack |
| F13 | `hooks/organize_skills.sh` **nguy hiểm nếu chạy** | Di chuyển skill vào `categorized/<nhóm>/<skill>` = sâu 2 cấp, trong khi discovery chỉ quét **1 cấp** → toàn bộ skill biến mất. Dòng cuối là đường dẫn lạc (`.../antigravity_guide/SKILL.md`) sẽ bị thực thi như lệnh. Đặt sai thư mục (không phải hook) |

### P2 — Vệ sinh

| # | Phát hiện | Bằng chứng |
|---|---|---|
| F14 | `models.env` được **track trong git**, format `provider:model:key`, repo không có `.gitignore` | `git ls-files models.env` → có. Quét pattern key phổ biến (`sk-`, `AIza`, `ATATT`, `ghp_`) → 0 kết quả (có thể là placeholder) — cần anh xác nhận |
| F15 | Sandbox chặn liệt kê thư mục `memory/` | `ls memory` và linter → `Operation not permitted`; đọc file trực tiếp thì được → mọi bước governance tự động bị kẹt |
| F16 | Overhead tài liệu | Full Doc Set 13–14 file/phase; ngay cả câu hỏi tư vấn cũng bị Rule #4/#22 yêu cầu workspace + linter |

---

## 2. Kiến trúc đích (1 phương án duy nhất)

Nguyên tắc: **Rules ngắn & luôn bật → Skills cho quy trình chi tiết (nạp khi cần) → Hooks để cưỡng chế bằng máy.** `agent/` vẫn là single source of truth (có git), cấu hình global chỉ *trỏ* vào đó để áp dụng cho cả `agent` lẫn `data-hub`.

```text
~/.gemini/config/
├── skills.json        → entries: /Users/trainguyen/Documents/work/agent/skills
├── hooks.json         → 4 hook trỏ tới agent/hooks/agy/*.sh
└── (global rule)      → ≤ 1KB: "Trả lời tiếng Việt + BẮT BUỘC đọc agent/GEMINI.md"

/Users/trainguyen/Documents/work/agent/
├── GEMINI.md          → CORE ≤ 12KB (luôn inline đủ, không bị cắt)
├── skills/            → 8–10 skill lõi (migrate từ workflows/)
│   ├── brain-delegate/SKILL.md
│   ├── muscle-execute/SKILL.md
│   ├── debug-agent/SKILL.md        (gộp bug-handling-sop)
│   ├── qa-agent/SKILL.md           (chứa chi tiết DoD G1–G8)
│   ├── security-agent/SKILL.md
│   ├── audit-propose-only/SKILL.md (codify lesson 2026-10-14)
│   ├── context-manager/SKILL.md    (format + bảo trì lessons)
│   ├── workspace-docs/SKILL.md     (doc registry + template + tier)
│   └── session-wrapup/SKILL.md     (gộp save/resume-session + linter)
├── hooks/agy/         → script theo contract Antigravity
├── memory/global/lessons/
│   ├── INDEX.md       → ≤ 15KB, 1 dòng/pattern (đọc đầu phiên)
│   └── topics/*.md    → chi tiết theo nhóm (đọc khi cần, grep theo tag)
└── workflows/_archive/ → 96 file cũ (giữ nguyên, không xoá)
```

---

## 3. Đề xuất chi tiết + Code demo

### U1 (F1, F2, F10) — Tách GEMINI.md thành Core ≤ 12KB

**Giữ trong Core** (viết nén, mỗi rule 2–4 dòng): Rule 0 (ngôn ngữ, không đoán mò, planning), vai trò Brain/Muscle + Propose-Only, Workspace tối thiểu theo tier, Immutable log format **duy nhất**, DoD G1–G8 (1 dòng/gate), No push, Memory protection, No secret, VCS restore-point, Escalation >3 lần.

**Chuyển ra Skill** (nạp khi cần): Doc Registry chi tiết → `workspace-docs`; Lesson format/GC/KPI → `context-manager`; DoD chi tiết → `qa-agent`; Workflows Reference → bỏ (skill tự hiện trong menu `/`).

**Sửa/bỏ**: tên tool lỗi thời (Rule 11/12), `/muscle-dispatch` + `.agent/` (Rule 3), Rule 8 (CLAUDE.md không tồn tại — xem Câu hỏi Q1).

**Global rule** thay bằng pointer (loại trùng lặp hoàn toàn):

```markdown
# Global
- Luôn trả lời bằng tiếng Việt.
- Trước mọi task: đọc /Users/trainguyen/Documents/work/agent/GEMINI.md (nếu chưa có trong context)
  và /Users/trainguyen/Documents/work/agent/memory/global/lessons/INDEX.md.
```

**Verify**: mở hội thoại mới → rules trong context không còn `<truncated`; `wc -c GEMINI.md` ≤ 12000.

### U2 (F3, F6, F7, F8) — Migrate workflow lõi thành Skills

`~/.gemini/config/skills.json` (áp dụng cho mọi workspace, trỏ về repo `agent/`):

```json
{
  "entries": [
    { "path": "/Users/trainguyen/Documents/work/agent/skills" }
  ]
}
```

Mẫu `agent/skills/muscle-execute/SKILL.md` (đã sửa mâu thuẫn):

```markdown
---
name: muscle-execute
description: >-
  Use when executing an APPROVED solution (09_tasks_solution_*.md) — implement code,
  run real tests, verify runtime, update 05_progress. Never pushes to git.
---
# Muscle Execute
1. Đọc 09_tasks_solution_*.md đã APPROVE + 08_tasks_*.md. Thiếu APPROVE → DỪNG.
2. Append 05_progress: `- [<ISO-ts>] [Agent:<Model>] START <task>` TRƯỚC khi sửa code.
3. Restore-point: `git -C <service> commit` local (KHÔNG push).
4. Implement theo pattern hiện có của service.
5. Verify thật: `go build ./... && go test ./...` (Go) | `npx tsc -b && npm run build` (web);
   restart process, `lsof -nP -iTCP:<port> -sTCP:LISTEN`, bắn request thật.
6. Gọi skill security-agent → qa-agent (DoD G1–G8).
7. Ghi 06_validation + 11_report; append 05_progress `DONE`.
```

Ánh xạ: `brain-delegate`, `muscle-execute`, `debug-agent`(+`bug-handling-sop`), `qa-agent`, `security-agent`, `context-manager`, `session-wrapup`(+`save-session`,`resume-session`,`scripts/wrapup.sh`), `go-review`, `go-test`, `e2e`. Các workflow còn lại → `workflows/_archive/` (mv, không xoá).

**Verify**: gõ `/` trong chat thấy `/brain-delegate`, `/muscle-execute`…; grep toàn bộ skills không còn `git push`, `npm` cho Go, path `agent/memory/project_context.md`.

### U3 (F4, F5) — Chuyển hooks sang `hooks.json` của Antigravity

`~/.gemini/config/hooks.json`:

```json
{
  "memory-protect": {
    "PreToolUse": [
      { "matcher": "write_to_file",
        "hooks": [{ "command": "/Users/trainguyen/Documents/work/agent/hooks/agy/memory_protect.sh", "timeout": 5 }] }
    ]
  },
  "secret-scan": {
    "PreToolUse": [
      { "matcher": "write_to_file\\|replace_file_content",
        "hooks": [{ "command": "/Users/trainguyen/Documents/work/agent/hooks/agy/secret_scan.sh", "timeout": 5 }] }
    ]
  },
  "startup-reminder": {
    "PreInvocation": [
      { "command": "/Users/trainguyen/Documents/work/agent/hooks/agy/startup_reminder.sh", "timeout": 5 }
    ]
  },
  "done-gate": {
    "Stop": [
      { "command": "/Users/trainguyen/Documents/work/agent/hooks/agy/done_gate.sh", "timeout": 20 }
    ]
  }
}
```

`hooks/agy/memory_protect.sh` (Rule 17 — chặn Overwrite memory file):

```bash
#!/usr/bin/env bash
set -euo pipefail
in=$(cat)
path=$(jq -r '.toolCall.args.TargetFile // empty' <<<"$in")
ovw=$(jq -r '.toolCall.args.Overwrite // false' <<<"$in")
if [[ "$ovw" == "true" && "$path" == */agent/memory/* && -f "$path" ]]; then
  jq -n --arg p "$path" '{decision:"deny",
    reason:("Rule 17: CẤM Overwrite memory file " + $p + ". Chỉ APPEND bằng replace_file_content.")}'
else
  echo '{"decision":"ask"}'   # passthrough — xem lưu ý Phase 0
fi
```

`hooks/agy/done_gate.sh` (chặn báo Done khi đã sửa source mà chưa log/verify; chỉ chặn 1 lần/hội thoại để tránh loop):

```bash
#!/usr/bin/env bash
set -euo pipefail
in=$(cat)
conv=$(jq -r '.conversationId' <<<"$in"); tr=$(jq -r '.transcriptPath' <<<"$in")
flag="/tmp/agy_done_gate_${conv}"
if [[ -f "$flag" || ! -f "$tr" ]]; then echo '{}'; exit 0; fi
if grep -Eq '"TargetFile":"[^"]+\.(go|ts|tsx|py|sql)"' "$tr" && ! grep -q '05_progress' "$tr"; then
  touch "$flag"
  jq -n '{decision:"continue",
    reason:"Done-gate: phiên này đã sửa source nhưng CHƯA append 05_progress và CHƯA chạy verify_governance.py --workspace <ws>. Hoàn tất trước khi kết thúc."}'
else
  echo '{}'
fi
```

Đánh số trong mọi message hook phải theo GEMINI.md Core mới (hết lệch F5). Các script Claude Code cũ chuyển vào `hooks/_claude_code/` (giữ nếu anh còn dùng Claude Code).

> [!WARNING]
> **Phase 0 bắt buộc trước U3**: cài 1 hook log `matcher: "*"` ghi stdin ra file để xác minh (a) tên tool thực tế mà matcher nhận, (b) cấu trúc `toolCall.args`, (c) passthrough nên trả `ask` hay `allow` mà không phá policy duyệt hiện tại. Không đoán — đo trước rồi mới chốt script.

### U4 (F11) — Tái cấu trúc lessons: INDEX + topics

- Script Python (đọc `lessons.md`, KHÔNG ghi đè) parse từng `### [date] title` + `**Đúng:**` + `**Tags:**` → sinh:
  - `lessons/INDEX.md`: 1 dòng/pattern, nhóm theo taxonomy, ≤ 15KB.
  - `lessons/topics/{process-governance,go-backend,react-frontend,cdc-pipeline,scheduler,db-schema,security}.md`: nội dung đầy đủ.
- `lessons.md` gốc → copy thành `lessons_archive_2026-10.md` (giữ nguyên, có git).
- Thăng cấp các pattern tái phạm nhiều lần (Propose-Only khi Audit, Báo Done khi chưa verify runtime, Mid-session fix) vào Core GEMINI.md đúng như Rule #5 đã quy định.
- Đối soát: số entry INDEX = 237 = số `### [` trong file gốc (G6).

Mẫu dòng INDEX:

```markdown
## Go Backend
- [2026-10-15] #shutdown-goroutine-teardown-race — Shutdown phải WaitGroup chờ goroutine nền rồi mới Close DB/Redis; check port bằng lsof trước khi báo Done → topics/go-backend.md
```

### U5 (F9, F16) — Linter + phân tầng tài liệu

| Tier | Khi nào | Tài liệu bắt buộc | Linter |
|---|---|---|---|
| T0 | Hỏi đáp/tư vấn, không sửa file dự án | Không | Không chạy |
| T1 | Hotfix/micro | 01, 05, 08 | Chạy |
| T2 | Feature | T1 + 09_solution, 06_validation, 11_report | Chạy |
| T3 | Epic/Phase | Full set | Chạy |

Sửa `verify_governance.py`: bắt buộc `--workspace` (bỏ auto-pick), đọc tier từ dòng `tier: T1|T2|T3` trong `01_requirements_*.md`, chấp nhận `12_implementation_plan*.md` thay cho `implementation_plan.md`, `ROOT_DIR = Path(__file__).resolve().parents[2]`.

### U6 (F12, F13) — Dọn skill global

- **Không chạy** `organize_skills.sh` → chuyển sang `scripts/_deprecated/`, xoá dòng path lạc.
- Skill ngoài domain (Go/TS-React/Postgres/Mongo/Kafka/Redis/K8s/Security/Agent-ops) → chuyển sang `~/.gemini/skills_archive/` (ngoài vùng discovery, không xoá).

### U7 (F14, F15) — Vệ sinh

- Thêm `.gitignore` cho `models.env`; tạo `models.env.example`; nếu có key thật trong lịch sử git → rotate key.
- Settings → Project-level **Permission Grants** cho phép đọc/ghi `/Users/trainguyen/Documents/work/agent/memory` để linter và các bước governance chạy được trong sandbox.

---

## 4. Lộ trình triển khai (mỗi phase cần APPROVE riêng)

| Phase | Nội dung | Restore-point | Tiêu chí PASS |
|---|---|---|---|
| 0 | `git commit` local repo `agent/`; cài hook log `*` để đo | commit `pre-optimization` | Có file log stdin của PreToolUse/Stop |
| 1 | U1 + U2 (Core GEMINI.md, global pointer, skills.json, 10 skill) | commit | Không còn `<truncated`; menu `/` có skill; grep không còn `git push` |
| 2 | U3 (hooks.json + 4 script) | commit | Test: Overwrite file memory giả → bị `deny`; ghi `token=abc...` vào memory → bị `deny`; sửa `.go` không log → Stop bị `continue` đúng 1 lần |
| 3 | U4 (lessons INDEX/topics) | commit + `wc -l` trước/sau | 237 = 237; INDEX ≤ 15KB; file gốc còn nguyên |
| 4 | U5 + U6 + U7 | commit | Linter PASS trên 1 workspace T1 thật; T0 không bị chặn |

## 5. Câu hỏi cần anh quyết định

- **Q1**: Anh còn dùng Claude Code không? Nếu có, `CLAUDE.md` đang ở đâu (không có trong `agent/`) → giữ Rule #8 + hooks Claude Code; nếu không → bỏ Rule #8 và archive hooks cũ.
- **Q2**: `models.env` có chứa API key thật không? (quét pattern phổ biến ra 0, nhưng format `provider:model:key` gợi ý có).
- **Q3**: Đồng ý phân tầng tài liệu T0–T3 (T0 = hỏi đáp không cần workspace/linter) thay cho quy định hiện tại?
