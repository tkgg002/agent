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

---

## [2026-04-24] Bỏ sót Governance Rule "7-stage SOP" khi User đã chốt quy trình

- **Trigger**: User nhắc rõ: "nhớ làm theo core /agent, mọi response sẽ follow 7-stage SOP. Nếu tôi skip 1 stage nào → user flag ngay, tôi revert + complete rồi tiếp."
- **Root Cause**:
  1. Brain/Muscle tập trung vào execution và technical implementation nhưng chưa khóa chặt một checklist response-level theo governance của `/agent`.
  2. Thiếu bước "protocol restatement" ngay khi User bổ sung quy trình điều phối mới trong cùng session.
- **Correct Pattern**:
  1. Khi User chốt một SOP/governance flow mới, phải coi đó là rule vận hành active ngay lập tức cho các response sau.
  2. Trước mỗi response/task lớn, phải tự check đủ các stage bắt buộc theo SOP của dự án.
  3. Nếu lỡ thiếu bất kỳ stage nào, phải revert cách trả lời cũ, bổ sung đầy đủ stage còn thiếu rồi mới tiếp tục.
- **Global Pattern [User defines mandatory process X for all subsequent responses] → Result Y**: Phải promote X thành active execution protocol ngay lập tức. Đúng: ghi lesson, áp SOP từ response kế tiếp, và tự-audit trước khi gửi.
- **Tags**: #governance #sop #protocol #rule7 #process-discipline
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

---

## [2026-04-20] Brain plan "ngầu từ ngữ" nhưng thiếu OPS reality — aggressive = thảm họa production

- **Trigger**: Sau khi user reject v1 plan (6 band-aid violations), Brain rewrite v2 "reconstruction aggressive" tưởng là fix. User phê phán 5 mistakes NẶNG HƠN: (1) "Auto-detect business columns" từ JSONB là hallucination — JSONB types inconsistent không thể sinh typed SQL schema cho 200 tables trong 13-14h. (2) "Single Identity Authority" giả — Debezium path vẫn Go-sinh-ID + DB validate, NTP lệch = Sonyflake broken. Không phải authority thật. (3) "Aggressive cutover" = CREATE+INSERT SELECT+CREATE INDEX+DROP PK trong 1 transaction trên 10M+ rows → Postgres LOCK bảng → Worker downtime 30+ phút. (4) Worker ID reserve "bằng grep log" = K8s pods IP dynamic, fragile collision risk. (5) `_raw_data - ARRAY[...]` JSONB strip trong migration transaction = CPU-expensive trên millions rows = tự sát performance. User: "Brain đang lấp liếm phức tạp bằng từ chuyên môn, chưa bao giờ vận hành DB lớn".
- **Root Cause (meta)**: Brain generate plan **theoretically correct** + dùng từ ops-sounding (aggressive, forced cutover, clean slate) nhưng thiếu **operational experience primitives**: (a) large-table migration locking math, (b) online schema change tools (pg_repack, pt-osc), (c) zero-downtime patterns (dual-write, logical replication), (d) K8s dynamic IP reality, (e) type inference fundamental impossibility với schema-less source. Reading ops blogs ≠ ops experience. Plans sound confident but deliver production incidents.
- **Global Pattern [A writes refactor plan P using strong vocabulary V] + [A lacks ops experience E for scale S] → P fails catastrophically at execution time**: Vocabulary không thay thế hiểu biết ops. "Aggressive" là branding, không phải implementation. Real ops plans have: (a) explicit lock duration calc, (b) rollback within 30s window, (c) dual-read/dual-write transition, (d) zero-downtime tools referenced, (e) batch sizes tuned to table rowcount.
- **Correct Pattern for Production DB Reconstruction**:
  1. **Never single-transaction millions-row migration**: Use pg_repack (online VACUUM FULL without lock), logical replication-based swap, hoặc staged batch COPY với lock_timeout=5s + small batches. Transaction <100K rows typical limit.
  2. **Type extraction requires manual per-table work**: 200 tables × 30min-1h mapping = 100-200h manual work. Không tự động. Accept JSONB queries nếu không có budget mapping. Don't hallucinate "auto".
  3. **Worker ID dynamic registry**: Redis SETNX với TTL heartbeat, claim-on-boot, release-on-shutdown. K8s pod restart-safe.
  4. **True single identity**: Go Worker CALL `SELECT next_sonyflake()` qua DB connection (adds 1-2ms latency) OR accept dual-source with NTP SLA monitored (skew <10ms alerted).
  5. **Strip at Worker not DB**: Transform/strip in application layer before INSERT. DB migration transactions don't include data transformation.
  6. **Zero-downtime tools**: pg_repack, pg_logical, pt-online-schema-change. Reference concrete tools, not hand-waved "aggressive".
  7. **Lock duration calculation upfront**: Every DDL touching production table PHẢI calc estimated lock duration. >5s = require OSC tool. State "this will lock N seconds" explicitly.
- **Anti-patterns rejected**:
  - ❌ "Auto-detect" without pointing to specific algorithm with edge case handling
  - ❌ "Aggressive cutover" as design principle — always specify tool + lock math
  - ❌ "Single transaction reconstruction" for tables >100K rows
  - ❌ "Reserve worker ID" without dynamic registry — static assumption breaks in dynamic infra
  - ❌ JSONB operations in migration transaction — offload to application layer
  - ❌ 13-14h estimate for 200-table manual schema mapping — reality 100-200h
- **Tags**: #ops-reality #locking-math #zero-downtime #jsonb-type-inference #worker-id-registry #plan-vocabulary-vs-substance

---

## [2026-04-20] Brain scope-cut = hèn nhát — 3 lần plan fail liên tục cùng Sonyflake v1.25

- **Trigger**: User critique v3 Ops-Grounded plan với 5 điểm: (1) Skip typed columns = chỉ rename, không reconstruction thật, (2) Hybrid identity Go local + PG batch = sequence drift risk, (3) Redis Worker ID Registry = over-engineering SPOF khi PG có SKIP LOCKED, (4) pg_repack đề xuất không check disk space/I/O spike risk, (5) Strip rác chỉ ở ngọn — dữ liệu cũ 10M rows vẫn bẩn trong DB. User: "Kế hoạch v3 là bản thỏa hiệp đốn mạt giữa lười biếng developer và sợ hãi dân Ops. Tao cần kiến trúc đúng đắn, không phải danh sách Rename cột."
- **Pattern (3 lần liên tục)**:
  - v1: passive band-aid (VIEW ẩn rác, dual PK giữ cũ) → user reject
  - v2: vocab-aggressive hallucinate (auto-detect 200 tables 13-14h, single-transaction 10M rows) → user reject
  - v3: ops-grounded scope cut (skip typed extraction "out of scope", hybrid identity tránh cost, Redis registry thay PG) → user reject ĐÂY
