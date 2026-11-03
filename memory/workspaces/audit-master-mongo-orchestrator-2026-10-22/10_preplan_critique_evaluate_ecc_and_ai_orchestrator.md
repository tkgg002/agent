---
tier: T2
suffix: evaluate_ecc_and_ai_orchestrator
---

# 10 — Pre-Plan Adversarial Critique (S2.5 — `DG0`): Phản biện Thực tế `affaan-m/ecc` vs `ai-orchestrator`

## 1. Phản biện Ngược các Tuyên bố trong `affaan-m/ecc` (`C-REQ`, `C-REALITY`, `C-CONTRA`)
1. **`C-REQ` (Bản chất `ecc` v2.2.3):** `package.json:1-5` và `AGENTS.md:1-6` định nghĩa `ecc-universal` (`v2.2.3`) là bộ plugin/harness gồm `71` agents, `311` skills, `97` commands kết hợp với `ecc2` (`ecc-tui` Rust crate tại `ecc2/Cargo.toml:1-6`).
2. **`C-REALITY` (Mâu thuẫn giữa Luật trong `AGENTS.md` và Code thực tế trong `ecc2/`):**
   - Tại `AGENTS.md:135,142`, `ecc` tuyên bố quy tắc vàng: *"Many small files over few large ones. 200-400 lines typical, 800 max... Functions small (<50 lines), files focused (<800 lines)"*.
   - Nhưng kiểm tra vật lý trên đĩa trong `ecc2/src/` cho thấy:
     - `ecc2/src/tui/dashboard.rs`: **`15,460` dòng** (gấp **19.3 lần** giới hạn 800 dòng!).
     - `ecc2/src/main.rs`: **`12,830` dòng** (gấp **16 lần** giới hạn 800 dòng!).
     - `ecc2/src/session/manager.rs`: **`8,181` dòng** (gấp **10.2 lần** giới hạn 800 dòng!).
   - Điều này chứng minh: **Rules chỉ viết bằng Markdown mà không có Hook/Linter cưỡng chế vật lý (`agy_guard.py` / `verify_governance.py` / `NOW <= HEAD`) thì chính tác giả bộ rules cũng vi phạm nghiêm trọng.**
3. **`C-CONTRA` (Không bê nguyên xi `ecc` vào `ai-orchestrator`):**
   - Tuyệt đối không bê nguyên 311 skills hay 71 agents của `ecc` vào `ai-orchestrator` vì sẽ gây phình nổ context window (vi phạm giới hạn `Always-On Surface <= 24KB` của `/agent`). Thay vào đó, `ai-orchestrator` chỉ nên chắt lọc cơ chế **Cross-SubTask File Overlap Detection & Caller Impact Check** ở tầng Go Engine nếu User muốn nâng cấp.
