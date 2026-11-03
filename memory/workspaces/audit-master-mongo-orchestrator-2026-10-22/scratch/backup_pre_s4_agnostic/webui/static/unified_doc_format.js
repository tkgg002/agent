// ============================================================================
// UNIFIED DOCUMENT & CO-ARCHITECT PRESENTATION ENGINE (UDF-v1)
// Single-Responsibility Renderer for #sri-markdown-content & .cas-messages-list
// ============================================================================

(function () {
  "use strict";

  function escapeUdfHtml(str) {
    return String(str || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function formatInlineUdfMarkdown(text) {
    let safe = escapeUdfHtml(text);
    // Inline code `...`
    safe = safe.replace(/`([^`]+)`/g, function (_, codeText) {
      const trimmed = codeText.trim();
      let cls = "udf-code-inline";
      if (
        /[a-zA-Z0-9_./-]+\.(go|ts|tsx|js|css|html|py|sql|md|json)(:\d+(-\d+)?)?/.test(trimmed) ||
        /^REF:\s*/.test(trimmed)
      ) {
        cls += " is-file-ref";
      } else if (/^(PASS|DONE|VERIFIED|CLEAN|✅|🟢)/i.test(trimmed) || trimmed.includes("PASS 🟢")) {
        cls += " is-status-pass";
      }
      return '<code class="' + cls + '">' + codeText + "</code>";
    });
    // Bold **...**
    safe = safe.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
    // Italic *...*
    safe = safe.replace(/(^|[\s(])\*([^*\n]+)\*(?=[\s).,;:!?]|$)/g, "$1<em>$2</em>");
    return safe;
  }

  function formatCodeBlockContent(codeStr, lang) {
    const lines = String(codeStr || "").split("\n");
    const isDiff =
      lang === "diff" ||
      lang === "patch" ||
      lines.some((l) => l.startsWith("diff --git ") || l.startsWith("@@ ") || l.startsWith("--- a/"));
    if (!isDiff) {
      return escapeUdfHtml(codeStr);
    }
    return lines
      .map((ln) => {
        const esc = escapeUdfHtml(ln);
        if (ln.startsWith("+") && !ln.startsWith("+++")) {
          return '<span class="udf-diff-add">' + esc + "</span>";
        }
        if (ln.startsWith("-") && !ln.startsWith("---")) {
          return '<span class="udf-diff-del">' + esc + "</span>";
        }
        if (ln.startsWith("@@") || ln.startsWith("diff --git")) {
          return '<span class="udf-diff-hunk">' + esc + "</span>";
        }
        return esc;
      })
      .join("\n");
  }

  function renderMarkdownTable(tableLines) {
    if (!tableLines || tableLines.length < 2) return "";
    const parseRow = (line) =>
      line
        .trim()
        .replace(/^\|/, "")
        .replace(/\|$/, "")
        .split("|")
        .map((c) => c.trim());

    const headers = parseRow(tableLines[0]);
    let startBodyIdx = 1;
    if (tableLines[1] && /^[\s|:-]+$/.test(tableLines[1])) {
      startBodyIdx = 2;
    }

    let html = '<div class="udf-table-wrap"><table class="udf-table"><thead><tr>';
    for (const h of headers) {
      html += "<th>" + formatInlineUdfMarkdown(h) + "</th>";
    }
    html += "</tr></thead><tbody>";

    for (let i = startBodyIdx; i < tableLines.length; i++) {
      const rowCells = parseRow(tableLines[i]);
      if (rowCells.length === 0 || (rowCells.length === 1 && !rowCells[0])) continue;
      html += "<tr>";
      for (let c = 0; c < headers.length; c++) {
        const val = rowCells[c] !== undefined ? rowCells[c] : "";
        html += "<td>" + formatInlineUdfMarkdown(val) + "</td>";
      }
      html += "</tr>";
    }
    html += "</tbody></table></div>";
    return html;
  }

  function renderSectionBodyLines(lines) {
    let html = "";
    let i = 0;

    while (i < lines.length) {
      const rawLn = lines[i];
      const trimLn = rawLn.trim();

      if (!trimLn || trimLn === "---") {
        i++;
        continue;
      }

      // Fenced code block ```...
      if (trimLn.startsWith("```")) {
        const lang = trimLn.slice(3).trim() || "code";
        const codeLines = [];
        i++;
        while (i < lines.length && !lines[i].trim().startsWith("```")) {
          codeLines.push(lines[i]);
          i++;
        }
        if (i < lines.length) i++; // skip closing ```
        const codeContent = codeLines.join("\n");
        html +=
          '<div class="udf-code-box">' +
          '<div class="udf-code-hdr"><span>' +
          escapeUdfHtml(lang.toUpperCase()) +
          "</span></div>" +
          '<pre class="udf-code-pre">' +
          formatCodeBlockContent(codeContent, lang.toLowerCase()) +
          "</pre></div>";
        continue;
      }

      // Markdown table | ... |
      if (trimLn.startsWith("|") && trimLn.endsWith("|")) {
        const tblLines = [];
        while (i < lines.length && lines[i].trim().startsWith("|")) {
          tblLines.push(lines[i]);
          i++;
        }
        html += renderMarkdownTable(tblLines);
        continue;
      }

      // Sub-heading ### ...
      if (trimLn.startsWith("### ")) {
        html += '<div class="udf-sub-heading">' + formatInlineUdfMarkdown(trimLn.slice(4)) + "</div>";
        i++;
        continue;
      }

      // Bullet / numbered list
      if (/^([-*•]|\d+\.)\s+/.test(trimLn)) {
        html += '<ul class="udf-list">';
        while (i < lines.length) {
          const itemTrim = lines[i].trim();
          if (!itemTrim) {
            i++;
            break;
          }
          if (/^([-*•]|\d+\.)\s+/.test(itemTrim)) {
            const cleanItem = itemTrim.replace(/^([-*•]|\d+\.)\s+/, "");
            html += "<li>" + formatInlineUdfMarkdown(cleanItem) + "</li>";
            i++;
          } else if ( lines[i].startsWith("  ") && !itemTrim.startsWith("#") && !itemTrim.startsWith("|") ) {
            const subClean = itemTrim.replace(/^([-*•]|\d+\.)\s+/, "");
            html += '<li style="margin-left:14px;list-style-type:circle;">' + formatInlineUdfMarkdown(subClean) + "</li>";
            i++;
          } else {
            break;
          }
        }
        html += "</ul>";
        continue;
      }

      // Regular paragraph / blockquote
      const cleanPara = trimLn.replace(/^>\s*/, "");
      html += '<p class="udf-p">' + formatInlineUdfMarkdown(cleanPara) + "</p>";
      i++;
    }

    return html;
  }

  function classifySectionCardClass(headingText, index) {
    const lower = String(headingText || "").toLowerCase();
    if (lower.includes("1.") || lower.includes("tóm tắt") || lower.includes("root cause") || lower.includes("nguyên nhân") || lower.includes("định danh")) {
      return "sec-1";
    }
    if (lower.includes("2.") || lower.includes("kiến trúc") || lower.includes("giải pháp") || lower.includes("điểm chạm") || lower.includes("bảng") || lower.includes("sub-task")) {
      return "sec-2";
    }
    if (lower.includes("3.") || lower.includes("code demo") || lower.includes("diff") || lower.includes("snippet") || lower.includes("trích xuất")) {
      return "sec-3";
    }
    if (lower.includes("4.") || lower.includes("kiểm định") || lower.includes("gate") || lower.includes("clean-room") || lower.includes("ràng buộc")) {
      return "sec-4";
    }
    return "sec-" + (((index % 4) + 1));
  }

  function renderUnifiedDocFormatHTML(rawText) {
    const raw = String(rawText || "").trim();
    if (!raw) {
      return '<div class="udf-p">Chưa có nội dung báo cáo.</div>';
    }

    // Raw git diff fallback
    if (raw.startsWith("diff --git ") || raw.startsWith("--- a/")) {
      return (
        '<div class="udf-section-card sec-3">' +
        '<div class="udf-section-hdr"><span>💻 Git Working Tree &amp; Isolated Patch Diff</span></div>' +
        '<div class="udf-section-body">' +
        '<div class="udf-code-box"><div class="udf-code-hdr"><span>GIT DIFF</span></div>' +
        '<pre class="udf-code-pre">' +
        formatCodeBlockContent(raw, "diff") +
        "</pre></div></div></div>"
      );
    }

    let lines = raw.split("\n");
    const metaChips = [];

    // 1. Extract YAML frontmatter if present
    if (lines[0] && lines[0].trim() === "---") {
      let endYaml = -1;
      for (let j = 1; j < Math.min(lines.length, 30); j++) {
        if (lines[j].trim() === "---") {
          endYaml = j;
          break;
        }
      }
      if (endYaml > 0) {
        const yamlMap = {};
        for (let j = 1; j < endYaml; j++) {
          const m = lines[j].match(/^([a-z0-9_]+):\s*(.+)$/i);
          if (m) yamlMap[m[1].trim()] = m[2].trim();
        }
        if (yamlMap.task_key) metaChips.push("<b>Task:</b> " + escapeUdfHtml(yamlMap.task_key));
        if (yamlMap.stage) metaChips.push("<b>Stage:</b> " + escapeUdfHtml(yamlMap.stage));
        if (yamlMap.tier || yamlMap.mode) {
          metaChips.push("<b>Tier/Mode:</b> " + escapeUdfHtml((yamlMap.tier || "T2") + " · " + (yamlMap.mode || "standard")));
        }
        if (yamlMap.model) metaChips.push("<b>Model:</b> " + escapeUdfHtml(yamlMap.model));
        if (yamlMap.laya_health_s0) metaChips.push("<b>Laya :8200:</b> " + escapeUdfHtml(yamlMap.laya_health_s0));
        lines = lines.slice(endYaml + 1);
      }
    }

    // 2. Extract top H1 title & "> 📌 ..." metadata strip
    let docTitle = "";
    const bodyLines = [];
    for (let i = 0; i < lines.length; i++) {
      const t = lines[i].trim();
      if (!docTitle && t.startsWith("# ")) {
        docTitle = t.slice(2).trim();
        continue;
      }
      if (t.startsWith("> 📌") || (t.startsWith("> ") && t.includes("·") && t.includes("**"))) {
        const cleanStrip = t.replace(/^>\s*📌?\s*/, "");
        const parts = cleanStrip.split("·").map((p) => p.trim()).filter(Boolean);
        for (const p of parts) {
          metaChips.push(formatInlineUdfMarkdown(p));
        }
        continue;
      }
      bodyLines.push(lines[i]);
    }

    // 3. Split bodyLines into Canonical Sections (by "## " or top-level "### 1.." when no "## " exists)
    const hasH2 = bodyLines.some((l) => l.trim().startsWith("## "));
    const sectionPrefix = hasH2 ? "## " : "### ";
    const sections = [];
    let currentSec = { title: "", lines: [] };

    for (const ln of bodyLines) {
      const t = ln.trim();
      if (t.startsWith(sectionPrefix)) {
        if (currentSec.title || currentSec.lines.some((x) => x.trim() !== "")) {
          sections.push(currentSec);
        }
        currentSec = { title: t.slice(sectionPrefix.length).trim(), lines: [] };
      } else {
        currentSec.lines.push(ln);
      }
    }
    if (currentSec.title || currentSec.lines.some((x) => x.trim() !== "")) {
      sections.push(currentSec);
    }

    let outHtml = "";
    if (docTitle) {
      outHtml += '<div class="udf-doc-title">' + formatInlineUdfMarkdown(docTitle) + "</div>";
    }
    if (metaChips.length > 0) {
      const uniqueChips = Array.from(new Set(metaChips)).slice(0, 7);
      outHtml +=
        '<div class="udf-meta-strip">' +
        uniqueChips.map((c) => '<span class="udf-meta-chip">' + c + "</span>").join("") +
        "</div>";
    }

    sections.forEach((sec, idx) => {
      const bodyHtml = renderSectionBodyLines(sec.lines);
      if (!sec.title && !bodyHtml) return;
      if (!sec.title) {
        outHtml +=
          '<div class="udf-section-card sec-1"><div class="udf-section-body">' +
          bodyHtml +
          "</div></div>";
        return;
      }
      const cardCls = classifySectionCardClass(sec.title, idx);
      outHtml +=
        '<div class="udf-section-card ' +
        cardCls +
        '">' +
        '<div class="udf-section-hdr"><span>' +
        formatInlineUdfMarkdown(sec.title) +
        "</span></div>" +
        '<div class="udf-section-body">' +
        bodyHtml +
        "</div></div>";
    });

    return outHtml || '<div class="udf-p">' + formatInlineUdfMarkdown(raw) + "</div>";
  }

  let isApplyingUdf = false;

  function applyUnifiedFormatToAllViews() {
    if (isApplyingUdf) return;
    isApplyingUdf = true;
    try {
      const mdEl = document.getElementById("sri-markdown-content");
      if (mdEl && !mdEl.querySelector(".udf-section-card, .udf-doc-title, .udf-meta-strip")) {
        const rawText = mdEl.textContent || "";
        if (rawText.trim()) {
          mdEl.classList.add("udf-doc-root");
          mdEl.innerHTML = renderUnifiedDocFormatHTML(rawText);
        }
      }

      const chatBodies = document.querySelectorAll("#co-architect-chat-messages .cas-msg-body");
      chatBodies.forEach((bodyEl) => {
        if (bodyEl.dataset.udfRendered === "1") return;
        const rawMsg = bodyEl.textContent || "";
        bodyEl.classList.add("udf-doc-root");
        bodyEl.innerHTML = renderUnifiedDocFormatHTML(rawMsg);
        bodyEl.dataset.udfRendered = "1";
      });
    } finally {
      isApplyingUdf = false;
    }
  }

  window.renderUnifiedDocFormatHTML = renderUnifiedDocFormatHTML;
  window.applyUnifiedFormatToAllViews = applyUnifiedFormatToAllViews;

  function installUnifiedFormatHooks() {
    if (typeof window.renderStepReportInspector === "function" && !window.renderStepReportInspector.__udfWrapped) {
      const origSri = window.renderStepReportInspector;
      const wrappedSri = function () {
        const res = origSri.apply(this, arguments);
        applyUnifiedFormatToAllViews();
        return res;
      };
      wrappedSri.__udfWrapped = true;
      window.renderStepReportInspector = wrappedSri;
    }

    if (typeof window.renderCoArchitectStudio === "function" && !window.renderCoArchitectStudio.__udfWrapped) {
      const origCas = window.renderCoArchitectStudio;
      const wrappedCas = function () {
        const res = origCas.apply(this, arguments);
        applyUnifiedFormatToAllViews();
        return res;
      };
      wrappedCas.__udfWrapped = true;
      window.renderCoArchitectStudio = wrappedCas;
    }

    if (typeof MutationObserver !== "undefined") {
      const obs = new MutationObserver(function () {
        if (!isApplyingUdf) {
          applyUnifiedFormatToAllViews();
        }
      });
      const mdEl = document.getElementById("sri-markdown-content");
      if (mdEl) obs.observe(mdEl, { childList: true, characterData: true, subtree: true });
      const chatEl = document.getElementById("co-architect-chat-messages");
      if (chatEl) obs.observe(chatEl, { childList: true });
    }

    applyUnifiedFormatToAllViews();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", installUnifiedFormatHooks);
  } else {
    installUnifiedFormatHooks();
  }
})();
