#!/bin/bash

# Đường dẫn thư mục chứa 781 skills của anh
SKILLS_DIR="/Users/trainguyen/.gemini/config/skills"
DEST_DIR="$SKILLS_DIR/categorized"

echo "🚀 Bắt đầu quy hoạch và dọn dẹp 781 skills (hỗ trợ cả File & Folder)..."

# 1. Tạo các thư mục chuyên đề (Domains)
CATEGORIES=(
  "00_Core_Global"
  "01_Security_Pentest"
  "02_Frontend_Mobile"
  "03_Backend_Data"
  "04_DevOps_Cloud"
  "05_AI_Agents_LLM"
  "06_Architecture_CodeQuality"
  "07_Testing_QA"
  "08_Marketing_SEO_Business"
  "09_Workflow_Productivity"
  "10_Uncategorized"
)

for cat in "${CATEGORIES[@]}"; do
  mkdir -p "$DEST_DIR/$cat"
done

# Bật chế độ không phân biệt hoa thường cho case matching
shopt -s nocasematch

# 2. Quét và di chuyển
cd "$SKILLS_DIR" || exit

# Lặp qua tất cả items (bỏ qua thư mục categorized và chính file bash)
for item in *; do
  if [ "$item" != "categorized" ] && [ "$item" != "organize_skills.sh" ]; then
    case "$item" in
      *prompt-optimizer*|*agent-memory*|*code-review-excellence*|*clean-code*|*context-manager*|*antigravity*)
        mv "$item" "$DEST_DIR/00_Core_Global/"
        ;;
      *security*|*pentest*|*hack*|*vulnerabilit*|*injection*|*xss*|*auth*|*metasploit*|*privilege*|*shodan*|*wireshark*|*burp*|*red-team*|*red_team*|*malware*|*forensics*|*threat*)
        mv "$item" "$DEST_DIR/01_Security_Pentest/"
        ;;
      *react*|*ui*|*ux*|*frontend*|*tailwind*|*swift*|*flutter*|*android*|*ios*|*vue*|*css*|*html*|*vite*|*nextjs*|*nuxt*|*avalonia*|*design*|*canvas*|*mobile*)
        mv "$item" "$DEST_DIR/02_Frontend_Mobile/"
        ;;
      *backend*|*node*|*python*|*go-*|*golang*|*rust*|*cpp*|*c-pro*|*csharp*|*java*|*sql*|*database*|*postgres*|*clickhouse*|*api*|*graphql*|*prisma*|*nestjs*|*django*|*laravel*|*springboot*|*supabase*|*db-*|*nosql*|*data-*|*scala*|*elixir*|*fastapi*)
        mv "$item" "$DEST_DIR/03_Backend_Data/"
        ;;
      *kubernetes*|*k8s*|*docker*|*terraform*|*aws*|*gcp*|*azure*|*cloud*|*ci-cd*|*deployment*|*github*|*gitlab*|*linux*|*bash*|*shell*|*devops*|*observability*|*prometheus*|*grafana*|*helm*|*istio*|*serverless*|*infra*)
        mv "$item" "$DEST_DIR/04_DevOps_Cloud/"
        ;;
      *agent*|*llm*|*prompt*|*claude*|*crewai*|*langchain*|*rag*|*ml-*|*hugging*|*model*|*ai-*|*gpt*|*notebooklm*|*vector*|*pytorch*|*machine-learning*)
        mv "$item" "$DEST_DIR/05_AI_Agents_LLM/"
        ;;
      *architecture*|*refactor*|*pattern*|*c4-*|*hexagonal*|*cqrs*|*system*|*monorepo*|*legacy-modernizer*)
        mv "$item" "$DEST_DIR/06_Architecture_CodeQuality/"
        ;;
      *test*|*qa*|*tdd*|*jest*|*cypress*|*playwright*|*debug*|*error*)
        mv "$item" "$DEST_DIR/07_Testing_QA/"
        ;;
      *seo*|*marketing*|*cro*|*copywrit*|*sales*|*startup*|*finance*|*business*|*pricing*|*content*|*ads*|*brand*|*customer*|*investor*|*billing*)
        mv "$item" "$DEST_DIR/08_Marketing_SEO_Business/"
        ;;
      *workflow*|*automation*|*zapier*|*slack*|*telegram*|*notion*|*doc*|*pdf*|*xlsx*|*csv*|*bot*|*discord*|*whatsapp*)
        mv "$item" "$DEST_DIR/09_Workflow_Productivity/"
        ;;
      *)
        mv "$item" "$DEST_DIR/10_Uncategorized/"
        ;;
    esac
  fi
done

# Tắt chế độ nocasematch
shopt -u nocasematch

echo "✅ Hoàn tất! Đã phân loại xong."

/Users/trainguyen/.gemini/antigravity/builtin/skills/antigravity_guide/SKILL.md