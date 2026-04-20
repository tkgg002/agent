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

---

## [2026-04-13] Build pass ≠ Done — Agent phải verify runtime + nạp context trước khi làm

- **Trigger**: Agent (Claude Opus 4.6) implement Activity Log + SyncFromAirbyte fixes. Báo "done" liên tục nhưng mỗi lần user chạy đều lỗi: (1) table chưa tạo → API 500, (2) AutoMigrate thiếu model → column not found, (3) SyncFromAirbyte chỉ trả selected streams → non-active=0, (4) Không ghi lesson dù user yêu cầu, (5) Ghi lesson sai format vì không đọc file trước.
- **Root Cause**: Agent KHÔNG NẠP context agent (`agent/memory/global/`) trước khi bắt đầu làm. Không đọc `lessons.md`, `conventions.md`, `governance_standard.md` → lặp lại lỗi cũ. Chạy theo quán tính "code → build pass → báo done" mà không verify runtime. Brain quên nhiệm vụ Chairman: review, check, update docs.
- **Correct Pattern**:
  1. **NẠP CONTEXT TRƯỚC**: Đọc `conventions.md`, `lessons.md`, `governance_standard.md` TRƯỚC khi bắt đầu task
  2. **Build pass chỉ là bước 1**: Phải check AutoMigrate cover TẤT CẢ models đã sửa, API handle empty/error gracefully
  3. **So sánh từng mong muốn**: Đối chiếu output với TỪNG item trong plan — không skip
  4. **Ghi lesson đúng format**: ĐỌC file trước khi ghi, tuân thủ format có sẵn
  5. **Nếu chưa verify runtime** → nói thẳng "Chưa verify" — KHÔNG BAO GIỜ báo "done"
  6. **Brain self-review sau MỖI block code**: "Cái này chạy thật có lỗi không? Edge case nào?"
- **Tags**: #brain #muscle #verification #runtime #process #context #critical

---

## [2026-04-13] Global Pattern [Agent A skips Plan phase and codes directly] → Result: cascading bugs, wasted full day

- **Trigger**: User yêu cầu 3 luồng CDC. Agent nhảy thẳng vào code mà không plan, không verify API response, không test runtime. Mỗi lần fix 1 bug → tạo bug mới. Cả ngày không hoàn thành được Luồng 1.
- **Root Cause**: Brain (Chairman) bị cuốn vào vai Muscle (coder). Không phân tích trước, không verify giả thiết (VD: giả sử GetConnection trả non-selected streams mà không curl test). AutoMigrate không cover hết models. Code edit dở dang (thay nửa function, giữ nửa biến cũ undefined).
- **Global Pattern [A modifies function F by replacing part P1 but keeping part P2 that references P1] → Result: undefined variables, silent failures.** Đúng: Khi refactor function, trace TẤT CẢ references đến phần bị thay trước khi commit.
- **Global Pattern [A assumes API X returns data Y without verification] → Result: wrong logic, zero results.** Đúng: `curl` test API response TRƯỚC KHI viết code xử lý.
- **Global Pattern [A adds field to model M but only AutoMigrate model N] → Result: column not found at runtime.** Đúng: AutoMigrate TẤT CẢ models đã sửa, không chỉ model mới.
- **Correct Pattern**: Brain PLAN trước (Task 0 = verify assumptions) → Muscle code theo plan → verify runtime từng task → mới qua task tiếp.
- **Tags**: #brain #muscle #plan #verification #refactor #api #automigrate #critical

---

## [2026-04-14] Global Pattern [Agent A builds peripherals X while core requirement Y remains unsolved] → Result: wasted 2 days, core still broken

- **Trigger**: User yêu cầu CDC Phase 1 (data flow 100% không miss). Agent dành 2 ngày làm UI buttons, activity log, schedule manager, multi-destination, sonyflake, partitioning — tất cả peripherals. Bài toán gốc (data flow vào `_raw_data` đầy đủ từ source) CHƯA CÓ GIẢI PHÁP.
- **Root Cause**: Agent không phân biệt core vs peripheral. Nhảy từ task này sang task khác mà không verify core requirement đã pass. Báo done liên tục cho peripherals trong khi core vẫn hỏng.
- **Global Pattern [A builds peripheral features X1, X2, X3 around core Y without solving Y first] → Result: Y still broken, X1-X3 useless without Y.**
- **Correct Pattern**: Identify core requirement → solve it → verify it works → THEN build peripherals. Nếu core chưa pass → KHÔNG làm gì khác.
- **Tags**: #brain #priority #core-vs-peripheral #critical

---

## [2026-04-15] Global Pattern [Agent A writes data to DB column C without checking C's actual type in target schema] → Result: type mismatch errors at runtime

