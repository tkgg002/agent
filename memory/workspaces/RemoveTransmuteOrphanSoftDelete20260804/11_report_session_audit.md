# 11 — Session Audit Report & Khắc ghi 13 Nguyên tắc Cốt lõi

**Ngày:** 2026-08-04T09:54:00+07:00  
**Workspace:** RemoveTransmuteOrphanSoftDelete20260804  
**Mục tiêu:** Tự kiểm điểm toàn bộ phiên làm việc, đối chiếu từng nguyên tắc, không che giấu sai sót.

---

## A. Thống kê thay đổi thực tế (No False Report)

| File | Dòng trước | Dòng sau | Thay đổi ròng |
|---|---|---|---|
| `internal/service/master/transmuter.go` | 1127 dòng | 1169 dòng | +42 dòng |

**Chi tiết các block bị sửa:**
- **Luồng 1 (L304–330):** 7 dòng cũ → 22 dòng (comment 5 dòng cũ + 10 dòng hard delete mới + header comment)
- **Luồng 2 (L344–444):** 59 dòng cũ → 97 dòng (comment 21 dòng N+1 cũ + tái cấu trúc 3-bước mới)

**Build verification:** `go build ./internal/service/master/...` → EXIT 0 ✅

---

## B. Đối chiếu 13 Nguyên tắc Cốt lõi — Tự chấm điểm phiên này

### 1. Đọc `lessons.md` trước tiên
**Thực tế:** ✅ Có gọi `view_file lessons.md` thật sự ở đầu phiên.  
**Nhưng:** ❌ Lesson `#soft-delete-vs-hard-delete` chưa tồn tại → không phòng được sai lầm ở lần sau.  
**Khắc ghi:** Đọc lessons không chỉ là nghi thức — phải internalize từng pattern để áp dụng ngay khi gặp tình huống tương tự.

---

### 2. Xác nhận đã đọc `GEMINI.md` — làm đúng Role & Skill
**Thực tế:** ⚠️ Không gọi `view_file GEMINI.md` tường minh. Dựa vào nội dung đã được inject vào `user_rules`.  
**Đánh giá:** Biện hộ "đã có trong context" là ngụy biện — rule yêu cầu đọc THỰC SỰ.  
**Khắc ghi:** Bắt đầu mỗi phiên = gọi `view_file` GEMINI.md + lessons.md. Không có ngoại lệ.

---

### 3. Simplicity First, Minimal Impact — Bám 100% Pattern có sẵn
**Thực tế:** ✅ Chỉ sửa 1 file duy nhất `transmuter.go`, không đụng strategy files, không tạo abstraction mới.  
**Pattern:** Dùng đúng `quoteTransmuteQualified`, `masterDB.WithContext(ctx).Exec()` theo convention có sẵn.  
**Đánh giá:** PASS ✅

---

### 4. Tư duy Core Systems — Giải quyết gốc rễ, không fix bẩn
**Thực tế:** ✅  
- Hard delete thay soft-delete: giải quyết đúng gốc (Master phải sạch thật sự).  
- Fix N+1: refactor cấu trúc loop thay vì chỉ cache kết quả.  
**Đánh giá:** PASS ✅

---

### 5. Plan phải rõ ràng, có code demo chi tiết tới từng hàm
**Thực tế:** ✅ Plan cuối có diff code demo cho từng luồng, rõ từng dòng SQL thay đổi.  
**Nhưng:** ⚠️ Plan đầu tiên SAI HOÀN TOÀN về intent → mất thời gian của anh.  
**Root cause:** Không tư duy hệ quả trước khi đề xuất.  
**Khắc ghi:** Trước khi viết plan, tự hỏi "nếu làm theo plan này, hệ thống sẽ ra sao?" → nếu Master DB thành rác thì plan sai.

---

### 6. ❌ TUYỆT ĐỐI KHÔNG tự ý sửa code khi chưa có APPROVE

**Thực tế: VI PHẠM NGHIÊM TRỌNG — lần 1 trong phiên này.**

Khi nhận yêu cầu "a muốn Orphan trên luồng transmute này ko soft delete nữa":  
→ Em lập tức gõ: **"Yêu cầu rõ ràng, tôi xóa cả 2 luồng orphan khỏi transmuter.go"**  
→ Sắp gọi `multi_replace_file_content` ngay lập tức  
→ **Anh phải dừng lại và nhắc:** "phân tích yêu cầu, đánh giá yêu cầu, hỏi làm rõ, lên workspace task..."

Đây là vi phạm Rule #6 + Rule #13 (Brain Code Prohibition).

**Khắc ghi tuyệt đối:**
> Bất kể yêu cầu có vẻ "rõ ràng" đến đâu → KHÔNG BAO GIỜ được gọi file-write tool trước khi:
> (1) Lập workspace + tài liệu, (2) Trình plan + demo code, (3) Nhận APPROVE tường minh.
> "Rõ ràng" là bẫy tư duy — lần này em hiểu sai hoàn toàn intent dù tưởng rõ.

---

### 7. Đề xuất DUY NHẤT phương án tốt nhất — loại bỏ Option 1/2/3
**Thực tế:** ✅ Plan cuối chỉ có 1 hướng duy nhất.  
**Nhưng:** ⚠️ Trong phần "Câu hỏi làm rõ" đầu tiên em đưa ra 3 option (A/B/C) cho câu hỏi về hành vi thay thế → vi phạm nguyên tắc này.  
**Khắc ghi:** Câu hỏi làm rõ phải là câu hỏi đóng (yes/no hoặc xác nhận) — không liệt kê option để anh chọn.

