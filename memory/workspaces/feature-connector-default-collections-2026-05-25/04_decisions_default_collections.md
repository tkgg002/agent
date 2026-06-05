# 04_decisions_default_collections — ADR Log

> **Phase**: `default_collections`
> **Format**: ADR-XXX (Architecture Decision Record)

---

## ADR-001: Chọn Phương án A — FE-only hint UX improvement

**Date**: 2026-05-25
**Status**: ✅ ACCEPTED (chờ user confirm)

### Context

Audit pipeline cho thấy:
- FE `compactConfig` drop empty value → BE không nhận `collection.include.list` khi user để trống.
- BE handler không inject default → forward as-is.
- Debezium connector mặc định CDC all khi missing config key.

→ **Runtime behavior đã đúng yêu cầu user**. Chỉ thiếu UX hint.

### Options considered

| # | Option | Pros | Cons |
|---|---|---|---|
| A | FE-only: thêm `extra` text + list view display fallback | Minimal impact (CLAUDE §6), không đụng BE, không risk regression runtime | Phụ thuộc Debezium default — nếu version đổi default sẽ silent break |
| B | BE-side: handler inject explicit `collection.include.list: "*.*"` khi missing | Explicit > implicit, behavior độc lập Debezium version | "*.*" có thể không hợp lệ Debezium 1.x — cần test version-specific; risk regression |
| C | BE inject `database.include.list` only, skip collection key | Đảm bảo Mongo DB filter chuẩn | Đã làm điều này — Mongo connector chỉ có database.include.list + collection.include.list; phương án không thực sự khác A |
| D | UI redesign: multi-select picker từ Mongo runtime | UX cao nhất | Cần BE endpoint mới, scope lớn → defer |

### Decision

**Chọn A**.

### Rationale

1. **Audit chứng minh runtime đúng** — không có lý do thay đổi BE/Debezium nếu chỉ là UX gap.
2. CLAUDE.md §6 "Simplicity First & Demand Elegance — minimal impact". B/D over-engineer cho gap nhỏ.
3. Risk silent break khi Debezium đổi default (cons của A) được mitigate bằng:
   - Smoke test M4 verify thực tế topic có message.
   - Note version Debezium trong M0.3 và `report` để future readers track.
4. D defer chính thức sang phase riêng (xem requirements §4 Out of scope).

### Consequences

- ✅ Code change rất nhỏ (2-3 file edit, < 20 lines).
- ✅ KHÔNG cần migration / restart BE.
- ⚠️ Nếu upgrade Debezium version → cần re-verify default behavior.
- ⚠️ Hint text phải đa ngôn ngữ trong tương lai nếu codebase i18n.

### Related lessons

- L-CDC-golden-rule (lessons.md): Mongo read-only, không đụng source. Phương án A tuân thủ.
- L-cheat-DB-ALTER-in-report (lessons.md): không hack DB. Phương án A không đụng DB.

---

## ADR-002: Wording của hint text + placeholder

**Date**: 2026-05-25
**Status**: ⏳ PENDING user input (default-suggested)

### Context

Cần thống nhất wording để UX rõ ràng, không gây hiểu lầm.

### Options

| # | `extra` text | `placeholder` |
|---|---|---|
| 1 | "Để trống nếu muốn CDC toàn bộ collections của database. Phân cách bằng dấu phẩy nếu chỉ muốn CDC một số collection cụ thể (vd: users,orders)." | `users,orders (để trống = tất cả)` |
| 2 | "Để trống = CDC tất cả collections." | giữ nguyên `users,orders,payments` |
| 3 | "Bỏ trống nếu muốn theo dõi tất cả collections trong database này." | `(tùy chọn) users,orders,payments` |

### Decision

**Suggest option 1**. Có thể downgrade option 2 nếu Antd Form layout chật.

### Rationale

- Option 1 đầy đủ context (giải thích cả 2 case empty + filled), phù hợp Ops admin chưa quen.
- Option 2 ngắn nhưng có thể user vẫn không hiểu "CDC" là gì.
- Option 3 dùng "theo dõi" friendly nhưng ko match terminology trong project (project dùng "CDC").

### Consequences

- Wording chốt qua user feedback ở M2 review.
- Nếu i18n có → tạo key `connector.form.collections.extra.v1` để tránh conflict version sau.

---

## ADR-003: List view fallback display `(All collections)`

**Date**: 2026-05-25
**Status**: ✅ ACCEPTED

### Context

R3 yêu cầu phân biệt visual "không filter" vs "đã filter".

### Options

| # | Display khi empty | Style |
|---|---|---|
| A | `(All collections)` | italic gray |
| B | `*` | bold |
| C | `Tất cả` | normal |
| D | `—` (em dash) | normal — cùng style hiện tại |

### Decision

**Chọn A**.

### Rationale

- A tự giải thích, không cần học convention.
- B dễ nhầm với regex pattern Mongo / Debezium.
- C OK nhưng inconsistent nếu UI có chỗ khác đang dùng English.
- D không phân biệt được "chưa cấu hình" vs "intentional all" — fail R3.

### Consequences

- Hardcode string `(All collections)`. Nếu i18n → key `connector.list.collections.all`.
- Italic gray để de-emphasize (không phải data thật của user nhập).

---

## ADR-004: KHÔNG validate format collection.include.list trong phase này

**Date**: 2026-05-25
**Status**: ✅ ACCEPTED (defer)

### Context

User có thể nhập sai format (vd: thiếu prefix DB, có dấu cách, regex sai). Validation sẽ improve UX.

### Decision

**Defer sang phase `connector-filter-validate`**.

### Rationale

- Out of scope (xem requirements §4).
- Phase hiện tại chỉ giải quyết "empty = all", validation là vấn đề độc lập.
- Cần research Debezium version để biết format chuẩn (1.x vs 2.x khác).

### Consequences

- Risk: user nhập sai format → connector tạo lỗi → phải fix manual. Acceptable cho phase này.

---

## ADR-005: KHÔNG đụng BE / Debezium config

**Date**: 2026-05-25
**Status**: ✅ ACCEPTED

### Context

Một số reviewer có thể propose "BE should explicitly inject collection.include.list as wildcard for clarity".

### Decision

**KHÔNG** đụng BE / Debezium config trong phase này.

### Rationale

1. CLAUDE.md §6 minimal impact.
2. User directive: "đảm bảo ko sửa code rồi hẵng chạy tiếp néh" — đặc biệt phase Brain planning chỉ ra phương án minimal.
3. ADR-001 đã chứng minh runtime đúng. Thêm code = thêm risk.
4. "Trust framework default" là pattern accepted khi default rõ ràng và documented (Debezium docs có note).

### Consequences

- Implicit dependency on Debezium default. Mitigate: note version trong report + smoke test.
- Nếu future Debezium đổi default → tạo phase `connector-explicit-defaults` để inject explicit.
