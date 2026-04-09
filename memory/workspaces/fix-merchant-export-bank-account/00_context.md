# Context
Sửa lỗi xuất merchant export bị nhầm lẫn bankAccount. 
Khi export file merchant, ở `logics/export/merchant/merchant-export.pure.ts` đang có vấn đề với thông tin bankAccount của merchant bị map sai hoặc nhầm.
- Handler xử lý bankAccounts: `GetMerchantExportAuxiliaryHandler`
- Bank accounts query: `merchantBankAccountModel.getAll`
- Mapping dựa trên Map: `const bankMap = new Map(bankAccounts.map((ba: any) => [ba.merchantId?.toString(), ba]));`
- Nếu có duplicate bankAccount (merchantId trùng) hoặc object bankAccount trả về lồng nhau `{ data: [...] }` thì sẽ gây sai. Do đó cần check data structure trả về từ `merchantBankAccountModel.getAll`.