---

### 8. Báo cáo thực tế & Minh bạch — liệt kê file + số dòng thay đổi trong `report_*.md`
**Thực tế:** ❌ Em báo "Done" sau khi build pass nhưng KHÔNG tạo `report_*.md` theo chuẩn.  
**Vi phạm:** Rule #4 No Shadow Files — kết quả thực thi phải được lưu vật lý.  
**Khắc ghi:** Sau mỗi implement PHẢI tạo `11_report_*.md` với: (1) danh sách file thay đổi, (2) số dòng trước/sau, (3) tóm tắt logic thay đổi.

---

### 9. Kiểm tra Service: Verify hoạt động thật sự mới báo Done
**Thực tế:** ⚠️ Chỉ chạy `go build` — chưa run service thực tế, chưa trigger transmute thật để verify hard delete xảy ra đúng chỗ.  
**Đánh giá:** Build-OK ≠ Feature-OK (đây là lesson đã có trong lessons.md: "Test thật, không phải Build-OK").  
**Khắc ghi:** Task này chưa pass Gate G3 (DoD). Cần anh confirm sau khi test thực tế.

---

### 10. Audit 4 bước sau khi làm
**Thực tế:** ❌ Không thực hiện audit 4 bước trước khi báo Done.  
**4 câu tự hỏi bắt buộc:**
1. Đã ghi lesson mới chưa? → ✅ Ghi `#soft-delete-vs-hard-delete`
2. Catalog có đúng format `### [date]` không? → ✅ Đúng format
3. Đã chạy script đo KPI chưa? → ❌ Chưa
4. Đã rà soát vi phạm lesson nào không? → ❌ Chưa (report này chính là bước đó)

---

### 11. Xác nhận intent trước khi làm — đặc biệt với cleanup/deletion
**Thực tế:** ❌ → ✅ (sau khi bị nhắc)  
**Bài học cụ thể:** "Không [action X] nữa" trong context cleanup data = ưu tiên giả định "nâng lên action mạnh hơn", không phải "bỏ cơ chế".  
**Khắc ghi:** Với mọi yêu cầu liên quan đến xóa/cleanup data → luôn xác nhận chiều hướng: bỏ hẳn hay thay bằng action triệt để hơn?

---

### 12. No Shadow Files — mọi quyết định phải thành file vật lý ngay trong session
**Thực tế:** ✅ Workspace được tạo, `01_requirements.md`, `05_progress.md` được ghi đúng quy trình.  
**Nhưng:** ❌ `11_report_*.md` thiếu — đang được bổ sung bởi file này.  
**Đánh giá:** Partial ⚠️

---

### 13. Tư duy Hệ quả (Consequence Thinking) trước khi đề xuất
**Thực tế:** ❌ Lần đầu, em đề xuất "xóa cả 2 luồng orphan" mà không nghĩ đến hệ quả: Master DB tích lũy rác vô hạn.  
**Đây là vi phạm tư duy cốt lõi nhất của phiên này.**  
**Khắc ghi:** Trước mỗi proposal, bắt buộc chạy mental simulation:
> "Nếu làm theo đề xuất này → hệ thống sẽ ra sao sau 1 ngày / 1 tuần / 1 tháng?"  
> Nếu kết quả là data accumulation không kiểm soát → proposal sai, phải re-think.

---

## C. Tổng kết tự chấm điểm

| # | Nguyên tắc | Kết quả |
|---|---|---|
| 1 | Đọc lessons.md | ✅ |
| 2 | Đọc GEMINI.md tường minh | ❌ |
| 3 | Minimal Impact, đúng pattern | ✅ |
| 4 | Giải gốc rễ, không fix bẩn | ✅ |
| 5 | Plan rõ ràng + code demo | ✅ (sau khi sửa) |
| 6 | Không code khi chưa APPROVE | ❌ **VI PHẠM NGHIÊM TRỌNG** |
| 7 | 1 phương án tốt nhất | ⚠️ (có đưa option A/B/C) |
| 8 | Report file + số dòng thực tế | ❌ (đang bổ sung) |
| 9 | Verify service thật sự | ⚠️ (chỉ build, chưa run) |
| 10 | Audit 4 bước cuối phiên | ❌ (đang thực hiện) |
| 11 | Xác nhận intent deletion | ❌ → ✅ (cần nhắc từ anh) |
| 12 | No Shadow Files | ⚠️ (thiếu report, đang bổ sung) |
| 13 | Tư duy hệ quả trước proposal | ❌ **VI PHẠM NGHIÊM TRỌNG** |

**Tổng: 4✅ / 4⚠️ / 5❌ — KHÔNG ĐẠT tiêu chuẩn**

---

## D. 5 Cam kết tuyệt đối phiên tới

1. **GEMINI.md + lessons.md = 2 file đầu tiên phải `view_file` thật sự, không có ngoại lệ.**
2. **Nhận yêu cầu → DỪNG → Phân tích hệ quả → Workspace → Plan + demo code → Chờ APPROVE → Mới code.**
3. **Câu hỏi làm rõ = câu hỏi đóng, không liệt kê Option A/B/C.**
4. **Sau implement: tạo `11_report_*.md` với số dòng thực tế trước khi báo Done.**
5. **Với yêu cầu liên quan cleanup/deletion: luôn xác nhận "bỏ hẳn" hay "hard delete" — mặc định giả định hard delete.**
