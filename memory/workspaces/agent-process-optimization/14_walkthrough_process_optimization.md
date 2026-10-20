# 14 — Walkthrough: Hướng dẫn Kiểm tra Thư mục `/agent` Sau Tối ưu

## 1. Cấu trúc Thư mục `/Users/trainguyen/Documents/work/agent` Đã Gọn Gàng
```text
agent/
├── .agent-project.json          # Manifest khai báo checks & runtime_check của repo agent
├── .agents/
│   ├── hooks.json               # 5 lifecycle hooks (PreInvocation, PreToolUse, PostInvocation, Stop)
│   └── skills.json              # Đăng ký agent/skills và ~/.agent/skills
├── GEMINI.md                    # Hiến pháp lõi 11,928B (< 12KB, đủ 23 Rules 0–22, Master Workflow S0–S6)
├── OPERATOR_MAP.md              # Bản đồ điều phối 10 Core Skills & Tooling
├── hooks/
│   ├── README.md                # Tài liệu vận hành H1–H7
│   ├── _archive_claude/         # Lưu trữ 7 script .sh cũ của Claude Code (chmod -x)
│   └── agy/
│       ├── agy_guard.py         # Dispatcher Python thực thi H1–H7
│       └── tests/
│           └── test_agy_guard.py # 16 fixture tests (16/16 PASS)
├── skills/                      # 10 Core Skills chuẩn Antigravity (<name>/SKILL.md)
│   ├── audit-propose-only/SKILL.md
│   ├── brain-delegate/SKILL.md
│   ├── context-manager/SKILL.md
│   ├── debug-agent/SKILL.md
│   ├── laya-router/SKILL.md
│   ├── muscle-execute/SKILL.md
│   ├── qa-agent/SKILL.md
│   ├── security-agent/SKILL.md
│   ├── session-wrapup/SKILL.md
│   └── workspace-docs/SKILL.md
├── tooling/
│   ├── laya_gate.py             # CLI Decision Gate /laya_router (Mode A & Mode B)
│   ├── build_lessons_index.py   # Sinh chỉ mục 2 tầng INDEX.md (<15KB) + topics/*.md
│   ├── verify_governance.py     # Process Linter theo Tier T0–T3
│   ├── governance_metrics.sh    # Đo KPI tuân thủ hệ thống Antigravity
│   └── tests/                   # 11 unit tests cho laya_gate, build_lessons_index, verify_governance
├── workflows/
│   ├── README.md                # Giải thích chuyển đổi sang skills/
│   └── _archive/                # Lưu trữ 96 file workflow phẳng cũ
└── memory/
    ├── global/
    │   ├── lessons.md           # Bản gốc 241 bài học (Read-Only khi đánh chỉ mục)
    │   └── lessons/
    │       ├── INDEX.md         # Chỉ mục 14,576B (<15KB, 241/241 bài học)
    │       └── topics/          # 5 file chủ đề T1–T5
    └── workspaces/
        └── agent-process-optimization/ # Hồ sơ đầy đủ của đợt tối ưu này
```

## 2. Các Lệnh Kiểm tra Nhanh cho User
```bash
# 1. Kiểm tra toàn bộ 27 unit & fixture tests (chạy trong ~0.04s)
python3 -m unittest discover -s /Users/trainguyen/Documents/work/agent/tooling/tests -p 'test_*.py' -v
python3 -m unittest discover -s /Users/trainguyen/Documents/work/agent/hooks/agy/tests -p 'test_*.py' -v

# 2. Kiểm tra kích thước GEMINI.md (< 12,000B) và số lượng Rule (= 23)
wc -c /Users/trainguyen/Documents/work/agent/GEMINI.md
grep -cE '^### [0-9]+\.' /Users/trainguyen/Documents/work/agent/GEMINI.md

# 3. Kiểm tra chỉ mục bài học 2 tầng (241 = 241, INDEX.md < 15,000B)
python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py

# 4. Chạy Process Linter cho workspace hiện tại
python3 /Users/trainguyen/Documents/work/agent/tooling/verify_governance.py --workspace agent-process-optimization

# 5. Chạy bảng đo KPI hệ thống
bash /Users/trainguyen/Documents/work/agent/tooling/governance_metrics.sh
```
