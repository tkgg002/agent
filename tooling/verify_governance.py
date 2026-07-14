#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# verify_governance.py — Bộ Process Linter tự động để kiểm soát quy trình Agent.
# Usage: python3 agent/tooling/verify_governance.py [--workspace <workspace_name>]

import os
import sys
import re
import glob
from datetime import datetime

ROOT_DIR = "/Users/trainguyen/Documents/work"
WORKSPACES_DIR = os.path.join(ROOT_DIR, "agent/memory/workspaces")
LESSONS_PATH = os.path.join(ROOT_DIR, "agent/memory/global/lessons.md")

def log_info(msg):
    print(f"🟢 [GOVERNANCE] {msg}")

def log_error(msg):
    print(f"🔴 [GOVERNANCE] ERROR: {msg}", file=sys.stderr)

def get_active_workspace(ws_arg=None):
    if ws_arg:
        target_dir = os.path.join(WORKSPACES_DIR, ws_arg)
        if os.path.isdir(target_dir):
            return ws_arg
        log_error(f"Workspace chỉ định '{ws_arg}' không tồn tại.")
        sys.exit(1)

    # Quét tất cả các thư mục con trong workspaces
    if not os.path.exists(WORKSPACES_DIR):
        log_error(f"Không tìm thấy thư mục workspaces tại {WORKSPACES_DIR}")
        sys.exit(1)

    subdirs = [d for d in os.listdir(WORKSPACES_DIR) if os.path.isdir(os.path.join(WORKSPACES_DIR, d))]
    
    # Lọc bỏ các file ẩn/hệ thống
    subdirs = [d for d in subdirs if not d.startswith(".")]

    if not subdirs:
        log_error("Không tìm thấy bất kỳ thư mục workspace nào.")
        sys.exit(1)

    # Tìm workspace được sửa đổi gần nhất dựa trên mtime của các file bên trong nó
    latest_ws = None
    latest_mtime = 0

    for ws in subdirs:
        ws_path = os.path.join(WORKSPACES_DIR, ws)
        # Lấy file có mtime lớn nhất trong thư mục này
        files = glob.glob(os.path.join(ws_path, "*"))
        if not files:
            mtime = os.path.getmtime(ws_path)
        else:
            mtime = max(os.path.getmtime(f) for f in files)

        if mtime > latest_mtime:
            latest_mtime = mtime
            latest_ws = ws

    return latest_ws

def verify_workspace_docs(ws_name):
    ws_path = os.path.join(WORKSPACES_DIR, ws_name)
    log_info(f"Đang kiểm tra workspace: '{ws_name}'")

    # 1. Kiểm tra sự hiện diện của các tài liệu bắt buộc
    req_files = glob.glob(os.path.join(ws_path, "01_requirements*.md"))
    progress_files = glob.glob(os.path.join(ws_path, "05_progress*.md"))
    task_files = glob.glob(os.path.join(ws_path, "08_tasks*.md"))

    if not req_files:
        log_error(f"Thiếu file 01_requirements_*.md trong {ws_path}")
        return False
    if not progress_files:
        log_error(f"Thiếu file 05_progress_*.md trong {ws_path}")
        return False
    if not task_files:
        log_error(f"Thiếu file 08_tasks_*.md trong {ws_path}")
        return False

    # Kiểm tra sự hiện diện của file implementation_plan.md
    impl_plan = os.path.join(ws_path, "implementation_plan.md")
    if not os.path.exists(impl_plan):
        log_error(f"Thiếu file implementation_plan.md trong {ws_path} (Yêu cầu đồng bộ từ Artifact)")
        return False

    log_info("✓ Đầy đủ tài liệu bắt buộc (01_requirements, 05_progress, 08_tasks, implementation_plan.md).")

    # 2. Đọc progress file và kiểm tra format + log hôm nay
    progress_file = progress_files[0]
    today_str = datetime.now().strftime("%Y-%m-%d")
    
    # Định dạng mong đợi: - [YYYY-MM-DD...] [Agent:Model] Action
    # Chấp nhận cả múi giờ (ví dụ +07:00 hoặc Z)
    log_pattern = re.compile(rf'^-\s*\[{today_str}.*\]\s*\[Agent:.*\]')
    
    has_today_log = False
    malformed_lines = []

    with open(progress_file, "r", encoding="utf-8") as f:
        lines = f.readlines()
        for idx, line in enumerate(lines, 1):
            line_str = line.strip()
            if line_str.startswith("- ["):
                # Kiểm tra xem có đúng format [YYYY-MM-DD...] [Agent:...] hay không
                generic_pattern = re.compile(r'^-\s*\[\d{4}-\d{2}-\d{2}.*\]\s*\[Agent:[a-zA-Z0-9.\-_]+\]\s+.*')
                if not generic_pattern.match(line_str):
                    malformed_lines.append((idx, line_str))
                
                # Check xem có log của ngày hôm nay không
                if log_pattern.match(line_str):
                    has_today_log = True

    if malformed_lines:
        for idx, line in malformed_lines:
            log_error(f"Dòng {idx} trong {os.path.basename(progress_file)} sai format audit log:")
            print(f"   👉 '{line}'")
            print("   👉 Định dạng chuẩn: '- [YYYY-MM-DDTHH:MM:SS+ZZ:ZZ] [Agent:Model] Action'")
        return False

    if not has_today_log:
        log_error(f"Không tìm thấy dòng log nào của ngày hôm nay ({today_str}) trong {os.path.basename(progress_file)}")
        log_error("Bắt buộc phải cập nhật nhật ký tiến độ trước khi hoàn thành task!")
        return False

    log_info(f"✓ File progress log hợp lệ và đã cập nhật ngày hôm nay ({today_str}).")
    return True

def main():
    ws_arg = None
    if len(sys.argv) > 1:
        if sys.argv[1] == "--workspace" and len(sys.argv) > 2:
            ws_arg = sys.argv[2]
        else:
            ws_arg = sys.argv[1]

    ws_name = get_active_workspace(ws_arg)
    
    if not ws_name:
        log_error("Không tìm thấy workspace hoạt động nào.")
        sys.exit(1)

    success = verify_workspace_docs(ws_name)
    
    if success:
        print("════════════════════════════════════════════════")
        print(f" ⛳ GOVERNANCE AUDIT PASSED 🟢 (Workspace: {ws_name})")
        print("════════════════════════════════════════════════")
        sys.exit(0)
    else:
        print("════════════════════════════════════════════════")
        print(f" ⛳ GOVERNANCE AUDIT FAILED 🔴 (Workspace: {ws_name})")
        print("   Vui lòng sửa các lỗi quy trình trên trước khi bàn giao task!")
        print("════════════════════════════════════════════════")
        sys.exit(1)

if __name__ == "__main__":
    main()
