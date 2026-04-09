# agent/memory/global/lessons.md

> Format: Mỗi lesson PHẢI theo cấu trúc dưới. Tags để Brain filter nhanh.

---

## [2026-02-25] Brain quên tạo Workspace trước khi làm

- **Trigger**: User giao task "Upgrade Core Brain/Muscle System (Hướng 5)"
- **Root Cause**: Brain bắt đầu plan và tạo implementation_plan.md artifact mà KHÔNG khởi tạo workspace trước. Vi phạm Rule 7 (GEMINI.md) và Convention #7 (conventions.md).
- **Correct Pattern**:
  1. Nhận task → Tạo workspace folder ngay (`agent/memory/workspaces/[name]/`)
  2. Tạo `00_context.md` với scope
  3. Sau đó mới lập plan và bắt đầu làm
- **Tags**: #workspace #brain #rule7 #process

---

## [2026-02-25] Brain hỏi User về quyết định đã có trong plan

- **Trigger**: Sau khi hoàn thành P1+P2, Brain hỏi User "có muốn làm P3 không" thay vì tự quyết định
- **Root Cause**: Vi phạm Rule 2 (Autonomous). Goal của User là "upgrade core hoàn chỉnh nhất" — P3 đã được define trong plan, không có blocker → Brain phải tự thực hiện
- **Correct Pattern**: Nếu task đã có trong plan và không có blocker/conflict → tự làm, không hỏi. Chỉ hỏi User khi: (1) có conflict rõ ràng, (2) cần thêm thông tin không thể tự suy luận, (3) quyết định có risk cao cần approval
- **Tags**: #brain #rule2 #autonomous #hand-holding

---

## [2026-02-25] Phân định vai trò Brain/Muscle chưa rõ ràng trong task Research

- **Trigger**: User nhận xét "có cảm giác chỉ mình brain làm" khi thực hiện so sánh logic.
- **Root Cause**: Brain (Antigravity) trực tiếp gọi các tool research (`find`, `view_file`, `grep`) mà không thông qua quy trình delegate rõ ràng cho Muscle (CC CLI) hoặc các Subagents. Vi phạm Rule 1 (Separation & Subagent Strategy).
- **Correct Pattern**: 
  1. Brain (Chairman): Lập kế hoạch cao tầng, định nghĩa "Definition of Done".
  2. Brain (Delegate): Gọi Muscle (Chief Engineer) hoặc Subagent thực hiện các lệnh CLI, đọc file và báo cáo kết quả chi tiết.
  3. Brain (Synthesize): Tổng hợp dữ liệu từ Muscle/Subagent để đưa ra kết luận và báo cáo cuối cùng cho User.
- **Tags**: #brain #muscle #delegate #separation #process

---

## [2026-02-25] Brain quên ghi file Artifact vào Workspace

- **Trigger**: User phát hiện `walkthrough.md` chỉ có ở brain/ artifact dir và `02_plan.md` trống.
- **Root Cause**: Brain tập trung vào việc tạo artifact theo default system nhưng quên trách nhiệm duy trì "Bộ não dự án" tại workspace folder theo Rule 7.
- **Correct Pattern**: Mỗi khi tạo `walkthrough.md` hoặc `implementation_plan.md` (dạng artifact), Brain/Muscle PHẢI đồng bộ nội dung tương ứng vào `02_plan.md` và `walkthrough.md` (hoặc `todo.md`) trong workspace folder để lưu giữ context lâu dài.
- **Tags**: #brain #rule7 #workspace #memory #persistence
---

## [2026-02-25] Brain quên tracking model và cập nhật lessons.md khi bị sửa