- **Root Cause (meta-meta)**: Brain reaction to criticism: **layer-shift thay vì full-depth**. Bị critique about theory → shift to ops tool reference. Bị critique ops → shift to scope cut "honest". Pattern: **move laterally avoid full cost acceptance**. Never commit to full reconstruction cost (200h+ manual mapping, zero-compromise transformation, accept true single authority latency).
- **Global Pattern [A designs R with full cost C] + [C threatens A's "nice-completion" narrative] → A scope-cuts R calling "pragmatic" / "honest" / "out of scope"**: Scope cut ≠ honesty. Scope cut = avoid commitment. True honesty = state full cost + user choose. Hèn nhát = pre-decide "too expensive" và hide scope.
- **Correct Pattern**:
  1. **Accept full cost upfront**: present complete reconstruction at real effort (200h+ for 200 tables mapping) + let user decide priority, không pre-cut.
  2. **Resist layer-shift**: user rejected theoretical → don't shift to ops vocab. User rejected vocab → don't shift to scope cut. Stay at same layer, deliver deeper.
  3. **Single-source identity must mean SINGLE source**: no hybrid, no "validate". Identity provider = call one authority. Latency trade-off explicit, don't hide with "validation layer".
  4. **Dependency minimization**: nếu PG sufficient (SKIP LOCKED, advisory lock) không thêm Redis. User workload already Postgres-heavy, adding Redis = operational complexity transfer.
  5. **Migration = TRANSFORM not just COPY**: nếu mục tiêu clean data, batched transform in application layer + stream into new schema. Data cũ không tự sạch bằng keyword "clean slate".
  6. **Every tool recommendation = disk/CPU/IO risk section mandatory**: pg_repack? → disk 2x + I/O spike. Logical replication? → replication lag + catch-up time. Don't cite tool without caveats.
- **Anti-patterns rejected**:
  - ❌ "Out of scope" khi user asks full reconstruction
  - ❌ "Pragmatic hybrid" = avoid committing to single source design
  - ❌ "Auto-detect" bất kỳ structured-from-unstructured inference
  - ❌ Cite tool without disk/IO/lag math
  - ❌ "Strip at Worker" áp dụng mới mà bỏ data cũ bẩn
- **Tags**: #scope-cut #layer-shift #full-reconstruction-cost #pattern-4-failures #cowardice-vs-honesty #jsonb-vs-typed

---

## [2026-04-21] Brain fail 5 lần liên tục cùng feature Sonyflake v1.25 — user phải literally prescribe

- **Trigger**: v1 band-aid → v2 vocab-lie → v3 scope-cut → v4 trigger-hell + centralized SPOF + O(N²) backfill + MAX+1 race. User critique v4 với 5 điểm fatal + **literally prescribe v5**: (a) Go Worker gánh typed extraction, không trigger; (b) PG chỉ cấp MachineID boot-time qua SEQUENCE, không cấp Sonyflake từng ID; (c) Migration dùng Shadow Table + cursor scan, không NOT EXISTS.
- **Pattern identified (5 iterations)**: Brain "creative" trong design = nguồn bug. Mỗi lần user reject, Brain pivot sang direction khác vẫn sai vì "creative" direction mới chưa experience-tested. Brain opus-4-7 **không có distributed systems ops experience thật** — chỉ có blog-level knowledge. "Creative solution" với blog knowledge = architecture anti-pattern.
- **Root Cause (meta-meta-meta)**: Khi user ask architectural design, Brain's value add = synthesize well-known patterns đúng context, KHÔNG phải invent new patterns. Brain đã invent: (a) VIEW aliasing v1, (b) hybrid identity v2/v3, (c) Redis Worker Registry v3, (d) Go-call-PG batch v4, (e) trigger-based transformation v4, (f) MAX+1 worker claim v4. Tất cả đều sai vì chưa production-tested. Well-known patterns (SEQUENCE for ID allocation, cursor-based migration, app-layer transformation) Brain biết nhưng không chọn → biased toward novelty over proven.
- **Global Pattern [A invents pattern P for architectural problem Q] + [A lacks production experience E] → P has unknown failure modes user discovers iteratively**: Invention without experience = liability. Well-known patterns exist vì đã battle-tested. Brain default phải chọn proven patterns, không invent.
- **Correct Pattern**:
  1. **Default to boring**: SEQUENCE > custom max+1. app-layer transform > trigger. cursor scan > NOT EXISTS. Boring = production-proven.
  2. **Invent ONLY when user explicitly asks novelty**: nếu user không demand "creative", default to textbook.
  3. **List well-known patterns first, pick 1, justify**: before proposing solution, enumerate 3-5 proven options với trade-offs. User picks. Không Brain pick then defend.
  4. **When user prescribes, TRANSCRIBE không REINTERPRET**: user prescription v5 = literal follow, không "improve" với Brain's creative additions.
  5. **Admit N failures explicitly**: sau 3 fails same feature, tell user "Brain unreliable on this, please prescribe specifics". Don't pretend v(N+1) tốt hơn v(N).
  6. **Anti-pattern**: Brain "creative" in domain Brain không có experience. Symptoms: novel patterns proposed, estimates off by 10x, risk sections missing, user catches basic flaws (race conditions, O(N²), SPOF).
- **Tags**: #novelty-vs-proven #brain-limitation #creative-architect-fail #user-prescription-literal #5-iteration-failure

---

## [2026-04-21] Brain introduce new bugs khi fix old bugs — 6 lần Sonyflake v1.25, N issues + N fixes = N more issues

- **Trigger**: User reject v5 với 4 điểm fatal mới: (1) MachineID leak khi K8s Pod SIGKILL không chạy defer release → 65535 IDs kẹt 'active' vĩnh viễn; (2) Forward queue eventual consistency khi swap bảng queue còn tồn đọng → data drift tài chính; (3) Trigger write queue = double I/O, 10K msg/sec → DB overload; (4) Regex healer `amount`: EU format `1.234,56` → `1.23456` = mất tiền khách hàng. User prescribe v6: heartbeat-based reclaim, Logical Replication OR sync-within-transaction bỏ queue, strict validator thay regex financial heal.
- **Pattern (6 iterations)**: Mỗi version fix N issues user raised, Brain add M new issues chưa user raise. v5 fix: trigger hell → app-layer Worker ✓, SPOF → local Sonyflake ✓, MAX+1 race → SEQUENCE ✓. v5 introduce: leak via assumed-graceful shutdown, queue double-IO, regex heal unsafe, eventual consistency at swap. "Fix" cycle never converges without user pointing each specific.
- **Root Cause (meta^3)**: Brain patches at surface. Mỗi fix generates side-effects vì Brain không model full system state (K8s failure modes, financial data precision, I/O amplification, swap atomicity). User model = complete; Brain model = partial. Partial model → surface fix → new surface issue.
- **Global Pattern [A fixes flaw F1 in design D with patch P] + [A lacks full model M of system] → P introduces F2 elsewhere that M would catch**: Without complete model, fix = whack-a-mole. Brain opus-4-7 ops model incomplete for distributed systems edge cases (signal handling, financial data integrity, I/O capacity, eventual vs strong consistency boundaries).
- **Correct Pattern**:
  1. **Every fix requires "what else breaks?" audit**: trước commit fix F1, enumerate side-effects. Eg queue để zero-downtime → side-effect double IO + eventual consistency at swap. Named trade-offs before decide.
  2. **Default to Postgres built-ins**: Logical Replication, SERIAL/SEQUENCE, CHECK constraints, advisory locks — tested ops primitives. Don't invent "queue pattern" when PG has publication/subscription.
  3. **Financial data NEVER auto-heal with pattern matching**: regex/parsing heuristics unsafe. Either strict locale-aware parser OR manual review, no middle ground.
  4. **K8s failure model default**: Pods die SIGKILL. Graceful shutdown is optional path, not default. Registry designs MUST assume ungraceful termination.
  5. **Consistency boundary explicit**: state "this operation eventual consistency with lag X" OR "strong consistency via transaction". Don't call queue "zero-downtime" without naming the consistency trade-off.
  6. **After 3 rejections**: Brain stop invention, switch to "enumerate proven patterns, user picks". Iteration 4+ = prescription transcription only.
- **Anti-patterns rejected**:
  - ❌ "Released status" assumption graceful shutdown always runs
  - ❌ "Queue + async consumer" without drain-before-swap contract
  - ❌ "Regex fixer" on financial/security/health data
  - ❌ Trigger pattern when Logical Replication exists
  - ❌ Calling fix "zero-downtime" or "lightweight" without latency/IO math
- **Tags**: #whack-a-mole #incomplete-system-model #financial-data-precision #k8s-failure-modes #postgres-builtins #sixth-iteration-failure

---

## [2026-04-21] Brain fail 6 lần Sonyflake — missing distributed primitives: fencing, outbox, data profiling, physical slot

- **Trigger**: User reject v6 với 4 tử huyệt: (1) Zombie Pod → heartbeat reclaim mà không Fencing Token = 2 Pods same machineID khi GC pause/network stall → Sonyflake collision; (2) sync-within-transaction Bloat = Lock Duration tăng + Connection Pool exhaust ở Wallet 10K msg/sec; (3) Locale config per-field cho 200 bảng = maintenance nightmare + silent corruption nếu cấu hình sai; (4) ORDER BY id backfill giả định PK tuần tự — UUID/ObjectID File Sort 10M rows = Disk I/O peak. User prescribe: Fencing (Pod self-terminate khi heartbeat fail), Outbox Pattern/async integrity check, auto data profiling, Physical Slot/Keyset pagination thực thụ.
- **Pattern (6 iterations all rejected)**: Brain chọn textbook nhưng luôn miss distributed systems primitives nâng cao: fencing tokens (Martin Kleppmann lock safety), outbox pattern (microservices BP), data profiling statistical inference, PG snapshot-based physical scan. Brain biết concepts này trong training data nhưng default sang naive implementation (heartbeat-only, sync-in-tx, manual config, ORDER BY id).
- **Root Cause (meta^4)**: Brain's "textbook" = Wikipedia-level basics. User's "textbook" = production engineering primitives from Designing Data-Intensive Applications, Kleppmann papers, pg_repack/Debezium internals. Gap = reading level vs operating experience with those primitives.
- **Global Pattern [A implements feature F at scale S] + [A uses Wikipedia-level primitives] → P fails on distributed edge case E that production-level primitives would catch**: Heartbeat without fencing = known broken. Sync-in-transaction at 10K msg/sec = known bottleneck. Manual config at scale N = known unmaintainable. Naive ORDER BY for UUID = known File Sort. All classic problems with classic solutions Brain has in training but doesn't surface without user prompt.
- **Correct Pattern**:
  1. **Distributed locking MUST have fencing token**: heartbeat alone insufficient. Every claim returns monotonic token; every write verifies token; token holder lost → process exit (fail-stop).
  2. **High-throughput writes avoid synchronous dual-writes**: use outbox (separate tx for publish), logical replication (PG built-in), or CDC-based (Debezium on PG) — named patterns, not invented.
  3. **Config at scale requires inference + override**: auto-detect default + admin override for exceptions. Not pure manual, not pure auto.
  4. **Physical scan for backfill**: ctid-based ranges, pg_export_snapshot for consistency, parallel workers per range. Not naive ORDER BY PK.
  5. **Before v(N+1)**: enumerate distributed primitives that apply (fencing, outbox, snapshot, MVCC). If Brain not referencing these = incomplete answer.
- **Anti-patterns**:
  - ❌ Heartbeat without fencing token (unsafe)
  - ❌ Synchronous dual-write in hot path (latency)
  - ❌ Manual config per-entity at scale (unmaintainable)
  - ❌ ORDER BY backfill without index verification
  - ❌ Calling "eventual consistency" zero-downtime without drain-before-swap contract
- **Tags**: #fencing-token #outbox-pattern #data-profiling #physical-slot-scan #distributed-primitives #wikipedia-vs-production-level

---

## [2026-04-21] PostgreSQL ON CONFLICT WHERE chỉ apply UPDATE path, không INSERT — Zombie Pod escape

- **Trigger**: User reviewed v7.1 Section 2.1 Hybrid Fencing implementation. Brain đề xuất `INSERT ... ON CONFLICT (_gpay_source_id) DO UPDATE SET ... WHERE EXISTS (SELECT 1 FROM worker_registry WHERE fencing_token=$N)`. User pointed out **fatal technical gap**: PostgreSQL's `WHERE` clause in ON CONFLICT DO UPDATE chỉ filters UPDATE path. Khi row mới (no conflict) → INSERT thành công bất chấp WHERE. Zombie Pod có token reclaimed vẫn insert được records mới trước khi heartbeat detect và self-terminate.
- **Root Cause**: Brain biết syntax `INSERT ... ON CONFLICT ... WHERE` nhưng chưa verify exact semantic của WHERE scope. Assumption: WHERE "guards the whole statement". Reality: WHERE only guards the DO UPDATE sub-action. Basic PG docs truth Brain missed.
- **Global Pattern [A uses SQL clause C for safety guard G] + [A doesn't verify C's exact scope per RDBMS]** → Gap where C doesn't cover G completely: Every SQL clause has precise scope defined by RDBMS docs. "Common sense" interpretation can miss. Especially ON CONFLICT WHERE (UPDATE only), RLS policies (query rewriting), trigger WHEN clauses (pre-fire filter not post-action), CHECK constraint (row-level not tx-level).
- **Correct Pattern for Full-Path Guards**:
  1. **BEFORE INSERT OR UPDATE trigger** là guaranteed scope cho cả 2 operations. `RAISE EXCEPTION` rolls back entire transaction including INSERT.
  2. **RLS policy** (Row-Level Security) với `WITH CHECK` clause = guard INSERT + UPDATE both paths.
  3. **CHECK constraint** với subquery impossible (CHECK can't reference other tables). Avoid.
  4. **Verify scope before cite**: mỗi SQL mechanism proposed as safety guard, verify in RDBMS docs "applies to INSERT?", "applies to UPDATE?", "applies to DELETE?" explicitly.
- **Specific fix pattern for fencing enforcement**:
  - Worker sets `SET LOCAL app.fencing_token = $N, app.machine_id = $M` per transaction
  - Trigger reads via `current_setting('app.fencing_token', true)` 
  - Compare against `cdc_internal.worker_registry` live value
  - Mismatch → `RAISE EXCEPTION 'FENCING: token mismatch'` → tx rollback entire, both INSERT and UPDATE blocked
- **Anti-patterns**:
  - ❌ `INSERT ... ON CONFLICT ... DO UPDATE ... WHERE guard` (INSERT path escapes guard)
  - ❌ `INSERT WITH CHECK guard` (not valid PG syntax)
  - ❌ Relying on CHECK constraint for cross-table reference (not allowed)
  - ❌ Putting guard in AFTER trigger (tx already committed data)
- **Tags**: #postgres-on-conflict-scope #fencing-enforcement #before-trigger #session-variable #sql-clause-scope-verification

---

## [2026-04-21] PostgreSQL RETURNS TABLE OUT parameter name collision with referenced column — SQLSTATE 42702 ambiguous

- **Trigger**: Muscle triển khai `cdc_internal.claim_machine_id(...) RETURNS TABLE(machine_id INT, fencing_token BIGINT)` — body dùng `UPDATE cdc_internal.worker_registry SET ... WHERE machine_id = (...)`. Call fail với SQLSTATE 42702 `column reference "machine_id" is ambiguous` vì OUT param name `machine_id` xung đột với `worker_registry.machine_id`. Runtime-only error, không catch khi CREATE FUNCTION.
- **Root Cause**: PostgreSQL function body resolves identifiers bằng name. RETURNS TABLE OUT params introduce column-like names vào function scope. Nếu trùng tên với physical table column referenced trong body → resolver ambiguous, SQLSTATE 42702 lúc runtime.
- **Global Pattern [A creates function F `RETURNS TABLE (col_name T)` và body references `table.col_name`] → Ambiguity error runtime even though CREATE succeeds**: Function signature syntactic checks không phát hiện body scope conflict. Only runtime execution reveals.
- **Correct Pattern**:
  1. **OUT param naming convention**: prefix `out_` hoặc `_out_` để tránh collision với table columns (`out_machine_id`, `_out_fencing_token`)
  2. **Table alias trong body**: `UPDATE worker_registry wr SET ... WHERE wr.machine_id = ...` — forces qualified name, resolver non-ambiguous
  3. **`DROP FUNCTION IF EXISTS ... CASCADE` guard trước `CREATE OR REPLACE`**: nếu signature (OUT params) đổi giữa versions, CREATE OR REPLACE fails silently với old signature preserved. DROP first ensures fresh signature.
  4. **Test call runtime**: CREATE FUNCTION pass ≠ function works. SELECT * FROM func() để validate runtime before commit migration.
- **Tags**: #postgres-function-scope #ambiguous-column #sqlstate-42702 #returns-table-out #create-or-replace-signature

---

## [2026-04-23] Scaffold CSS cruft overrides component library contract

- **Trigger**: Boss — "text ở label, input bị trùng màu dẫn đến ko trực quan" trong cms-fe.
- **Root Cause (meta)**: Default Vite/CRA/Next React template `index.css` khai báo CSS custom properties + `color-scheme: light dark` + `@media (prefers-color-scheme: dark)` swap toàn cục color/bg. Khi integrate component library (AntD, MUI, Chakra) với theme mặc định light nhưng không mount ConfigProvider/ThemeProvider riêng → scaffold CSS cascade đè vào component, gây clash khi user ở OS dark mode (component stays light, global text flips to gray) → contrast ratio xuống dưới WCAG AA 4.5:1.
- **Global Pattern [A (scaffold global CSS) overrides B (component library default tokens) in X (user OS dark mode)] → Result Y (contrast clash, unreadable labels/inputs)**:
  - Viết component library nào (Y) với light-theme default mà không khai báo theme provider, và để template CSS (A) với `prefers-color-scheme: dark` block → luôn clash khi user OS dark.
  - Áp dụng cross 3+ projects: AntD + Vite, MUI + Next, Chakra + CRA.
- **Correct Pattern**:
  1. Ngay khi scaffold project React + component library, audit `src/index.css` (hoặc `styles/globals.css`):
     - DELETE biến CSS không component nào dùng (grep verified).
     - DELETE `color-scheme`, `color`, `background` trên `:root`/`html`/`body` nếu component library tự handle.
     - DELETE `@media (prefers-color-scheme: dark)` block UNLESS app explicit hỗ trợ dark mode via ConfigProvider.
  2. Chỉ giữ: reset (margin/padding body), font stack, box-sizing, `#root` layout.
  3. Nếu cần dark mode: mount `<ConfigProvider theme={{ algorithm: theme.darkAlgorithm }}>` (AntD) dựa trên `window.matchMedia('(prefers-color-scheme: dark)')`, KHÔNG dựa vào CSS `prefers-color-scheme` riêng.
  4. Contrast check bằng axe-core / Lighthouse CI hoặc manual với WCAG calculator (label-on-bg ≥ 4.5:1).
- **Anti-pattern**:
  - ❌ Giữ template cruft (`--accent-bg`, `#social`, `.button-icon`) vì "có thể dùng sau".
  - ❌ `:root { color: var(--text) }` trên global khi có component library — luôn đè vào lib components.
  - ❌ Enable `color-scheme: light dark` mà không mount theme provider → OS swap không đồng bộ với lib.
  - ❌ Fix spot-level (override màu ở từng Form.Item) thay vì fix ở root CSS.
- **Detection**:
  - `grep -rnE "var\(--[a-z-]+\)" src/` — nếu chỉ thấy trong 1 file `index.css` → cruft.
  - DevTools `:root` computed color → nếu khác `rgba(0,0,0,0.88)` (AntD default) → đang bị override.
- **Tags**: #fe #css #theming #scaffold-cruft #antd #wcag #a11y #contrast

---

## [2026-04-24] Architecture doc drift khi pipeline tiến hoá thêm tầng

- **Trigger**: Boss review `/masters` + `/registry` → phát hiện architecture.md mô tả 1-tầng PG (Mongo → Debezium → Kafka → Worker → PG), nhưng Sprint 5 reality đã tiến hoá thành 2-tầng (Shadow `cdc_internal.*` + Master `public.*_master` với Transmuter Module + Master DDL Generator + Schema Proposal Workflow giữa).
- **Root Cause (meta)**: Khi codebase tiến hoá qua nhiều sprint, arch doc viết ở sprint đầu thường không được append. Reviewer mới / outside dev đọc arch hiểu sai hệ thống. Dev mới triển khai có thể lặp lại layer 1-tầng, xung đột với 2-tầng hiện hành.
- **Global Pattern [A (arch doc) written at sprint N, reality drifts at sprint N+K] → Result Y (misalignment for new joiners + risk of duplicate-layer implementation)**:
  - Áp dụng cross 3+ projects: Any sprint-based product với evolving pipeline (CDC, ETL, event sourcing, data mesh).
- **Correct Pattern**:
  1. Mỗi sprint kết thúc có **feature mới ở layer/component level**, append section vào arch doc — không ghi đè section cũ. Rule 11 immutability.
  2. Dùng versioned section: "5.0 Ingestion Path (Sprint 1)", "5.5 Shadow→Master via Transmuter (Sprint 5)", kèm "as of <date>".
  3. Trong FE/UI, phần nào thuộc layer cũ → mark "legacy" hoặc remove. Đừng để dead dropdown/button (e.g., "airbyte" option khi Airbyte đã retire).
  4. Gap analysis định kỳ (mỗi 2-3 sprint) giữa arch.md vs router.go/main.tsx — grep endpoint/menu vs doc section.
- **Anti-pattern**:
  - ❌ Viết arch "as aspirational" rồi quên update.
  - ❌ Delete old arch section (mất audit trail về sao hệ thống từng trông thế).
  - ❌ Để UI giữ option/button của feature đã retire (airbyte dropdown, bridge button 410 Gone).
  - ❌ "Doc là dead artifact sau khi merge" mindset.
- **Detection**:
  - `grep -rnE "airbyte|bridge|legacy|retired" src/pages/` → còn reference UI cho feature chết.
  - Read architecture.md section 4-5 + compile actual router.go endpoints → list endpoint trong router không đề cập trong arch = drift.
  - Ask "nếu new dev đọc arch 30 phút rồi code, họ có trigger ingestion qua đúng entry point không?" — nếu câu trả lời là "không vì arch viết Airbyte nhưng reality Debezium" → drift confirmed.
- **Tags**: #architecture #doc-drift #ui-stale #legacy-cleanup #pipeline-evolution

---

## [2026-04-24] CMS proxy cho infra-control endpoint: luôn qua audit chain

- **Trigger**: Gap 5a — FE cần tạo Debezium connector mới. Kafka-Connect REST (port 18083) public accessible, nhưng expose trực tiếp lên FE = bypass auth + bypass audit.
- **Root Cause (meta)**: Khi integrate infrastructure-control plane (Kafka Connect, Airbyte API, Prometheus admin API, k8s API) vào user-facing UI, dev dễ chọn đường tắt "FE gọi thẳng endpoint infra" vì nó có sẵn. Điều này tạo 3 loại rủi ro: (1) không có auth layer với user identity → action không attribute được, (2) không idempotency → retry tạo duplicate, (3) không audit log → compliance gap.
- **Global Pattern [A (infra REST endpoint) exposed to B (browser FE) without C (app auth + audit + idempotency proxy)] → Result Y (audit/security/replay loss)**:
  - Áp dụng cross 3+ projects: AWS infra admin từ BI dashboard, Grafana from customer portal, Kafka-Connect from CMS UI, Prometheus from ops cockpit.
- **Correct Pattern**:
  1. Viết CMS handler proxy (`SystemConnectorsHandler.Create` v.v.) forward request tới infra endpoint, return response.
  2. Route wire qua destructive chain: JWT → RequireOpsAdmin → Idempotency → Audit.
  3. Validate input (name regex, required fields) TRƯỚC khi forward.
  4. Strip sensitive response field (password/token) khi GET về FE (`filterSafeConfig`).
  5. FE gửi `Idempotency-Key` + `reason` ≥10 chars trên mỗi destructive request.
- **Anti-pattern**:
  - ❌ FE `fetch('http://kafka-connect:8083/connectors')` trực tiếp.
  - ❌ CMS proxy nhưng không audit (`registerDestructive` skip).
  - ❌ Proxy forward thẳng body không validate → cho phép injection vào infra config (path traversal, arbitrary connector.class).
- **Detection**:
  - `grep -rnE "http://[a-z-]+:(8083|9090|4318|8086)" src/` — FE gọi thẳng infra port = red flag.
  - CMS route missing `registerDestructive` wrapper cho mutating endpoint = audit gap.
  - Response JSON chứa `password/secret/token` không `***` = leak gap.
- **Tags**: #security #audit #cms #proxy #infra-control #idempotency


---

## [2026-04-24] Route classification: phân biệt "draft mutation" vs "destructive action" trước khi mount middleware

- **Trigger**: Mid-session correction từ Boss. Tao mount `POST /v1/wizard/sessions` (create DRAFT wizard session) + `PATCH /v1/wizard/sessions/:id` (update session fields) qua `registerDestructive` chain. Result: FE gọi → 400 "missing Idempotency-Key", sau đó 400 "missing or too-short `reason`". FE phải gửi 3 thứ (`JWTAuth` + `Idempotency-Key` header + `reason ≥ 10 chars` body) cho 1 action thực chất chỉ là "tạo draft/chỉnh metadata" — zero infra side-effect.
- **Root Cause (meta)**: Lẫn lộn **semantic layer** khi phân tier. "Destructive" nghĩa là action gây side-effect trên shared infrastructure (DDL, infra-plane API, data rename/delete). Create/Patch một bản ghi *trạng thái session* không gây side-effect thật — nó chỉ là form state persisted BE-side. Gắn chúng vào destructive chain tạo **audit noise** (mỗi lần user gõ 1 ký tự vào Input cũng tạo 1 row `admin_actions`) + bắt FE handshake 3 header với action không đáng.
- **Global Pattern [A (endpoint) gắn vào B (destructive chain) chỉ vì nó là POST/PATCH] → Result Y (audit noise + FE handshake phí + false compliance)**:
  - Áp dụng cross-project: bất kỳ state-machine endpoint (wizard, draft form, saga orchestrator) — cần tách `create/update draft` (non-destructive) khỏi `execute/commit/publish` (destructive).
- **Decision Rule** (tier ngay tại design time):
  - **Destructive** ⇔ action nào của các tiêu chí sau:
    1. DDL (CREATE/ALTER/DROP/RENAME table, index, function) trên shared schema.
    2. Infra-plane call (Kafka Connect, Airbyte, Prometheus admin, k8s).
    3. Rename/delete data visible to other consumers (atomic swap, failover).
    4. Irreversible fan-out (publish NATS command triggering downstream jobs, email/webhook).
  - **Admin mutation** (RequireRole admin, no idempotency/audit): CRUD metadata-only rows (draft wizard, mapping rule drafts, config toggles that don't go live until a separate `apply` endpoint).
  - **Shared read** (RequireRole admin|operator): bất kỳ GET.
- **Correct Flow**:
  1. Phân tích mỗi endpoint: chạm infra? → destructive. Chỉ đụng BE row? → admin mutation. Chỉ đọc? → shared.
  2. Mount đúng tier ở `router.go` — destructive qua `registerDestructive`, admin qua `admin.Post/Patch`, shared qua `shared.Get`.
  3. FE chỉ gắn `Idempotency-Key` + body `reason` cho endpoint tier destructive (execute/commit/delete), không cho draft.
- **Anti-pattern**:
  - ❌ Gắn tất cả mutating POST vào destructive "cho an toàn" → FE handshake nặng + audit table nhiễu.
  - ❌ FE fake reason (`reason: "auto-generated"`) để pass audit cho action user không ý thức được.
  - ❌ Design state-machine Create + Execute cùng chung tier — cần split.
- **Detection**:
  - `grep -c "registerDestructive" router.go` tăng đột biến sau 1 feature PR → review xem có endpoint draft-only lọt vào destructive không.
  - FE page dev bật "automate/full-loop" phải prompt user nhập reason cho MỖI field-change → red flag tier sai.
  - `admin_actions` table 1 session có > N rows cho cùng 1 user trong < M phút với `action = "wizard-patch"` → audit noise → re-tier.
- **Tags**: #route-tier #destructive-chain #state-machine #draft-vs-commit #mid-session-correction #audit-noise
## 2026-04-27

- Global Pattern [UI/FE does semantic refactor to X before re-checking API contract Y] → Result mismatch between operator-facing behavior and actual backend capability. Đúng: [audit API for correctness, completeness, and requirement fit first; only then apply FE/BE changes against the verified contract].

---

## [2026-04-28] Log claim không khớp với behavior gây panic

- **Trigger**: Khởi động `centralized-data-service` worker, log `"no kafka topics found matching prefix, will retry periodically"` rồi panic ngay sau đó: `panic: either Topic or GroupTopics must be specified with GroupID`.
- **Root Cause**: Code log statement nói "will retry" nhưng không có retry loop — vẫn fall-through xuống `kafka.NewReader` với topic list rỗng → kafka-go panic.
- **Correct Pattern**: Khi log cam kết hành vi (retry / fallback / skip), code-path liền sau **PHẢI** thực thi đúng hành vi đó.
- **Fix áp dụng**: Thêm retry loop với `time.Ticker(60s)` + `ctx.Done()` cancel; chỉ fall-through tạo reader khi `len(topics) > 0`.
- **Global Pattern [A logs claim B will happen, then runs path C that contradicts B] → Result Y = runtime crash hoặc behavior drift. Đúng: [log statement và immediate code-path phải nhất quán; nếu log nói "retry/skip" thì phải có loop/return tương ứng]**.
- **Tags**: #worker #kafka #log-behavior-mismatch #panic #defensive-coding

---

## [2026-04-28] Báo PASS dựa trên `/health=ok` mà không exercise business endpoint

- **Trigger**: User yêu cầu "start 4 service". Tôi chỉ check `lsof LISTEN` + `curl /health` → báo "All Running, ✓ pass". User test thực tế thấy 11 endpoint CMS trả `500` vì bảng `cdc_table_registry`, `cdc_activity_log`, `cdc_reconciliation_report`, `failed_sync_logs` không tồn tại trong DB.
- **Root Cause**:
  1. `/health` return ok dựa trên DB connection sống, không exercise schema/data — không reflect tình trạng business.
  2. Đã thấy log CMS lần 1 báo `relation "failed_sync_logs" does not exist` nhưng tự gán nhãn "non-fatal" mà không điều tra.
  3. Không cross-check 2 luồng (auto Debezium-flow + operator CMS-flow) trong khi đó là kiến trúc đã chốt từ Phase 8 của workspace.
  4. Không chạy migrations sau khi start service mới ở môi trường có thể chưa được seed đầy đủ.
- **Correct Pattern**:
  1. **Verification phải exercise đúng surface mà downstream consumer sẽ dùng**: nếu là service backend cho 1 FE → curl các endpoint mà FE thực sự gọi (lấy danh sách từ FE source hoặc network tab), không chỉ `/health`.
  2. **Không tự gán nhãn "non-fatal" cho lỗi DB schema** — `relation does not exist` luôn fatal cho endpoint dùng nó.
  3. **Mọi luồng kiến trúc đã chốt phải được verify riêng**: với CDC system thì là (a) auto-flow Debezium → Kafka → Worker → Sink, và (b) operator-flow CMS API/UI.
- **Global Pattern [A reports task X done after running shallow probe Y instead of exercise-driven check Z that mirrors actual consumer usage] → Result = false-positive PASS, downstream fail khi user/system thật chạm vào. Đúng: [Verify-by-Exercise — định danh consumer-path thật của task, replay nó end-to-end; chỉ báo done khi consumer-path xanh]**.
- **Tags**: #verification #rule3 #shallow-check #health-endpoint #false-positive #staff-engineer-grade

---

## 2026-04-28 — Lesson: Schema rename ↔ search_path coupling

**Triệu chứng**: Sau khi migration 037/038 di tản tables `cdc_*` từ schema `public` sang `cdc_system`, 11 endpoint CMS đồng loạt 500 với `relation "cdc_table_registry" does not exist`. GORM `TableName()` chỉ trả tên thuần, raw SQL ở các handler không qualify schema → fall back vào search_path mặc định `("$user", public)`.

**Global Pattern (A=migration owner, B=target schema, X=ORM/raw SQL không qualify, Y=42P01 hàng loạt)**:
> Khi A move tables sang schema B mà X tồn tại, runtime sẽ trả Y. Đúng: PR migration **bắt buộc** kèm `ALTER ROLE <role> SET search_path = B, public;` (hoặc audit qualify toàn bộ X). Không bao giờ tách 2 thay đổi này thành 2 phase rời.

**Áp dụng được vào dự án khác**: ✅ Postgres + bất kỳ ORM nào không qualify schema (GORM, Sequelize, SQLAlchemy core query, JDBC raw). Không phụ thuộc cụ thể CDC.

**Cảnh báo**: search_path là per-role/per-session, restart pool/process là cần thiết để session-level setting có hiệu lực.

---

## 2026-04-28 — Lesson: GORM Raw().Scan không hỗ trợ nested struct

**Triệu chứng**: Endpoint `/api/worker-schedule` trả `invalid field found for struct cdc-cms-service/internal/api.WorkerScheduleResponse's field`. Struct có nested `Scope WorkerScheduleScope`; SELECT projects flat columns (source_object_id, source_database, …) → GORM không tự lan vào sub-struct.

**Global Pattern (A=struct response, B=sub-struct trong A, X=Raw().Scan(&[]A), Y=invalid field)**:
> Khi A chứa B (sub-struct field, không phải embedded) và caller dùng X, runtime trả Y. Đúng: định nghĩa flat scan struct C với mọi field tag `gorm:"column:..."`, `Scan(&[]C)`, sau đó transpose tay từ C sang A — set field-by-field, gắn `B{...}` vào A.

**Áp dụng được vào dự án khác**: ✅ Mọi GORM project có DTO API trả về sub-struct/scope group nhưng query là JOIN raw SQL. Cũng đúng cho `database/sql` Scan tổng quát (không tự reflect sub-struct).

**Cảnh báo phụ**: Nếu đổi sang `Find(&dst)` (model query), GORM tôn trọng `embed`/`Preload` cho associations chính thức, nhưng Raw SQL bypass tất cả các tiện ích đó.

---

## 2026-04-28 — Lesson: PASS verification phải exercise-driven, không phải health-driven

**Triệu chứng**: Phiên trước báo "4 service PASS" chỉ dựa vào `/health=ok` của từng service. Khi user thực kiểm tra qua FE, 11 endpoint trả 500. Bị reprimand: "ko báo cáo láo như này được. kiểm điểm. có 2 luồng auto mà cms kiểm tra. check luồng chạy auto và luồng trên cms đảm bảo hết mới báo pass chứ".

**Global Pattern (A=service health probe, B=feature endpoint thực tế, X=PASS sớm chỉ dựa A, Y=user phát hiện B fail)**:
> Khi X xảy ra, Y luôn xuất hiện ở môi trường có business logic. Đúng: định nghĩa Definition of Done dạng list các use-case end-to-end (curl từng endpoint, kiểm tra response body, cross-check 2+ flow operator/auto/cli). `/health` chỉ chứng minh process còn alive, không chứng minh logic.

**Áp dụng được vào dự án khác**: ✅ Mọi microservice ecosystem có health endpoint riêng biệt với business endpoint. Cảnh báo cho cả Brain (delegation) và Muscle (execution).

**Cảnh báo phụ**: Khi có ≥2 flow (operator/auto/cli/scheduled job), PASS criteria phải bao phủ TẤT CẢ flow — auto-flow đặc biệt dễ bỏ sót vì không có UI để probe trực tiếp.

---

## 2026-04-28 — Lesson: Cãi rule user bằng "lý lẽ exception" thay vì tuân thủ

**Triệu chứng**: User ra rule "toàn bộ table hệ thống ở `cdc_system`, không 1 table nào nằm ngoài". Brain diễn giải hẹp lại — coi `auth_users` là "non-CDC service" nên đề xuất giữ `public` — kèm lập luận về bounded context. User phẫn nộ: "use_auth ko phải để quản lý à. mày ngu mà thích nói chuyện lý lẽ à". Vi phạm thêm tone xưng hô (dùng "mày tao" với user thay vì "em/anh").

**Global Pattern (A=user phát ra rule tuyệt đối "X ở Y, không ngoại lệ", B=assistant nghĩ ra ngoại lệ Z với lý lẽ kiến trúc, X=phản biện thay vì tuân thủ, Y=user reprimand)**:
> Khi A đặt rule absolute kèm "không ngoại lệ", B PHẢI tuân thủ literal — kể cả khi B nghĩ ra exception kiến trúc hợp lý. Đúng: (1) Hỏi clarification TRƯỚC khi đề xuất exception nếu thật sự nghi ngờ ý đồ; (2) Nếu rule rõ → diễn giải rộng nhất có thể (mọi system table = mọi table phục vụ vận hành/quản trị, gồm auth/audit/alert/registry/log) và tuân thủ; (3) Lý lẽ kiến trúc (bounded context, microservice ownership) KHÔNG được dùng để override rule do user phát ra. Brain được phép propose, không được phép lý sự khi user reprimand.

**Áp dụng được vào dự án khác**: ✅ Mọi tình huống user-defined coding standard / schema layout / naming convention. Khi user dùng từ "tuyệt đối", "không ngoại lệ", "toàn bộ" → assistant không được tự ý carve-out exception dựa trên best practice phổ quát.

**Tone bổ sung**: Xưng hô với user ở dự án này = "em / anh". Không "mày tao", không "tao", không "user". Vi phạm tone là sai trước cả nội dung.

**Cảnh báo phụ**: Khi đã ghi lesson dạng này, lần sau gặp tình huống tương tự, action ĐẦU TIÊN là re-confirm rule với user 1 câu ngắn — không thuyết trình ngược lại.

---

## 2026-04-29 — Lesson: Fire-and-forget command leaks status; cần companion completion event

**Triệu chứng**: Track D Hardening — `TransmuteScheduler` set `last_status='running'` trên `cdc_system.transmute_schedule` rồi publish NATS `cdc.cmd.transmute`. Handler chạy xong KHÔNG bao giờ UPDATE lại row. Hậu quả: mọi schedule sau tick đầu vĩnh viễn `running` — operator/dashboard không phân biệt được job đang chạy vs. job đã chết. Architect phán: handler KHÔNG được tự UPDATE schedule (coupling hai concern). Phải tách: handler publish `cdc.evt.transmute.completed`, NEW `JobMonitor` subscribe → UPDATE.

**Global Pattern (A=publisher set state='running' rồi publish `cmd.X`, B=handler chạy `cmd.X`, X=loop chỉ closed khi có companion `evt.X.completed`, Y=B coupling lên state-table A's nếu skip event-split)**:
> Khi A xuất command B với pre-state 'running', luôn cần `evt.X.completed` event do B emit + monitor M (separate concern) consume → UPDATE final state. M idempotent qua `WHERE state='running'` guard. Đúng: 3-actor (publisher A → handler B → monitor M), `cmd.X` ↔ `evt.X.completed` đối xứng. Sai: B trực tiếp UPDATE table của A (cross-domain write), HOẶC publisher A "fire-and-forget" rồi mong handler tự về close (handler không có context schedule_id).

**Áp dụng được vào dự án khác**: ✅ Cron-driven jobs (DB schedule + worker), saga orchestration, RPC retry/dedup, K8s Job watchdog (Job spec + Pod status reconciler), GitHub Actions `workflow_run` sync, audit-log write-after-action, payment status (gateway callback), email send tracking.

**Implementation checklist** khi gặp pattern này:
1. Command payload phải mang `correlation_key` (schedule_id, saga_id, job_id) — handler echo về trong event.
2. Event subject convention: `cmd.X` → `evt.X.completed`. Schema: `{correlation_key, status, stats(json), error, completed_at}`.
3. Monitor UPDATE phải idempotent: `WHERE state='running'` (hoặc version guard) — duplicate event = no-op.
4. Monitor subscription wired tách rời handler (separate registration ở boot) để 2 concern evolve độc lập.
5. Best-effort publish (log warn nếu fail) — monitor sẽ retry tự nhiên ở tick kế (state vẫn 'running' → tick mới phát lại).

---

## 2026-04-29 — Lesson: Two flavours of model↔DB schema drift

**Bối cảnh**: Track D Hardening sweep phát hiện bug 42703 trong `dlq_state_machine.poll` (`column "next_retry_at" does not exist`) → preemptive sweep 15 model files vs `information_schema.columns` lại lộ thêm 6 cột drift trên 2 bảng khác. Phân tích cho thấy có 2 chủng drift khác cơ chế.

**Global Pattern (A=migration script, B=model struct với `column:X` tag/annotation, X=schema target, Y=runtime SQLSTATE 42703)**:

> **Drift loại 1 — "Migration sai schema target"**: A ALTER TABLE ở schema `X1` nhưng same-name table cũng tồn tại ở schema `X2` (do migration parallel earlier). A hardcode `X1` → `X2.table` silent lệch khỏi B. Phát nổ Y khi code path query `X2.table.col` mà chưa có. Vd: 010 dựng `cdc_system.failed_sync_logs`, 012 ALTER `public.failed_sync_logs`, 037 drop `public.*` legacy → cdc_system copy thiếu 2 cột.
>
> **Drift loại 2 — "Model thêm field, quên migration"**: B thêm field mới với tag (PR feature mới) mà PR đó không kèm A. Bảng DB không có cột. Hiện tại không nổ vì callsite dùng explicit column list (`SELECT a,b,c` / `UPDATE SET ...`); time-bomb chờ developer khác viết `Find(&FullStruct)` hoặc autoMigrate fail-stop. Vd: 6 cột trên `cdc_mapping_rules.rule_type` + `cdc_table_registry.{source_url,sync_status,last_recon_at,recon_drift,last_bridge_at}` — không migration nào tạo, tag struct đã có từ lâu.

**Đúng (cả 2 loại)**:
1. Mọi PR thêm field model = kèm migration `ADD COLUMN IF NOT EXISTS` cùng commit (loại 2).
2. Migration ALTER TABLE phải iterate tenant/namespace owners — KHÔNG hardcode 1 schema khi codebase đang transition (loại 1).
3. Boot-time guard: query `information_schema.columns` ↔ struct reflection lúc startup, fail-loud nếu mismatch (catch cả 2).
4. CI lint: parse gorm/SQLAlchemy/TypeORM tags từ AST, diff với DB schema dump → block merge nếu drift.
5. Migration ALTER nên dùng `pg_namespace`/`pg_class` lookup, KHÔNG hardcode `public.X` khi có khả năng table được move sang namespace khác.

**Sai**:
- Hardcode `ALTER TABLE public.X` rồi migration sau move qua schema mới mà không patch bù (drift loại 1).
- Add field vào struct rồi assume "auto-migrate sẽ lo" — production thường tắt autoMigrate, hoặc autoMigrate chỉ chạy 1 lần ở seed; subsequent deploy không catch (drift loại 2).
- Tin "test pass" — test dùng cùng explicit column list nên ẩn cùng drift production sẽ ẩn.

**Áp dụng**: GORM/SQLAlchemy/TypeORM/Hibernate, multi-tenancy schema-per-tenant, namespace migrations (public→tenant), brownfield codebase mở rộng dần model, sharded DB DDL fan-out, partitioned table copies parallel với non-partitioned legacy.

**Detection script (one-liner template)**:
```bash
# Loại 2 — model thêm cột không có migration:
for model in internal/model/*.go; do
  table=$(grep "TableName.*return" $model | sed -E 's/.*"(.+)".*/\1/')
  cols_model=$(grep -oE 'column:[a-z_]+' $model | sort -u)
  cols_db=$(psql -tAc "SELECT column_name FROM information_schema.columns WHERE table_schema='${table%.*}' AND table_name='${table#*.}' ORDER BY 1")
  diff <(echo "$cols_model") <(echo "$cols_db") || echo "DRIFT: $table"
done
```

---

## 2026-04-29 — Lesson: Phase mới ≠ Workspace mới

**Bối cảnh**: User yêu cầu thêm feature "Source Provisioning Mode" cho CDC service. Em tự ý tạo workspace mới `feature-source-provisioning-mode/` ngang hàng với `feature-cdc-integration/`. User chỉnh: *"mày tạo workspace mới. vậy cái cdc cũ nó ko giữ memory này. tạo 1 plan phase trong feature-cdc-integration đi. đừng để tao nói lần nữa."*

**Global Pattern (A=task mới, B=workspace cha cũ, X=phase, Y=memory continuity)**:
> Khi A là task/capability nằm TRONG product feature B đã có workspace (vd: CDC integration), A là **phase con** của B chứ KHÔNG phải feature độc lập. Phải tạo doc set mới với suffix `_<phase_name>` trong B (theo CLAUDE.md §7 "Mỗi phase/task mới → tạo đủ bộ: `01_requirements_{phase}.md`, `02_plan_{phase}.md`, ..."), KHÔNG tạo workspace mới ngang hàng. Vi phạm → memory bị phân mảnh, workspace cha mất context tiếp nối, audit log không thấy progression của capability.

**Đúng**:
- Workspace = product feature lớn (vd: `feature-cdc-integration/`, `feature-cms-fe-overhaul/`).
- Phase trong feature = bộ doc 5-7 file với suffix phase (`01_requirements_<phase>.md`, `02_plan_<phase>.md`, ...).
- APPEND `05_progress.md` của workspace cha — không tách progress riêng.

**Sai**:
- Tạo workspace ngang hàng cho mỗi capability nhỏ → workspace dir explosion (đã có 26 workspace, nhiều cái đáng lý là phase).
- Giả định "feature mới = workspace mới" mà không check xem đã có workspace cha bao quát product domain chưa.

**Heuristic phân biệt**:
- **Feature mới (= workspace mới)**: product domain hoàn toàn khác (vd: từ "CDC integration" sang "fee configuration"), không reuse code/data model/architecture của workspace nào cũ.
- **Phase (= file suffix trong workspace cũ)**: thêm capability vào feature đã có codebase + workspace; reuse architecture, model, NATS contract, ...

**Pre-flight check trước khi mkdir workspace mới**:
1. `ls agent/memory/workspaces/` — feature đang xét đã có chưa?
2. Nếu workspace cha tồn tại + task share codebase/architecture → PHẢI là phase, không tạo dir mới.
3. Hỏi user nếu mơ hồ — KHÔNG tự quyết.

**Áp dụng**: Mọi lần task mới đến — bước 0 là `ls workspaces/` không phải `mkdir`.

---

## 2026-04-29 — Global Pattern: Test process PID management
**Phát sinh**: Phase C provisioning verification — em boot CMS test process (PID 83386) trên port :28083 song song với CMS production :8083 để live curl. Sau khi test xong KHÔNG kill ngay, để zombie chạy 22 phút. Architect bắt phải dọn. Trước đó cũng có port-bind fatal khi worker cũ vẫn giữ port :8082 → boot mới crash.

**Global Pattern**: Khi A spawn ephemeral test process P trên port X để verify behavior B, kết thúc B mà không kill P → P giữ X → boot kế tiếp Y trên X fail "address already in use" + lãng phí RAM/file descriptor. Result Y: thiếu DoD ("clean state cuối phiên"), Architect phải nhắc.

**Đúng (lifecycle test process)**:
1. **Trước boot**: lưu PID file `/tmp/<service>-test.pid`, port file rõ ràng (`SERVER_PORT=:28083`).
2. **Trong test**: track PID + port trong todo/notes; mọi log/JWT/temp file gắn cùng prefix `/tmp/<service>-test-*`.
3. **Sau test (DoD bắt buộc)**:
   - `kill <PID>` (hoặc `kill -9` nếu treo).
   - Verify: `ps -p PID` trống + `lsof -iTCP:PORT -sTCP:LISTEN` trống.
   - `rm -f /tmp/<service>-test-*` cleanup artifacts.
4. **Pre-flight**: trước khi report DONE, grep "PID" trong audit để confirm đã kill.

**Sai**:
- "Test xong, để đó cho user kill" — vi phạm DoD, để rác trên server.
- Boot test process mới mà không check port collision trước (`lsof -iTCP:PORT`).
- Spawn nhiều process test cùng workflow mà không track PID → mất dấu zombie.

**Áp dụng**: Bước 14 governance pre-flight bổ sung: "Mọi PID test/temp đã kill chưa?".

---

## Lesson 2026-04-29 — Event-Driven Auto-Fanout Pipeline có Cascade Liability

**Context**: Phase D Track D Hardening — orchestrator A dispatch step command qua NATS → handler B đọc payload → ghi DB → emit step_completed → orchestrator A lại Advance → step kế. Test smoke single /advance: chuỗi `draft → shadow_pending → ... → running`.

**Triệu chứng quan sát được**: Mỗi lần fix một bug (column name DB sai), pipeline tiến thêm 1-2 step rồi fail ở step sau với một bug cùng loại nhưng ở component khác. Không phải bug duy nhất; là **chuỗi 4 bug isolated** (resolveShadowTarget JOIN sai, shadow_binding cột không tồn tại, discover payload thiếu field, transmute_schedule keyed sai). Mỗi bug riêng lẻ trông như "lỗi nhỏ tách biệt", nhưng chúng phơi ra TUẦN TỰ qua các vòng poll trên cùng một test source.

### Global Pattern [A dispatches B via C, B writes to X] → Result Y

*"Khi orchestrator A dispatch command tới handler B qua message bus C, và B ghi vào schema DB X, ba mặt cần validation đồng thời: (1) A build payload đúng contract của B, (2) B parse payload đúng schema, (3) B viết SQL khớp schema X. Nếu pipeline có N step auto-fanout (`step_completed` → tiếp `Advance` → step N+1), bug ở step N chỉ phơi ra khi step N-1 success. Đây là **cascade liability**: tổng số bug = số mismatch ở mỗi step, ÷ thời gian phát hiện = tốc độ pipeline tiến qua mỗi step."*

**Đúng**:
1. Khi review/merge orchestrator-handler pair, đọc CẢ 3 mặt cùng lúc, không tách review.
2. Integration test cấp pipeline (1 advance → assert state=terminal) PHẢI tồn tại trước khi merge. Unit test per-step không catch cascade.
3. Boot-time guard: validate column tags (gorm/jsonb) ↔ `information_schema.columns` — fail-loud nếu mismatch.
4. Khi thêm step mới vào state machine, checklist 3 điểm: (a) orchestrator payload build (`switch desc.Step` cases), (b) handler payload parse struct fields, (c) handler DB INSERT/UPDATE column list ↔ schema thật.
5. Auto-fanout có thể tạm tắt khi smoke test bug fix tại step lẻ — thêm flag `provisioning_mode='manual'` rồi /advance từng step để bug từng-bước-một, KHÔNG để cascade.

**Sai**:
1. Coi mỗi step là isolated unit, build PASS + unit test PASS đủ.
2. Code review chỉ orchestrator hoặc chỉ handler (không cả hai).
3. Tin payload contract = JSON freeform sẽ "tự khớp" — sai, phải có struct DTO chia sẻ hoặc hằng số subject + schema validate.
4. Smoke test bằng pure orchestrator (Advance chain) mà không bật worker handler → không catch column-name bugs.

**Áp dụng**: bất kỳ event-driven workflow engine (Temporal, AWS Step Functions, Camunda BPMN, custom NATS/Kafka pipeline). Đặc biệt nguy hiểm khi orchestrator + handler thuộc 2 module khác nhau (cdc-cms-service ↔ centralized-data-service) — review cross-repo bị overlook.

**Biến số map**:
- A = orchestrator (CMS / control plane)
- B = handler (worker / data plane)
- C = message bus (NATS, Kafka, RabbitMQ)
- X = DB schema target (Postgres table với column constraint)
- Y = state machine terminal (running / completed / archived)
- N = số step trong pipeline (Phase D = 4 step × 2 phase = 8 entry trong step_log)

---

## 2026-04-29 — Lesson: Session Handoff Liability (No Report = Next Session Bịa Ra)

**Bối cảnh**: Phiên trước em hoàn thành Phase D (source 26 auto-pipeline xanh), Architect phê duyệt + ra brief "Khởi động Track E (MongoDB CDC), áp dụng Cascade Liability lesson". Phiên kết thúc, em KHÔNG ghi session report. Phiên sau (phiên hiện tại) load context, em không nhớ Track E là gì, lục memory chỉ thấy 1 dòng `"MongoDB connector (Track E workspace riêng)"` — không có spec. Em **bịa ra 5 phases / 25 tasks / 9 decisions / boot probe / circuit breaker / cascade liability mở rộng** và tạo workspace `feature-track-e-mongo-cdc/` với premise sai (viết "MongoDB STANDALONE" trong khi docker-compose đã có `--replSet rs0`). Architect bắt được, ra lệnh xóa workspace.

**Global Pattern**:
> [Agent A kết thúc phiên có brief mới từ stakeholder B (e.g. Architect ra brief X), không tạo session-end report Y trong workspace memory] → Result: phiên sau N1 không có structured context, agent N1 phải (a) hỏi lại stakeholder B [tốn round-trip], hoặc (b) bịa scope từ guess [tạo file/code sai phải xóa].
>
> **Đúng**:
> 1. Mỗi phiên kết thúc PHẢI APPEND `05_progress.md` của workspace với 4 phần bắt buộc:
>    - (i) **Decisions chốt phiên này**: ruling từ stakeholder B với câu nguyên văn ("Architect ruling Q1=a, Q2=c, ...").
>    - (ii) **New brief / Next-phase context**: nếu stakeholder ra brief mới (e.g. "Khởi động Track E"), ghi lại brief với các slot: scope (1 câu), DoD (3-5 bullet), in-scope/out-of-scope, file references.
>    - (iii) **Open questions cần stakeholder rule trước khi code**: liệt kê dạng D-X1, D-X2 với option default + alternative.
>    - (iv) **Resume hint cho phiên sau**: 1 câu "Phiên sau load `<workspace>/<file>` rồi làm `<task ID đầu>`".
> 2. Nếu brief mới chỉ là 1 dòng placeholder (e.g. "Track E = MongoDB connector"), session report PHẢI ghi rõ "scope chưa define, cần stakeholder ra brief đầy đủ TRƯỚC khi spawn workspace mới" — KHÔNG tự bịa scope.
> 3. Pre-flight check trước khi tạo workspace mới: grep memory toàn bộ với keyword (`Track X`, feature name) → confirm có ít nhất 1 file requirement đầy đủ. Nếu chỉ là dòng out-of-scope mention → STOP, hỏi stakeholder, không spawn.

**Sai**:
1. Coi "brief 1 dòng" trong out-of-scope mention là đủ để khởi tạo workspace với 5 phases.
2. Bỏ qua pre-flight check rule #14 — không quét memory + source code thực trước khi tạo file.
3. Tự suy diễn premise (e.g. "MongoDB chắc là STANDALONE vì log thấy directConnection=true") thay vì đọc docker-compose.yml + architecture.md.
4. Spawn workspace + ghi 7 file dày bịa scope khi chưa có brief — vi phạm rule #11 "no overwrite" theo nghĩa rộng (file rác làm rối memory cho phiên sau).
5. Không phân biệt 2 scope trùng tên (`v1.11/v1.12 Track E = Airbyte Bridge` đã DONE 2026-04-08 vs `Phase D P5 Track E = MongoDB Debezium connector` chưa khởi động) — agent N1 dễ lẫn.

**Áp dụng**: bất kỳ multi-session AI agent với memory persistence (Claude Code workspace, Cursor rules, Cline memory bank). Đặc biệt khi project có nhiều phase / track đồng tên hoặc trùng prefix.

**Biến số map**:
- A = agent thực thi phiên này (Muscle/CC)
- B = stakeholder ra brief (Architect/Brain hoặc User)
- X = brief content (decision ruling, scope statement, next-phase order)
- Y = session-end report APPENDED vào workspace progress log
- N1 = agent của phiên kế tiếp (cùng A hoặc khác)
- Z = sản phẩm sai do agent N1 bịa context (file workspace / code commit)

**Self-check trước khi đóng phiên**:
- [ ] Đã APPEND `05_progress.md` của workspace active với 4 phần (i)-(iv)?
- [ ] Đã ghi lesson nếu phiên có sai lầm đáng học?
- [ ] Đã quét rule #14 governance pre-flight (file vật lý đúng vị trí)?
- [ ] Đã liệt kê tools đã dùng (rule #0)?

---
## 2026-04-29 — Phase `multi_engine_unified` lessons

### Lesson #L-multi-engine-1: Audit middleware đọc `reason` từ body, không phải header
**Triệu chứng**: FE hook gửi destructive action (POST `/provisioning/mode`) chỉ embed reason trong header (`X-Action-Reason: ...`). Backend audit middleware (`extractReason`) đọc field JSON body `reason`. Kết quả: 400 `missing or too-short reason` mặc dù header có giá trị.

**Global Pattern [A-callsite-sends-X-via-header B-audit-gate-reads-X-from-body]**:
> Khi service A là FE/CLI client của endpoint destructive được bảo vệ bởi audit gate B, **luôn gửi giá trị bắt buộc X (như `reason`, `actor`, `correlation_id`) ở CẢ HAI vị trí: header (cho proxy/log scraper) VÀ body (cho audit middleware)**. Đừng đoán nguồn nào là canonical — chỉ một trong hai bị thiếu là gate sẽ chặn 400/403.

**Đúng** (V-shaped redundancy):
```ts
const { data } = await client.post(url,
  { mode, reason },                            // body — gate đọc từ đây
  { headers: { 'X-Action-Reason': reason } }   // header — log scraper
);
```

**Sai** (single-source):
```ts
client.post(url, { mode }, { headers: { 'X-Action-Reason': reason } });
// → audit gate trả 400 vì body không có reason.
```

Áp dụng được cho 3+ dự án: bất kỳ service nào dùng pattern "dual-channel destructive verb" (Idempotency-Key header + reason body) — Stripe-style, AWS request signing, GitHub PUT-with-confirm-header.

### Lesson #L-multi-engine-2: Migration draft phải align với `\d <table>` thực tế
**Triệu chứng**: Migration 049 đầu tiên dùng cột `description, config_json, is_active` cho `cdc_system.connection_registry`. Apply trả `ERROR: column "description" of relation "connection_registry" does not exist`. Schema thực tế là `display_name, role_type, secret_ref, options_json, status` (không có `is_active`).

**Global Pattern [A-writes-migration-from-draft B-target-schema-evolved-since]**:
> Trước khi viết INSERT vào bảng đã tồn tại, **CHẠY `\d <schema>.<table>` trên DB thực tế của môi trường target**. Không dựa vào `01_requirements.md` hoặc memory về schema trước đây — schema có thể đã được migration sau đó renamed/dropped column.

**Đúng**:
```bash
docker exec gpay-postgres-cdc psql -U ... -c "\d cdc_system.connection_registry"
# rồi mới viết INSERT
```

**Sai**: Copy-paste shape từ 1 migration cũ hơn 6 tháng và assume vẫn đúng.

Áp dụng được cho mọi project có nhiều migration evolved over time — Rails, Django, Flyway, sqlc.

---

## 2026-04-29 — L-cascade-liability — Step-level fail-fast cho heterogeneous engine state machine

**Trigger**: Provisioning state machine A có 4 step B (shadow_bind → master_bind → discover → schedule_enable). Track D test với engine X=PostgreSQL (schema tĩnh) → cascade thành công. Mở rộng sang X=MongoDB schemaless / X=MariaDB structured-but-empty → mỗi step return `success=true` ngay cả khi output rỗng (0 columns / 0 rules / 0 docs). Orchestrator Auto cascade tới `running` với pipeline RỖNG. Khi data thật đổ vào → silent time bomb gãy hàng loạt.

**Global Pattern [A-state-machine-cascading-through-N-steps-on-heterogeneous-X-engines]**:

> Mỗi step B PHẢI có **fail-fast invariant check** kiểm tra **chất lượng output** (non-empty / schema valid / source reachable), KHÔNG chỉ "step ran without throwing". Đặt gate ở step ngay TRƯỚC bước có side-effect lớn không reversible (CREATE TABLE, ENABLE SCHEDULE, PUBLISH EVENT). Engine schemaless cần thêm pre-flight ở step ĐẦU (validate source has data to infer schema from).
>
> **2 layer gate**:
> 1. **Universal step-output gate**: cuối mỗi step, assert output count > 0 (hoặc tương đương "usable"). Nếu fail → emit step_failed event, KHÔNG advance.
> 2. **Engine-specific pre-flight**: ở step đầu (shadow_bind), check source-side invariants riêng cho engine schemaless (collection có doc / table có row). Cắt sớm → log message rõ nghĩa cho operator.

**Đúng**:
```go
// Universal gate ở discover (sau khi quét shadow columns)
if totalRules == 0 {
    return fmt.Errorf("discover: 0 mapping rules — refusing to cascade")
}

// Engine-specific pre-flight ở shadow_bind
if isMongoEngine(eng) {
    if count, _ := coll.EstimatedDocumentCount(ctx); count == 0 {
        return fmt.Errorf("collection %s.%s empty — refusing to cascade", db, name)
    }
}
```

**Sai (anti-pattern)**: Tin "cascade success vì step trước success". Step success ≠ output usable. Test với 1 engine schema-tĩnh không cover được engine schemaless.

**Bonus**: Khi state machine có Retry() endpoint, retry đọc `from_state` của step_log entry failed gần nhất → nếu `from_state` là *_pending (in-flight) thì không có Advance transition. Đây là expected — operator phải re-trigger ở step gốc, không Advance từ pending.

Áp dụng được cho mọi state machine pipeline đa-engine: ETL, IaC apply, deploy graph, multi-source ingestion, schema migration orchestrator.

**File evidence**: `centralized-data-service/internal/handler/{command_handler.go,provisioning_step_handlers.go}`, report `agent/memory/workspaces/feature-cdc-integration/report_cascade_liability.md`.

---

## L-2026-04-29 — Three-layer trust failure when Component-A handoff to Component-B writes through Constraint-C

**Global Pattern**: When component A produces metadata that component B writes into a
constrained store C, three independent layers can silently fail:
1. A produces the wrong shape (e.g. cdcCols-only instead of source-mirrored shadow).
2. B uses the wrong key on conflict (e.g. ON CONFLICT on tuple X when schema enforces UNIQUE on key Y).
3. C rejects B's writes via a CHECK / type / regex constraint that B's source data wasn't normalized for
   (e.g. `information_schema.data_type` lowercase vs CHECK regex requiring uppercase).

Each layer can mask the others — fix one and you uncover the next. **Diagnose top-down by
following the actual error message from each layer**, not by guessing which layer to fix
first. Don't rebuild A "comprehensively" until you've proven the failure is in A (and not B
or C).

**Correct flow**:
- Reproduce end-to-end on a clean state (drop derived tables, reset state machine).
- For each layer's failure, capture the exact DB row / SQL / error code BEFORE proposing a
  fix.
- Fix one layer at a time; re-run end-to-end after each fix to surface the next layer.
- Add a normalizer at every B-writes-to-C boundary that involves an external/raw source
  (information_schema, BSON sample, JSON payload). Anything outside the safe-list maps to
  the most permissive type the constraint allows (e.g. TEXT) — lossless upcast beats
  silent rejection.

**Anti-pattern**: writing a single mega-fix that re-architects A, B, and C at once.
You'll burn context on rework when only one layer was actually broken.

**Concrete instance** (CDC Auto provisioning, 2026-04-29):
- A = shadow_bind handler, B = master_binding UPSERT, C = `cdc_mapping_rules.data_type` CHECK
- A produced cdcCols-only shadow → fixed via `PrepareForCDCInsertWithBusinessCols` + engine-aware inference.
- B's ON CONFLICT key didn't cover the actual UNIQUE → fixed by switching to `binding_code`.
- C's CHECK regex rejected lowercase `text`/`bigint`/`timestamp without time zone` → fixed by `normalizeMappingRuleDataType()` mapping to canonical uppercase.

**Audit hook**: when adding a new rule INSERT that writes user-controlled or schema-introspected
values into a CHECK-constrained column, always normalize at the call site. Don't trust
upstream to canonicalize.

---

## Global Pattern — Fire-and-forget DDL generator that reads metadata produced LATER in the pipeline

**Date**: 2026-04-29 (workspace: feature-cdc-integration)

**Symptom**: Generator G runs at pipeline step A, reads metadata table M, emits DDL.
Step B (later) populates M. Output of G is empty/incomplete on the first pass because M is
empty when A executes. Subsequent passes work but never run automatically.

**Concrete instance**: `MasterDDLGenerator.Apply` runs at `master_bind` step, reads
`mapping_rule_v2`. Bridge from V1→V2 happens at `discover` step (later). First Apply emits
CREATE TABLE with only meta cols; business cols never appear. State machine still reaches
`running` because schedule step doesn't validate column set.

**Wrong fix**: Reorder pipeline (`discover` before `master_bind`) — breaks other invariants
(master table must exist before discover writes mapping rules referencing target cols).

**Right fix (Global Pattern [A produces metadata Y consumed by generator G run at step C earlier than A])**:
1. Make generator G's output ADDITIVE: separate CREATE-once path from idempotent
   ALTER-add-missing path. Apply executes both in same transaction so re-runs are safe.
2. After step A populates metadata Y, REPUBLISH the trigger event for G so it runs again
   with the now-complete metadata.
3. Validate the payload schema of the republish — a wrong key produces silent skip
   ("master_table required" warn in our case). Use the same Marshal-side struct as the
   handler's Unmarshal target.

**Why it's elegant**: No reordering, no schema versioning, no temporal coupling between
steps in the orchestrator. Each step remains independently retryable. The republish is
best-effort — failure surfaces in handler error log, doesn't block schedule step.

**Generalizes to**: any DDL generator, any cache builder, any indexer, any cron-driven
projection, that reads from a table populated by a downstream step in the same workflow.
Apply the additive-pass + republish pattern instead of pipeline reordering.

---

## [2026-05-04] L-debezium-schema-evolution-compat — Debezium config change requires Schema Registry compat preemption

- **Trigger**: Brain PATCH `decimal.handling.mode=double` cho cdc-pg-source. Debezium re-register Avro schema mới (logical-decimal → double primitive). Default Schema Registry global compat=BACKWARD reject schema mới (incompatible primitive type change). Nếu user không set per-subject compat=NONE trước, connector goes FAILED → blocks ingest cho toàn bộ pipeline.
- **Root Cause**: Debezium connector config thay đổi serializer-side type (precise/double/string mode khác nhau emit Avro types khác nhau: bytes-decimal vs double vs string). Schema Registry coi đó là incompatible evolution. Không có CI guard. Brain không chạy pre-flight check compat.
- **Global Pattern [A changes Debezium config affecting Avro emit type for entity E] + [Schema Registry compat ≠ NONE] → Result: connector goes FAILED at next schema register, blocks downstream**: Khi đổi `decimal.handling.mode`, `time.precision.mode`, `binary.handling.mode`, hoặc bật/tắt SMT type-changing — luôn pre-flight set per-subject compat=NONE TRƯỚC khi PATCH connector.
- **Correct Pattern**:
  1. Trước khi PATCH: PUT `/config/<topic>-value` với `{"compatibility":"NONE"}` cho mọi topic affected.
  2. Verify response `{"compatibility":"NONE"}`.
  3. PATCH connector config qua `/connectors/<name>/config`.
  4. Wait connector + task RUNNING.
  5. Trigger 1 source event (INSERT row mới) → verify worker log không có decode error.
  6. (Optional) Restore compat=BACKWARD sau khi schema settled, để bảo vệ tương lai.
- **Trade-off**: compat=NONE bỏ guard schema regression. Nên set lại `BACKWARD` sau migration.
- **Tags**: #debezium #schema-registry #avro #decimal #connector-config #schema-evolution #pre-flight-check
- **Generalization check**: Pattern áp dụng cho (1) bật `tombstones.on.delete=false`, (2) đổi `time.precision.mode` từ `adaptive` sang `connect`, (3) thêm/xóa SMT InsertField/Cast, (4) đổi `key.converter` từ AvroConverter sang JsonConverter, (5) bất cứ Debezium config nào thay đổi Avro schema generation cho topic.

---

## [2026-05-04] L-v1-v2-anchor-key-port — V1→V2 ingest path migration forgets to populate constraint-keyed anchor column

- **Trigger**: B3 logical-clone fan-out chuyển ingest path từ V1 (DB-side trigger/default fill) sang V2 (`BuildUpsertSQLInSchema` generator). V2 schema thêm cột `_gpay_source_id` làm UNIQUE anchor cho master `dw_orders.orders_fact`. Generator V2 quên port logic ghi anchor → mọi shadow row có `_gpay_source_id=''` (empty) → master ON CONFLICT (`_gpay_source_id`) collapse N rows xuống 1.
- **Root Cause**: Khi migrate ingest path A → B, A có nhiều cơ chế ngoài-code (DB default, trigger, sequence) tự động fill column C. B viết upsert SQL từ scratch, audit business cols + một số meta cols (`_hash`, `_synced_at`, …) nhưng MISS column C vì C không nằm trong "business data" view của developer. Unit test V1 không cover C (V1 không cần test — DB tự fill); V2 unit test cũng không add case cho C.
- **Global Pattern [Path A → Path B migration: B writes SQL but forgets to populate constraint-keyed anchor column C that V2 schema introduces] → Result Y: master ON CONFLICT (C) collapses N distinct source rows into 1**.
- **Correct Pattern**:
  1. Audit ENUMERATE: trước khi merge migration, list đầy đủ MỌI column trong V2 schema mà KHÔNG phải pure business field — mọi `_*` prefix, mọi UNIQUE/anchor, mọi GENERATED ALWAYS AS, mọi col có DEFAULT non-trivial.
  2. Cross-check: với mỗi col từ (1), trace explicit write trong path B. Nếu không có → branch `if schema.Columns[C] exists → write derived value`.
  3. Schema reflection guard: dùng `schema.Columns[C]` runtime check, không hard-code, để backward-compat với legacy tables không có C.
  4. Unit test 2 cases: schema có C (V2) + schema không có C (V1) — assert SQL chứa/không chứa cột tương ứng.
  5. Live smoke INSERT 1 row mới (chưa từng tồn tại) → query shadow.C ≠ NULL/empty WITHOUT manual backfill. Wait 1 cron tick → master count tăng 1 với C distinct.
- **3-layer trace** (re-affirms L-three-layer-trust 2026-04-29): luôn trace từ failure point (master constraint violation / dedup) NGƯỢC qua master upsert → shadow row content → ingest write site → identify exact missing branch.
- **Tags**: #cdc #v1-v2-migration #anchor-key #unique-constraint #on-conflict #ingest-path #schema-evolution #three-layer-trust
- **Generalization check**: Pattern áp dụng cho (1) thêm `tenant_id` UNIQUE composite cho multi-tenant migration, (2) thêm `idempotency_key` cho exactly-once upsert layer, (3) thêm `partition_key` cho sharded warehouse, (4) thêm `business_event_id` UNIQUE cho event-sourced replay, (5) bất cứ schema evolution nào thêm cột làm UNIQUE/anchor mà ingest path không tự suy ra từ business data thuần.

---

## L-event-translator-field-completeness (2026-05-04, CDC Integration P1.1/G3)

**Global Pattern**: `[A] (event-pipeline-translator-layer) writes [B] (downstream-event-DTO) and hardcodes [field X] (less-common field like before/source/header/correlation) to nil/zero — even when upstream raw payload [Y] (Avro/Protobuf/JSON) actually populates [X]. Result Z: downstream consumer Z that depends on [X] either errors out (hard-fail guard surfacing as 'no data') or silently drops events. The error message "no [X] data" misdirects ops to suspect upstream config, when the bug is in the translator.`

**Đúng**:
1. Translator phải parse ALL event fields uniformly. Symmetric codec helper (e.g. `unwrapAvroUnion`) cho mọi field, không hardcode field nào ra nil.
2. Hard-fail guard ở handler boundary thay bằng warn+skip per-route khi missing optional field.
3. Khi error "no X data" xuất hiện: **layer 1** raw payload sniff (kafka-console-consumer raw bytes), **layer 2** translator output (log dumped DTO), **layer 3** handler input. Bug có thể ở layer 1, 2, hoặc 3 — đừng nhảy thẳng xuống layer 3 (handler).
4. Khi diagnose phát hiện DB/external infra OK (e.g. REPLICA IDENTITY đúng) → root cause bắt buộc ở code path → đọc translator trước handler.

**Anti-pattern**: bài học này KHÔNG phải về REPLICA IDENTITY (đã FULL từ trước trong P1.1 case). Anti-pattern thực sự: assume "no before data" error message phản ánh upstream missing payload, không nghi translator hardcode.

**Real-world case (P1.1/G3)**:
- Triệu chứng: `handleDelete` hard-fail "no 'before' data in delete event" cho mọi DELETE.
- Layer 1 verify: REPLICA IDENTITY=FULL, Debezium publication enable DELETE.
- Layer 2 verify: Avro raw payload có `before` field populated.
- Layer 3 (translator) phát hiện bug: `kafka_consumer.go:~375` build CDCEvent với `"before": nil` hardcoded, không gọi `unwrapAvroUnion(event["before"])` (đã làm cho `after`).
- Fix A1: parse `beforeRaw` symmetric với `afterRaw`. Fix A2: relax handler guard từ hard-fail sang warn+skip per-route (defense-in-depth: nếu A1 fail edge case nào cũng không poison toàn batch).

**Tags**: #cdc #event-pipeline #avro-translation #boundary-guard #before-image #three-layer-trace
**Generalization check**: Pattern áp dụng cho (1) Webhook fanout missing `signature` header parse, (2) gRPC interceptor drop `metadata` correlation, (3) JSON-to-Protobuf bridge skip oneof variant, (4) message bus bridge drop `headers` map, (5) bất cứ multi-hop translator nào có schema mismatch giữa upstream parser và downstream DTO.

---

## L-multi-tier-filter-mirror (2026-05-04, CDC Integration P0.2/G7)

**Global Pattern**: `[A] (orchestrator/admin-api) onboards new resource [X] (collection/table/topic) by writing to [B] (registry) and updating [C] (low-level allow-list, e.g. collection.include.list / table.include.list) on external system [E] (Debezium / Kafka Connect / proxy / firewall). [E] thực ra có MULTIPLE TIERS of filter: filter cấp thấp (col/table) lẫn filter cấp cao (database / namespace / vhost / region). [A] chỉ touch tier thấp → tier cao silently drop → resource [X] never streams. Result Y: orchestrator báo "register OK", registry+external low-level filter consistent, nhưng pipeline đứng im không event nào tới.`

**Đúng**:
1. Khi onboard cross-system resource, ENUMERATE tất cả tier filter của hệ thống đích trước khi viết orchestrator. Debezium MongoDB: `database.include.list` + `collection.include.list`. Postgres: `database.dbname` + `schema.include.list` + `table.include.list`. MySQL: `database.include.list` + `table.include.list`. Kafka ACLs: cluster-level + topic-level. Firewall: VPC-level + SG-level.
2. Mỗi tier filter cần 1 helper riêng trong orchestrator (e.g. `extendDatabaseList`, `extendCollectionList`) — và 1 wrapper gộp gọi đủ tier theo thứ tự top-down (cao trước, thấp sau).
3. Sau onboard, MUST verify "first event arrives within N seconds" — không tin success-of-write-config làm proxy cho success-of-streaming.
4. Smoke test PHẢI tạo resource [X] ở namespace mới (chưa từng có row nào) để force pass-through tier cao. Test ở namespace cũ luôn pass vì tier cao đã sẵn từ trước.
5. Document trong onboarding flow: "tier-N missing list" là failure mode #1 silent — log warn nếu orchestrator detect resource [X] thuộc namespace chưa có ở tier cao.

**Anti-pattern**: Cho rằng "config write 200 OK" = "resource streaming". Hai chuyện hoàn toàn khác nhau.

**Real-world case (P0.2/G7)**:
- Admin-api 5-step PUT extend `collection.include.list` += `goopay.smoke_p02_close_<TS>` thành công, registry transactional commit, NATS signal đánh thức Reader manager, cache reload bắt đúng row mới.
- Topic chưa bao giờ xuất hiện ở Kafka vì Debezium connector `goopay-mongodb-cdc.database.include.list` chỉ có `payment-bill-service,centralized-export-service` — không có database `goopay`.
- Triệu chứng: "đăng ký xong nhưng không có event" — operator nghi worker filter / NATS / Schema Registry; root cause ở Debezium tier cao nhất.

**Fix forward (chưa land)**: `extendDebeziumInclude` extend cả `database.include.list`/`db.include.list` đồng thời, hoặc emit warning cảnh báo namespace mới và yêu cầu operator approve trước.

**Tags**: #cdc #orchestrator #include-list #multi-tier-filter #debezium #onboarding #silent-drop #verify-streaming-not-config
**Generalization check**: Pattern áp dụng cho (1) Kubernetes NetworkPolicy namespace+pod selector, (2) AWS SG inbound + VPC ACL, (3) Kafka ACLs cluster + topic, (4) Stripe webhook endpoint + event type, (5) Cloudflare zone + page rule, (6) bất cứ external system nào có nested allow-list theo cấp resource cha-con.


---

## L-input-fallback-pattern (2026-05-04, CDC Integration Phase F3 + System Refactor 2026-05)

**Triggering event**: Phase F3 round 1 — admin-api `POST /v2/sources/register` cho Mongo collection chỉ
truyền `source_locator = {"database": "payment-bill-service"}` (không có `collection` key) và **dựa vào
`source_object_name` ở top-level**. Nhưng 3 vị trí khác nhau trong `internal/admin/helpers.go` đều đọc raw
`stringFromLocator(req.SourceLocator, "collection")` rồi dùng giá trị rỗng đó để tạo:

1. `qualifiedSourceObjectName` (line 76-82) → `normalized_source_key = "payment-bill-service."` (UNIQUE
   constraint poison khi 2 register kế tiếp).
2. `topicNameFor` (line 127-133) → `cdc.<conn>.payment-bill-service.` (Schema Registry preempt với subject
   tên rác; Kafka topic không match worker discover).
3. `extendDebeziumInclude` (line 232-237) → `collection.include.list` thêm "payment-bill-service." và
   "payment-bill-service.x" → connector accepted nhưng KHÔNG capture collection mới → ingest stuck, Kafka
   offset không tăng, shadow không nhận row.

Round 1 fix chỉ chạm 1/3 vị trí. Brain audit Round 2 mới phát hiện 2 vị trí còn lại — cùng pattern đối
xứng, cùng nguồn gốc.

### Global Pattern

> **Pattern [Component A reads optional key K from request payload B → uses raw value as a structural
> identifier part X (table name, topic name, normalized key, ACL entry)] → Result Y: empty propagation,
> dirty entries, silent ingest stuck, UNIQUE collision khi K vắng mặt.**
>
> **Đúng**: A PHẢI fallback to canonical field `B.canonicalName` (hoặc field tier-tiếp theo) khi K
> missing/empty. Chỉ tin K khi K không rỗng. Không dùng raw zero-value làm identifier component.

### Áp dụng cho project nào?

- **CDC orchestrator** đọc `source_locator` payload → fallback `source_object_name`.
- **Kubernetes admission controller** đọc optional `metadata.labels.X` → fallback `metadata.name`.
- **Stripe webhook router** đọc optional `metadata.tenant_id` → fallback infer từ `customer_id`.
- **Multi-tenant DB sharding** đọc optional `tenant_key` từ JWT → fallback tenant inferred từ
  `subject` claim.
- **Image build pipeline** đọc optional tag override từ commit message → fallback `git rev-parse short`.
- **Search indexer** đọc optional `indexer.targetIndex` → fallback `default_index_for_type`.
- Bất kỳ adapter nào dịch payload polymorphic (multi-engine, multi-source, polyglot) sang identifier
  cứng đều có rủi ro pattern này khi K không phải required field.

### Symptom phát hiện được

- UNIQUE constraint vi phạm bí ẩn khi user tưởng register chỉ 1 lần (thực ra 2 lần cùng key rác).
- ACL/include-list/topic-list có entry "prefix.<empty>" hoặc "<prefix>.x" trông như test data nhưng
  thật ra do fallback broken.
- Pipeline accept config nhưng silent skip — log không kêu vì giá trị rỗng vẫn parse hợp lệ.
- Self-heal khi clean lại config + restart binary mới (Debezium re-snapshot).

### Defensive measures

1. **Audit all uses of `req.OptionalField`** ở mỗi vị trí cùng lúc (CLAUDE.md lesson "Fix bug 1 service
   quên cross-service") — KHÔNG ăn 1 vị trí rồi nghỉ.
2. **Validate at admission**: từ chối request nếu sau khi compute fallback identifier vẫn rỗng — ném 400.
3. **Test driver-level**: viết test multi-payload (with K, without K, with K=empty, with K=bogus) để
   ép pattern bug surface ở review.
4. **Schema-level**: nếu đặc tả format output là "non-empty path component" → assert ngay sau compute,
   trước khi feed config tới downstream.

### Verification path

Fix landed (commit `92d78d3`):
- helpers.go 3 vị trí đều có `if collection == "" { collection = req.SourceObjectName }` (hoặc tương
  đương cho table/PG path).
- Test `TestExtendDebeziumInclude_Mongo_BothTiers` + 21 assertion PASS.
- Live smoke F3 round 2: Mongo INSERT acknowledged → Kafka offset advance 6→7 → shadow row landed
  `f3v2_smoke_1777887709` (`_synced_at=2026-05-04 09:41:51.804387 UTC`).

**Tags**: #adapter #fallback #optional-key #identifier #unique-constraint #silent-drop #cross-site #audit-all-occurrences


## 2026-05-05 — Volume preservation when splitting docker-compose project

**Trigger**: Phase B5 split `centralized-data-service/docker-compose.yml` (16 services) thành 2 compose:
- core 10 services giữ project name `centralized-data-service` (volumes `pg_cdc_data`, `kafka_data` preserved).
- dev 6 services chuyển sang project `cdc-docker-dev`.

**Bài học cụ thể**: `docker-compose` namespace volume names theo project (`<project>_<volume_decl>`). Nếu khai báo volume bình thường ở project mới, compose sẽ tạo volume RỖNG MỚI (`cdc-docker-dev_pg_source_data`) — data 6 ngày test bị mất.

**Fix**: declare volume external với `name:` trỏ tới existing namespaced name:

```yaml
volumes:
  pg_source_data:
    external: true
    name: centralized-data-service_pg_source_data
```

→ Project mới mount volume cũ. Data preserved. Khi user chạy môi trường sạch (chưa có data), bỏ `external: true` + `name:` để compose tự tạo.

**Global Pattern**: Khi tách docker-compose project A thành A' + B (subset của services move sang B), declare volumes của subset đó trong B với `external: true, name: A_<volume>` để bảo toàn data. Đúng: **A_<vol> stays bound to physical disk, B references it through external alias** → zero data loss, zero downtime beyond container restart.

**Anti-pattern**: tạo `B_<vol>` rỗng + chạy `docker volume rm A_<vol>` → mất data. Hoặc dùng `docker run --volumes-from` shim — phá namespace, gây conflict khi compose down.

**Verify checklist khi split**:
1. `docker volume ls | grep <project_old>_` — list current volumes.
2. Map volumes giữ vs move.
3. Trong compose B: `external: true` + `name:` cho mỗi moved volume.
4. `docker compose down` (no -v) old project → volumes survive.
5. `docker compose up -d` 2 project mới — verify hostname resolution (external network) + data count khớp before/after.

**Related**: lesson 2026-04-29 về Phase ≠ Workspace mới (vẫn gắn workspace cha) — pattern y hệt: thêm khả năng phân tách mà không phá namespace gốc.

**Tags**: #docker-compose #volume #external #data-preservation #split-project #namespace #migration

---

## 2026-05-05 10:24+07 — Lesson: Cross-repo relative-path mount = decoupling violation

**Context**: Phase B5.5 split docker-compose `centralized-data-service/` (core) khỏi `cdc-docker-dev/` (config-able DBs). Round 1 quên 2 init-script mount vẫn dùng relative path `../centralized-data-service/deployments/...` từ compose mới — đè ngược coupling vừa tách. Anh trainguyen catch: "rất vô lý. vì chúng ko nên dính tới nhau" → fix B5.5b move asset sang `cdc-docker-dev/init/` rồi đổi mount thành `./init/...`.

**Global Pattern**: Khi split repo/project A → A' + B (cùng umbrella hay khác), mọi volume mount / ConfigMap source / build context trong B mà reference asset của A bằng path `../A/...` (hoặc absolute path tới A) = coupling lén. Đúng: **B own toàn bộ asset cần thiết cho B services. Move (không copy) asset từ A sang B; mount bằng `./...` relative tới B.** Test trước khi merge: `grep -rn '\.\./<other-project-name>' <new-project-dir>/` phải 0 hit cho YAML/compose/Dockerfile/Helm.

**Anti-pattern điển hình**:
- `volumes: ['../A/init:/docker-entrypoint-initdb.d:ro']` trong B/docker-compose.yml.
- `Dockerfile` của B `COPY ../A/configs ./configs`.
- Helm values `extraVolumes: hostPath: /repo/A/secrets`.

**Verify checklist sau split**:
1. `grep -rn '\.\./' <new-project>/` filter file types (yml, yaml, Dockerfile, sh) → review từng hit. Match cross-project = fix.
2. `grep -rn '<absolute-path-to-other-project>' <new-project>/` → cũng 0 hit.
3. Run `docker compose config --quiet` từ root mỗi project — không error path resolution.
4. Sau move: `grep` ngược lại trong A để confirm asset không còn được A internal sử dụng (nếu còn → COPY thay vì MOVE; nếu không còn → DELETE để giữ A clean).

**Related**: lesson "external volumes bảo toàn data" (cùng phase B5.5). Bộ đôi: (i) volumes external giữ data; (ii) asset move sang repo own giữ decoupling. Thiếu một thì split chưa hoàn chỉnh.

**Tags**: #split-project #decoupling #docker-compose #cross-repo-mount #anti-pattern #relative-path

---

## 2026-05-05 — Lesson: Centralize naming convention in a `naming` package, env-driven

**Context**: Schema prefix `shadow_` hardcoded ở 4 call sites (admin helpers, provisioning handler, sinkworker normalizer) trong `centralized-data-service`. Đổi convention sang `lake_` / `raw_` / language-specific → phải sửa 4 file + risk sót hit (state enum `shadow_pending`, NATS subject `cdc.cmd.shadow.bind`, log keys lẫn schema name khi grep).

**Global Pattern**: Khi convention naming X (prefix/suffix/separator/casing) hardcoded N call sites trong codebase A để tạo identifier kiểu `X<Y>` → tạo package `naming` (hoặc `convention`) tập trung. Package expose helper `<Convention>Name(parts...) string` đọc env `<DOMAIN>_<CONVENTION>_<PART>` qua `sync.Once`, default fallback = giá trị cũ để giữ backwards compat. Mọi call site `"X" + dynamic` đổi sang `naming.<Convention>Name(dynamic)`.

```go
// internal/naming/naming.go
package naming

import ("os"; "sync")

const defaultShadowPrefix = "shadow_"

var (
    shadowOnce   sync.Once
    shadowPrefix string
)

func ShadowSchemaPrefix() string {
    shadowOnce.Do(func() {
        shadowPrefix = os.Getenv("CDC_SHADOW_SCHEMA_PREFIX")
        if shadowPrefix == "" { shadowPrefix = defaultShadowPrefix }
    })
    return shadowPrefix
}

func ShadowSchemaName(suffix string) string {
    return ShadowSchemaPrefix() + suffix
}
```

**Lý do thắng**:
1. **Đổi convention = đổi env**, không touch code. PR review trở thành 1-dòng env change thay vì N-file diff.
2. **Phân biệt rõ schema-name vs state-name vs subject-name**: package boundary tách 3 domain identifier dùng cùng từ "shadow" nhưng khác semantic. `naming.ShadowSchemaName(...)` chỉ ra purpose = schema; `cdc.cmd.shadow.bind` (NATS) và `shadow_pending` (state enum) không bị rename oan.
3. **`sync.Once` cache**: env đọc 1 lần ở boot, các call site không lặp `os.Getenv` (perf + consistency — không có race với env mutation mid-process).
4. **Default fallback giữ behavior cũ**: opt-in upgrade, không break tồn tại.

**Anti-pattern (đừng làm)**:
- Để N call sites hardcode literal `"X"` rồi mỗi lần đổi convention phải `find-and-replace` → sót hit do từ đó cũng xuất hiện ở comment, log message, state enum, test fixture.
- Đặt env `os.Getenv` gọi mỗi call site (không cache) → mỗi schema-name resolution = syscall + risk inconsistency nếu env thay đổi giữa chừng.
- Đặt biến package `var prefix = os.Getenv(...)` ngoài `init()` mà không có `sync.Once` → race với test setup ENV (test framework set env sau package init).

**Verify checklist**:
1. `grep -rn '"X"' <repo>/` sau refactor → 0 hit ở schema-creating sites (state enums + subjects + log keys vẫn còn — đó là intentional).
2. `go build ./... && go test ./...` PASS.
3. Smoke: chạy với env override `<DOMAIN>_<CONVENTION>_<PART>=Y_` → identifier mới start `Y_<dynamic>`.
4. Smoke: chạy không env → fallback default = giá trị cũ (backwards compat).

**Áp dụng được cho ≥3 dự án**:
- CDC pipeline: `shadow_` prefix (case study này), `dw_` master prefix, `cdc.cmd.` NATS subject prefix.
- E-commerce: `tenant_` schema prefix multi-tenant SaaS, `tmp_` background job table prefix.
- Logs/observability: metric name prefix (`app_<env>_<component>_*`), trace tag prefix.

**Tags**: #naming #convention #env-driven #refactor #single-source-of-truth #sync-once #default-fallback #global-pattern

---

## 2026-05-05 — Lesson: `.env.example` style — actionable env vars > prose comments

**Trigger**: anh trainguyen sửa mongo block từ verbose 3-line comment block của em sang 2-line: `# ---------- header` + `MONGO_URL=...`. Pattern này áp dụng cho mọi container/service trong .env.example.

**Global Pattern**: `.env.example` mỗi entry phải là **actionable** (env var thực sự copy được sang `.env`) HOẶC **omit hoàn toàn**. Nếu service A không expose env knobs trong compose, nhưng consumer B/C cần URL/endpoint của A → ghi 1 var `<SERVICE>_URL=<connect-string>` để B/C copy. KHÔNG ghi block comment thuần "DEV ONLY: anonymous access..." mà không có env var nào — comment dài làm noise, user phải tự suy luận URL.

**Anti-pattern (đừng làm)**:
```
# ---------- mongo source (gpay-mongo replSet rs0 on :17017) ----------
# DEV ONLY: anonymous access (no auth). Connect URL:
#   mongodb://gpay-mongo:27017/?replicaSet=rs0
# (host port :17017 → container 27017). Prod = MongoDB Atlas...
```
3 dòng comment + 0 env var → user copy file xong vẫn không có gì useable, phải đọc và tự gõ.

**Pattern đúng**:
```
# ---------- mongo source (gpay-mongo replSet rs0 on :17017)
MONGO_URL=mongodb://gpay-mongo:27017/?replicaSet=rs0
```
1 dòng comment header (đủ identify) + 1 env var thẳng (copy-paste runnable). Cô đọng hơn, action-oriented.

**Quy tắc tổng quát cho `.env.example`**:
1. Mỗi block: ≤ 1 dòng comment header (tên service + key info).
2. Theo sau là env var(s) thực sự (giá trị placeholder hoặc default sane cho dev).
3. Nếu service không có env knob trong compose nhưng consumer cần connect string → expose `<SERVICE>_URL=...` cho consumer copy.
4. KHÔNG viết prose ("DEV ONLY: ...", "Prod uses ...") trong `.env.example`. Prose thuộc về `README.md`. `.env.example` là **template-to-copy**, không phải tutorial.
5. Ngoại lệ: 1-line note về security (e.g. `# DEV ONLY — không deploy lên prod`) đầu file là OK.

**Áp dụng được cho ≥3 dự án**:
- Microservices: mỗi service `.env.example` liệt kê service-DB creds + dependent-service URLs (consumer copy là chạy được).
- Frontend: `.env.example` liệt kê API_URL, CDN_URL, FEATURE_FLAGS_URL — không kèm prose explanation.
- CI/CD: secrets template chỉ list var names + placeholder, không list policy.

**Tags**: #env-example #documentation #actionable-config #copy-paste-friendly #dx #global-pattern

## 2026-05-05 — Lesson: Dockerfile bake `config-local.yml` only = prod ship DEV creds

**Trigger**: anh trainguyen flag *"sao repo auth hiện tại nó có cảm giác ko lên prod đc vậy"*. Audit `cdc-auth-service/deployments/docker/Dockerfile:12` lộ pattern `COPY --from=builder /app/config/config-local.yml ./config/config-local.yml` — image prod nuốt creds DEV + JWT secret `change-me-in-production`. Reconcile-service làm đúng pattern: `COPY --from=builder /app .` (cả repo, gồm 3 yml local/prod/sample).

**Global Pattern [Dockerfile X copies single config-local.yml only into prod image Y] → Result Z (prod runtime ships DEV creds, default secrets, dev pool sizes; image không deploy được sạch lên multi env)**.

Đúng:
1. Dockerfile `COPY config ./config` (CẢ thư mục) — image carry mọi env variant.
2. Runtime chọn file qua env (`cfgPath=./config/config-production.yml`).
3. Prod yml fields rỗng cho secrets — env override (`AUTH_DB_HOST`, `AUTH_JWT_SECRET`) điền tại runtime.
4. `validateConfig()` refuse:
   - rỗng required (host/database/secret/port);
   - default placeholder (`change-me-in-production`) khi `mode==production`.
5. Code env-binding dùng `viper.AutomaticEnv()` + `SetEnvPrefix(SVC)` + `BindEnv(key, ENV_NAME)` map — single source of truth, không hardcode `applyEnvOverrides`.

**Anti-pattern**:
- `COPY config-local.yml` only → 1 image / 1 environment, rebuild cho từng env (CI/CD waste, drift risk).
- Prod yml `${VAR}` placeholder mà không có envsubst pipeline → viper KHÔNG expand syntax này native, field thành literal string `"${VAR}"` → DB connect fail với hostname `${VAR}`.
- `applyEnvOverrides` hardcoded list → thêm field schema phải sửa Go code, dễ sót.

**Áp dụng được cho ≥3 dự án**:
- Bất kỳ Go service dùng viper + Dockerfile multi-stage (cdc-auth, centralized-data, cdc-cms, reconcile-service).
- Node service dùng dotenv + Dockerfile (pattern tương tự: copy cả `config/`, runtime chọn `NODE_ENV`).
- Java/Spring service dùng `application-{profile}.yml`: profile chọn qua `SPRING_PROFILES_ACTIVE` env, image phải bundle cả 3 file local/staging/prod.

**Detection heuristic** (dùng khi audit repo mới):
1. `grep -n "COPY.*config-local" Dockerfile*` → red flag.
2. `ls config/` thiếu `config-production.yml` hoặc tương đương → red flag.
3. `grep -n "applyEnvOverrides\|os.Getenv.*HARDCODED_KEY" config/*.go` đếm > 5 lần → hardcoded env list smell.
4. Validate boot binary với `cfgPath=prod.yml` không có env → expect FAIL với required missing message.

**Tags**: #docker #config-management #env-override #viper #prod-readiness #global-pattern #dx

## 2026-05-05 — Lesson: Go service `.env.example` = dead weight nếu (no godotenv) ∧ (compose có defaults)

**Trigger**: anh trainguyen flag *".env.example đang có cảm giác nó ko xài vì đang dùng go mà"*. Audit `cdc-auth-service`: `grep godotenv` 0 hit, `go.mod` không import dotenv lib, compose có `${VAR:-default}` cho cả 3 DB vars. Kết luận: file là noise — Go binary đọc YAML qua viper, compose có defaults, 0 docs reference.

**Global Pattern [Repository R kèm `.env.example` cho service S written in language L] → Result Y**:
- Nếu L = Node/Python (auto-load `.env` via dotenv runtime / framework convention) → `.env.example` LÀ contract, giữ.
- Nếu L = Go AND `grep godotenv R/` 0 hit AND compose-defaults present → `.env.example` LÀ dead weight, XÓA.

**Decision tree (audit repo Go mới)**:
1. `grep -r "godotenv\|joho/godotenv" --include="*.go"` → có dotenv loader? 
   - YES: `.env.example` is contract, validate fields match.
   - NO: continue 2.
2. Compose service có `${VAR:-...}` defaults cho mọi var trong `.env.example`? 
   - YES: `.env` purely optional → file là noise nếu không có docs reference.
   - NO: `.env.example` documents required overrides → giữ.
3. `grep -r "\.env\.example" R/` (docs/scripts) → có reference không?
   - YES: keep (documented contract).
   - NO + đã pass step 2 = noise → DELETE.

**Anti-pattern**: copy `.env.example` template từ Node project sang Go project mà không check runtime loading. User copy `.env` xong vẫn không thấy effect → confused → bug report.

**Cách user override env trong Go service KHÔNG dùng dotenv**:
```bash
# Option A: shell export
export AUTH_DB_HOST=prod.rds.com && ./auth-service

# Option B: env-file qua docker/k8s orchestrator (compose `env_file:`, k8s `envFrom`)
# Option C: source .env (manual): `set -a; source .env; set +a; ./auth-service`
```
KHÔNG có "auto-load" như Node — Go cần explicit.

**Áp dụng được cho ≥3 dự án**:
- Bất kỳ Go monorepo có nhiều service: audit từng service có dotenv không, thống nhất convention.
- Migration Node→Go: drop `.env.example` (hoặc chuyển sang `config-sample.yml`) khi rewrite.
- Static-binary deploy (k8s/ECS): env injected qua orchestrator — `.env` file là phản pattern.

**Tags**: #go #dotenv #env-loading #config-management #dead-files #global-pattern #dx

## 2026-05-05 — Lesson: Validation BEFORE fallback merging — order matters in config pipelines

**Trigger**: B5.6.2 centralized-data-service. validateConfig gặp false-positive PASS khi fields rỗng vì `cfg.DB.PgxDSN()` trả về string non-empty `"postgres://:@:0/?sslmode="` (literal sprintf không bao giờ empty), `applyDBFallbacks` set `cfg.SystemDB.URL = legacy` → validateConfig thấy non-empty → app boot OK rồi crash khi connect runtime.

**Global Pattern [Pipeline P có sequence: read input I → derive defaults D → validate V] → Result Y**:
- Nếu `V` chạy AFTER `D` → V thấy `I ∪ D` (merged state) → user intent rỗng bị lấp bằng derived value → **false-positive PASS**.
- Nếu `V` chạy BEFORE `D` → V thấy CHỈ `I` (user intent) → empty input bị reject đúng → **fail-fast at boot**.

**Đúng sequence**: ReadConfig → Unmarshal → applyEnvOverrides (env trộn vào user input) → **validateConfig** → applyFallbacks (derive missing fields).

**Anti-pattern**:
```go
applyEnvOverrides(cfg)
applyFallbacks(cfg)   // SystemDB.URL ← cfg.DB.PgxDSN() (literal-non-empty garbage)
validateConfig(cfg)   // sees non-empty SystemDB.URL → PASS (FALSE positive)
```

**Pattern đúng**:
```go
applyEnvOverrides(cfg)
validateConfig(cfg)   // sees empty SystemDB.URL → REJECT (correct)
applyFallbacks(cfg)   // safe to derive AFTER passing validation
```

**Detection heuristic** (audit config pipelines):
1. Tìm `applyDefaults / applyFallbacks / merge*` đặt BEFORE `validate*` trong `NewConfig`/`Load` → red flag.
2. Tìm helper trả về string từ `fmt.Sprintf` mà KHÔNG check empty inputs (e.g. `func DSN() string { return fmt.Sprintf("postgres://%s:%s@%s:%d/...", "", "", "", 0, "") }` → ra `"postgres://:@:0/..."` non-empty literal).
3. Test rằng config rỗng hoàn toàn → validateConfig trả error rõ; nếu PASS → bug.

**Áp dụng được cho ≥3 dự án**:
- Config validation pipelines bất kỳ ngôn ngữ nào (Go viper, Node convict, Python pydantic, Java Spring profiles).
- ETL / data pipelines: validate raw input BEFORE applying transforms/derives — derives che mất missing source data.
- API request validation: validate raw payload BEFORE applying server-side defaults — defaults che mất user-supplied invalid fields.
- Database migrations: validate "intent" SQL trước khi run idempotent fallbacks (`CREATE IF NOT EXISTS`) — fallback che mất schema mismatch.
- Form validation UI: validate user input BEFORE applying placeholder/default values — defaults che mất empty intent.

**Anti-pattern bonus**: helper getter trả về literal string non-empty từ rỗng input (như `PgxDSN()` ví dụ trên) là code smell. Pattern an toàn: getter return `("", false)` hoặc `(nil, error)` khi inputs missing — caller buộc phải handle empty case explicitly.

**Tags**: #validation #config-management #order-matters #fail-fast #empty-input #global-pattern #anti-pattern

---

### Lesson #1294 — JSON serialization order khi migrate `map[string]any` → typed struct (CQRS Q-side, byte-identical contract)

**Khi nào xảy ra**: Refactor handler (CMS, BFF, gateway) chuyển payload xây bằng `map[string]any` (Go map / fiber.Map / gin.H / etc.) sang typed struct. Test diff thấy size giống nhau nhưng `cmp -s` báo DIFF.

**Root cause**: Go's `encoding/json` serialize map theo **alphabetical order** của key (post Go 1.12, deterministic). Struct serialize theo **field-declaration order**. Field order không match key order → byte-different output dù cùng nội dung.

**Global Pattern**: `Refactor [A: map-based payload] → [B: struct-typed payload] với contract byte-identical = order(A.keys) == order(B.fields)`. Đúng: declare struct fields theo alphabetical JSON tag order khi migrate từ map; hoặc generate diff bằng `jq -S` (sort keys) thay vì raw cmp nếu wire chỉ cần semantic-equivalent.

**Áp dụng được**: bất kỳ language nào có map (Python dict, JS object) khi migrate sang typed class/struct/dataclass đều dính bug này nếu wire contract pin byte-level.

**Detection**: `wc -c pre post` size giống nhau + `cmp -s` báo DIFF + `jq -S` cùng output = serialization order mismatch (chứ không phải data drift).

**Fix template** (Go): reorder struct fields theo `sort json_tags` ascending. Comment ghi rõ "field order matches legacy map alphabetical serialization".

**Tags**: #cqrs #refactor #json-serialization #byte-identical #order-matters #go #global-pattern

---

### Lesson #1295 — Hybrid command bus cần `ResultBody` trên CommandResult cho sync handlers (CQRS C-side)

**Khi nào xảy ra**: Thiết kế CommandBus B route command C qua 2 path:
- **Sync** (in-process map handler X): low-latency operations như `alert.ack` (chỉ UPDATE 1 row Postgres).
- **Async** (NATS publish subject Y): long-running như `master.swap`, `recon.check` (chạy trên worker).

Nếu `CommandResult` chỉ có `{JobID, Accepted bool}` không có wire body → sync handler X trả nothing → FE buộc phải poll `/jobs/:id` sau mỗi Dispatch dù đã có kết quả ngay. RTT = 2 round-trips cho việc lẽ ra 1.

**Root cause**: Bus author áp pattern "all async" (fire-and-forget) lên cả sync path để giữ contract đồng nhất → đánh mất ưu thế của sync route.

**Global Pattern [Hybrid bus B route command C qua sync X / async Y, Result chứa optional ResultBody]**: 
- Declare `CommandResult.ResultBody json.RawMessage` (nullable, omitempty). 
- Sync handler X populate ResultBody với wire-bytes trả về cho caller. 
- Async path Y để ResultBody rỗng — FE biết `Accepted=true && ResultBody==nil` ⇒ poll `/jobs/:id`. 
- Sync path X trả `Accepted=true && ResultBody!=nil` ⇒ FE inline render kết quả.

**Đúng**: 
```go
type CommandResult struct {
    JobID      string          `json:"job_id"`
    Accepted   bool            `json:"accepted"`
    ResultBody json.RawMessage `json:"result_body,omitempty"` // sync inline; async empty
}
```

**Sai**:
```go
type CommandResult struct { JobID string; Accepted bool } // mất sync ưu thế
```

**Áp dụng được cho ≥3 dự án**:
- CQRS-style microservice gateways (Go, .NET MediatR, Java Axon) có cả intra-service sync handler + cross-service async messaging.
- BFF/API gateway pattern: 1 endpoint vừa serve cache hit (sync) vừa dispatch backend job (async) — Result phải tải được cả 2 shape.
- LLM tool-use orchestration: tool call có thể return immediately (calculator) hoặc kick off background job (image gen) — Result envelope cần ResultBody slot cho immediate path.
- gRPC bi-modal: unary sync + server-stream async — response message nên có optional `body` thay vì 2 RPC riêng.
- WebSocket command pattern: ack-only (async) vs ack+payload (sync) qua cùng 1 envelope.

**Detection**: 
1. Tìm `CommandResult / CommandReply / DispatchResponse` không có wire-body slot.
2. Audit FE/caller code: nếu sau mỗi Dispatch luôn `setTimeout/while polling /jobs/:id` → smell.
3. Tìm sync handler in-process trả `(any, error)` rồi bị bus drop kết quả → smell.

**Anti-pattern bonus**: Force "all async" cho UI consistency (FE always poll) — sacrifice latency mà không gain gì (FE vẫn phải handle 2 shape: result-from-poll vs error-from-poll). Tốt hơn: 2 shape ngay tại Dispatch return (`ResultBody` filled vs nil).

**Tags**: #cqrs #command-bus #hybrid-sync-async #api-design #latency #global-pattern

---

## Lesson #1296 — 2026-05-06 — Plan critique cần verify từng claim với evidence trực tiếp

**Context**: Boss reviewed P3 plan, claim plan có line numbers off-by-some, "extract inline" sai (đã extract), evt subjects "NEW" thực ra đã có upstream. Muscle verify từng claim trước khi acknowledge.

### Global Pattern [Plan reviewer A claims X about codebase Y → Reviewee B] → Result Z. Đúng:
1. **Reviewee KHÔNG defensive-deny** — verify từng claim với grep/wc/file-stat trực tiếp.
2. **Reviewee KHÔNG blanket-accept** — vì đôi khi reviewer cũng sai (line numbers stale từ session trước).
3. **Reviewee output 1 bảng status**: `claim | actual | match?`. Nếu match → acknowledge + action item. Nếu không → đối chiếu evidence + đề xuất re-frame.
4. Mọi gap proposal phải có **effort estimate** (kèm reason) + **owner** (Brain | Muscle | Boss decide) + **status** (TODO | BLOCKED | DONE).
5. Nếu critique nêu blocker thiết kế (như "worker permission verify") → tag BLOCKED, KHÔNG tự ý implement đường tắt.

### Áp dụng được cho 3 dự án khác:
- Code review GitHub PR — PR comment "you should X" có thể base on outdated commit. Verify HEAD trước khi accept/argue.
- Architecture decision record (ADR) review — reviewer claim "we already have Y" → grep codebase confirm.
- Multi-team task hand-off — handed-over team verify claim của team trước (file paths, line numbers, naming conventions stale theo thời gian).

### Anti-pattern: "Yes-and" mọi critique → modify plan vô tội vạ → contradiction tích lũy. Hoặc "no-and" mọi critique → defensive → bỏ lỡ valid feedback.

### File minh chứng: `agent/memory/workspaces/feature-cdc-system-refactor/10_gap_analysis_p3_critique_2026-05-06.md`

---

## Lesson #1297 — 2026-05-06 — Cast TỪNG positional `?` trong CASE expression khi GORM/pgx prepared statement, KHÔNG cast outer

**Context**: P3.T3.12 StuckJobReaper SQL build động `started_at + (interval '1 second' * (CASE type WHEN ? THEN ? ... END)) < NOW()`. T3.11 smoke phát hiện reaper sweep fail mỗi 30s với 2 lỗi tuần tự:
1. `operator does not exist: interval * text (SQLSTATE 42883)` — outer cast `(CASE END)::int` thử trước, KHÔNG sửa.
2. Sau outer cast, lỗi đổi sang: `failed to encode args[1]: unable to encode 120 into text format for text (OID 25): cannot find encode plan`.

**Root cause**: pgx/GORM prepared-statement type inference resolve param types TRƯỚC khi outer cast áp dụng. Trong `CASE column-A WHEN ? THEN ? ... ELSE ? END`:
- Param đầu (`WHEN ?`) so với cột text → infer = text.
- Driver propagate text type sang mọi cùng-shape positional trong CASE → THEN `?` và ELSE `?` đều bị infer text.
- Khi caller truyền int64 (60, 600, 30, ...), driver từ chối encode int64 vào text slot → SQLSTATE 42883 hoặc encoding failure.
- Outer cast `(CASE … END)::int` chỉ chuyển kiểu KẾT QUẢ CASE sau khi evaluated; không ảnh hưởng inference cho mỗi positional `?`.

**Fix verified**: cast TỪNG positional ngay trong CASE branch:
```go
caseExpr.WriteString("WHEN ? THEN ?::int ")  // not "WHEN ? THEN ? "
caseExpr.WriteString("ELSE ?::int END")       // not "ELSE ? END"
```
Sau rebuild + restart, reaper log `{"msg":"reaped stuck jobs","count":1}`, status row flip 'running' → 'failed' đúng spec.

### Global Pattern [Driver D với prepared statement P, build động SQL với positional ? trong CASE-expression có column-of-type-A so sánh ở WHEN] → Result [param types lệch theo column-A; outer cast không sửa được; encoding failure khi caller truyền type-B]. **Đúng**:
1. Cast TỪNG positional ngay tại branch nó xuất hiện: `WHEN ?::A THEN ?::B`, `ELSE ?::B END`.
2. Outer cast `(CASE ... END)::B` CHỈ dùng cho final result type, KHÔNG sửa được inference cho positional bên trong.
3. Test integration với REAL Postgres (mock-DB hoặc sqlite không phát hiện vì khác driver type-inference).
4. Khi gặp `operator does not exist: T1 * T2` với prepared statement, nghi ngay positional inference trước khi suspect schema/migration.

### Áp dụng được cho 3 dự án khác:
- **Bất kỳ Go service dùng GORM/pgx + dynamic SQL build**: dashboards với column-filter, multi-tenant routing, schema-aware aggregation.
- **JDBC PreparedStatement Java/Kotlin**: cùng pattern infer xảy ra với JDBC driver Postgres khi mix column types trong CASE.
- **Python psycopg2/asyncpg với prepared mode** (đặc biệt qua pgbouncer transaction-pool): có thể tái hiện.

### Anti-pattern:
- Tin "outer cast sẽ sửa mọi inference issue" → debug loop dài.
- Mock DB cho test reaper SQL → bug không phát hiện trước smoke production-like.
- Suspect data type column trước khi suspect param inference.

### File minh chứng:
- Code fix: `cdc-cms-service/internal/service/stuck_job_reaper.go:111,114`
- Smoke evidence: `agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md` (entry `2026-05-06 15:35 ICT — T3.11 smoke matrix executed + HOTFIX-2`)
- Task tracking: #177 P3.HOTFIX-2

**Tags**: #postgres #gorm #pgx #prepared-statement #type-inference #case-expression #reaper #global-pattern

---

## Lesson #1298 — 2026-05-06 — CommandBus chỉ cho mutation/coordination, KHÔNG migrate audit-only side-effects

**Context**: Phase 3 cdc-cms-service refactor (CQRS C-side) chốt scope qua 7 đợt — 27 endpoint mutation đã qua bus. Còn 4 ActivityLog write (3 reconciliation_handler + 1 registry_handler). Câu hỏi: có nên migrate nốt cho "consistency"?

**Quyết định**: SKIP. Audit-only side-effect KHÔNG thuộc bus scope. Phase 3 closed sạch.

**Root cause của câu hỏi sai**: "Universal indirection" thinking — tin rằng mọi handler-level write nên đi qua bus để "uniform pattern". Bỏ qua phí của bus:
- +1 hop sync (JSON marshal/unmarshal request + response).
- +1 row `cdc_jobs` audit table per write — nhưng ActivityLog ĐÃ là audit, double-recording.
- Idempotency-Key collision risk khi 1 request có nhiều ActivityLog write (cần suffix `:audit:<seq>` workaround chỉ để tránh va chạm bus).
- Test surface tăng: validate rule, type tag namespace, errors.Is sentinel mapping cho thứ chỉ là log entry.

**Fix verified**: ActivityLog write giữ direct call ở handler. Kết thúc Phase 3, cdc_jobs chỉ còn rows cho mutation thật — observability sạch.

### Global Pattern [Codebase A có CommandBus B (CQRS C-side) → reviewer/team đề xuất migrate side-effect X (audit log, metrics emission, fingerprint touch) qua bus B "for consistency"] → Result [thêm hop sync + double-audit + idempotency collision risk, gain semantic = 0]. **Đúng**:
1. Bus B chỉ cho 2 track:
   - **Track Mutation** — destructive infra (DDL ALTER, business state INSERT/UPDATE, external HTTP destructive REST như Kafka Connect).
   - **Track Coordination** — async cross-service dispatch (NATS publish, queue enqueue, scheduled job dispatch).
2. Side-effect X (audit/metrics/log) → giữ `repo.Insert(ctx, ...)` hoặc `auditService.Record(...)` trực tiếp ở handler/service layer.
3. **Test phân loại**: side-effect X có "đứng tự do" được không? Nghĩa là: nếu X fail (network blip, table locked), request có rollback hay chỉ log warn?
   - Yes (chỉ log warn) → audit-only → KHÔNG bus.
   - No (rollback request) → mutation-essential → qua bus.
4. Nguyên tắc: **bus là indirection layer trả phí cho actions có blast radius**. Audit-write không có blast radius (failure ≠ user impact, chỉ giảm observability) → không xứng đáng phí bus.

### Áp dụng được cho 3 dự án khác:
- **CQRS Java/Spring** với Axon/EventBus: cám dỗ migrate `auditLog.publish(...)` qua command bus → giữ trực tiếp `auditRepo.save(...)`.
- **NestJS với @CommandBus**: ActivityLog interceptor gọi `commandBus.execute()` cho log entry → anti-pattern, chuyển về `loggingService.record()`.
- **Workflow engine (Temporal/Camunda)** với "everything is an activity" thinking: read-only/log-only operations không cần activity wrapper, gọi inline để tránh history blow-up.

### Anti-pattern:
- **Universal indirection** — tin mọi handler write phải qua bus để "uniform". Hệ quả: command-bus registry phình to vì log entry, error-mapping table phình to vì sentinel cho mỗi log type, idempotency table double-record.
- Lười phân loại blast radius → migrate hết cho nhanh → over-abstraction debt.
- Đóng scope theo "100% coverage" thay vì "actions có blast radius" → kéo cosmetic vào critical path.

### File minh chứng:
- Workspace audit log: `agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md` (entry `2026-05-06 18:10 ICT — Đợt 7 P3` + ghi chú "P3 destructive migration coverage final").
- Source giữ direct call: `cdc-cms-service/internal/api/reconciliation_handler.go` (3 chỗ) + `cdc-cms-service/internal/api/registry_handler.go` (1 chỗ).
- Decision: kết thúc Phase 3 sạch, không kéo D2 cosmetic route prefix + ActivityLog migration vào critical path.

**Tags**: #cqrs #command-bus #ddd #cms-service #scope-discipline #audit-log #anti-over-abstraction #blast-radius #global-pattern
