# Decisions: Merchant Export Activation Info

## 1. Activation Date Logic
- **Decision**: Logic calculation is moved to the **Auxiliary Handler** (`GetMerchantExportAuxiliaryHandler`) instead of the main Handler.
- **Why**: Existing patterns in this service use `BaseExportProcessor` hooks (`subQueryClass`, `mergeData`) to perform joins. This keeps the domain handlers clean and follows the "Minimal Impact" principle (Rule #6).

## 2. Combined Joins
- **Decision**: Merged `BusinessLine` and `MerchantHistory` lookups into a single `MerchantExportAuxiliaryQuery`.
- **Why**: Reduces RPC/DB roundtrips by fetching all required external data in a single composite query.