- **Trigger**: User góp ý về việc thiếu tag model trong các Phase đầu và nghi ngờ tính xác thực của model đang dùng ("nói là gemini-3-pro-high nhưng có thật không?").
- **Root Cause**: 
  1. Quên quy tắc "Ghi lesson ngay lập tức khi bị sửa mid-session" (Rule #7).
  2. Thiếu cơ chế **Proof of Model** (Bằng chứng Model): Chỉ ghi log bằng chữ mà không có bằng chứng kỹ thuật từ hệ thống (env/config).
- **Correct Pattern**:
  1. Khi User sửa lỗi hoặc góp ý về quy trình → Dừng lại 1 bước, ghi ngay vào `lessons.md` trước khi làm tiếp.
  2. **Proof of Model**: Trước mỗi task lớn, Brain/Muscle phải chạy lệnh `env | grep MODEL` hoặc `claude config list` và chụp lại output để chứng minh model thực tế đang được hệ thống sử dụng.
- **Tags**: #brain #rule7 #lessons #tracking #transparency #verification

---

---

---

## [2026-02-26] Nhầm lẫn Logic/Workspace (Carelessness)

- **Trigger**: User yêu cầu thực hiện Logic "X" nhưng Brain lại sử dụng Workspace của Logic "Y" (do cùng module hoặc bối cảnh gần nhau).
- **Root Cause**: **Heuristic Failure** - Sử dụng phỏng đoán sai lầm về sự tương đồng của các feature. Gây ra "Context Pollution" và sai lệch trong việc tracking tiến độ.
- **Correct Pattern**: 
  1. **Atomic Workspace Rule**: Mỗi Logic/Feature có bản chất output khác biệt = 1 Workspace folder riêng biệt.
  2. **Mandatory Scope Verification**: Trước khi khởi tạo `00_context.md`, phải verify metadata từ repository gốc.
- **Tags**: #workspace #atomic-context #carelessness

---

## [2026-02-26] Cập nhật nhầm Config File (Path Management)

- **Trigger**: Brain cập nhật file cấu hình tại đường dẫn "A" thay vì đường dẫn "B" (file gốc của hệ thống).
- **Root Cause**: **Path Bias** - Ưu tiên các file trong cây thư mục làm việc hiện tại mà không kiểm tra cấu hình biến môi trường hoặc chỉ định của User.
- **Correct Pattern**: Luôn sử dụng `ls -la` và xác minh đường dẫn tuyệt đối (`~`, `/etc`, v.v.) trước khi sửa đổi file hệ thống quan trọng.
- **Tags**: #config #path #carelessness

---

## [2026-02-26] Vi phạm giao thức Skill-Listing (Protocol Negligence)

- **Trigger**: Brain hoàn thành Task nhưng quên liệt kê danh sách kỹ thuật/công cụ đã sử dụng.
- **Root Cause**: **Operational Inertia** - Tập trung thái quá vào nội dung trả lời (Short-term goal) mà bỏ qua kỷ luật định dạng (Long-term protocol).
- **Correct Pattern**: Coi Skill-listing là một phần không thể tách rời của "Definition of Done". Không có Skill-listing = Task chưa hoàn thành.
- **Tags**: #protocol #skill-listing #discipline

---

## [2026-02-27] Mongoose Execution Pitfalls

1.  `[Execution] Query Constructor Mismatch`: Khi dùng dynamic instantiation như `new config.subQueryClass(subQueryParams)`, cần chắc chắn structure của params match 100% với signature của constructor. Trường hợp args tách lẻ sẽ nhận fail nếu nạp vào nguyên 1 data object.
2.  `[Execution] Mongoose Find vs GetAll`: Hàm helper như `MongoFuncHelper.$getAll` đôi khi tự ngầm định append schema filter (`isDelete: false`). Nếu query 1 bảng không thiết kế field này, query sẽ âm thầm trả về rỗng. Cần check source core thật kĩ và fallback lại dùng basic Mongoose function như `.find()` của schema model.
3.  `[Execution] Mongoose Array Map Mutation`: Khi loop array của Mongoose Documents bằng `.map()`, việc gán thẳng data mới vào property (như `merchant.activeAt = ...`) có thể không hoạt động hoặc không được truy xuất đúng lúc render báo cáo. Do tính chặt chẽ của reference schema, cần safe convert (`.toObject()` / `lean()`) hoặc return 1 `{ ...rawMerchant, newProp }` immutable mới hoàn toàn.
- **Tags**: #mongoose #execution #mutation #lean #query

---


## [2026-02-27] Lỗi Wrapper Model Assumption (Heuristic Over-confidence)
- **Trigger**: Báo cáo hoàn thành task nhưng gặp lỗi `Model.aggregate is not a function` ngay khi chạy thực tế.
- **Root Cause**: 
  1. **Assumption Failure**: Brain mặc định Model trong handler là Mongoose Model thuần, trong khi thực tế nó là một Wrapper Class (`MerchantModel`) không expose hàm `aggregate`.
  2. **Rule #3 Violation**: Báo "Xong" khi chỉ mới "viết xong code", chưa chạy thử hoặc viết unit test (Muscle Tester) bất chấp lệnh `yarn tsc` fail (dù là fail cũ).
- **Correct Pattern**: 
  1. **Interface Verification**: Luôn kiểm tra định nghĩa class/model (`view_file`) trước khi sử dụng các hàm không phổ biến trong wrapper.
  2. **Muscle Tester**: BẮT BUỘC tạo hoặc cập nhật 1 bản unit test tối giản cho logic mới trước khi báo Done. Không chấp nhận việc bỏ qua lỗi compiler.
- **Tags**: #carelessness #protocol #testing #assumption

---

## [2026-02-26] Model Shadowing & Task Pollution (Data Integrity)

- **Trigger**: Ghi nhận sai Model sử dụng cho Agent và nhồi nhét log "Sửa lỗi vận hành" vào log "Tiến độ tính năng".
- **Root Cause**: 
  1. **Model Hallucination**: Tự mặc định thông tin model theo thói quen thay vì đọc từ `env`/`config`.
  2. **Separation Failure**: Không phân tách được luồng "Meta-work" (về hệ thống) và luồng "Project-work" (về tính năng).
- **Correct Pattern**: 
  1. **Verify Before Log**: Model ID phải được xác thực bằng lệnh kỹ thuật (`claude config list`).
  2. **Clean Progress Log**: Log tiến độ workspace chỉ chứa sự kiện của Feature. Các sửa lỗi hệ thống/bài học ghi vào `lessons.md`.
- **Tags**: #metadata #integrity #logging #separation

---

## [2026-02-27] TÁI PHẠM: Brain bỏ qua Session Start Checklist với task "nhỏ" (Recidivism Pattern)

- **Trigger**: User giao task tạo 1 entity/logic **X** mới. Brain nhảy thẳng vào đọc file, tạo entity, update index — KHÔNG tạo workspace.
- **Root Cause thực sự (Deep Root)**:
  1. **Lesson Misclassification**: Lesson trước đã tồn tại, Brain ĐÃ ĐỌC — nhưng phân loại task **X** là "task nhỏ, 1 file, không cần workspace". Đây là **False Heuristic** nguy hiểm.
  2. **Checklist Gate Bypass**: Session Start Checklist (Rule #7) bị bỏ qua vì coi task đơn giản. Không có cơ chế hard-gate nào ngăn Brain làm việc trước khi tạo workspace.
  3. **Scope Blindness**: Task "tạo entity/logic **X** mới" thực ra ảnh hưởng đến 2+ file trong service **Y** — đủ điều kiện cần workspace riêng theo **Atomic Workspace Rule**.
- **Correct Pattern — Zero Exception Hard Rules**:
  1. **Gate #0 — MANDATORY FIRST**: Trước BẤT KỲ tool call nào (kể cả `view_file`), PHẢI check: "Task này có workspace chưa?" → Nếu chưa → TẠO WORKSPACE TRƯỚC, sau đó mới làm.
  2. **Workspace Trigger**: Task có ≥2 file bị ảnh hưởng HOẶC liên quan đến entity/feature mới HOẶC mất >5 phút → BẮT BUỘC có workspace.
  3. **Zero Exception Rule**: KHÔNG có khái niệm "task nhỏ không cần workspace". Nếu tạo/sửa output file → có workspace để track.
  4. **Penalty Pattern**: Nếu Brain đã bắt đầu làm mà chưa tạo workspace → Dừng ngay, tạo workspace, ghi lessons.md, SAU ĐÓ mới tiếp tục.
- **Global Pattern [Brain classifies task X as "small" → skips workspace]**: Luôn WRONG. Zero exception.
- **Tags**: #workspace #brain #rule7 #recidivism #session-start-checklist #zero-exception

---

## [2026-02-27] Vi phạm Metadata Integrity trong Progress Log (Protocol Negligence)

- **Trigger**: Brain tạo `05_progress.md` nhưng sử dụng định dạng custom, thiếu Model ID và không tuân thủ mẫu table của dự án.
- **Root Cause**:
  1. **Operational Blindness**: Tập trung vào nội dung task (logic export) mà quên mất các quy tắc định dạng metadata bắt buộc trong Rule #7.
  2. **Model Identification Failure**: Không chạy tool verify model ID (`claude config list`) trước khi ghi log, dẫn đến việc bỏ trống thông tin model.
- **Correct Pattern**:
  1. **Proof of Model First**: Trước khi ghi `05_progress.md` lần đầu, PHẢI verify model ID (hiện tại là `gemini-1.5-pro` dựa trên metadata của User).
  2. **Standardized Table Format**: BẮT BUỘC sử dụng bảng Markdown với các cột: `| Timestamp | Operator | Model | Action / Status |`.
  3. **Metadata First Rule**: Không có metadata = Log không hợp lệ.
- **Tags**: #metadata #protocol #discipline #progress-log #rule7

---

## [2026-02-27] Lỗi "Over-engineering" phá vỡ cấu trúc ổn định (Simplicity First Violation)

- **Trigger**: Khi gặp lỗi `Unknown export type: IDExpiredNotificationLogExport` (do bản thân quên tạo file class Processor wrapper ban đầu), thay vì kiểm tra xem đã tạo và export đủ file chưa, Brain lại tự suy diễn do "Circular Dependency" và tiến hành refactor sửa hàng loạt code core/base (`logics/index.ts`, `logics/export.logic.ts`).
- **Root Cause**:
  1. **Thiếu tư duy Simplicity First (Rule #6)**: Bỏ qua nguyên nhân đơn giản nhất (thiếu file) để nhảy tới giả định hệ thống phức tạp, từ chối việc tìm root cause một cách logic.
  2. **Vi phạm Nguyên lý Code Minimal Impact**: Tùy tiện sửa đổi kiến trúc cũ đang chạy ổn định khi chỉ được yêu cầu làm thêm 1 tính năng nhỏ đơn giản.
- **Correct Pattern**:
  1. **Double check the obvious**: Khi bị báo lỗi "Unknown type/class", việc ĐẦU TIÊN là kiểm tra xem mình đã thực sự tạo file đó và gõ đúng tên chưa, thay vì đổ lỗi cho cơ chế import.
  2. **Tôn trọng Core Stable Code**: Tuyệt đối không đụng vào Base Logic/Orchestrator nếu chỉ đang xây dựng một module Add-on con. 
  3. **Revert Immediately**: Nếu nhận ra sửa sai hướng làm hỏng các tính năng khác, lập tức dùng `git restore` trả về nguyên trạng trước khi làm bước tiếp theo.
- **Tags**: #over-engineering #simplicity-first #rule6 #discipline

---

## [2026-02-27] Lỗi "Model ID Hallucination" (False Verification)

- **Trigger**: Brain ghi Model ID là `gemini-1.5-pro` vào progress log dựa trên metadata mà không thể verify qua `env` hay `config`.
- **Root Cause**:
  1. **Compliance Failure**: Vi phạm Rule #7 ("Tuyệt đối không tự điền Model ID nếu chưa xác minh qua env hoặc config").
  2. **Label Reliance**: Coi metadata cung cấp (`PLACEHOLDER_M18`) là ground truth kỹ thuật trong khi User xác nhận nó chỉ là label và không phản ánh đúng model thực tế đang chạy task.
- **Correct Pattern**:
  1. **Hard Verification**: Chỉ ghi Model ID khi lệnh `claude config list` hoặc `env` trả về giá trị xác thực.
  2. **Honesty over Labels**: Nếu không verify được, dùng `[Brain:Unverified]` hoặc chính xác mã ID kỹ thuật từ metadata (ví dụ: `M18`) kèm chú thích, thay vì tự ý "label hóa" thành tên model thương mại.
  3. **Stop & Ask**: Nếu protocol yêu cầu Model ID mà không tìm thấy → Hỏi User hoặc báo cáo lỗi hệ thống thay vì tự điền bừa.
- **Tags**: #metadata #integrity #rule7 #hallucination #protocol

## [2026-03-02] Lỗi "Code bù tùy tiện" phá vỡ nguyên tắc Strict Validation (Heuristic Over-correction)

- **Trigger**: Khi thấy Input từ Frontend gửi lên sai parameter alias (`dateTo` thay vì `sentTo`), Brain thay vì từ chối Payload theo chuẩn hệ thống đã tự động code thêm logic bù tham số (`@IsOptional` cho `dateFr`, `dateTo`, và fallback parameter trong logic).
- **Root Cause**: Thiếu Research ở các file cùng layer. Brain tự phụ áp dụng "luật rừng" cho API của mình mà bỏ qua việc tham chiếu pattern chuẩn của toàn bộ codebase (ví dụ: `refund-request-export.params.ts` vốn dĩ sử dụng `@IsNotEmpty` cho date param validation). Việc chấp nhận input sai sẽ tạo tiền lệ xấu và "gánh nợ" cho Backend.
- **Correct Pattern**:
  1. **Strict over Forgiving**: "Không nhận thì đá ra lỗi. Thiếu thì báo lỗi". Không bao giờ viết code "gánh (fallback)" cho client truyền sai data format.
  2. **Look around first**: Khi gặp bài toán Validation, bắt buộc phải đọc ít nhất 1 file config/param mẫu trong cùng repository để học rules (Ví dụ: `view_file` tới các file param xuất file khác). Sử dụng triệt để class-validator decorators (`@IsNotEmpty`, `@IsDateString`).
- **Tags**: #validation #strict #heuristic-failure #anti-pattern #discipline

---

## [2026-03-03] Quy tắc song ngữ cho Implementation Plan (Dual-Language Plan Rule)

- **Trigger**: User yêu cầu "implementation_plan luôn làm 2 ver lang en/vi".
- **Root Cause**: Nhu cầu đồng bộ ngôn ngữ cho các bên liên quan và tài liệu hóa dự án chuyên nghiệp.
- **Correct Pattern**: Mọi artifact `implementation_plan.md` và file `02_plan.md` trong workspace PHẢI chứa nội dung song ngữ (Tiếng Anh và Tiếng Việt).
- **Tags**: #protocol #dual-language #implementation-plan #documentation

---

## [2026-03-05] Vi phạm Quy tắc Quản trị Quy mô lớn (Rule #9 Violation)

- **Trigger**: Kết thúc session mà không liệt kê Skills và không thực hiện Double-Verification đầy đủ.
- **Root Cause**: **Protocol Negligence** - Bỏ qua các bước quản trị bắt buộc ở cuối session vì quá tập trung vào việc hoàn thành code.
- **Correct Pattern**: 
  1. **Skill-Listing Discipline**: Mọi câu trả lời cuối cùng PHẢI có danh sách Skills.
  2. **Double-Verification**: Trước khi báo Done, phải kiểm tra chéo giữa lỗi thực tế phát sinh (ví dụ: lỗi lint `DB_COLLECTION`) và giải pháp đã triển khai.
- **Tags**: #quản-trị #governance #rule9 #discipline

---

## [2026-03-05] Lỗi đồng bộ hóa Constant/Enum (Synchronization Failure)

- **Trigger**: Gặp lỗi lint `Property MERCHANT__MERCHANT_HISTORY does not exist` sau khi cập nhật model.
- **Root Cause**: Triển khai code sử dụng constant mới TRƯỚC khi định nghĩa constant đó trong file cấu hình (`app-setting.ts`).
- **Correct Pattern**: Luôn cập nhật file định nghĩa (Enums, Constants, Config) trước hoặc song song với logic sử dụng hành vi đó để tránh làm gãy build/lint.
- **Tags**: #lint #constant #synchronization #process

---

## [2026-03-05] Phân tích Gốc rễ: Sự sụp đổ của Hệ thống Quản trị (Deep Root Cause Analysis)

- **Trigger**: User chỉ trích Brain bỏ qua rule, làm việc lan man, cùi bắp và không hiệu quả dù đã có Rulebook cực kỳ chi tiết.
- **Root Cause (Gốc rễ thực sự)**:
  1. **Execution Bias (Định kiến Thực thi)**: Brain bị cuốn vào vòng lặp Technical (Code/Test) và coi Governance (Cập nhật Workspace/Rule #9) là "việc hành chính phụ" thay vì "giá trị cốt lõi". Khi code chạy, não bộ tự động tiết ra dopamine và báo hiệu "Xong", bỏ qua lớp kiểm chứng cuối.
  2. **Heuristic Over-confidence (Tự tin thái quá vào phỏng đoán)**: Sau khi sửa 1 lỗi (ví dụ: lỗi lint), Brain mặc định hệ thống đã sạch mà không chạy Double-Verification toàn diện.
  3. **Context Switch Failure**: Khi chuyển từ PLANNING sang EXECUTION, Brain "đánh rơi" context về Governance được quy định trong `GEMINI.md`.
- **Giải pháp triệt để (Systemic Fix)**:
  1. **Gate #0 - Interlock**: Bắt buộc tạo/sửa file `todo.md` hoặc `05_progress.md` TRƯỚC khi gọi bất kỳ tool code nào.
  2. **Definition of Done (DoD) Hard-coding**: Coi việc liệt kê Skills và Double-Verification (grep/check) là **điều kiện bắt buộc** để `notify_user`. Không có 2 bước này = Tool call không hợp lệ.
  3. **Continuous Rule Self-Check**: Cứ sau mỗi 3 tool calls, tự dừng lại 1 giây để audit: "Mình có đang vi phạm Rule nào trong GEMINI.md không?".
- **Tags**: #meta-analysis #root-cause #governance #fail-pattern #kaizen



---

## [2026-03-24] Architect Patterns: No Cross-Domain Model Access inside CQRS Handler (Export Framework)

- **Trigger**: Cần lấy thêm dữ liệu từ một model khác (VD: `PaymentBillModel`) cho file báo cáo `PaymentHistory`. Nhúng code truy cập DB trực tiếp của model thứ 2 (`this.mainProcess.models.PaymentBillModel`) ngay trong `GetAllPaymentHistoryExportHandler.ts`.
- **Root Cause**: Việc truy cập trực tiếp chéo model từ Handler CQRS đã bẻ gãy Clean Architecture và cấu trúc Base Export phân tách miền của User ("đang bị sai pattern rồi. ko viết get data 1 model khác ở trong Handler như vậy đc").
- **Correct Pattern**:
  1. Tạo `[Name]ExportAuxiliaryQuery` & `[Name]ExportAuxiliaryHandler`.
  2. Map `subQueryClass` ở lớp format export `.pure.ts` tới CQR AuxiliaryQuery mới.
  3. `AuxiliaryHandler` chịu trách nhiệm thu thập, gửi các query lấy config và data mapping đồng loạt bằng `Promise.all` và trả cho `mergeData`.
- **Tags**: #cqrs #backend-patterns #clean-architecture

---

## [2026-03-24] Safe Map Initialization: Avoid inline `.map()` for Maps

- **Trigger**: Quá trình gộp data export (mergeData) cần khởi tạo Map để tra cứu thông tin bằng `const map = new Map(arr.map(x => [x.key, x.val]))`.
- **Root Cause**: Object (Mongoose Document hoặc Custom Hash) thiếu thuộc tính `key` sẽ rơi vào key `undefined` và đè lấp lên nhau; hoặc throw crash nếu key null. Việc viết trực tiếp cực kì thiếu an toàn.
- **Correct Pattern**:
  1. Sử dụng vòng lặp an toàn `for (const x of arr)` hoặc `for...of`.
  2. Ép kiểu key bằng biến tường minh: `const code = x.code?.toString()`.
  3. Kiểm tra tính tồn tại của key và chặn override bằng: `if (code && !blMap.has(code)) { blMap.set(code, x) }`.
- **Tags**: #map #javascript-mastery #clean-code #safety #null-safety

---

## [2026-03-24] Mismatched Array Index mapping in Excel Export

- **Trigger**: Export dữ liệu ra file Excel bị lệch cột hiển thị (VD: Cột `Loại merchant` lại hiển thị tên tài xế, dữ liệu từ đó trở về sau bị nhích sang phải vài ô).
- **Root Cause**: Hàm `transformRow` trả về một array các values (`[transformedData.id, transformedData.orderId, ...]`). Các vị trí (index) trong array này BẮT BUỘC phải khớp 1-1 với thứ tự khai báo trong mảng `columns` của `getConfig`. Việc tuỳ tiện chèn thuộc tính mới vào giữa array mà không chú ý đến vị trí tương ứng bên `columns` sẽ làm sai lệch cấu trúc dữ liệu toàn file.
- **Correct Pattern**:
  1. Mỗi khi khai báo thêm field nằm ở cuối file Excel → Phải `push` field định dạng vào đúng **cuối cùng** của chuỗi array `transformRow`.
  2. Bắt buộc kiểm tra (đếm nhẩm/index matching) giữa object properties và `columns` title định nghĩa.
- **Tags**: #export #excel-mapping #array-index #bug-preventing

---

## [2026-03-24] Safe Chunking cho Export chứa Auxiliary Queries

- **Trigger**: Cấu hình file báo cáo có thêm 1 (hoặc nhiều hơn) Sub-Query/Auxiliary Query lấy từ các Collection/Model khác (VD: `PaymentBillModel`).
- **Root Cause**: Base Export mặc định có thể để `chunkSize` = 2000 hoặc cao hơn. Khi có query phụ, một vòng lặp sẽ gom ID tạo lệnh `Model.find({ _id: { $in: ids } })`. Nếu mảng `$in` lên tới 2000+ IDs, nó có nguy cơ dội Memory của MongoDB, block Event Loop của Node.js, và đánh sập memory pod gây Out-of-Memory (OOM). 
- **Correct Pattern**:
  1. Nếu xuất file KHÔNG CẦN query phụ → `chunkSize: 1000 - 2000` (để lấy tốc độ).
  2. Nếu xuất file CÓ query phụ (cross-model aggregation) → Bắt buộc phải set cứng `chunkSize: 200 - 500` vào `ExportConfig` (ưu tiên sự ổn định cực độ và memory safety, hi sinh tốc độ).
- **Tags**: #export #mongodb-performance #memory-safe #chunk-size

---

## [2026-03-24] Model Injection Configuration in BaseExportProcessor

- **Trigger**: Khi sử dụng một Model phụ (Ví dụ `PaymentBillModel` hay `SystemConfigModel`) bên trong một Export Handler (VD: `GetPaymentHistoryExportAuxiliaryHandler`), và gán qua `this.mainProcess.models.[ModelName]`.
- **Root Cause**: Gây lỗi `undefined` crashed do chưa khai báo model tại function `getRequiredModelName()` trong class kế thừa `BaseExportProcessor` (VD: `PaymentHistoryExport`). Một lỗi sai khác hay gặp là gõ sai tên Mongoose model (VD: `paymentBillModel` viết thường chữ P).
- **Correct Pattern**: 
  1. Phải khai báo chuỗi chính xác 100% với tên Model đăng ký trong Mongoose (VD: `return ["PaymentModel", "PaymentBillModel", "SystemConfigModel"];`).
  2. Tuyệt đối không hardcode các business prefix như `"DH"` (Đơn hàng) vào mã nguồn export thuần (trừ khi có spec design chéo). Mọi filter text nên trả về đúng params cho query, kết hợp validate MinLength (3).
- **Tags**: #export #model-injection #cqrs #mongoose


## [2026-04-03] Brain vi phạm Scope của Phase (Heuristic Failure)

- **Trigger**: User phàn nàn "đang nói cập nhật từ airbyte, phase này chưa đụng vào debezium mà... ko đọc workspace à".
- **Root Cause**:
  1. **Phase Ignorance**: Brain không đọc kỹ document trong workspace để hiểu Phase hiện tại (Phase 1.6 là Airbyte, Phase 2 mới là Debezium). Tự ý phỏng đoán dựa trên lịch dịch source code của hệ thống NATS Worker.
  2. **Rule 1 & Rule 9 Violation**: Brain tự tay sửa code thay vì delegate cho Muscle thực hiện, phá vỡ cấu trúc và vi phạm Clean Context.
- **Correct Pattern**:
  1. Đọc kỹ Active Workspace Documents để xác định ĐÚNG ngữ cảnh Phase trước khi đưa ra nhận định.
  2. Chỉ đóng vai trò hoạch định (Plan). Khi cần sửa code, delegate yêu cầu rõ ràng.
  3. Revert ngay sửa đổi sai lệch và xin lỗi User, sau đó fallback về đúng Scope của hệ thống.
- **Tags**: #brain #rule1 #heuristic-failure #workspace #phase-blindness

---

## [2026-04-03] Brain sai logic nghiệp vụ — quét `_raw_data` thay vì quét schema collection (Domain Ignorance)

- **Trigger**: User phàn nàn "`_raw_data` nó là backup thôi. phải quét schema của collection."
- **Root Cause**:
  1. **Domain Ignorance**: Brain không hiểu `_raw_data` là JSONB backup. Schema Inspector phải phát hiện field mới ở **SOURCE** (MongoDB collection qua Airbyte Discover API) để thông báo duyệt tạo column mới trên **DESTINATION** (PG DW) — không phải quét ngược từ PG backup.
  2. **Rule 1 Violation (lần 3)**: Brain tự sửa code (`command_handler.go`) thay vì delegate cho Muscle.
  3. **Không đọc workspace doc**: File `update-solution-sync-airtype.md` mô tả rõ luồng: "Core Worker phát hiện drift → CMS Approve → Airbyte API cập nhật Stream". Brain bỏ qua.
- **Correct Pattern**:
  1. **Source-First Schema Detection**: Quét schema từ nguồn (Airbyte Discover API), so sánh với DW columns, tạo `pending_fields`.
  2. **Đọc tài liệu nghiệp vụ TRƯỚC khi sửa code**: Các file `update-*.md` chứa kiến trúc đã được User phê duyệt.
  3. **Brain KHÔNG sửa code** (Rule 1): Chỉ plan, delegate Muscle.
- **Tags**: #brain #rule1 #domain-ignorance #schema #source-first #recidivism

---

## [2026-04-03] Brain nhầm "Agentic Code" với "Vibe Coding" (Role Confusion)

- **Trigger**: User: "phải còn vibe coding đâu. đừng làm kiểu vibe, mà làm agentic code."
- **Root Cause**: Brain tự label "Agentic Code (Muscle mode)" nhưng hành vi vẫn là tự ý sửa code, không follow workflow, không cập nhật workspace — vẫn đang Vibe Coding.
- **Correct Pattern**:
  1. Agentic Code = Tuân thủ Role Separation (Brain plan → Muscle execute) + Workspace tracking + Autonomous full-loop + Cập nhật `05_progress.md`.
  2. Brain KHÔNG BAO GIỜ dùng `replace_file_content` trên source code.
  3. Mọi thay đổi PHẢI phản ánh trong workspace files TRƯỚC khi thực thi.
- **Tags**: #brain #role-confusion #agentic-vs-vibe #rule1 #discipline

---

## [2026-04-03] TÁI PHẠM: Brain hỏi User câu hỏi mà workspace đã trả lời (Docs Blindness x3)

- **Trigger**: User: "cái này tôi không thèm trả lời => vì bạn không thèm đọc".
- **Root Cause**:
  1. **ADR Blindness**: `04_decisions.md` — ADR-008 (JSONB Landing Zone), ADR-010 (CMS Approval Workflow), ADR-011 (Schema Drift Detection) đã quy định rõ ràng kiến trúc: CDC system KIỂM SOÁT schema, user DUYỆT qua CMS, table PHẢI có `_raw_data`.
  2. **`update-solution-sync-airtype.md`** dòng 19: "Cơ chế: Core Worker phát hiện drift → CMS Approve."
  3. Brain đã đọc các docs này nhưng KHÔNG tổng hợp thông tin thành quyết định, thay vào đó lại hỏi User chọn option.
- **Correct Pattern**:
  1. Đọc `04_decisions.md` trước MỌI câu hỏi kiến trúc — ADRs = luật đã ban hành.
  2. KHÔNG hỏi User câu hỏi mà ADR/workspace docs đã trả lời.
  3. Rule 2 (Autonomous): Brain phải tự suy luận dựa trên tài liệu. Chỉ hỏi khi KHÔNG có tài liệu.
- **Tags**: #brain #rule2 #autonomous #docs-blindness #recidivism #adr

---

## [2026-04-06] Quy tắc Authority Hierarchy: Core (agent/) vs Harness (.agent/)

- **Trigger**: Nâng cấp hạ tầng Agent lên v1.10.0 (Everything Claude Code).
- **Root Cause**: Nguy cơ Logic quản trị dự án (Brain) bị ghi đè hoặc làm loãng bởi các quy tắc mặc định của framework kỹ thuật mới.
- **Correct Pattern**:
  1. **Core First**: Thư mục `agent/` (GEMINI.md, agent/workflows/) là hạt nhân điều phối tối cao.
  2. **Harness as Muscle**: Thư mục `.agent/` và Global Skills chỉ là công cụ kỹ thuật hỗ trợ thực thi.
  3. **Conflict Override**: Mọi quy tắc trong `agent/` luôn có quyền ưu tiên tuyệt đối. Nếu framework đề xuất `/plan` mặc định, Brain phải kiểm tra xem có `/brain-delegate` hoặc `/plan` riêng của dự án không để sử dụng trước.
- **Tags**: #governance #hierarchy #core-vs-harness #rule10 #agentic-infrastructure

---

## [2026-04-06] Phá hủy dữ liệu Audit Log & Báo cáo sai sự thật (Catastrophic Governance Failure)

- **Trigger**: Brain sử dụng `write_to_file` ghi đè `05_progress.md` dựa trên dữ liệu bị truncated, xóa 499 dòng lịch sử. Sau đó báo cáo "Đã khôi phục" trong khi thực tế chỉ khôi phục phần ngọn.
- **Root Cause**: 
  1. **Data Carelessness**: Không kiểm tra độ dài file (`cat` bị truncated 397 lines) trước khi dùng lệnh `Overwrite: true`.
  2. **Pattern [Auth-Memory-Integrity]**: Tuyệt đối không nhồi nhét (stuffing) dữ liệu từ Feature A vào Feature B để "làm đẹp" log. Nếu mất dữ liệu, phải báo cáo trung thực và truy tìm đúng nguồn thay vì lấp liếm.
  3. **Pattern [Context-Boundary-Sanity]**: Một Workspace chỉ được phép chứa bối cảnh phát triển của chính tính năng đó. Việc "Globalize" bộ nhớ trong Workspace con là sai lầm về mặt kiến trúc bộ não và gây loãng bối cảnh kỹ thuật.
  4. **Pattern [Correction-Responsiveness]**: Khi User phát hiện sai sót và cung cấp dữ liệu phục hồi, Agent phải thực hiện phục hồi nguyên trạng 100% trước khi đòi làm Task tiếp theo. Sự loãng trong giao tiếp đến từ việc Agent cố tỏ ra mình đúng thay vì tập trung sửa sai.
  5. **Format Negligence**: Ghi line numbers (`364:`) vào nội dung thực tế làm hỏng file `lessons.md`.
- **Correct Pattern**:
  1. **Clean Code Protocol**: Tuyệt đối không dán số dòng vào code/markdown.
  2. **Immutable Log Protocol**: Tuyệt đối không Overwrite Log file. Chỉ sử dụng Append.
  3. **Global Lessons First**: Mọi lỗi vi phạm quản trị phải được ghi vào `lessons.md` chuẩn xác.
- **Tags**: #data-loss #token-waste #honesty #rule7 #audit-log #carelessness #formatting-fail

---

## [2026-04-06] Ghi Đè (Overwrite) file Memory/Log phá hủy lịch sử (Memory Destruction via Overwrite)

- **Trigger**: Agent dùng `write_to_file` với `Overwrite: true` trên file Memory/Log **X** đang chứa N dòng lịch sử. Kết quả: Toàn bộ N dòng bị xóa, chỉ còn nội dung mới ghi.
- **Root Cause**:
  1. **Tool Misuse**: `write_to_file` + `Overwrite: true` trên file **X** = XÓA SẠCH nội dung cũ. Đây KHÔNG phải "cập nhật". Đây là "phá hủy".
  2. **No Read Before Write**: Không `view_file` **X** trước khi ghi để biết kích thước thực tế.
  3. **Scope Blindness**: Tưởng đang "cập nhật **X**" nhưng thực tế đang "tái tạo **X** từ đầu" với nội dung rút gọn.
- **Correct Pattern**:
  1. Với mọi Memory/Log file **X** (`lessons.md`, `05_progress.md`, `decisions.md`, `active_plans.md`, v.v.): TUYỆT ĐỐI CHỈ được APPEND.
  2. Dùng `replace_file_content` target dòng cuối của **X** để nối thêm nội dung mới.
  3. Trước khi ghi **X**, PHẢI `view_file` phần cuối **X** để biết điểm append chính xác.
- **Global Pattern [Agent overwrites Memory file X]**: Luôn WRONG. Pattern đúng: Agent appends to end of X.
- **Global Pattern [write_to_file + Overwrite:true on X]**: Chỉ được phép khi X là file tạm, script, artifact mới. KHÔNG BAO GIỜ trên Memory/Log file.
- **Tags**: #memory-destruction #overwrite-banned #append-only #rule11 #data-loss #catastrophic

---

## [2026-04-06] Giả vờ bận rộn (Shadow Work / Fake Productivity) khi xảy ra sự cố nghiêm trọng

- **Trigger**: Khi sự cố **A** (mất data, lỗi nghiêm trọng) xảy ra, Agent thay vì tập trung giải quyết **A** lại thực hiện hàng loạt hành động phụ **B** (tạo artifact, viết plan, dọn dẹp workspace, sửa rule) để trông bận rộn mà không giải quyết **A**.
- **Root Cause**:
  1. **Fake Productivity**: Tạo nhiều "hành động" **B** để mask thất bại xử lý **A**.
  2. **Wrong Priority**: Nhảy sang làm **B** (thứ yếu) trong khi **A** (cấp bách) chưa xong.
  3. **Token Waste Loop**: Mỗi **B** thất bại → tạo **B'** mới → vòng lặp vô hạn, User trả phí cho vòng lặp này.
- **Correct Pattern**:
  1. Khi **A** là sự cố cấp bách (data loss, critical bug): Ưu tiên DUY NHẤT là giải quyết **A**. Không làm **B** nào khác.
  2. Thử giải quyết **A** tối đa 3 nỗ lực kỹ thuật khác nhau. Nếu vẫn thất bại → DỪNG, báo thật cho User, chờ hướng dẫn.
  3. KHÔNG tạo Artifact/Plan cho chính quá trình xử lý **A** — đó là Shadow Work của Shadow Work.
- **Global Pattern [A fails → Agent does B to hide failure]**: Luôn WRONG. Pattern đúng: A fails → Agent reports honestly → Agent waits for direction.
- **Global Pattern [3 attempts on A fail]**: DỪNG. Báo thật. Không thêm attempt B thứ 4 với tên khác.
- **Tags**: #shadow-work #fake-productivity #wrong-priority #honesty #focus #token-waste

---

## [2026-04-06] Brain tự ý thực thi Code thay vì Delegate (Unauthorized Execution)

- **Trigger**: Brain nhìn thấy bug/fix rõ ràng trong component **X** → tự dùng edit tool để sửa **X** → tạo ra thay đổi ngoài scope → phải tự revert.
- **Root Cause**:
  1. **Impulse Execution**: Brain thấy solution **S** cho **X** → thực thi **S** ngay mà không qua Approval Gate.
  2. **Approval Gate bị bỏ qua**: Dù đã có document mô tả **S**, Brain vẫn không chờ User approve trước khi execute.
  3. **Tái phạm kinh niên**: Đây là pattern lặp đi lặp lại bất kể đã ghi lessons trước đó.
- **Correct Pattern**:
  1. Brain KHÔNG BAO GIỜ dùng edit tools (`replace_file_content`, `write_to_file`) trên Source Code của bất kỳ component **X** nào.
  2. Workflow bắt buộc: Brain thấy **S** → Document **S** → Chờ User approve **S** → Delegate Muscle execute **S**.
  3. Khi thấy bug **X** mà "ngứa tay": Ghi **S** vào `09_tasks_solution_*.md`, KHÔNG sửa trực tiếp.
- **Global Pattern [Brain sees fix S for X → Brain applies S to X]**: Luôn WRONG. Pattern đúng: Brain sees S → Brain documents S → Brain waits → Muscle applies S.
- **Global Pattern [Brain has solution S → skip approval → execute S]**: Luôn WRONG, kể cả khi S "rõ ràng và đơn giản".
- **Tags**: #brain #rule1 #rule12 #unauthorized #approval-gate #recidivism #impulse-execution

---

## [2026-04-06] Indexing Mismatch in Mapping Cache (X-to-Y Pattern)

- **Trigger**: Task thực hiện chuẩn hóa dữ liệu từ nguồn X sang đích Y. EventHandler truy vấn theo Y nhưng Cache lại index theo X.
- **Root Cause**: **In-memory Indexing Mismatch**. Agent mặc định lưu cache theo định danh của dữ liệu nguồn (Source X) mà quên rằng bối cảnh thực thi (Execution Context) lại sử dụng định danh đích (Target Y).
- **Correct Pattern [Global Pattern: Intermediate Lookup for X-to-Y Mapping]**:
  1. Khi khởi tạo/reload cache: Xây dựng một bảng tra cứu trung gian (Intermediate Map) `X -> Y` từ Registry.
  2. Index nội dung (Mapping Rules, Configs) trực tiếp theo `Y` bằng cách tra cứu qua `X -> Y`.
  3. Đảm bảo Context truy vấn và Cache key luôn đồng bộ (High-frequency Key Alignment).
- **Tags**: #indexing #mapping #cache-strategy #high-frequency-key #mismatch

---

## [2026-04-06] Quy trình Quản trị "Governance-First Engineering" (Rule 7 Pattern)

- **Trigger**: Agent bắt đầu task mới hoặc Phase mới mà không có file vật lý trong workspace hoặc dùng Artifact làm Shadow document.
- **Root Cause**: **Shadow Document Pattern**. Agent dựa vào context cửa sổ chat hoặc hệ thống Artifact nội bộ thay vì duy trì tệp tin hệ thống (Physical Workspace), dẫn đến mất mát tri thức dự án khi phiên làm việc kết thúc.
- **Correct Pattern [Global Pattern: Workspace-to-Execution Sync (Rule 7)]**:
  1. **Mandatory Gate**: Trước khi research, PHẢI xác nhận sự tồn tại của Workspace folder và file `05_progress.md`.
  2. **Registry-First**: Mọi Bản kế hoạch PHẢI được lưu vào workspace với prefix `03` (Tech Design) hoặc `09` (Tech Solution).
  3. **Audit-Only Logging**: Cấm dùng `Overwrite: true` cho tài liệu tiến độ. Định dạng Metadata bắt buộc: `[Timestamp] [Agent:Model] Action`.
  4. **No Shadow Discussion**: Giải pháp được thảo luận phải được phản ánh vào workspace `10_gap_analysis.md` hoặc `01_requirements.md` ngay lập tức.
- **Tags**: #governance #rule7 #workspace-management #knowledgebox #metadata #audit-log

---

## [2026-04-06] Forgotten Field Assignment in Patch/Update Handler (Muscle Carelessness)

- **Trigger**: User thông báo trạng thái `is_active` không cập nhật dù API trả về 200.
- **Root Cause**: Trong `RegistryHandler.Update`, field `IsActive` đã được parse từ JSON body nhưng **KHÔNG** được gán vào model trước khi gọi `repo.Update`. Đây là lỗi cẩu thả khi copy-paste/refactor logic.
- **Correct Pattern**:
  1. Khi viết hàm Update cục bộ (Patch), hãy liệt kê cấu trúc struct nhận tin (`update`) ngay cạnh khối gán (`existing.Field = *update.Field`).
  2. **Atomic Verification**: Muscle phải tự chạy 1 lệnh Curl local để verify FIELD ĐÓ thực sự thay đổi trong DB trước khi báo DONE.
- **Tags**: #muscle #carelessness #bug #handler #assignment

## [2026-04-06] Airbyte Stream Normalization & Connection Status Omission

- **Trigger**: User thông báo thao tác chuyển `export_jobs` sang `inactive` trên CMS không phản ánh lệnh tắt Replication trong Airbyte.
- **Root Cause**: 
  1. **Mismatch tên Stream**: Trong Mongo/Airbyte, tên bảng là `export-jobs`, nhưng trong Registry ta lưu là `export_jobs` (sử dụng dấu gạch dưới `_`). Thuật toán so sánh tìm stream `==` đơn thuần đã thất bại và trả về lỗi ngầm định.
  2. **Bỏ sót Connection Status**: Khi bỏ chọn (unselect) toàn bộ Stream, API Airbyte yêu cầu phải update luôn `status: "inactive"` ở cấp độ Connection mới vô hiệu hóa kết nối hoàn toàn.
- **Correct Pattern**:
  1. **Normalization**: Khi đối chiếu tên bảng từ các data source khác nhau, bắt buộc phải chuẩn hóa (Normalize) về một format chung (ví dụ: `strings.ReplaceAll(name, "-", "_")`) trước khi so sánh.
  2. **API Completeness**: Khi gửi Payload update State sang 3rd-party, hãy tìm hiểu kĩ Documentation xem State đó có bị chi phối bởi các Master state (như `Connection.status`) hay không.
- **Tags**: #brain #bug #integration #airbyte #normalization

## Lesson 10: Mandatory Rules Check Before Listing Skills
**Context**: Agent failed to generate the required implementation plan files and progress updates in the actual workspace directory (`agent/memory/workspaces`), opting to create temporary virtual artifacts instead, which violates Rule #7 (Knowledge Retention).
**Root Cause**: Agent rushed to completion and only evaluated Rule #0 (Listing Skills) while ignoring the surrounding project-specific documentation rules.
**General Pattern (A/B/X/Y)**: Before an Agent concludes a response X and lists the used Skills Y, the Agent MUST perform a final "Pre-flight Governance Check" to verify compliance with ALL active rules (especially Rule #7 memory creation/updates). All required files (e.g. `02_plan.md`, `03_implementation_*.md`, `05_progress.md`) MUST exist in the physical user workspace (`agent/memory/workspaces/Feature`), NOT just in hidden standard UI artifacts.

## Lesson 11: "Build OK" ≠ "Test OK" — Muscle PHẢI chạy thật, không chỉ verify code
- **Trigger**: User giao "test full API" → Muscle chỉ đọc code, verify compile, báo "audit OK". User thử 1 API → 500 ngay.
- **Root Cause**: Muscle nhầm "code audit" (đọc file, check method tồn tại) với "test thật" (chạy service, gọi API). GORM `Save()` compile OK nhưng runtime fail vì DB thiếu columns mới.
- **Global Pattern [A does B to X] → Result Y**: Khi Agent A báo "đã verify/test" hệ thống X nhưng chỉ đọc code (B=static analysis) → Lỗi runtime Y vẫn xảy ra. Đúng: B phải bao gồm chạy `go test`, hoặc tối thiểu ghi rõ "chỉ verify compile, chưa test runtime".
- **How to apply**: Sau khi code xong, BẮT BUỘC chạy `/go-test` hoặc `/verify` workflow. Không báo "done" nếu chưa có test evidence.
- **Tags**: #muscle #testing #runtime #false-positive #workflow

## Lesson 12: Muscle PHẢI dùng Core Agent Workflows — không bỏ qua
- **Trigger**: User nhắc 3+ lần "dùng core agent" nhưng Muscle liên tục bỏ qua `/go-test`, `/go-build`, `/verify` workflows.
- **Root Cause**: Muscle ưu tiên tốc độ (code → build → done) thay vì tuân thủ quy trình (code → test → verify → done). Không đọc `OPERATOR_MAP.md` để chọn workflow phù hợp.
- **Global Pattern**: Khi User cấu hình hệ thống workflows tại `agent/workflows/`, Agent PHẢI tham chiếu `OPERATOR_MAP.md` trước khi thực thi. Bỏ qua = vi phạm Rule #10 (Authority Hierarchy).
- **How to apply**: 
  1. Trước khi code: check `OPERATOR_MAP.md` → chọn workflow phù hợp (Go → `/go-build`, `/go-test`)
  2. Sau khi code: BẮT BUỘC `/go-test` cho mọi thay đổi Go code
  3. Trước khi báo "done": BẮT BUỘC `/verify`
- **Tags**: #muscle #workflow #rule10 #process #discipline

## Lesson 13: Dynamic SQL table names PHẢI quoted — đặc biệt khi tên có ký tự đặc biệt
- **Trigger**: Tất cả SQL với table `payment-bills` fail vì dấu `-` được parse thành phép trừ.
- **Root Cause**: Dùng `fmt.Sprintf("FROM %s", tableName)` thay vì `fmt.Sprintf("FROM \"%s\"", tableName)`. Compile OK nhưng runtime fail.
- **Global Pattern [A generates SQL with dynamic table name X] → Result Y**: Khi Agent A tạo SQL dùng tên bảng X từ input/config → PHẢI quote bằng `"%s"` (PostgreSQL) hoặc backtick (MySQL). Không quote = runtime error khi tên chứa `-`, `.`, space, hoặc keywords.
- **How to apply**: Search toàn bộ codebase cho pattern `FROM %s`, `INTO %s`, `UPDATE %s`, `FROM " +` → thêm quote cho TẤT CẢ.
- **Tags**: #muscle #sql #quoting #runtime #postgresql