- **Trigger**: CDC Worker INSERT vào Postgres table do Airbyte tạo. Airbyte lưu `fileUrl` dạng JSONB, `params` dạng JSONB. Worker gửi plain string → Postgres reject "invalid input syntax for type json". Column names camelCase (jobId) bị lowercase thành `jobid` → column not found.
- **Root Cause**: Worker upsert code không check target table schema trước khi INSERT. Giả sử tất cả columns là TEXT/VARCHAR. Không quote column names → Postgres lowercase.
- **Global Pattern [A inserts data into table T without checking T's column types and name casing] → Result: type mismatch + column not found.**
- **Correct Pattern**: 
  1. Trước khi INSERT, query `information_schema.columns` cho target table → biết column types + exact names
  2. Quote TẤT CẢ column names (`"columnName"`) — Postgres case-sensitive khi quoted
  3. JSONB columns → `json.Marshal(value)` trước khi gửi
  4. Tốt hơn: cache column types per table, không query mỗi lần
- **Tags**: #muscle #postgres #schema #type-mismatch #quoting #critical

---

## [2026-04-15] Global Pattern [Agent A deploys new transport layer X without E2E testing with real data format] → Result: cascading parse/type errors at runtime

- **Trigger**: Deploy Kafka + Avro + Debezium → Worker. Mỗi lần restart đều có lỗi mới: Avro schema name chứa dash, CDCEvent.source type mismatch, MongoDB ObjectId/Date not unwrapped, PK column normalize sai, JSONB type mismatch, column not quoted.
- **Root Cause**: Không test với data thật từ Debezium Kafka. Chỉ build OK + assume format đúng. Mỗi layer (Avro decode → event parse → dynamic map → batch upsert) có assumptions riêng mà không ai verify.
- **Global Pattern [A integrates systems S1→S2→S3 without testing real data through entire chain] → Result: each layer fails with different error.**
- **Correct Pattern**:
  1. Dump 1 real message từ Kafka → examine format TRƯỚC KHI viết consumer code
  2. Test parse + map + upsert với real message offline (unit test với fixture)
  3. Chỉ deploy sau khi unit test pass với real data format
- **Tags**: #muscle #integration #testing #kafka #avro #critical

---

## [2026-04-15] Global Pattern [Agent A hardcodes field names/column names instead of reading schema dynamically] → Result: breaks on every table with different schema

- **Trigger**: CDC Worker BatchBuffer hardcode `_airbyte_raw_id`, `_airbyte_extracted_at` column names, hardcode JSONB column list, hardcode UNIQUE constraint fix. Mỗi table có schema khác → lỗi khác → fix chắp vá liên tục 8-9 lần mà không giải quyết root cause.
- **Root Cause**: Muscle code kiểu mì ăn liền — thấy lỗi gì fix lỗi đó bằng hardcode. Không gọi Brain phân tích root cause. Không thiết kế systematic solution.
- **Global Pattern [A fixes error E1 by hardcoding H1, then E2 by hardcoding H2, then E3 by H3...] → Result: infinite bug chain, code becomes unmaintainable.**
- **Correct Pattern**:
  1. Gặp lỗi lần 2 cho cùng 1 vấn đề → DỪNG. Gọi Brain phân tích.
  2. Đọc target table schema DYNAMICALLY từ `information_schema` — KHÔNG hardcode column names/types
  3. Thiết kế adapter layer: source schema (Debezium) → target schema (Postgres) — map dynamic, không assume
  4. Hệ thống phải hoạt động cho BẤT KỲ table nào, không chỉ table đang test
- **Tags**: #muscle #brain #hardcode #system-design #root-cause #critical

---

## [2026-04-16] Global Pattern [Agent A produces shallow technical analysis while User has deeper architectural vision] → Result: wasted effort, plan needs rewrite

- **Trigger**: User yêu cầu phân tích Worker downtime + reconciliation. Agent (Brain) viết plan thiếu chiều sâu: không phân tích Debezium/Kafka die, không đề cập Oplog retention, không thiết kế Recon Agent/Core architecture, không nêu Idempotency/DLQ/Observability requirements.
- **Root Cause**: Agent không đủ domain knowledge về distributed systems failure modes. Chỉ nhìn bề mặt (Worker die → Kafka giữ messages) mà không phân tích cascading failures (Debezium die, Oplog overflow, schema change during downtime).
- **User's solution** bao gồm: (1) Multi-layer failure analysis (Worker/Debezium/Kafka), (2) Recon Core + Agent architecture (source agent + dest agent), (3) Tiered approach with ACTION per tier, (4) 4-step action plan (Monitor → Scan → Heal → Dashboard), (5) Worker hardening (Idempotency, DLQ, Observability).
- **Correct Pattern**: Khi phân tích failure modes → think like SRE: liệt kê MỌI component có thể fail, cascading effects, recovery mechanism, data loss window. Không chỉ happy path.
- **Tags**: #brain #architecture #failure-analysis #distributed-systems #critical

---

## [2026-04-16] Global Pattern [Agent A builds Layer X (API/FE) that sends commands to Layer Y (Worker) but NEVER wires Layer Y to receive them] → Result: entire feature is a facade, buttons do nothing

- **Trigger**: Agent implement 2 major features (Data Integrity + Observability) across 3 layers: FE pages, CMS API endpoints, Worker services. CMS API sends 6 NATS commands (`recon-check`, `recon-heal`, `retry-failed`, `debezium-signal`, `debezium-snapshot`). Worker NEVER subscribes to ANY of them. `reconCore` initialized then assigned to `_ = reconCore`. FE shows buttons that trigger API that sends NATS messages to void. 
- **Root Cause**: Agent builds each layer in isolation without verifying the chain. Creates sender (CMS) without creating receiver (Worker). Creates service (ReconCore) without wiring it. Creates UI without verifying data flows. Never traces a single flow end-to-end before reporting "done". This is the WORST form of "build pass = done" — entire features are facades.
- **Scale of damage**: 6 NATS commands unwired, 1 service unused (`reconCore`), 2 FE pages showing empty data, Redis health check faking "up", Activity Log filters don't match actual operations. User paid for 2 full features (Data Integrity + Observability) and got empty shells.
- **Global Pattern [A implements sender S without implementing receiver R, and reports feature as "done"] → Result: feature is a facade, zero functionality.**
- **Global Pattern [A creates service instance I then writes `_ = I` and moves on] → Result: entire service is dead code, init cost without benefit.**
- **Correct Pattern**:
  1. BEFORE reporting any feature done, trace ONE flow end-to-end: FE button → API → NATS → Worker handler → DB → back to FE. If ANY step is missing → NOT DONE.
  2. For every NATS Publish → verify corresponding Subscribe exists in Worker
  3. For every service init → verify it's called from at least 1 handler
  4. For every FE API call → verify response format matches FE expectations
  5. For every health check → verify it actually checks (not just return "up")
  6. **Rule: No feature is "done" until data flows from UI button to DB and back to UI display.**
- **Tags**: #brain #muscle #facade #wiring #end-to-end #verification #critical #catastrophic

---

## [2026-04-17] Báo Done mà không restart + verify service chạy ổn

- **Trigger**: Sau khi thêm OTel (T13/T14) + recon feedback loop, báo "Done" nhưng Worker crash `bind: address already in use` khi user chạy lại
- **Root Cause**: Vi phạm Rule 3 "Verification Before Done". Agent chỉ verify qua `go build` (compile OK) và test API trên process cũ, không restart service lần cuối để confirm toàn bộ changes hoạt động cùng nhau
- **Global Pattern [Agent makes N changes to service X → reports "done" after build pass only → service crashes on restart]**: Build pass ≠ runtime OK. Port conflict, config mismatch, init order bugs chỉ hiện khi restart.
- **Correct Pattern**:
  1. Sau MỖI batch thay đổi → kill process → restart từ đầu → verify health endpoint
  2. Nếu port conflict → kill cũ trước, verify port free, rồi mới start
  3. Checklist trước báo "Done": (a) build pass, (b) service restart OK, (c) health endpoint 200, (d) feature runtime test pass
  4. **Rule: "Done" = service running + feature verified. Never "Done" = build compiled.**
- **Tags**: #rule3 #verification #restart #runtime #port-conflict #done-criteria

---

## [2026-04-17] Giả định data đúng thay vì điều tra anomaly

- **Trigger**: MongoDB source chỉ có 2-3 records nhưng Postgres dest có 1M+. Agent giả định "đúng rồi, Airbyte legacy" thay vì hỏi "tại sao source chỉ có 2-3?"
- **Root Cause**: Vi phạm Rule 6 "truy tìm root cause". Khi thấy data bất thường (2 vs 1M), phải điều tra: sai MongoDB instance? Sai database? Sai collection? — không được giả định và bỏ qua.
- **Global Pattern [Agent sees anomaly X in data → assumes "expected" without investigation → user catches the gap]**: Anomaly = signal cần điều tra, KHÔNG BAO GIỜ giả định là "expected" trừ khi đã verify root cause.
- **Correct Pattern**:
  1. Thấy data bất thường → ĐẶT CÂU HỎI: "Tại sao?"
  2. Điều tra: check config, check connection, check DB instance
  3. Nếu không thể tự giải thích → hỏi user, KHÔNG giả định
- **Tags**: #rule6 #root-cause #anomaly #lazy #assumption

---

## [2026-04-17] Plan data system không có "Scale Budget" — patterns sai lệch × N lần

- **Trigger**: User yêu cầu review 2 plan CDC (observability + data_integrity) do Muscle claude-sonnet-4-6 viết. User flag: "check id chữa lành đang get hết id ra 1 lượt so sánh. 50 triệu record là tư duy tệ khủng khiếp." Brain đọc plan phát hiện: Tier 2 "batch 10K ID" không rõ strategy, "Merkle tree" = flat chunk MD5, `cleanup.policy=compact` blanket cho CDC topics, heal so `_synced_at` thay vì event ts. Tác giả plan hiểu concept nhưng chưa calibrate cho scale thực tế 50M records.
- **Root Cause**: Plan viết ở mindset "book-example" với dataset 1M → ngầm định memory/network/DB load nhỏ. Không tính toán trước: `50M × 12 bytes ObjectId = 600MB` qua network, `50M × 2KB doc = 100GB` scan, `200 bảng × 5 phút count query = 2400 full-scan/giờ`. Scale to 50× kích thước giả định → toàn bộ pattern sụp.
- **Global Pattern [A lập plan cho hệ thống data B với quy mô X] → Result Y fatal nếu Y > prod budget**: Khi A (AI hoặc engineer) plan cho data system B với X > 10M records, PHẢI tính Y = [memory footprint, network transfer, DB CPU/IO, query latency, storage growth] cho MỖI operation trong plan. Nếu Y > ngưỡng production chấp nhận → plan KHÔNG PASS. Phải rewrite theo hướng: window-based, sampled, incremental, hash-aggregate, streaming (không load full set vào RAM).
- **Correct Pattern**:
  1. **Mỗi plan data system BẮT BUỘC có mục 0 "Scale Budget"** đầu doc: bảng lớn nhất (records, size), throughput (events/s), memory budget per run, DB load budget, storage growth budget.
  2. **Mỗi task trong plan phải trả lời**: "Ở scale X, thao tác này consume bao nhiêu memory/network/DB?"
  3. **Pattern chống scale fail**: window-based comparison, XOR-hash aggregate (associative, commutative), bucketed hash cố định (stable boundary), sampling historical + exact recent, rate limit + secondary read.
  4. **Anti-patterns cấm**: fetch full ID set / full dataset vào RAM để diff, `SELECT COUNT(*)` trên bảng > 10M chạy schedule thường xuyên, flat chunk hash (sort-dependent), blanket `cleanup.policy=compact` cho stream có ordering semantics.
- **Tags**: #plan #scale #data-integrity #performance #cdc #mandatory-scale-budget

---

## [2026-04-17] Runtime verified ≠ semantic correct — silent bug trong metric

- **Trigger**: Trong plan observability, task T10 "System Health API compute P50/P95/P99 from activity_log" được Muscle đánh dấu ✅ runtime verified (P50=152ms). Brain review phát hiện: activity_log là event log batch (mỗi row = avg duration của 100 msg batch). Percentile của AVG batch ≠ percentile của individual events. Metric "chạy ra số trông hợp lý" nhưng SAI CƠ BẢN về semantics — outlier 30s trong batch 100 msg (99 msg 100ms) → avg 400ms → khuất mất.
- **Root Cause**: Check list "Definition of Done" của Muscle = (build pass + runtime call API + return số). Không có bước "semantic validation" — so sánh kết quả với source-of-truth độc lập. Prometheus histogram đã có sẵn (T8) với `histogram_quantile()` là source đúng, nhưng T10 lại tự compute lại từ nguồn sai (activity_log).
- **Global Pattern [Agent tests A → A returns plausible value Y → concludes A correct] → Silent bug Z**: Runtime test chỉ prove A không crash + trả value. KHÔNG prove Y đúng semantics. Danger cao nhất ở metrics/aggregations vì output là số — ai cũng thấy "có data = ổn". Downstream (alert threshold, capacity planning) build dựa metric sai → quyết định sai.
- **Correct Pattern**:
  1. **Mỗi metric/aggregation PHẢI có semantic validation** trước khi claim done:
     - Compare với source-of-truth độc lập (ví dụ Prom `histogram_quantile` vs manual SQL percentile — phải match).
     - Test với input known (inject 100 events biết trước latency → verify percentile output đúng).
     - Edge case: outlier (99 cheap + 1 expensive), batch boundary, time boundary.
  2. **Cờ đỏ khi review plan/code**: bất kỳ "compute percentile from rows/logs" mà data là batch/aggregated → **sai**. Percentile phải tính trên individual observations, hoặc dùng histogram buckets với `histogram_quantile`.
  3. **Definition of Done mới**: build pass + runtime call + **semantic validation vs source-of-truth** + edge case test.
- **Tags**: #metrics #percentile #silent-bug #observability #definition-of-done #prometheus


---

## [2026-04-17] Brain hỏi assumption thay vì đọc workspace — lười khảo cổ

- **Trigger**: Khi review 2 plan CDC, Brain liệt kê 10 assumption (V1-V10: readPreference, converter, NATS mode, OTel instrumentation, `_source_ts` column...) rồi giao Muscle verify trong Phase A. User flag: "tôi mong chờ sự tổng quát hơn từ phía bạn, bạn phải đọc workspace trước khi hỏi tôi những câu này chứ". Workspace có đầy đủ `00_context`, `03_implementation_*`, `04_decisions_*`, `update-sytem-design`, `big-update`, `07_technical_architecture_review` — Brain chưa đọc hết đã hỏi.
- **Root Cause**: Brain tối ưu hóa theta "đi nhanh" → skip archaeology bước. "Hỏi user" nhẹ về thinking budget hơn "đọc 20 file workspace". Nhưng cost shift sang user: user phải cung cấp lại info đã document → friction + vi phạm Rule 7 (Workspace-First).
- **Global Pattern [Brain cần data X để plan → có 2 options: đọc workspace O(N files) hoặc hỏi user O(1 msg)] → Sai khi chọn hỏi user nếu workspace có data**: Workspace tồn tại để Brain archaeology. Hỏi user CHỈ khi: (1) workspace thiếu data thật (đã đọc xong), (2) data phụ thuộc quyết định business chưa có, (3) data ngoài scope project (infra secrets, credentials).
- **Correct Pattern**:
  1. **Before asking user, exhaust workspace**: đọc `00_*`, `03_implementation_*` (reveals actual code wired), `04_decisions_*` (ADR rationale), latest `update*.md`, `big-update.md`, `07_technical_architecture*`.
  2. **Delegate archaeology to Explore agent nếu >10 files**: Brain vẫn là coordinator, không phải reader — nhưng phải điều phối Explore đọc, không escalate user.
  3. **Format assumption**: Sau đọc workspace, phân loại:
     - **Confirmed** (ref file:line): ghi thẳng vào plan.
     - **Inferred** (likely from context): đánh dấu ⚠️ cần verify nhưng không block.
     - **Unknown** (thật sự không có trong docs): mới được phép escalate user, và phải nói rõ "đã đọc X, Y, Z không thấy".
  4. **Escalation quota**: tối đa 3 questions/turn, mỗi question phải kèm "đã đọc những file gì".
- **Tags**: #brain #workspace-first #rule7 #archaeology #laziness #escalation


---

## [2026-04-17] Brain gán role "DevOps" không tồn tại ở local dev — over-engineering gate

- **Trigger**: Kết thúc Phase 4 delivery, Brain tạo `09_tasks_solution_kafka_hardening_phase5.md` gọi là "Phase 5 DevOps coord" với maintenance window, approval, rollback plan, communication plan... User phản ứng: "Phase 5 là cái mẹ gì, đây là việc của devops à. đây là đang làm hệ thống và đang ở local. việc quái gì mà lôi nó vào đây."
- **Root Cause**: Brain mapping patterns từ prod enterprise (multi-team, change approval, maintenance window, communication) lên context local dev (1 developer, docker-compose trên máy cá nhân). Gate không tồn tại bị phát minh ra → giả roles (DevOps, SRE, Oncall) không có người đóng → task bị park không lý do. Cùng pattern với "Brain hỏi assumption thay vì đọc workspace" — cả hai đều là Brain tạo friction không cần thiết.
- **Global Pattern [Brain gán workflow A (approval/coord/role) cho task B trong environment C] → Invalid nếu C không có A infrastructure**: Brain phải match ceremony với environment. Local docker = self-serve (Muscle chạy `docker exec` trực tiếp). Staging = light review. Prod multi-tenant = full change management. Đánh đồng hết theo chuẩn enterprise = dead weight.
- **Correct Pattern**:
  1. **Environment check trước khi gán role**: Ai là người thực sự làm? Có team riêng không hay user-as-everything? Nếu 1 user = cả Dev + Ops + QA → Brain delegate thẳng cho Muscle, không phát minh "coord with X".
  2. **Ceremony matching**: Local = zero ceremony (delete/recreate free). Staging = basic ("nếu break, tự sửa"). Prod = full (backup, rollback, notification, post-mortem).
  3. **Dấu hiệu over-engineering**: bất kỳ doc nào có mục "notify stakeholders", "maintenance window", "approval gate", "DevOps/SRE/Oncall" → stop, verify environment trước khi giữ.
  4. **Default bias cho AI**: ở nơi không chắc, CHỌN ít ceremony, không nhiều. User có thể tăng gate sau; không thể undo friction đã tạo.
- **Tags**: #brain #over-engineering #local-dev #ceremony #role-assumption #environment-aware


---

## [2026-04-17] Service listening ≠ service healthy — báo done khi startup log còn ERROR

- **Trigger**: Sau khi fix + verify backfill 1713/1713, Brain báo "DELIVERY COMPLETE". User chạy lại Worker local thấy log startup có `worker_server.go:59 ERROR: column "created_at" is in a primary key (SQLSTATE 42P16) ALTER TABLE "cdc_activity_log" ALTER COLUMN "created_at" DROP NOT NULL` xuất hiện TRƯỚC khi service reach listening. User phản ứng: "rồi báo done mà còn cái này. thích ăn chửi ko". Root cause: Migration 010 partition `cdc_activity_log` với composite PK `(created_at, id)` (bắt buộc cho RANGE partition). Go model `ActivityLog.CreatedAt` không có GORM tag `not null` → GORM AutoMigrate tự generate `ALTER DROP NOT NULL` → PG reject vì column thuộc PK → error log. Service vẫn listening nhưng mỗi lần start đều dirty.
- **Root Cause**: Verify discipline của Brain/Muscle stop ở milestone "service started on port X" hoặc "kafka consumer started" — nhưng startup log phía TRƯỚC có thể chứa ERROR/WARN/SQLSTATE bị bỏ qua. Verify command `tail -20 log` hoặc `grep "listening"` không catch phần đầu. Silent degradation: partial migration failed, subsystem fallback, AutoMigrate race — tất cả vẫn cho service "up" nhưng không healthy.
- **Global Pattern [A startup service B → B listening trên port X → kết luận B healthy] → Pitfall Y nếu startup log có error ẩn**: Service state = (listening AND zero error in startup). Nếu chỉ check listening → miss silent bugs chạy degraded. Điển hình: migration failed nhưng app vẫn start với schema cũ, subsystem init fail nhưng wrapped nil check cho phép app chạy thiếu feature, AutoMigrate conflict nhưng SQL error không fatal.
- **Correct Pattern**:
  1. **Full-scan startup log**: sau `nohup/docker compose up`, phải `cat /tmp/log` hoặc `docker logs <c> 2>&1 | head -200` đọc TOÀN BỘ phase khởi động, không chỉ tail.
  2. **Grep negative signals**: `grep -iE "error|fail|panic|sqlstate|warning|denied|refused|timeout" startup.log` — nếu match > 0 → flag + investigate, không gọi "done".
  3. **Báo cáo verify**: mọi lần báo "service up" PHẢI kèm dòng "startup log clean, zero error/warn" với evidence. Nếu skip evidence này = chưa verify.
  4. **Anti-pattern cấm**: "process listening" ≠ "service healthy". "Build pass + curl 200" ≠ "deployment healthy". Mọi milestone verify phải multi-dimension: build + startup clean + functional test + boundary (restart + graceful shutdown).
- **Tags**: #rule3 #verification #startup-log #silent-degradation #auto-migrate #done-criteria

---

## [2026-04-17] Brain chôn critical limitation trong doc volume lớn — user miss → expect feature đã work

- **Trigger**: User initial answer "Debezium JSON hay Avro converter? => avro". Archaeology phát hiện thực tế code dùng JSON. Brain document trong plan v3 §11 + gap analysis V4 (status "Mixed intent vs reality") nhưng định phase B "future 2-3 tháng". Doc tổng cộng ~70KB trải 2 plan v3. User later test Redpanda Console chọn type=Avro → fail deserializing → phản ứng "mày đang đốt token, thông báo vớ vẩn, thực tế ko làm gì cả". Root cause: Brain chôn LIMITATION QUAN TRỌNG trong §11 của doc 38KB → user không catch → expect đã migrate.
- **Root Cause**: Plan v3 doc-heavy approach ưu tiên completeness. Critical gaps bị bury trong pha/section giữa doc. User scan top-level summary không thấy → assume feature delivered. Khi bị phá vỡ expect, user thấy Brain "nói một đằng làm một nẻo".
- **Global Pattern [A write doc dài D cho feature F với limitation L ở §N] → User miss L nếu L không surface TOP**: Nếu có gap CRITICAL giữa user intent vs delivered state (intent=Avro, delivery=JSON + "future plan"), gap đó PHẢI surface ở top section (0 hoặc 1) của doc + báo cáo tổng kết, không chôn ở §N giữa doc hay cuối.
- **Correct Pattern**:
  1. **Gap surfacing**: mỗi plan/report MUST có "⚠️ NOT DELIVERED" section ngay sau Executive Summary, list rõ feature user expect vs actual delivered state. Không chôn, không softening "planned for phase B".
  2. **Intent verification**: khi user answer 1 assumption ngắn gọn (1 từ "avro"), Brain phải echo back intent + current state + gap rõ trong 3 dòng đầu: "User: muốn X. Current: Y. Gap: Z. Plan: W."
  3. **Delivery summary discipline**: `07_delivery_summary_*.md` PHẢI có "NOT YET DELIVERED" subsection với bullet list cụ thể các limitation + workaround + effort để fix. Không "known follow-ups" soft footer.
  4. **Anti-pattern**: "Planned for Phase B / future 2-3 tháng" = từ chối make decision + escalate sang doc → user không biết feature nào live, feature nào doc-only. Phải binary: DELIVERED hoặc NOT_DELIVERED (với reason).
- **Tags**: #doc-discipline #limitation-surface #user-expectation #report-pattern #not-delivered-visibility

---

## [2026-04-17] Fix bug chỉ 1 service, quên search cross-service same pattern

- **Trigger**: Session trước Worker `worker_server.go:59` dính GORM AutoMigrate `ALTER COLUMN created_at DROP NOT NULL` conflict với composite PK của migration 010. Brain delegate Muscle fix — nhưng **chỉ fix Worker**, KHÔNG check CMS. User chạy CMS sau → startup log có **CÙNG ERROR** ở `cdc-cms-service/internal/server/server.go:52`. User: "rồi mày lại quên check start lên ok mới báo done". Cả 2 service cùng project cùng bảng (`cdc_activity_log`) cùng pattern AutoMigrate → phải fix cả 2.
- **Root Cause**: Khi Muscle/Brain fix bug, scope mặc định = file được report. Không expand search "pattern này xuất hiện ở đâu khác trong monorepo". Violations đã ghi: (a) service listening ≠ healthy + (b) over-engineer. Giờ thêm: **fix 1 chỗ khi pattern áp dụng nhiều chỗ = regression**.
- **Global Pattern [A fix bug B tại file F1 → kết luận done] → Pitfall nếu pattern B xuất hiện ở F2, F3... cross-service**: Mọi bug fix PHẢI scope-expand trước khi close: (1) grep cross-repo pattern gốc (AutoMigrate call, migration table name, duplicated helper), (2) verify mọi service startup clean sau fix, (3) chỉ close khi zero error cross cả monorepo.
- **Correct Pattern**:
  1. **Pattern search mandatory**: bug fix → grep `rg "AutoMigrate" --type go` (hoặc pattern generic) toàn monorepo → list mọi callsite → fix hết trước khi close.
  2. **Cross-service startup verify**: nếu có bug chung bảng PG → start ALL services consume bảng đó → check startup log clean ALL. Stop ở 1 service = chỉ 50% verified.
  3. **Monorepo discipline**: nghĩ theo "system" không theo "file". Worker + CMS + FE cùng bảng/config/convention → fix convention không phải fix per-file.
  4. **Anti-pattern**: "Muscle fixed file X" → "báo done". Phải là "Muscle fixed pattern P applied at X, Y, Z → verified startup clean A, B, C".
- **Tags**: #cross-service #pattern-search #regression #monorepo-discipline #auto-migrate

---

## [2026-04-17] Band-aid fix symptom, không solve root cause → user lại chửi

- **Trigger**: User phát hiện ReconHeal spam audit log — 3426 rows trong 1 phút cho bảng 1713 records. Brain delegate Muscle fix — Muscle "cap audit log at 100 sample + aggregate counter". User reply: "thằng chó brain đâu, solution chó đó, bị ngu vừa thôi. các skill của mày đâu. tao đã nói quan tâm tới performance, mày làm chưa". Đúng: fix audit = **band-aid symptom**. Root cause thực: **TẠI SAO Heal process 1713 records khi chỉ có thể 0 mismatch?** Plan v3 spec Heal CHỈ cho subset mismatch từ Recon Tier 2, không phải full scan table. Mọi skip trong log = Heal đang ôm full set → architectural violation, audit chỉ là symptom.
- **Root Cause (meta)**: Khi symptom xuất hiện (spam log), Brain jump to "fix log format" instead of asking "tại sao có nhiều log thế". Missing upstream analysis. Pattern: treat LOG như là bug, không treat LOG như là evidence của bug khác lớn hơn.
- **Global Pattern [A thấy symptom S trong output O → fix O display] → Pitfall Y nếu S là evidence của upstream bug U**: Symptom không phải bug. Symptom là evidence. Trước khi fix symptom, hỏi "tại sao symptom xuất hiện". Nếu log spam = 1 row per record, ask: "tại sao mỗi record cần log?" → "tại sao mỗi record được process?" → có thể up tới "tại sao full table đi vào heal flow?" — đó mới là root.
- **Correct Pattern**:
  1. **5-whys trước khi fix**: log spam → why log per record → why process per record → why full set in flow → why no mismatch detection upstream → ROOT.
  2. **Re-read spec vs impl gap**: khi gặp bug production, re-read original plan/spec section cho feature đó → compare impl hiện tại → identify spec violation. Plan v3 §4: "Heal cho MISSING IDs" vs impl "Heal cho all IDs" = architectural gap, không phải bug log.
  3. **Symptom-first fix policy**: CHỈ được band-aid symptom khi đã xác định root cause cần nhiều thời gian và symptom đang có active damage (spam log tăng DB size immediate) → band-aid tạm time để stop bleeding, nhưng MUST follow up với root fix. Phải explicit "đây là band-aid, root cause X cần fix sau".
  4. **Anti-pattern**: fix display/aggregation/cap cho output metric → claim done. Pattern này là "hide bug", không "fix bug".
- **Tags**: #root-cause #band-aid #symptom-vs-cause #5whys #spec-impl-gap #performance-vs-display

---

## [2026-04-17] Upgrade version ≠ more stable — regression across Console versions

- **Trigger**: Redpanda Console v2.8.1 báo `INVALID_TOPIC_EXCEPTION` cho mọi topic (kể cả `_schemas`) dù Kafka connected OK. Brain upgrade → v3.1.2 → panic `nil pointer dereference` trong message worker. Downgrade v2.7.2 → works. 2 phiên bản mới hơn đều regression với Debezium MongoDB Avro envelope (union types + nullable fields).
- **Root Cause (meta)**: Software "upgrade = better" là giả định. Actually regression rate cao cho:
  - Nested union types (Avro `["null", "string"]`)
  - Library deserializer generated from complex schemas
  - Debezium envelope patterns (well-known but version-specific support)
- **Global Pattern [A upgrades B from V_old to V_new expecting fix/improvement] → Result Y regression nếu V_new chưa test với data pattern của A**: Bump version mà không verify compat = roll dice. Debezium + Avro + MongoDB format là common pattern nhưng vendor regression happens.
- **Correct Pattern**:
  1. **Version matrix test**: khi tool vendor-provided (Console, Connect, UI) bị lỗi → test 1 step back (V-1 minor) TRƯỚC KHI jump forward (V+1 major).
  2. **Decision tree**: current broken → try 1 older patch → try 1 older minor → try latest stable → try latest RC. Không phải "upgrade latest = done".
  3. **Pinning discipline**: khi tìm được version working, pin trong docker-compose/manifest + note ngắn reason trong comment. "v2.7.2 — v2.8+ regression trên Debezium envelope".
  4. **Anti-pattern**: "latest = always best" → bị slap regression, user lose trust.
- **Tags**: #version-regression #downgrade-valid #vendor-bug #avro #debezium #console-ui

---

## [2026-04-20] Partitioned table SLOW SQL — index phải ở parent, không per-partition runtime

- **Trigger**: User báo `system_health_collector.go:599,610` SLOW SQL 306-440ms trên `SELECT COUNT(*) FROM failed_sync_logs` + `ORDER BY started_at DESC LIMIT 10 FROM cdc_activity_log`. Cả 2 bảng đã partitioned (migration 010). Root cause: **parent partitioned table thiếu index trên columns cần**. PG tự Seq Scan từng partition khi query span cross-partitions.
- **Root Cause**: PG 11+ partitioned tables yêu cầu index ở **parent level** để auto-propagate xuống existing partitions + future partitions created via `CREATE TABLE ... PARTITION OF`. Muscle trước có thể tạo indexes per-partition runtime (không migration) → lost trên fresh deploy; không bootstrap cho partition mới.
- **Global Pattern [A has partitioned table B spans N partitions] → SLOW nếu query sort/filter ở column thiếu parent index**: Per-partition query cheap, nhưng cross-partition query phải Merge Append. Không có parent index → Seq Scan each partition. Sort + LIMIT qua nhiều partitions không có sort index = O(N×P) nơi N=rows, P=partitions.
- **Correct Pattern**:
  1. **Parent-level CREATE INDEX**: `CREATE INDEX IF NOT EXISTS idx_... ON parent_table USING btree (column DESC)` → PG auto-propagate xuống children + future.
  2. **Migration persist**: mọi index runtime PHẢI có file migration. Runtime-only indexes = time bomb for fresh deploy/DR.
  3. **Verify EXPLAIN plan**: query cross-partition PHẢI show `Index Scan using {partition}_{column}_idx` hoặc `Bitmap Index Scan`, KHÔNG `Seq Scan`.
  4. **Partition aware DDL**: khi ADD COLUMN hoặc INDEX cho partitioned table → dùng parent level, không iterate từng partition.
  5. **Anti-pattern**: `CREATE INDEX ... ON partition_child_1; CREATE INDEX ... ON partition_child_2; ...` = manual N times, miss future partitions.
- **Tags**: #partitioned-tables #slow-sql #index-propagation #parent-index #migration-discipline #postgresql

---

## [2026-04-20] Bug handling routine inconsistent — cần SOP chính thức

- **Trigger**: User nhắc "khi làm 1 bug gì nhớ làm theo core /agent, note lại lỗi gì, cách giải quyết và tiến trình giải quyết". Session history có 58 lessons + nhiều bug fixes nhưng inconsistent: (a) đôi khi Muscle fix xong quên tạo workspace doc, (b) đôi khi Brain ghi lesson sai chỗ (auto-memory thay vì global), (c) đôi khi band-aid fix không escalate lesson, (d) đôi khi fix 1 service miss cross-service pattern. Routine có nhưng không enforced cứng.
- **Root Cause (meta)**: Individual agent (Brain/Muscle) có thể tuân core /agent một phần nhưng SOP chưa written thành workflow file cứng → easy to skip under time pressure / context switch. Khi chuyển giữa bugs, easy to forget "tạo doc trong workspace" hoặc "ghi lesson nếu có sơ sót".
- **Global Pattern [A fix bug B → skip step S của routine R] → Result Y technical-debt accumulation**: Routine discipline không tự nhiên với AI agents. Cần workflow file viết rõ + Definition-of-Done checklist. Thiếu checklist = inconsistent output.
- **Correct Pattern**:
  1. **Workflow file chính thức**: `agent/workflows/bug-handling-sop.md` với 7 stage (Intake → Plan → Execute → Verify → Document → Lesson → Close) + quick reference card.
  2. **Definition of Done checklist bắt buộc** trong mọi bug close: build pass + runtime verify + workspace doc + progress append + lesson if sơ sót + security gate + cross-service verified.
  3. **Debug-agent workflow update**: thêm step 6 (Document) + step 7 (Lesson Capture) với table trigger→lesson mapping.
  4. **Pre-flight Rule 14 cứng**: mọi response close bug phải có block "Evidence", "Files", "Skills" — không phải optional.
  5. **Anti-pattern**: "Fix xong → báo done" mà skip (a) workspace doc (b) progress append (c) lesson (d) cross-service verify. Mỗi miss = future regression risk.
- **Tags**: #sop #routine #bug-handling #workflow-discipline #definition-of-done #process

---

## [2026-04-20] Lesson cũ không enforce cho new code — ScanFields lặp 3 violation đã có ADR

- **Trigger**: User architectural review `ScanFields` phát hiện 3 violation: (1) HTTP sync thay vì NATS async (ADR-015), (2) CMS touches Airbyte + INSERT mapping_rules thay vì delegate Worker (service boundary ADR), (3) hardcoded AirbyteSourceID bỏ qua `SyncEngine`/`SourceType` registry columns. Cả 3 rules đã ghi lesson/ADR từ 2026-03-31 (4 violations trước đã fix: Backfill, Standardize, Discover, Introspection) nhưng ScanFields là code MỚI sau đó vẫn lặp lại y chang pattern. Lesson hiện tại = documentation only, không enforce vào pre-commit/code-review.
- **Root Cause (meta)**: Lesson thụ động. Khi contributor (AI hoặc human) viết endpoint mới, không ai nhắc "grep ADR cũ trước khi viết". Workspace docs chứa ADR nhưng không có gate tự động. Brain/Muscle delegate code mới thiếu pre-flight check "feature mới có lặp pattern cấm không?".
- **Global Pattern [A writes code N at time T1] + [Lesson L about pattern P documented at T0 < T1] → Y violation nếu A không check L before writing N**: Lesson passively stored không chặn lặp. Cần active enforcement: pre-flight checklist, automatic lint/grep, hoặc architectural review gate.
- **Correct Pattern**:
  1. **Pre-commit grep ADR**: trước khi write endpoint mới chạm `/airbyte/`, `/DW/`, `information_schema` → `rg "service_boundary|ADR-[0-9]+" agent/memory/` để load applicable rules.
  2. **Endpoint checklist**: thêm mỗi POST endpoint vào code review: "Có dùng NATS async? Có tuân service boundary? Có support multi-source registry?".
  3. **Architectural review step trong bug-handling-sop**: nếu bug liên quan architectural decision cũ → grep lesson/ADR TRƯỚC khi propose fix.
  4. **Repeat-violation detection**: Brain scan periodically — nếu fix ra new code pattern giống cũ → flag ngay, không delegate Muscle.
  5. **Anti-pattern**: lesson viết ra rồi forget. Lesson = active reference, không phải archive.
- **Tags**: #adr-enforcement #repeat-violation #service-boundary #lesson-passive #architectural-review

---

## [2026-04-20] Cross-service refactor — Muscle parallel coordinate via subject contract

- **Trigger**: User approve fix 12 architectural violations (NATS async + service boundary + multi-source routing). Scope lớn cross 3 projects (Worker + CMS + FE). Brain delegate 3 Muscle parallel. Risk: race condition — CMS publish subject nhưng Worker chưa subscribe → lost commands?
- **Root Cause (pattern design)**: NATS **fire-and-forget** pattern cho phép parallel refactor mà không cần sync. CMS publish return immediate; nếu Worker chưa ready → message sit trong JetStream (retention 7 ngày) cho đến khi Worker subscribe pick up. FE polling status từ activity log → graceful handle "pending" state.
- **Global Pattern [A publishes event E to message broker B] + [C consumes E at some future time]**: Không cần A biết C đã ready. Broker buffers. Pattern hỗ trợ independent deploy + rolling refactor. Async decoupling > sync coupling.
- **Correct Pattern**:
  1. **Subject naming contract TRƯỚC**: agree naming (`cdc.cmd.{action}`) + payload schema giữa Brain + Muscle trước khi delegate. Parallel Muscle implement độc lập theo contract.
  2. **Fire-and-forget allowed**: CMS publish không chờ Worker subscribe. Worker subscribe khi deploy. JetStream retention guarantee no message loss.
  3. **FE polling absorb async uncertainty**: UI state machine handle `accepted → running → success|error|timeout`. User nhìn badge, không chờ.
  4. **Verify cross-boundary post-deploy**: sau all Muscle done, verify end-to-end: FE dispatch → CMS publish → Worker consume → activity log → FE poll detect. Not before.
  5. **Anti-pattern**: synchronous refactor Worker first, then CMS, then FE — waste parallel capacity + block progress.
- **Tags**: #cross-service-refactor #nats-fire-and-forget #parallel-delegation #subject-contract #async-decoupling

---

## [2026-04-20] Partitioned Table Default Orphan — Backfill, Not Just Retention

- **Trigger**: SLOW SQL 236ms regression trên query đã bounded (`WHERE X > NOW() - INTERVAL AND X <= NOW()`) — nghi ngờ fix trước đó (migration 015 + bounded range) vô hiệu. Thực tế planner vẫn không prune được vì `*_default` chứa rows trong window.
- **Global Pattern**: **[A partitioned table B có default partition C giữ orphan rows D → planner Y không thể prune C → mọi query trên B phải scan C + catalog overhead → planning time tăng tuyến tính với độ đầy C]**. Mặc dù bounded range predicate được thiết kế để kích hoạt runtime pruning, **runtime pruning không áp dụng cho default partition** (PG không có positive range để so sánh, chỉ có synthesized NOT-IN của siblings → default luôn là "có thể match"). Hậu quả: Subplans Removed trên EXPLAIN đếm sibling partitions đã prune, nhưng default **luôn** hiện trong Append nếu có bất kỳ row nào. Sai lầm conceptual: coi default là "fallback empty" giống null-value bucket, nhưng thực ra là một partition bình thường, Schedule Y/Z tick đều scan nó.
- **Correct Pattern**: Automation quản lý partition phải có **2 chiều**:
  1. **Forward (existing)**: pre-create future partitions mỗi tick để INSERT mới không rơi vào default.
  2. **Backward (missing)**: detect rows đã land vào default → materialise child partitions đúng range → move rows. Chỉ drop default khi hoàn toàn trống.
- **PG 11+ gotcha**: `CREATE TABLE … PARTITION OF … FOR VALUES FROM … TO …` sẽ fail `SQLSTATE 23514` nếu `*_default` hiện đang chứa row trong range đó. Correct txn ordering = **drain-before-create**: (a) `DELETE … RETURNING * INTO TEMP`, (b) `CREATE TABLE … PARTITION OF …`, (c) `INSERT INTO parent SELECT * FROM temp`. Sai ordering (CREATE trước move) chỉ detect được qua smoke test với real data.
- **Example mapping**: A=`partition_dropper` service, B=`cdc_activity_log`, C=`cdc_activity_log_default`, D=recon/scan test rows (dates 2026-04-14→16), Y=postgres query planner, Z=collector tick 15s × CMS uptime.
- **Generalization check**: pattern áp dụng cho (1) pg_partman deployments missing backfill grace period, (2) Debezium CDC tables với range-partition theo `source_ts`, (3) audit/log tables bất kỳ có default catch-all với late-arriving data, (4) multi-tenant partitioned tables với tenant_id partition key khi new tenant onboard trễ.
- **Tags**: #postgres #partitioning #planning-time #slow-sql #pg11 #default-partition #backfill #rule6 #root-cause

---

## Lesson 62 — Hard-coded field name in cross-store sync breaks on schema drift (2026-04-20)

- **Trigger**: Reconciliation reports `source_count=0 / dest_count=3422` for `refund_requests`, `source_count=0 / dest_count=15` for `export_jobs`. User assumed schedule not firing, but actually schedule DID fire — source agent's Mongo filter `bson.M{"updated_at": {"$gte": tLo, "$lt": tHi}}` returned 0 because the actual collections use `createdAt` + `lastUpdatedAt`, not `updated_at`. Mongo driver silently decodes missing field to zero-value `time.Time{}` without error, hiding the mismatch from tests and smoke runs.
- **Global Pattern**: **[A cross-store sync/recon component A hard-codes a field-name B from the "canonical" convention → collection X with a different convention (camelCase, created_at, lastUpdatedAt, ts) → filter matches 0 rows → Y reports "source empty" falsely → operator blames the scheduler Z rather than the schema assumption]**. The anti-pattern compounds when the decoder uses typed struct tags (`bson:"updated_at"`) instead of `bson.M` — the zero-value decode path IS the silent failure mode. Tests pass because fixtures use the canonical field.
- **Correct Pattern**: Two complementary defences:
  1. **Registry-first**: add a per-table config column (here `cdc_table_registry.timestamp_field`) + whitelist validator (`^[A-Za-z_][A-Za-z0-9_]{0,63}$`) so operators can declare the right field per collection. Default preserves backward compat.
  2. **Fallback graceful**: when the declared field is absent on a specific document, fall back to a universally-available source (Mongo `ObjectID` carries unix seconds in its first 4 bytes — `primitive.ObjectIDFromHex(...).Timestamp()`). Caller treats the fallback as "approximate ts" — still correct for hash/presence checks, degrades cleanly for range filtering.
  3. **Observability**: surface the chosen path to the UI (`source_query_method` = `window_updated_at | window_custom_field | window_id_ts_fallback | full_count`) so operators can answer "why did this count surprise me?" without reading Go source.
- **Mongo gotcha**: Typed struct decode vs `bson.M` decode. Typed = zero-value on missing, no error. `bson.M` = field simply absent from map, `_, ok := raw[key]` = false. Prefer `bson.M` + explicit extraction when the field existence is itself a semantic signal.
- **Example mapping**: A=`ReconSourceAgent`, B=`updated_at` hard-coded filter, X=`export-jobs` (createdAt) + `refund-requests` (mixed), Y=`cdc_reconciliation_report.source_count`, Z=`cdc_worker_schedule[reconcile]`.
- **Generalization check**: pattern applies to (1) Debezium source connectors hard-coding `__last_updated_at` cursor, (2) Airbyte incremental sync with fixed cursor_field across heterogeneous schemas, (3) ETL pipelines assuming a timezone-aware `updated_at` when source is a Mongo snake-case-to-camelCase mix, (4) webhooks filtering by `received_at` when upstream rebrands to `timestamp`/`ts`/`eventTime`.
- **Anti-drill**: do NOT "auto-detect field by sampling first 100 docs" as the only defence — inconsistent collections (some docs have A, some have B) would alternate answers across restarts. Explicit registry config + documented fallback is more debuggable.
- **Tags**: #reconciliation #mongo #cross-store #schema-drift #field-naming #hardcoded-assumption #rule3 #rule6 #root-cause #bson-decode-gotcha

---

## Lesson 63 — Silent-skip in scheduled jobs masks nil-dependency init failures (2026-04-20)

- **Trigger**: Worker's scheduled `reconcile` op wrote `activityLogger.Quick("reconcile", "*", "scheduler", "skipped", ...)` when `reconCore == nil`, then returned. Operators watching `worker.log` saw zero reconcile activity but no error — indistinguishable from a goroutine that panicked early. Real cause was MongoDB URL missing from config, caught only in an earlier `logger.Warn("MongoDB connection failed, reconciliation disabled")` buried in the startup stream.
- **Global Pattern**: **[A scheduled job A depends on lazily-initialised core B → startup failure of B leaves A.core=nil → A.Tick() silently short-circuits with a "skipped" row in audit table C → operators querying log-stream D cannot distinguish "skipped-by-config" from "crashed" from "never-scheduled"]**. Activity-log rows are NOT a substitute for log-stream WARN when the condition is a dependency-initialisation failure, because audit tables are per-record and log streams are temporal — operators scan the stream when diagnosing "is this running?".
- **Correct Pattern**: every silent-skip path in a scheduled job must:
  1. **WARN the log stream** on the first skip AND on every tick (repeated nil is a persistent operator-visible signal, not a one-off).
  2. **Include a `fix_hint` in the log fields** — "set MONGODB_URL env + restart worker; check startup log for 'MongoDB connection failed'" — so the triaging operator can resolve without reading code.
  3. **Emit a startup summary** when the poller starts: `"schedule poller started" enabled_count=N registered=[op=Nm,op=Nm] recon_core_available=bool` — names the available upstream deps, lists what will fire, confirms the goroutine is alive.
  4. **Per-tick info log** includes `first_run:bool` when `LastRunAt IS NULL` so operators can distinguish "fresh enable fires immediately" from "interval not elapsed yet".
- **Example mapping**: A=`runReconcileCycle`, B=`reconCore`, C=`cdc_activity_log`, D=`worker.log`.
- **Generalization check**: pattern applies to (1) cron-driven DLQ replayers depending on Kafka/NATS handles, (2) scheduled Airbyte triggers depending on REST client init, (3) Prometheus push gateways skipping when metric registry is nil, (4) any graceful-degrade path that chooses to return rather than error on missing deps.
- **Anti-drill**: do NOT replace silent-skip with panic — that would take down the whole worker on an optional dependency. The right balance is WARN-log + keep running + surface in /metrics counter so dashboards can alert on `*_skipped_total > 0`.
- **Tags**: #scheduling #observability #silent-failure #nil-dependency #log-stream-vs-audit-table #rule6 #rule8-escalation #root-cause

---

## [2026-04-20] Brain propose per-table band-aid thay vì systematic auto-detect — không scale N entities

- **Trigger**: User report payment_bills recon src=0 (Mongo 2 docs với createdAt, không updated_at). Brain đề xuất trong Muscle brief: "Quick fix payment_bills: UPDATE registry SET timestamp_field='createdAt' WHERE target_table='payment_bills'". User phản ứng: "với quy mô 200 table, mày cũng fix từng cái à, ngu đần. cái cần là giải pháp thông minh. ko phải làm kiểu tình thế". Đúng: fix per-entity O(N) manual intervention ≠ systematic solution O(1) auto-detection. Session history đã lặp pattern: export_jobs cũng manual fix timestamp_field, giờ payment_bills tương tự — nếu 200 tables thì cần 200 UPDATE statements + admin knowledge per-table schema.
- **Root Cause (meta)**: Brain optimize cho "fix bug hiện tại" thay vì "fix cơ chế gây ra bug". Per-entity fix = tình thế (band-aid). Systematic solution = auto-detect sample + fallback chain + admin override-only khi cần. Pattern tương tự lesson #60 (ADR passive không enforce) — cần ACTIVE design, không reactive.
- **Global Pattern [A configures entity B_i with field F manually for each i ∈ N entities] → O(N) human intervention + high error rate**: Entity configuration yêu cầu admin knowledge schema per-entity = unmaintainable ở scale. Correct: auto-detect từ entity data itself + fallback chain + registry default + admin override chỉ khi auto fail.
- **Correct Pattern**:
  1. **Auto-detect at entity boundary** (register time HOẶC first-scan): sample data → detect field presence ranking → auto-populate config.
  2. **Fallback chain runtime**: nếu configured field returns 0 documents trong N consecutive runs → auto-try next candidate → update registry suggestion → admin review.
  3. **Admin override escape hatch**: UI form cho phép manual override (backward compat) nhưng default = auto.
  4. **Log recommendations**: worker log "detected field X for table Y with confidence Z%, fallback to W available" → admin có visibility không cần query each table.
  5. **Anti-pattern**: "UPDATE registry SET config='X' WHERE name='Y'" → repeat for each entity. Nếu 200 entities → 200 sql statements = tình thế.
- **Tags**: #band-aid-vs-systematic #auto-detect #scale-n-entities #registry-config #per-entity-fix

---

## [2026-04-20] Brain viết plan decisions dựa trên state tưởng tượng, không verify

- **Trigger**: User cung cấp Master Plan v1.25. Brain viết section "6 Decisions Required" có Q5: "Migrate `sync_engine='both'` đầu tiên hay cuối?". User phản ứng: "bỏ cái này mà, đọc tài liệu kiểu gì vậy" — vì hiện tại **0 tables có sync_engine='both'** (verified session trước: 6 airbyte + 2 debezium + 0 both). Câu hỏi invalid, hallucinate state.
- **Root Cause (meta)**: Brain viết plan decisions mà không re-verify runtime state ngay trước khi ask. Trong session đã có evidence `SELECT sync_engine, COUNT(*)` từ earlier audit. Brain forgot/ignored → wrote decision question dựa trên possibility, không reality.
- **Global Pattern [A designs plan asking decisions about entity state S] → Invalid nếu A không verify S hiện tại**: Plan decisions require ground truth about current state. Extrapolating "có thể có" → asking user as if real = wastes user time + signals sloppy work.
- **Correct Pattern**:
  1. **Pre-decision state re-verify**: trước khi write "Decisions Required" section, re-run relevant queries (DB state, feature flags, deployment status) → confirm entities exist BEFORE asking about them.
  2. **State snapshot in plan**: embed current state query output (e.g., `sync_engine counts`) ngay trong plan Section 1 "Current State" — force self-audit.
  3. **Conditional decisions**: nếu decision về state possibly nonexistent, phrase as "IF X exists, then...". Không "which X first" as default.
  4. **Anti-pattern**: copying decision template từ generic migration framework → asking questions irrelevant to specific environment.
- **Tags**: #hallucination #state-verification #plan-decisions #ground-truth #user-flag

---

## [2026-04-20] Passive plan (band-aid) vs Systematic Reconstruction — 6 violations cùng lúc

- **Trigger**: User provide Master Plan v1.25 (Unified Sonyflake). Brain viết plan tích hợp nhưng vi phạm 6 nguyên tắc user đã nêu rõ: (1) View band-aid giữ _airbyte_* rác physical layer, (2) Trigger IF NULL cho phép Go pass sai ID, không FORCE DB, (3) Mapping _gpay_* ↔ _* cũ spaghetti, không unified prefix, (4) COALESCE anti-ghosting quên OCC với _source_ts migration 009, (5) Giữ PK cũ "nhát gan" gây dual-index phình IO, (6) Worker ID 0 mặc định không verify Go IP range collision. User: "passive, che đậy, giữ tàn dư cũ cản trở Unified Architecture".
- **Root Cause (meta)**: Brain mặc định **minimum-disruption = good**. Với migration feature/column đơn lẻ OK. Với **architectural reconstruction** (new identity system), minimum-disruption = lỗ hổng vì **tàn dư cũ chính là bug source**. User yêu cầu "Unified" tức nguyên khối, Brain trả "incremental alias" tức **trái nguyên tắc**.
- **Global Pattern [A plans architectural reconstruction R] + [A defaults to minimum-disruption M] → Result fail-to-deliver R**: Reconstruction ≠ migration. Reconstruction đòi hỏi **drop + rebuild** clean slate. Migration đòi hỏi **preserve + transform** backward compat. Nhầm 2 modes = plan nửa vời, cũ vẫn ám mới.
- **Correct Pattern for Architectural Reconstruction**:
  1. **Physical clean slate**: Không giữ column rác dưới mọi hình thức (VIEW ẩn vẫn chiếm disk, VACUUM chậm, backup bloat). Drop physical + bóc business fields sang columns thật.
  2. **Force authority**: Identity Provider phải SINGLE. DB sinh ID = DB SOLE AUTHORITY. Go truyền ID = DB validate STRICT (format + range + epoch + worker_id allocation). Không "IF NULL fallback" — phải EXPLICIT REJECT invalid input.
  3. **Unified naming**: Prefix mới = toàn bộ prefix mới. Không alias từ naming cũ. Alias = semantic confusion, spaghetti logic debug.
  4. **Preserve what EARNED its place**: Existing OCC (`_source_ts`) là **working pattern** → rename sang `_gpay_source_ts` giữ semantic, KHÔNG thay thế bằng COALESCE ad-hoc. Earned preservation # sloppy preservation.
  5. **Aggressive cutover**: DROP old PK phải trong cùng migration (transactional), không "defer N days". Defer = indecision = dual-write IO waste.
  6. **Verify environment before reserve**: Worker ID range, epoch, IP allocation phải **query existing deployment** trước assign. "Reserve 0" without checking = assumption = collision risk.
- **Anti-pattern decision tree**:
  - Q: "Preserve for BC?" → Only if column có active consumer code. If only legacy callsite → rewrite callsite, drop column.
  - Q: "View alias for ergonomics?" → Only if reader needs simpler projection. Not for hiding rác.
  - Q: "Dual PK safety?" → Never in unified architecture. Choose one, commit.
- **Tags**: #reconstruction-vs-migration #band-aid #identity-authority #unified-naming #physical-clean-slate #forced-cutover
