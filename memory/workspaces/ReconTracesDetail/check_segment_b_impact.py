#!/usr/bin/env python3
import os
import sys
import re

# ==============================================================================
# SCRIPT STATIC ANALYSIS KIỂM TRA ẢNH HƯỞNG SEGMENT B (RECON STRUCT)
# ==============================================================================
# Script này được lưu trong workspace và dùng để kiểm tra ảnh hưởng của cấu trúc
# cũ lên toàn bộ codebase dự án centralized-data-service.
# ==============================================================================

PROJECT_DIR = "/Users/trainguyen/Documents/work/data-hub/centralized-data-service"
SRC_DIR = os.path.join(PROJECT_DIR, "internal")

# Các mẫu regex tìm kiếm dấu vết cấu trúc cũ (đã lọc các trường hợp hợp lệ)
PATTERNS = {
    "segment_b_window": re.compile(r'"segment_b_window"'),
    "OrphanInMaster": re.compile(r'\.OrphanInMaster\b'),
    "StaleIDs_usage": re.compile(r'\bstaleB\.StaleIDs\b|\bstaleObj\.StaleIDs\b'), # Chỉ bắt trên object staleB/staleObj
    "SourceDB_segment_b": re.compile(r'\b(rpt|report)\.SourceDB\b'),
}

# Các file loại trừ khỏi kiểm tra (ví dụ file base definition hoặc backup)
EXCLUDE_FILES = [
    "recon_base_handler.go", # Nơi định nghĩa struct mới/cũ và helper parse
]

def scan_files():
    errors = []
    
    if not os.path.exists(SRC_DIR):
        print(f"Thư mục nguồn không tồn tại: {SRC_DIR}")
        return errors
        
    for root, _, files in os.walk(SRC_DIR):
        for file in files:
            if not file.endswith(".go") or file.endswith("_test.go"):
                continue
            
            if file in EXCLUDE_FILES:
                continue
                
            file_path = os.path.join(root, file)
            relative_path = os.path.relpath(file_path, PROJECT_DIR)
            
            try:
                with open(file_path, "r", encoding="utf-8") as f:
                    content = f.read()
                    
                lines = content.splitlines()
                
                # Check 1: segment_b_window
                if PATTERNS["segment_b_window"].search(content):
                    for idx, line in enumerate(lines):
                        if "segment_b_window" in line:
                            errors.append(f"{relative_path}:{idx+1} - Phát hiện CheckType cũ 'segment_b_window'. Phải đổi sang 'hash_window' hoặc 'bucket_hash'.")
                            
                # Check 2: OrphanInMaster
                if PATTERNS["OrphanInMaster"].search(content):
                    for idx, line in enumerate(lines):
                        if ".OrphanInMaster" in line:
                            errors.append(f"{relative_path}:{idx+1} - Phát hiện sử dụng trường cũ '.OrphanInMaster'. Phải đổi sang '.MissingFromSrc'.")
                
                # Check 3: StaleIDs (chỉ báo lỗi nếu gọi từ staleB/staleObj)
                if PATTERNS["StaleIDs_usage"].search(content):
                    for idx, line in enumerate(lines):
                        if "staleB.StaleIDs" in line or "staleObj.StaleIDs" in line:
                            errors.append(f"{relative_path}:{idx+1} - Phát hiện sử dụng trường cũ '.StaleIDs' trên object. Phải đổi sang '.Mismatched'.")
                            
                # Check 4: SourceDB được gọi cùng rpt/report trên Segment B
                if PATTERNS["SourceDB_segment_b"].search(content):
                    for idx, line in enumerate(lines):
                        # Bỏ qua phép gán gán rỗng hoặc định nghĩa khởi tạo rỗng
                        if 'SourceDB: ""' in line or 'SourceDB = ""' in line:
                            continue
                        
                        # Bỏ qua dòng là chuỗi text log/error message hoặc trong nhánh else của Segment A
                        if 'report.source_db' in line or 'shadowRel = rpt.SourceDB' in line:
                            continue
                        
                        # Cảnh báo nếu sử dụng report.SourceDB trong file chứa logic shadow_master
                        if ("SourceDB" in line or "source_db" in line) and ("SegmentShadowMaster" in content or "shadow_master" in content):
                            # Cho phép nếu là ghi nhận ở handler check_handler.go
                            if "recon_check_handler.go" in file:
                                continue
                            errors.append(f"{relative_path}:{idx+1} - Cảnh báo: Sử dụng 'SourceDB' trong logic Segment B. Phải đổi sang shadow FQN ('ShadowSchema.ShadowTable').")
                            
            except Exception as e:
                print(f"Lỗi khi đọc file {file_path}: {e}")
                
    return errors

def main():
    print("🔍 Đang quét tĩnh codebase centralized-data-service...")
    errors = scan_files()
    
    if errors:
        print("\n❌ PHÁT HIỆN SỰ ẢNH HƯỞNG / SÓT CẤU TRÚC CŨ:")
        for err in errors:
            print(f"  {err}")
        print("\n👉 Yêu cầu: Quay lại bước làm plan, cập nhật các file trên để phù hợp với plan hiện tại trước khi thực thi code!")
        sys.exit(1)
    else:
        print("\n✅ Không phát hiện ảnh hưởng. Codebase đã sạch cấu trúc cũ!")
        sys.exit(0)

if __name__ == "__main__":
    main()
