# FE CMS — Label/Input Contrast Fix Plan

> Date: 2026-04-23 02:15 ICT
> Trigger: Boss — "text ở label, input bị trùng màu dẫn đến ko trực quan"
> Workspace: feature-cdc-integration
> SOP: Bug Handling 7-stage

## 1. Symptom

Trong các modal/form FE CMS (MasterRegistry, SchemaProposals, TransmuteSchedules, Preview, AddMappingModal, Login), text ở `<label>` và placeholder input bị trùng / xỉn màu làm người dùng khó phân biệt label vs value. Không runtime broken, không gãy chức năng — usability/a11y regression.

## 2. 5-whys Root Cause

| # | Question | Answer |
|:--|:---------|:-------|
| 1 | Tại sao label/input trùng màu? | `src/index.css:21` đặt `color: var(--text)` trên `:root` → cascade vào mọi AntD component. |
| 2 | Tại sao `--text` gây low contrast? | Dark-mode swap ở `@media (prefers-color-scheme: dark)` đổi `--text: #9ca3af` (gray) trên nền AntD trắng → ~3.4:1 < WCAG AA 4.5:1. |
| 3 | Tại sao CSS này tồn tại? | Template cruft `create-vite` starter — CSS vars (`--accent`, `--social-bg`, `.button-icon`) không component nào xài (grep verified). |
| 4 | Tại sao build không catch? | tsc/lint/E2E chỉ check structure + logic; không có visual regression / contrast audit. |
| 5 | Tại sao tích luỹ? | Sprint 1-5 priority backend; FE style nợ kỹ thuật chưa dọn. |

**Root cause (meta)**: Scaffold cruft + missing theme contract (no AntD ConfigProvider) + OS dark-mode auto-swap xung đột light-only AntD components.

## 3. Cross-service Scope Check

```bash
grep -rnE "var\(--text|var\(--bg|var\(--accent|var\(--border|var\(--code-bg" src/ | head -20
# → ALL hits ONLY in src/index.css. Zero component usage.

grep -rnE "#social|button-icon" src/
# → ALL hits ONLY in src/index.css. Dead selector.
```

Scope: **1 file** — `cdc-cms-web/src/index.css`. Không cross-service.

## 4. Approach comparison

| | Option A — Rewrite minimal | Option B — AntD ConfigProvider + dark algo | Option C — Spot-remove `color` |
|:-|:-|:-|:-|
| Invasive | LOW (1 file) | MEDIUM (main.tsx + index.css + theme.ts) | LOW (1 file) |
| Dead code | Removed | Kept (vars may confuse) | Kept |
| Dark-mode support | No (light only, explicit) | Yes | Ambiguous |
| Risk | Low | Medium (must test both modes) | Low but incomplete |
| Rule 6 fit | ✅ Simplest | ❌ Over-engineer | ⚠️ Half-done |

**Chọn Option A**.

## 5. Plan (Stage 3 execution)

### File: `cdc-cms-web/src/index.css`
- DELETE: all CSS custom properties (`--text`, `--text-h`, `--bg`, `--border`, `--code-bg`, `--accent*`, `--social-bg`, `--shadow`, `--sans`, `--heading`, `--mono`).
- DELETE: `color-scheme: light dark`, `color: var(--text)`, `background: var(--bg)` từ `:root`.
- DELETE: `@media (prefers-color-scheme: dark)` block.
- DELETE: `#social .button-icon` selector.
- DELETE: `h1, h2, p, code, .counter` overrides (AntD Typography handles).
- KEEP: `#root { width:100%; min-height:100vh; box-sizing:border-box }`.
- KEEP: `body { margin: 0 }`.
- ADD: font stack fallback (system-ui) + antialiased rendering.

**Expected diff**: 122 LOC → ~15 LOC.

## 6. Acceptance criteria (Stage 4 VERIFY)

- [ ] `npx tsc --noEmit` EXIT=0.
- [ ] `npm run build` EXIT=0 (if run).
- [ ] Vite HMR reload `/src/index.css` HTTP 200.
- [ ] Visually: label + input text cả light OS + dark OS — AntD default light theme stays stable.
- [ ] Grep confirm dead vars không còn reference nào.
- [ ] Không regression ở Login/Dashboard render.

## 7. Anti-cases

- ❌ Không thêm AntD ConfigProvider theme (over-engineer, scope creep).
- ❌ Không chạm tsconfig/vite.config (không liên quan).
- ❌ Không đụng AntD `<Sider theme="dark">` trong App.tsx (đó là sidebar intentional dark).
- ❌ Không refactor màu component-level (MasterRegistry Tag color, etc.).

## 8. Files to edit

1. `cdc-cms-web/src/index.css` — rewrite minimal.

## 9. Skills

- CSS cleanup, AntD theming contract, WCAG contrast audit.

## 10. Stage 5+ docs plan

- `03_implementation_fe_contrast_fix.md` (sau Stage 4 VERIFY).
- APPEND `05_progress.md` entry timestamp + summary.
- APPEND `agent/memory/global/lessons.md` — lesson "Scaffold CSS cruft overrides component library" (Global Pattern).
