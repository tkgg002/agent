let dashboardState = null;
let selectedChainKey = null;
let selectedAgentSlot = null;
let selectedStepNum = null;
let selectedReportTab = "S3";
let userLockedReportTab = false;
let selectedSubTaskRepoFilter = "";
let selectedSubTaskID = "";
let inspectingSubTaskOverride = null;
let dimMode = "collapse";
let waveViewMode = "subtasks";
let selectedHierarchyEpic = "";
let activeRunnerFilter = "";
let currentScreenMode = "autopilot";
let isLifecycleActionRunning = false;
let runningActionTaskKey = "";
let runningActionSubTaskID = "";
let currentActionAbortController = null;
let liveElapsedTimerHandle = null;
let liveElapsedStartMs = 0;
let liveElapsedBaseSec = 0;

function formatElapsedSeconds(sec) {
  const total = Math.max(0, Math.floor(Number(sec) || 0));
  const m = String(Math.floor(total / 60)).padStart(2, "0");
  const s = String(total % 60).padStart(2, "0");
  return `${m}:${s}s`;
}

function startLiveElapsedTicker(initialSec) {
  liveElapsedBaseSec = Math.max(0, Number(initialSec) || 0);
  liveElapsedStartMs = Date.now();
  if (liveElapsedTimerHandle) clearInterval(liveElapsedTimerHandle);
  liveElapsedTimerHandle = setInterval(() => {
    const delta = Math.floor((Date.now() - liveElapsedStartMs) / 1000);
    const curSec = liveElapsedBaseSec + Math.max(0, delta);
    const formatted = formatElapsedSeconds(curSec);
    const grbEl = document.getElementById("grb-elapsed-pill");
    if (grbEl) grbEl.textContent = `⏱ ${formatted}`;
    const hudEl = document.getElementById("hud-elapsed-sec");
    if (hudEl) hudEl.textContent = formatted;
  }, 500);
}

function stopLiveElapsedTicker() {
  if (liveElapsedTimerHandle) {
    clearInterval(liveElapsedTimerHandle);
    liveElapsedTimerHandle = null;
  }
}

async function fetchDashboard(optionalTaskKey) {
  try {
    if (optionalTaskKey) selectedChainKey = optionalTaskKey;
    const targetKey = optionalTaskKey || selectedChainKey;
    const res = await fetch(targetKey ? `/api/dashboard?task_key=${encodeURIComponent(targetKey)}` : "/api/dashboard");
    if (!res.ok) return;
    dashboardState = await res.json();
    if (dashboardState.active_task_key && !selectedChainKey) selectedChainKey = dashboardState.active_task_key;
    syncDefaultReportTabForStage();
    renderAll();
  } catch (err) {
    console.error("fetchDashboard error:", err);
  }
}

function syncDefaultReportTabForStage() {
  if (!dashboardState || userLockedReportTab) return;
  const st = dashboardState.active_task_stage || "READY";
  if (st === "NEEDS_YOU") {
    selectedReportTab = "S3";
  } else if (st === "DONE") {
    selectedReportTab = "S5";
  } else if (st === "RUNNING" || st === "UNDER_WAY") {
    selectedReportTab = "S4";
  } else if (st === "IN_REVIEW") {
    selectedReportTab = "S5";
  } else {
    selectedReportTab = "S1";
  }
}

function connectSSE() {
  try {
    const es = new EventSource("/api/stream");
    es.addEventListener("update", () => {
      fetchDashboard();
    });
    es.addEventListener("step_progress", () => {
      fetchDashboard();
    });
    es.addEventListener("task_stopped", () => {
      fetchDashboard();
    });
    es.addEventListener("task_deleted", () => {
      fetchDashboard();
    });
  } catch (e) {
    console.warn("SSE fallback polling:", e);
  }
}

function getSelectedRunner() {
  const sel = document.getElementById("global-runner-select");
  return sel?.value || dashboardState?.active_runner || "antigravity";
}

async function changeActiveRunner(runnerVal) {
  if (!runnerVal) return;
  try {
    const res = await fetch(`/api/dashboard?runner=${encodeURIComponent(runnerVal)}`);
    if (!res.ok) return;
    dashboardState = await res.json();
    renderAll();
  } catch (e) {
    console.error("changeActiveRunner error:", e);
  }
}

async function selectActiveTask(taskKey) {
  if (!taskKey) return;
  selectedChainKey = taskKey;
  selectedSubTaskRepoFilter = "";
  userLockedReportTab = false;
  await runTaskAction(taskKey, "select");
}

function triggerPrimaryTaskAction() {
  const stage = dashboardState?.active_task_stage || "READY";
  if (stage === "NEEDS_YOU") {
    return runTaskAction("", "approve_and_run");
  }
  return runTaskAction("", "autopilot");
}

async function stopRunningProcess(taskKey, subTaskID, ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  const lsp = dashboardState?.live_step_progress || {};
  const targetKey =
    taskKey ||
    runningActionTaskKey ||
    lsp.task_key ||
    selectedChainKey ||
    dashboardState?.active_task_key ||
    "";
  const targetSub = subTaskID || runningActionSubTaskID || lsp.sub_task_id || "";

  if (currentActionAbortController) {
    try {
      currentActionAbortController.abort();
    } catch (_) {}
    currentActionAbortController = null;
  }
  isLifecycleActionRunning = false;
  runningActionTaskKey = "";
  runningActionSubTaskID = "";
  stopLiveElapsedTicker();

  try {
    const res = await fetch("/api/tasks/stop", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        task_key: targetKey,
        sub_task_id: targetSub,
        reason: "Người dùng bấm nút [⏹ Dừng Tiến Trình] trên giao diện WebUI",
      }),
    });
    if (res.ok) {
      dashboardState = await res.json();
      selectedChainKey = dashboardState.active_task_key || targetKey;
      renderAll();
    } else {
      await fetchDashboard();
    }
  } catch (e) {
    console.error("stopRunningProcess error:", e);
    await fetchDashboard();
  }
}

async function deleteSingleTask(taskKey, ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  const targetKey = taskKey || selectedChainKey || dashboardState?.active_task_key || "";
  if (!targetKey) return;

  try {
    const res = await fetch("/api/tasks/delete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        mode: "single",
        task_key: targetKey,
      }),
    });
    if (!res.ok) {
      alert("Không thể xóa Task: " + (await res.text()));
      return;
    }
    dashboardState = await res.json();
    selectedChainKey = dashboardState.active_task_key || "";
    selectedSubTaskID = "";
    renderAll();
  } catch (e) {
    console.error("deleteSingleTask error:", e);
  }
}

async function deleteDoneTasksBulk(ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  try {
    const res = await fetch("/api/tasks/delete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        mode: "done",
        task_key: dashboardState?.active_task_key || "",
      }),
    });
    if (!res.ok) {
      alert("Không thể dọn các Task DONE: " + (await res.text()));
      return;
    }
    dashboardState = await res.json();
    selectedChainKey = dashboardState.active_task_key || "";
    renderAll();
  } catch (e) {
    console.error("deleteDoneTasksBulk error:", e);
  }
}

async function deleteSingleSubTask(taskKey, subTaskID, ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  const targetKey = taskKey || selectedChainKey || dashboardState?.active_task_key || "";
  if (!targetKey || !subTaskID) return;
  try {
    const res = await fetch("/api/tasks/delete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        mode: "subtask",
        task_key: targetKey,
        sub_task_id: subTaskID,
      }),
    });
    if (!res.ok) {
      alert("Không thể xóa Sub-Task: " + (await res.text()));
      return;
    }
    dashboardState = await res.json();
    if (selectedSubTaskID === subTaskID) {
      selectedSubTaskID = "";
    }
    renderAll();
  } catch (e) {
    console.error("deleteSingleSubTask error:", e);
  }
}

async function runTaskAction(taskKey, action, ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  const targetKey = taskKey || selectedChainKey || dashboardState?.active_task_key || "";
  const runner = getSelectedRunner();
  const isExecutionStep = Boolean(action && action !== "select" && action !== "stop" && action !== "delete");
  const stepDelayMs = isExecutionStep ? 3000 : 0;

  let pollTimer = null;
  if (isExecutionStep) {
    isLifecycleActionRunning = true;
    runningActionTaskKey = targetKey;
    runningActionSubTaskID = "";
    currentActionAbortController = new AbortController();
    startLiveElapsedTicker(0);
    renderCrossContextBar();
    renderGlobalRunningProgressBanner();
    renderSidebarTaskQueue();
    renderLiveTelemetryHUD();
    pollTimer = setInterval(() => {
      fetchDashboard();
    }, 800);
  }

  try {
    const res = await fetch("/api/tasks/action", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      signal: currentActionAbortController ? currentActionAbortController.signal : undefined,
      body: JSON.stringify({
        task_key: targetKey,
        action: action || "autopilot",
        runner,
        step_delay_ms: stepDelayMs,
      }),
    });
    if (!res.ok) {
      const msg = await res.text();
      alert("Không thể thực thi thao tác trên task: " + msg);
      return;
    }
    dashboardState = await res.json();
    selectedChainKey = dashboardState.active_task_key || targetKey;
    if (action === "plan" || action === "autopilot") {
      userLockedReportTab = false;
      if (dashboardState.active_task_stage === "NEEDS_YOU") {
        selectedReportTab = "S3";
      }
    } else if (action === "approve_and_run" || action === "qa") {
      selectedReportTab = dashboardState.active_task_stage === "NEEDS_YOU" ? "S3" : "S5";
      userLockedReportTab = true;
    } else if (action === "implement") {
      selectedReportTab = "S4";
      userLockedReportTab = true;
    } else {
      syncDefaultReportTabForStage();
    }
    if (action && action !== "select" && currentScreenMode !== "autopilot") {
      switchScreen("autopilot");
    } else {
      renderAll();
    }
    if (isExecutionStep) {
      const chatBox = document.getElementById("co-architect-chat-messages");
      if (chatBox) {
        chatBox.scrollTop = chatBox.scrollHeight;
      }
    }
  } catch (e) {
    if (e && e.name === "AbortError") {
      await fetchDashboard();
      return;
    }
    console.error("runTaskAction error:", e);
  } finally {
    if (pollTimer) clearInterval(pollTimer);
    isLifecycleActionRunning = false;
    runningActionTaskKey = "";
    runningActionSubTaskID = "";
    currentActionAbortController = null;
    if (!dashboardState?.live_step_progress?.active) {
      stopLiveElapsedTicker();
    }
    renderAll();
  }
}

function switchScreen(mode) {
  if (mode === "all") mode = "autopilot";
  currentScreenMode = mode;
  document.querySelectorAll(".screen-tab").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.screen === mode);
  });
  const sWave = document.getElementById("screen-wave");
  const sAuto = document.getElementById("screen-autopilot");
  const sVibe = document.getElementById("screen-vibe");

  if (sWave) sWave.classList.toggle("hidden", mode !== "wave");
  if (sAuto) sAuto.classList.toggle("hidden", mode !== "autopilot");
  if (sVibe) sVibe.classList.toggle("hidden", mode !== "vibe");

  if (mode === "wave") {
    requestAnimationFrame(() => drawWaveBezierEdges());
  }
}

function selectHierarchyEpic(epicCode) {
  const normalized = String(epicCode || "").toUpperCase();
  selectedHierarchyEpic = selectedHierarchyEpic === normalized && epicCode !== "" ? "" : normalized;
  const fE = document.getElementById("filter-epic");
  if (fE && fE.value !== selectedHierarchyEpic) {
    fE.value = selectedHierarchyEpic;
  }
  renderWaveBoard();
}

async function selectHierarchyTask(taskKey) {
  if (!taskKey) return;
  selectedChainKey = taskKey;
  await runTaskAction(taskKey, "select");
  if (currentScreenMode === "wave") {
    requestAnimationFrame(() => drawWaveBezierEdges());
  }
}

function setWaveMode(mode) {
  waveViewMode = mode;
  renderWaveBoard();
}

function setDimMode(mode) {
  dimMode = mode;
  renderWaveBoard();
}

function resetWaveHighlight() {
  selectedChainKey = dashboardState?.active_task_key || null;
  selectedHierarchyEpic = "";
  selectedAgentSlot = null;
  selectedStepNum = null;
  const fA = document.getElementById("filter-assignee");
  const fM = document.getElementById("filter-milestone");
  const fE = document.getElementById("filter-epic");
  const fS = document.getElementById("wave-search");
  if (fA) fA.value = "";
  if (fM) fM.value = "";
  if (fE) fE.value = "";
  if (fS) fS.value = "";
  renderAll();
}

function clearCrossSelection() {
  selectedAgentSlot = null;
  selectedStepNum = null;
  userLockedReportTab = false;
  renderAll();
}

function getAllWaveNodes() {
  const out = [];
  if (!dashboardState) return out;
  for (const wCol of dashboardState.waves || []) {
    for (const n of wCol.nodes || []) {
      out.push(n);
    }
  }
  return out;
}

function findWaveNodeByKey(taskKey) {
  if (!dashboardState || !taskKey) return null;
  for (const wCol of dashboardState.waves || []) {
    for (const n of wCol.nodes || []) {
      if (n.key === taskKey) return n;
    }
  }
  return null;
}

async function selectWaveCard(key) {
  // If user clicked a subtask key like TASK-9.2, keep parent task active
  const baseKey = String(key || "").includes(".") ? String(key).split(".")[0] : key;
  selectedChainKey = baseKey;
  const node = findWaveNodeByKey(baseKey);
  if (node) {
    selectedAgentSlot = node.agent_slot || null;
    selectedStepNum = node.workflow_step_num || null;
  }
  await runTaskAction(baseKey, "select");
}

function selectCrossAgent(slot, taskKey) {
  selectedAgentSlot = slot;
  if (taskKey) {
    selectedChainKey = taskKey;
    const node = findWaveNodeByKey(taskKey);
    if (node && node.workflow_step_num) {
      selectedStepNum = node.workflow_step_num;
    }
  }
  switchScreen("vibe");
  renderAll();
  setTimeout(() => {
    const paneEl = document.getElementById(`vp-slot-${slot}`);
    if (paneEl) {
      paneEl.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
  }, 50);
}

function mapStepNumToReportTab(stepNum) {
  if (stepNum <= 2) return "S1";
  if (stepNum === 3) return "S2";
  if (stepNum === 4) return "DG0";
  if (stepNum <= 10) return "S3";
  if (stepNum <= 12) return "S4";
  return "S5";
}

function selectCrossStep(stepNum) {
  selectedStepNum = stepNum;
  selectedReportTab = mapStepNumToReportTab(stepNum);
  userLockedReportTab = true;
  if (currentScreenMode !== "autopilot") {
    switchScreen("autopilot");
  }
  renderAll();
}

function selectReportTab(stepID) {
  selectedReportTab = stepID || "S3";
  userLockedReportTab = true;
  renderStepReportInspector();
}

async function selectTaskFromAutopilot(taskKey, agentSlot) {
  if (!taskKey) return;
  selectedChainKey = taskKey;
  if (agentSlot) selectedAgentSlot = agentSlot;
  const node = findWaveNodeByKey(taskKey);
  if (node && node.workflow_step_num) {
    selectedStepNum = node.workflow_step_num;
  }
  await runTaskAction(taskKey, "select");
}

async function jumpToScreenWithTask(ev, screenMode, taskKey, agentSlot) {
  if (ev) ev.stopPropagation();
  if (taskKey) {
    const baseKey = String(taskKey).includes(".") ? String(taskKey).split(".")[0] : taskKey;
    selectedChainKey = baseKey;
    await runTaskAction(baseKey, "select");
  }
  if (agentSlot) {
    selectedAgentSlot = agentSlot;
  }
  switchScreen(screenMode);
  renderAll();
}

function computeConnectedChain(rootKey, edges) {
  const visited = new Set([rootKey]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const e of edges) {
      if (visited.has(e.from) && !visited.has(e.to)) {
        visited.add(e.to);
        changed = true;
      }
      if (visited.has(e.to) && !visited.has(e.from)) {
        visited.add(e.from);
        changed = true;
      }
    }
  }
  return visited;
}

function formatRepoShortName(repoPath) {
  if (!repoPath) return "ai-orchestrator";
  const parts = String(repoPath).split("/").filter(Boolean);
  if (parts.length >= 2 && parts[parts.length - 2] === "data-hub") {
    return `data-hub/${parts[parts.length - 1]}`;
  }
  return parts[parts.length - 1] || repoPath;
}

function formatRepoBasename(repoPath) {
  if (!repoPath) return "ai-orchestrator";
  const parts = String(repoPath).split("/").filter(Boolean);
  return parts[parts.length - 1] || repoPath;
}

function formatAffectedReposSummary(repos, fallbackRepo) {
  if (Array.isArray(repos) && repos.length > 0) {
    const names = repos.map((r) => formatRepoBasename(r));
    return names.join(", ");
  }
  return formatRepoShortName(fallbackRepo);
}

function renderCrossContextBar() {
  if (!dashboardState) return;
  const activeKey = dashboardState.active_task_key || selectedChainKey || "TASK-1";
  const activeStage = dashboardState.active_task_stage || "READY";
  const iso = dashboardState.active_isolation || {};
  const targetRepo = dashboardState.active_target_repo || iso.target_repo || "";
  const affectedRepos = dashboardState.active_affected_repos || [];
  const lsp = dashboardState.live_step_progress || {};
  const le = dashboardState.live_execution || {};
  const isAnyRunning = Boolean(lsp.active || le.is_running || isLifecycleActionRunning);

  const taskSelect = document.getElementById("active-task-select");
  const allNodes = getAllWaveNodes();
  if (taskSelect) {
    taskSelect.innerHTML = allNodes
      .map((n) => {
        const sel = n.key === activeKey ? "selected" : "";
        const repoShort = formatAffectedReposSummary(n.affected_repos, n.target_repo);
        const runMark = (lsp.active && lsp.task_key === n.key) || (runningActionTaskKey === n.key && isLifecycleActionRunning) || n.stage === "RUNNING" ? "⟳ ĐANG CHẠY · " : "";
        return `<option value="${escapeAttr(n.key)}" ${sel}>${escapeHtml(runMark)}${escapeHtml(n.key)} · ${escapeHtml(n.title)} [${escapeHtml(n.stage)} · ${escapeHtml(repoShort)}]</option>`;
      })
      .join("");
  }

  const stagePill = document.getElementById("active-task-stage-pill");
  if (stagePill) {
    const displayStage = isAnyRunning && (lsp.task_key === activeKey || runningActionTaskKey === activeKey) ? "RUNNING" : activeStage;
    stagePill.className = `stage-pill ${displayStage}`;
    stagePill.innerHTML = displayStage === "RUNNING" ? `<span class="spin-inline">⟳</span> RUNNING` : escapeHtml(displayStage);
  }

  setText("hdr-target-repo", formatAffectedReposSummary(affectedRepos, targetRepo));
  setText("hdr-iso-worktree", iso.worktree_path || `.orchestrator/worktrees/${activeKey}`);
  setText("hdr-iso-branch", iso.branch_name || `task/${activeKey}`);

  // Update primary Autopilot button labels depending on whether the task is at NEEDS_YOU
  const isWaitingApprove = activeStage === "NEEDS_YOU";
  const hdrAutoBtn = document.getElementById("hdr-primary-autopilot-btn");
  if (hdrAutoBtn) {
    hdrAutoBtn.textContent = isWaitingApprove
      ? `✓ APPROVE & Chạy S4–S5 (${activeKey})`
      : `▶ Chạy Autopilot (${activeKey})`;
  }
  const ctxApproveBtn = document.getElementById("ctx-btn-approve-run");
  if (ctxApproveBtn) {
    ctxApproveBtn.classList.toggle("pulse-approve", isWaitingApprove);
  }
  const ctxStopBtn = document.getElementById("ctx-btn-stop-task");
  if (ctxStopBtn) {
    ctxStopBtn.classList.toggle("hidden", !isAnyRunning);
  }
  const consoleRunBtn = document.getElementById("console-run-btn");
  if (consoleRunBtn) {
    consoleRunBtn.textContent = isWaitingApprove
      ? `✓ APPROVE & Chạy S4–S5 (${activeKey})`
      : `▶ Chạy Phase 1 S1–S3 (${activeKey})`;
  }

  const clearBtn = document.getElementById("btn-clear-cross-filter");
  if (clearBtn) {
    clearBtn.classList.toggle("hidden", !selectedAgentSlot && !selectedStepNum);
  }
}

function renderGlobalRunningProgressBanner() {
  const bannerEl = document.getElementById("global-running-progress-banner");
  if (!bannerEl || !dashboardState) return;

  const lsp = dashboardState.live_step_progress || {};
  const le = dashboardState.live_execution || {};
  const isRunning = Boolean(lsp.active || le.is_running || isLifecycleActionRunning);
  const wasStopped = Boolean(!isRunning && (lsp.was_stopped || le.was_stopped));

  if (!isRunning && !wasStopped) {
    bannerEl.classList.add("hidden");
    bannerEl.classList.remove("is-running", "was-stopped");
    return;
  }

  bannerEl.classList.remove("hidden");
  bannerEl.classList.toggle("is-running", isRunning);
  bannerEl.classList.toggle("was-stopped", wasStopped);

  const runTaskKey = lsp.task_key || runningActionTaskKey || dashboardState.active_task_key || "TASK-11";
  const nodeObj = findWaveNodeByKey(runTaskKey);
  const runTaskTitle = nodeObj?.title || dashboardState.active_task_title || "";
  const subId = lsp.sub_task_id || runningActionSubTaskID || (isRunning ? le.active_sub_task_id : "") || "";
  const subTitle = lsp.sub_task_title || (isRunning ? le.active_sub_task_title : "") || "";
  const stepNum = lsp.step_index || lsp.step_num || le.active_step_num || 1;
  const totalSteps = lsp.total_steps || le.total_workflow_steps || 17;
  const pct = Math.max(6, Math.min(100, Number(lsp.percent || le.overall_progress_pct || 18)));
  const liveStepSummary = [lsp.stage_label, lsp.current_action].filter(Boolean).join(" — ");
  const detail = wasStopped
    ? lsp.stop_reason || le.stop_reason || "Đã dừng tiến trình theo yêu cầu người dùng — đang tạm dừng ở NEEDS_YOU"
    : liveStepSummary || lsp.detail || le.active_step_title || "Đang thực thi từng bước quy trình /agent...";
  const roleSlotText =
    lsp.active_role_slot ||
    (lsp.role_label
      ? `${lsp.role_label} · ${lsp.agent_slot || "Slot A"}`
      : `${le.active_role || "🧠 BRAIN · Architect"} · ${le.active_agent_slot || "Slot A"}`);
  const elapsedSec = Number(lsp.elapsed_sec ?? le.elapsed_sec ?? 0);

  if (isRunning && !liveElapsedTimerHandle) {
    startLiveElapsedTicker(elapsedSec);
  } else if (isRunning && elapsedSec > liveElapsedBaseSec) {
    liveElapsedBaseSec = elapsedSec;
    liveElapsedStartMs = Date.now();
  }

  const spinnerEl = document.getElementById("grb-spinner-icon");
  if (spinnerEl) {
    spinnerEl.textContent = isRunning ? "⟳" : "⏹";
    spinnerEl.className = isRunning ? "grb-spinner spinning" : "grb-spinner stopped";
  }

  const statusPill = document.getElementById("grb-status-pill");
  if (statusPill) {
    statusPill.textContent = isRunning ? "⚡ ĐANG CHẠY TIẾN TRÌNH" : "⏹ ĐÃ DỪNG TIẾN TRÌNH";
    statusPill.className = isRunning ? "grb-status-pill running" : "grb-status-pill stopped";
  }

  setText("grb-task-label", `${runTaskKey} — ${runTaskTitle}`);

  const subPill = document.getElementById("grb-subtask-pill");
  if (subPill) {
    if (subId) {
      subPill.classList.remove("hidden");
      subPill.textContent = `🧩 Sub-Task: ${subId}${subTitle ? " (" + subTitle + ")" : ""}`;
    } else {
      subPill.classList.add("hidden");
    }
  }

  setText("grb-step-badge", `Step ${stepNum}/${totalSteps} · ${pct}%`);
  setText("grb-action-detail", detail);
  setText("grb-role-pill", roleSlotText);

  const elapsedEl = document.getElementById("grb-elapsed-pill");
  if (elapsedEl) {
    elapsedEl.textContent = `⏱ ${formatElapsedSeconds(elapsedSec)}`;
  }

  const stopBtn = document.getElementById("grb-btn-stop");
  if (stopBtn) {
    stopBtn.classList.toggle("hidden", !isRunning);
    stopBtn.onclick = (ev) => stopRunningProcess(runTaskKey, subId, ev);
  }

  const fillEl = document.getElementById("grb-progress-fill");
  if (fillEl) {
    fillEl.style.width = `${pct}%`;
  }
}

let sidebarQueueFilter = "all";

function setSidebarQueueFilter(mode) {
  sidebarQueueFilter = mode || "all";
  document.querySelectorAll("#sidebar-queue-filter-bar .sb-qf-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.getAttribute("data-qfilter") === sidebarQueueFilter);
  });
  renderSidebarTaskQueue();
}

function renderSidebarTaskQueue() {
  const listEl = document.getElementById("sidebar-task-queue");
  if (!listEl || !dashboardState) return;

  const activeKey = dashboardState.active_task_key || selectedChainKey || "";
  const lsp = dashboardState.live_step_progress || {};
  const le = dashboardState.live_execution || {};
  const allNodes = getAllWaveNodes();

  let runningOrReadyCnt = 0;
  let needsYouCnt = 0;
  let doneCnt = 0;

  const enriched = allNodes.map((n, idx) => {
    const isSelected = n.key === activeKey;
    const isTaskRunning = Boolean(
      n.is_running ||
        (lsp.active && lsp.task_key === n.key) ||
        (isLifecycleActionRunning && runningActionTaskKey === n.key) ||
        n.stage === "RUNNING" ||
        n.stage === "IN_REVIEW" ||
        n.stage === "UNDER_WAY" ||
        (isSelected && le.is_running)
    );
    if (isTaskRunning) {
      runningOrReadyCnt++;
    } else if (n.stage === "NEEDS_YOU") {
      needsYouCnt++;
    } else if (n.stage === "DONE") {
      doneCnt++;
    } else {
      runningOrReadyCnt++;
    }
    const primaryRepo = formatRepoBasename(n.target_repo || (Array.isArray(n.affected_repos) && n.affected_repos[0]) || "ai-orchestrator");
    return {
      node: n,
      isSelected,
      isTaskRunning,
      primaryRepo,
      queuePos: n.queue_position || idx + 1,
    };
  });

  setText("sb-qs-all", enriched.length);
  setText("sb-qs-running", runningOrReadyCnt);
  setText("sb-qs-needs-you", needsYouCnt);
  setText("sb-qs-done", doneCnt);

  const filtered = enriched.filter((item) => {
    const st = item.node.stage;
    if (sidebarQueueFilter === "active" || sidebarQueueFilter === "running") {
      return item.isTaskRunning || (st !== "DONE" && st !== "NEEDS_YOU");
    }
    if (sidebarQueueFilter === "needs_you") {
      return st === "NEEDS_YOU";
    }
    if (sidebarQueueFilter === "done") {
      return st === "DONE";
    }
    return true;
  });

  if (filtered.length === 0) {
    listEl.innerHTML = `<div class="sb-empty-queue">Không có Task nào trong bộ lọc hiện tại. Bấm <b>Tất cả</b> để xem toàn bộ hàng đợi.</div>`;
    return;
  }

  // Group tasks by primary Target Repo for clear multi-repo queue visibility
  const repoGroups = [];
  const repoMap = new Map();
  filtered.forEach((item) => {
    let grp = repoMap.get(item.primaryRepo);
    if (!grp) {
      grp = { repoName: item.primaryRepo, items: [], running: 0, waiting: 0, done: 0 };
      repoMap.set(item.primaryRepo, grp);
      repoGroups.push(grp);
    }
    grp.items.push(item);
    if (item.isTaskRunning) grp.running++;
    else if (item.node.stage === "DONE") grp.done++;
    else grp.waiting++;
  });

  listEl.innerHTML = repoGroups
    .map((grp) => {
      const grpCardsHtml = grp.items
        .map(({ node: n, isSelected, isTaskRunning, queuePos }) => {
          const runPct = isTaskRunning ? Math.max(8, Math.min(100, Number(lsp.percent || le.overall_progress_pct || 24))) : 0;
          const stepIdx = lsp.step_index || lsp.step_num || le.active_step_num || 1;
          const stepTitle = lsp.stage_label || lsp.current_action || lsp.title || le.active_step_title || "Đang xử lý...";
          const runDetail = isTaskRunning
            ? lsp.sub_task_id
              ? `🧩 ${lsp.sub_task_id} · Step ${stepIdx}/${lsp.total_steps || 5} (${runPct}%) — ${stepTitle}`
              : `Step ${stepIdx}/${lsp.total_steps || le.total_workflow_steps || 17} (${runPct}%) — ${stepTitle}`
            : "";

          const subTotal = Number(n.sub_tasks_total || (n.sub_tasks || []).length || 0);
          const subDone = Number(
            n.sub_tasks_done ??
              (n.sub_tasks || []).filter((s) => String(s.status || "").toLowerCase() === "done").length
          );
          const diffFiles = Number(n.repo_diff_files || 0);
          const diffBadgeHtml =
            diffFiles > 0
              ? `<span class="sb-diff-pill" title="${escapeAttr(n.repo_diff_summary || `${diffFiles} files changed`)}">📝 ${diffFiles}f</span>`
              : `<span class="sb-diff-pill zero" title="Chưa có git diff cho task này">📝 0f</span>`;

          let btnLabel = "▶ Chạy";
          let btnAction = "autopilot";
          let statusLabel = `⏳ #${queuePos}`;
          if (n.stage === "NEEDS_YOU") {
            btnLabel = "⚡ All";
            btnAction = "approve_all";
            statusLabel = "⏸ DUYỆT S3";
          } else if (n.stage === "DONE") {
            btnLabel = "↻ Chạy lại";
            btnAction = "approve_all";
            statusLabel = "✓ XONG";
          }

          const stageBadgeHtml = isTaskRunning
            ? `<span class="running-pulse-pill"><span class="spin-inline">⟳</span> ${runPct}%</span>`
            : `<span class="wcard-stage ${escapeAttr(n.stage)}">${escapeHtml(statusLabel)}</span>`;

          const actionButtonsHtml = isTaskRunning
            ? `
              <button type="button" class="sb-stop-btn"
                      title="Dừng ngay tiến trình của ${escapeAttr(n.key)}"
                      onclick="stopRunningProcess('${escapeAttr(n.key)}', '', event)">
                ⏹ Dừng
              </button>
              <button type="button" class="sb-del-btn"
                      title="Xóa Task ${escapeAttr(n.key)}"
                      onclick="deleteSingleTask('${escapeAttr(n.key)}', event)">
                🗑
              </button>
            `
            : n.stage === "NEEDS_YOU"
            ? `
              <button type="button" class="sb-run-btn"
                      title="Approve All: Duyệt &amp; chạy toàn bộ ${subTotal} Sub-Tasks theo Wave -> S5"
                      onclick="runTaskAction('${escapeAttr(n.key)}', 'approve_all', event)">
                ⚡ All
              </button>
              <button type="button" class="mini-tg-btn"
                      title="Duyệt Từng Sub-Task (1/${subTotal} -> ${subTotal}/${subTotal}) tinh tế qua Telegram &amp; WebUI"
                      onclick="runTaskAction('${escapeAttr(n.key)}', 'step_subtasks', event)">
                🧩 Từng Sub
              </button>
              <button type="button" class="sb-del-btn"
                      title="Xóa Task ${escapeAttr(n.key)} khỏi hàng đợi"
                      onclick="deleteSingleTask('${escapeAttr(n.key)}', event)">
                🗑
              </button>
            `
            : `
              <button type="button" class="sb-run-btn"
                      title="Kích hoạt chạy ngay ${escapeAttr(n.key)}"
                      onclick="runTaskAction('${escapeAttr(n.key)}', '${btnAction}', event)">
                ${btnLabel}
              </button>
              <button type="button" class="sb-del-btn"
                      title="Xóa Task ${escapeAttr(n.key)} khỏi hàng đợi"
                      onclick="deleteSingleTask('${escapeAttr(n.key)}', event)">
                🗑
              </button>
            `;

          const liveBarHtml = isTaskRunning
            ? `
              <div class="sb-running-strip">
                <div class="sb-running-text"><span class="spin-inline">⟳</span> ${escapeHtml(runDetail)}</div>
                <div class="sb-running-track"><div class="sb-running-fill" style="width:${runPct}%"></div></div>
              </div>
            `
            : "";

          const cleanKey = String(n.key || "").trim();
          const cleanTitle = String(n.title || "").trim();
          const showSubtitle = cleanTitle && cleanTitle.toLowerCase() !== cleanKey.toLowerCase();

          return `
            <div class="sb-task-item ${isSelected ? "active-task" : ""} ${isTaskRunning ? "is-actively-running" : ""}"
                 onclick="selectActiveTask('${escapeAttr(n.key)}')">
              <div class="sb-task-top">
                <span class="sb-task-key" title="#${queuePos} · ${escapeAttr(cleanKey)}">#${queuePos} ${escapeHtml(cleanKey)}</span>
                <span class="sb-task-actions">
                  ${actionButtonsHtml}
                </span>
              </div>
              ${showSubtitle ? `<div class="sb-task-title" title="${escapeAttr(cleanTitle)}">${escapeHtml(cleanTitle)}</div>` : ""}
              ${liveBarHtml}
              <div class="sb-task-iso">
                ${stageBadgeHtml}
                <span>🧩 ${subDone}/${subTotal} Sub</span>
                ${diffBadgeHtml}
              </div>
            </div>
          `;
        })
        .join("");

      return `
        <div class="sb-repo-group">
          <div class="sb-repo-group-hdr">
            <span class="sb-repo-group-name">📦 ${escapeHtml(grp.repoName)}</span>
            <span class="sb-repo-group-counts">
              ${grp.running > 0 ? `<b class="c-run">⚡ ${grp.running} chạy</b> · ` : ""}
              ${grp.waiting > 0 ? `<b class="c-wait">⏳ ${grp.waiting} chờ</b> · ` : ""}
              <span class="c-done">✓ ${grp.done}/${grp.items.length}</span>
            </span>
          </div>
          <div class="sb-repo-group-body">
            ${grpCardsHtml}
          </div>
        </div>
      `;
    })
    .join("");
}

function renderAgentKernelCard() {
  if (!dashboardState) return;
  const ak = dashboardState.active_agent_kernel || {};
  const targetRepo = dashboardState.active_target_repo || dashboardState.active_isolation?.target_repo || "";
  const affectedRepos = dashboardState.active_affected_repos || [];

  setText("ak-role-label", ak.active_role || "BRAIN — Chairman & Architect (Propose-Only)");
  setText("ak-total-lessons", ak.total_lessons || 269);
  setText("ak-target-repo", formatAffectedReposSummary(affectedRepos, targetRepo));

  const hdrKernelBadge = document.getElementById("hdr-kernel-badge");
  if (hdrKernelBadge) {
    hdrKernelBadge.textContent = `🛡️ Kernel: /agent (${ak.total_lessons || 269} Lessons · H1..H7)`;
  }

  const chipsEl = document.getElementById("ak-skills-hooks-strip");
  if (chipsEl) {
    const skills = (ak.active_skills || []).map(
      (s) => `<span class="ak-chip">${escapeHtml(s)}</span>`
    );
    const hooks = (ak.active_hooks || []).map(
      (h) => `<span class="ak-chip hook">${escapeHtml(h)}</span>`
    );
    chipsEl.innerHTML = [...skills, ...hooks].join("");
  }

  const lessonsEl = document.getElementById("ak-matched-lessons-list");
  if (lessonsEl) {
    const matched = ak.matched_lessons || [];
    lessonsEl.innerHTML = matched
      .map(
        (m) => `
        <div class="ak-lesson-row" title="${escapeAttr(m.rule || m.summary || m.title)}">
          <span class="ak-lesson-code">${escapeHtml(m.lesson_id || m.code)}</span>
          <span class="ak-lesson-topic">[${escapeHtml(m.topic)}]</span>
          <strong>${escapeHtml(m.title)}</strong>
          <span class="ak-lesson-sum">— ${escapeHtml(m.rule || m.summary || "")}</span>
        </div>
      `
      )
      .join("");
  }
}

function renderIsolationInspector() {
  if (!dashboardState) return;
  const activeKey = dashboardState.active_task_key || "TASK-1";
  const activeTitle = dashboardState.active_task_title || "";
  const iso = dashboardState.active_isolation || {};
  const runners = dashboardState.available_runners || [];
  const totalLessons = dashboardState.active_agent_kernel?.total_lessons || 269;
  const targetRepo = dashboardState.active_target_repo || iso.target_repo || "";
  const affectedRepos = dashboardState.active_affected_repos || [];
  const repoSummary = formatAffectedReposSummary(affectedRepos, targetRepo);

  setText("iso-task-heading", `${activeKey}: ${activeTitle}`);
  setText(
    "iso-branch-val",
    `📦 ${repoSummary} · ⎇ ${iso.branch_name || "task/" + activeKey}`
  );
  setText("iso-worktree-val", `${repoSummary} → ${iso.worktree_path || ".orchestrator/worktrees/" + activeKey}`);
  setText("iso-spec-val", iso.spec_file || `.orchestrator/tasks/${activeKey}/01_requirements.md`);
  setText("iso-plan-val", iso.plan_file || `.orchestrator/tasks/${activeKey}/09_tasks_solution.md`);
  setText("iso-audit-val", iso.audit_file || `.orchestrator/tasks/${activeKey}/audit_report.md`);
  setText("iso-diff-val", iso.diff_file || `.orchestrator/tasks/${activeKey}/diff.patch`);

  const runnerSel = document.getElementById("global-runner-select");
  if (runnerSel && dashboardState.active_runner) {
    runnerSel.value = dashboardState.active_runner;
  }

  const detectedList = runners
    .filter((r) => r.installed)
    .map((r) => `${r.name} (${r.version || "ready"})`)
    .join(" · ");
  setText("iso-runners-detected", detectedList || "Local CLI Runners Ready");

  const statusEl = document.getElementById("iso-runner-status");
  if (statusEl) {
    const mdl = dashboardState.cli_settings?.planner_model || dashboardState.planner_model || "gemini-4-argon (antigravity)";
    statusEl.innerHTML = `Engine: <b>${escapeHtml((dashboardState.active_runner || "antigravity").toUpperCase())} (${escapeHtml(mdl)})</b> · Kernel: <b>/agent (GEMINI.md + ${totalLessons} Lessons)</b>`;
  }

  setText("console-log-path", iso.log_file || `.orchestrator/tasks/${activeKey}/terminal.log`);
  const consoleEl = document.getElementById("active-task-console");
  if (consoleEl) {
    const logs = dashboardState.active_task_logs || [];
    consoleEl.innerHTML = logs
      .map((ln) => `<div class="vp-log-line">${escapeHtml(ln)}</div>`)
      .join("");
    consoleEl.scrollTop = consoleEl.scrollHeight;
  }
}

function setSubTaskRepoFilter(repoName) {
  selectedSubTaskRepoFilter = repoName || "";
  renderStepReportInspector();
}

function selectSubTaskFractal(subTaskID, ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  selectedSubTaskID = subTaskID || "";
  renderStepReportInspector();
  renderWaveBoard();
}

function inspectSubTaskDedicatedPlan(subTaskID, tabMode, ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  selectedSubTaskID = subTaskID || selectedSubTaskID;
  inspectingSubTaskOverride = {
    subTaskID: selectedSubTaskID,
    tabMode: tabMode || "plan",
  };
  if (currentScreenMode !== "autopilot") {
    switchScreen("autopilot");
  } else {
    renderStepReportInspector();
  }
}

function clearSubTaskReportOverride(ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  inspectingSubTaskOverride = null;
  renderStepReportInspector();
}

async function runSubTaskLifecycle(subTaskID, subAction, ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-1";
  selectedSubTaskID = subTaskID || selectedSubTaskID;

  let pollTimer = null;
  isLifecycleActionRunning = true;
  runningActionTaskKey = activeKey;
  runningActionSubTaskID = selectedSubTaskID;
  currentActionAbortController = new AbortController();
  startLiveElapsedTicker(0);
  renderCrossContextBar();
  renderGlobalRunningProgressBanner();
  renderSidebarTaskQueue();
  renderLiveTelemetryHUD();
  renderStepReportInspector();
  pollTimer = setInterval(() => {
    fetchDashboard();
  }, 650);

  try {
    const res = await fetch("/api/tasks/subtasks", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      signal: currentActionAbortController ? currentActionAbortController.signal : undefined,
      body: JSON.stringify({
        task_key: activeKey,
        sub_task_id: selectedSubTaskID,
        action: subAction,
        step_delay_ms: 1600,
      }),
    });
    if (!res.ok) return;
    dashboardState = await res.json();
    renderAll();
  } catch (e) {
    if (e && e.name === "AbortError") {
      await fetchDashboard();
      return;
    }
    console.error("runSubTaskLifecycle error:", e);
  } finally {
    if (pollTimer) clearInterval(pollTimer);
    isLifecycleActionRunning = false;
    runningActionTaskKey = "";
    runningActionSubTaskID = "";
    currentActionAbortController = null;
    if (!dashboardState?.live_step_progress?.active) {
      stopLiveElapsedTicker();
    }
    renderAll();
  }
}

function buildSubTaskFractalInspectorHTML(st, activeKey) {
  if (!st) return "";
  const stId = st.sub_task_id || st.id || "TASK-M01";
  const stStage = String(st.stage || st.status || "READY").toUpperCase();
  const lsp = dashboardState?.live_step_progress || {};
  const isSubRunning = Boolean(
    stStage === "RUNNING" ||
      (lsp.active && lsp.task_key === activeKey && lsp.sub_task_id === stId) ||
      (isLifecycleActionRunning && runningActionTaskKey === activeKey && runningActionSubTaskID === stId)
  );
  const stBranch = st.branch_name || `task/${activeKey}/${stId.toLowerCase()}`;
  const stRepo = st.target_repo_name || formatRepoBasename(st.target_repo || "");
  const stPlanFile = st.plan_file || `.orchestrator/tasks/${activeKey}/subtasks/${stId}/09_tasks_solution.md`;
  const inContract = st.input_contract || {};
  const outContract = st.output_contract || {};
  const sh = st.self_heal_state || dashboardState?.active_self_heal_state || {};
  const retryCnt = sh.retry_count ?? 0;
  const maxRetries = sh.max_retries || 3;
  const circuitOpen = Boolean(sh.circuit_open);
  const reApproveReq = Boolean(sh.requires_human_reapprove);
  const crossVerified = st.cross_model_verified !== false && (dashboardState?.cross_model_verified !== false);

  const upDeliverables = Array.isArray(inContract.upstream_deliverables) && inContract.upstream_deliverables.length > 0
    ? inContract.upstream_deliverables
    : [`[Wave ${st.wave_index ?? 0} Entry] Nhận trực tiếp BRD/Spec gốc & điểm chạm ${st.context_file_line || stRepo}`];
  const outDeliverables = Array.isArray(outContract.deliverables) && outContract.deliverables.length > 0
    ? outContract.deliverables
    : [`[${stRepo}] Cập nhật & kiểm chứng ${st.context_file_line || stRepo} — ${st.title}`];
  const downstreamList = Array.isArray(outContract.downstream_sub_task_ids) && outContract.downstream_sub_task_ids.length > 0
    ? outContract.downstream_sub_task_ids.join(", ")
    : `Wave Cuối -> Bàn giao cho Task cha (${activeKey})`;

  const stages = Array.isArray(st.pipeline_stages) && st.pipeline_stages.length > 0 ? st.pipeline_stages : [];
  const pipelineHtml = stages
    .map((grp, idx) => {
      const cards = Array.isArray(grp.cards) && grp.cards.length > 0 ? grp.cards : [{ title: grp.id, status: "done" }];
      const cardsInner = cards
        .map((card) => {
          const cStatus = String(card.status || "pending").toLowerCase();
          let icon = "✓";
          if (cStatus === "running") icon = `<span class="spin-inline">⟳</span>`;
          else if (cStatus === "waiting") icon = "●";
          else if (cStatus === "pending") icon = "◌";
          const roleChip = card.role_label
            ? `<div class="sfi-stage-meta"><b>${escapeHtml(card.role_label)}</b> · <code>${escapeHtml(card.agent_slot || "")}</code> · ${escapeHtml(card.model || "")}</div>`
            : "";
          return `
            <div class="sfi-stage-node status-${escapeAttr(cStatus)} ${cStatus === "running" ? "is-actively-running" : ""}" style="margin-bottom:${cards.length > 1 ? "4px" : "0"};">
              <div class="sfi-stage-title"><span>${icon}</span> ${escapeHtml(card.title || "")}</div>
              <div class="sfi-stage-sub">${escapeHtml(card.subtitle || "")}</div>
              ${roleChip}
              <div class="sfi-stage-meta">${escapeHtml(card.meta || "")}</div>
            </div>
          `;
        })
        .join("");
      const foot = grp.round_badge ? `<div class="sfi-stage-meta" style="text-align:center;color:#93c5fd;">${escapeHtml(grp.round_badge)}</div>` : "";
      const arrow = idx < stages.length - 1 ? `<span class="sfi-arrow">→</span>` : "";
      return `
        <div class="sfi-stage-group">
          ${cardsInner}
          ${foot}
        </div>
        ${arrow}
      `;
    })
    .join("");

  const upReadyBadge = inContract.upstream_ready !== false
    ? `<span class="sfi-contract-badge ready">✓ Upstream &amp; Wave Sync Ready</span>`
    : `<span class="sfi-contract-badge waiting">⏳ Chờ Wave trước hoàn tất</span>`;
  const outReadyBadge = outContract.handoff_ready
    ? `<span class="sfi-contract-badge ready">✓ Handoff Ready (${escapeHtml(outContract.evidence_receipt_id || "EV-PASS")})</span>`
    : `<span class="sfi-contract-badge waiting">⏳ Đang thực thi (${escapeHtml(outContract.evidence_receipt_id || "EV-PENDING")})</span>`;

  const circuitBadge = circuitOpen
    ? `<span class="sfi-contract-badge waiting" title="Quá ${maxRetries} vòng tự sửa -> Ngắt mạch đẩy về NEEDS_YOU">🔌 Circuit OPEN (${retryCnt}/${maxRetries} Retries · Wave Rollback)</span>`
    : reApproveReq
    ? `<span class="sfi-contract-badge waiting" title="Thiết kế S3 đã thay đổi -> Bắt buộc duyệt lại tại Human Gate">🔒 Tier-2 S5→S3 Re-Locked Human Gate</span>`
    : `<span class="sfi-contract-badge ready" title="Tier-1 S5->S4->S5 (Max ${maxRetries} Retries) · Tier-2 S5->S3 Re-Locks Human Gate">🛡️ Circuit Breaker: Retry ${retryCnt}/${maxRetries} (Max ${maxRetries})</span>`;

  const crossModelBadge = crossVerified
    ? `<span class="sfi-contract-badge ready" title="Coder ≠ QA Auditor ≠ Laya Gate">⚖️ Cross-Model Audited</span>`
    : `<span class="sfi-contract-badge waiting">⚠️ Single-Model</span>`;

  const stagePillHtml = isSubRunning
    ? `<span class="running-pulse-pill"><span class="spin-inline">⟳</span> ĐANG CHẠY ${escapeHtml(stId)}</span>`
    : `<span class="wcard-stage ${escapeAttr(stStage)}">● ${escapeHtml(stStage)}</span>`;

  const stopSubBtnHtml = isSubRunning
    ? `<button type="button" class="sfi-btn stop-sub-btn" onclick="stopRunningProcess('${escapeAttr(activeKey)}', '${escapeAttr(stId)}', event)">⏹ Dừng ${escapeHtml(stId)}</button>`
    : "";

  return `
    <div class="sfi-card ${isSubRunning ? "is-actively-running" : ""}">
      <div class="sfi-hdr">
        <div class="sfi-hdr-left">
          <span class="sfi-fractal-tag">🔬 SUB-TASK LÀ TASK ĐỘC LẬP</span>
          <strong class="sfi-title">${escapeHtml(stId)} — ${escapeHtml(st.title || "")}</strong>
          ${stagePillHtml}
          <code class="sfi-branch-pill" title="Git Worktree được gắn ngay từ đầu pha S4 (Slot C)">🌿 Worktree Attached (Slot C): ${escapeHtml(stBranch)}</code>
          <span class="sri-repo-badge">📦 ${escapeHtml(stRepo)} · Wave ${st.wave_index ?? 0}</span>
          ${crossModelBadge}
          ${circuitBadge}
        </div>
        <div class="sfi-actions">
          ${stopSubBtnHtml}
          <button type="button" class="sfi-btn" onclick="inspectSubTaskDedicatedPlan('${escapeAttr(stId)}', 'plan', event)" title="Mở file 09_tasks_solution.md & Code Demo riêng của ${escapeAttr(stId)} lên khung chính">📐 Xem Plan &amp; Demo Riêng</button>
          <button type="button" class="sfi-btn" onclick="inspectSubTaskDedicatedPlan('${escapeAttr(stId)}', 'diff', event)" title="Xem diff.patch riêng của ${escapeAttr(stId)}">🔧 Xem Diff Riêng</button>
          <button type="button" class="sfi-btn" onclick="runSubTaskLifecycle('${escapeAttr(stId)}', 'subtask_plan', event)">1. Plan (S1–S3)</button>
          <button type="button" class="sfi-btn" onclick="runSubTaskLifecycle('${escapeAttr(stId)}', 'subtask_replan_s3', event)" title="Tier-2 S5->S3: Sửa Plan S3 và khóa lại ở Human Gate (NEEDS_YOU) chống Scope Drift">↺ S5→S3 Re-Plan</button>
          <button type="button" class="sfi-btn approve" onclick="runSubTaskLifecycle('${escapeAttr(stId)}', 'subtask_approve', event)">✓ APPROVE</button>
          <button type="button" class="sfi-btn" onclick="runSubTaskLifecycle('${escapeAttr(stId)}', 'subtask_implement', event)">3. Implement (S4)</button>
          <button type="button" class="sfi-btn" onclick="runSubTaskLifecycle('${escapeAttr(stId)}', 'subtask_qa', event)">4. 5A Hard + 5B QA</button>
          <button type="button" class="sfi-btn primary" onclick="runSubTaskLifecycle('${escapeAttr(stId)}', 'subtask_autopilot', event)">▶ Autopilot ${escapeHtml(stId)}</button>
          <button type="button" class="sfi-btn del-sub-btn" onclick="deleteSingleSubTask('${escapeAttr(activeKey)}', '${escapeAttr(stId)}', event)" title="Xóa Sub-Task ${escapeAttr(stId)}">🗑 Xóa</button>
        </div>
      </div>

      <!-- 6-Step Sub-Task Pipeline: Start -> Plan (S1-S3) -> You -> Implement (S4 + Slot C Worktree) -> 5A Hard Gate (Zero-LLM) + 5B Cross-Model QA/DG3 -> Finalize & Diff Review (S6) -->
      <div class="sfi-pipeline-row">
        ${pipelineHtml}
      </div>

      <!-- 3-Column Fractal Contract: 1. Input Contract -> 2. Dedicated Plan & Code Demo -> 3. Output Contract -->
      <div class="sfi-contracts-grid">
        <div class="sfi-contract-col">
          <div class="sfi-col-hdr">
            <span>📥 1. ĐẦU VÀO (<code>InputContract</code>)</span>
            ${upReadyBadge}
          </div>
          <div class="sfi-col-body">
            <div><b>Context (<code>file:line</code>):</b> <code>${escapeHtml(inContract.source_context || st.context_file_line || stRepo)}</code></div>
            <div><b>Phụ thuộc Wave trước (<code>DependsOn</code>):</b> <code>${escapeHtml((inContract.upstream_sub_task_ids || []).join(", ") || "Wave 0 Entry (Không phụ thuộc)")}</code></div>
            <div><b>Đầu vào bàn giao từ Wave trước:</b></div>
            <ul class="sfi-list">
              ${upDeliverables.map((d) => `<li>${escapeHtml(d)}</li>`).join("")}
            </ul>
          </div>
        </div>

        <div class="sfi-contract-col">
          <div class="sfi-col-hdr">
            <span>📐 2. PLAN GIẢI QUYẾT RIÊNG (<code>09_tasks_solution.md</code>)</span>
            <code class="sfi-path-code">${escapeHtml(stPlanFile)}</code>
          </div>
          <div class="sfi-col-body">
            <div>${escapeHtml(st.plan_summary || st.description || "")}</div>
            <pre class="sfi-mini-code">${escapeHtml(st.code_demo || "// Code demo riêng của Sub-Task")}</pre>
          </div>
        </div>

        <div class="sfi-contract-col">
          <div class="sfi-col-hdr">
            <span>📤 3. ĐẦU RA (<code>OutputContract</code> -> Wave kế tiếp)</span>
            ${outReadyBadge}
          </div>
          <div class="sfi-col-body">
            <div><b>Sản phẩm bàn giao (<code>Deliverables</code>):</b></div>
            <ul class="sfi-list">
              ${outDeliverables.map((d) => `<li>${escapeHtml(d)}</li>`).join("")}
            </ul>
            <div><b>Bàn giao cho Sub-Task Wave sau:</b> <code>${escapeHtml(downstreamList)}</code></div>
            <div><b>Hồ sơ cô lập:</b> <code>${escapeHtml(outContract.diff_file || "")}</code> · <code>${escapeHtml(outContract.audit_file || "")}</code></div>
          </div>
        </div>
      </div>
    </div>
  `;
}

async function toggleSubTaskStatus(subTaskID) {
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-1";
  try {
    const res = await fetch("/api/tasks/subtasks", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        task_key: activeKey,
        toggle_id: subTaskID,
      }),
    });
    if (!res.ok) return;
    dashboardState = await res.json();
    renderAll();
  } catch (e) {
    console.error("toggleSubTaskStatus error:", e);
  }
}

function renderStepReportInspector() {
  if (!dashboardState) return;
  const reports = dashboardState.active_step_reports || [];
  const activeKey = dashboardState.active_task_key || "TASK-1";
  const activeStage = dashboardState.active_task_stage || "READY";
  const subTasks = dashboardState.active_sub_tasks || [];

  if (subTasks.length > 0 && (!selectedSubTaskID || !subTasks.some((x) => (x.sub_task_id || x.id) === selectedSubTaskID))) {
    selectedSubTaskID = subTasks[0].sub_task_id || subTasks[0].id || "";
  }
  const activeSubTaskObj = subTasks.find((x) => (x.sub_task_id || x.id) === selectedSubTaskID) || subTasks[0] || null;

  // Highlight active tab
  document.querySelectorAll("#sri-tabs-bar .sri-tab").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.step === selectedReportTab);
  });

  const currentReport =
    reports.find((r) => r.step_id === selectedReportTab) ||
    reports.find((r) => r.step_id === "S3") ||
    reports[0] || {
      step_id: selectedReportTab,
      title: `${selectedReportTab} Report`,
      status: activeStage,
      artifact_path: `.orchestrator/tasks/${activeKey}/09_tasks_solution.md`,
      content_md: "Chưa có dữ liệu báo cáo.",
      code_demo: "",
    };

  if (inspectingSubTaskOverride && activeSubTaskObj && (activeSubTaskObj.sub_task_id || activeSubTaskObj.id) === inspectingSubTaskOverride.subTaskID) {
    const mode = inspectingSubTaskOverride.tabMode || "plan";
    const subFile = mode === "diff"
      ? activeSubTaskObj.output_contract?.diff_file || `${activeSubTaskObj.artifact_dir}/diff.patch`
      : activeSubTaskObj.plan_file || `${activeSubTaskObj.artifact_dir}/09_tasks_solution.md`;
    setText("sri-active-file", subFile);
    const titleEl = document.getElementById("sri-report-title");
    if (titleEl) {
      titleEl.innerHTML = `🔬 [${escapeHtml(activeSubTaskObj.sub_task_id)}] ${escapeHtml(activeSubTaskObj.title)} <button type="button" class="sfi-btn" style="margin-left:8px;" onclick="clearSubTaskReportOverride(event)">↩ Quay lại Báo cáo Task cha (${escapeHtml(activeKey)})</button>`;
    }
    const mdEl = document.getElementById("sri-markdown-content");
    if (mdEl) {
      mdEl.textContent = mode === "diff"
        ? activeSubTaskObj.diff_preview || "Chưa có diff.patch cho Sub-Task này."
        : activeSubTaskObj.plan_markdown || activeSubTaskObj.plan_summary || "";
    }
    const codeEl = document.getElementById("sri-code-demo-content");
    if (codeEl) {
      codeEl.textContent = activeSubTaskObj.code_demo || "// Chưa có Code Demo cho Sub-Task này.";
    }
  } else {
    setText("sri-active-file", currentReport.file_path || currentReport.artifact_path || `.orchestrator/tasks/${activeKey}/09_tasks_solution.md`);
    setText("sri-report-title", `${currentReport.title} (${activeKey})`);
    const mdEl = document.getElementById("sri-markdown-content");
    if (mdEl) {
      mdEl.textContent = currentReport.content_md || currentReport.markdown_content || "Đang tải nội dung báo cáo...";
    }
    const s3Report = reports.find((r) => r.step_id === "S3");
    const codeDemo = currentReport.code_demo || s3Report?.code_demo || "// Chưa có Code Demo cho bước này.";
    const codeEl = document.getElementById("sri-code-demo-content");
    if (codeEl) {
      codeEl.textContent = codeDemo;
    }
  }

  const stEl = document.getElementById("sri-report-status");
  if (stEl) {
    stEl.className = `stage-pill ${activeStage}`;
    stEl.textContent = `${currentReport.status} · ${activeStage}`;
  }

  const approveBtn = document.getElementById("sri-approve-btn");
  if (approveBtn) {
    approveBtn.classList.toggle("hidden", activeStage !== "NEEDS_YOU");
    approveBtn.textContent = `✓ APPROVE Giải Pháp S3 & Chạy S4–S5 Cho ${activeKey}`;
  }

  setText("sri-subtasks-count", subTasks.length);

  // Render Sub-Task Fractal Lifecycle Inspector in View 1 (#subtask-fractal-inspector)
  const fractalEl = document.getElementById("subtask-fractal-inspector");
  if (fractalEl) {
    fractalEl.innerHTML = buildSubTaskFractalInspectorHTML(activeSubTaskObj, activeKey);
  }

  // Count per-repo breakdown across sub-tasks
  const repoCountMap = new Map();
  for (const st of subTasks) {
    const rName = st.target_repo_name || formatRepoBasename(st.target_repo || dashboardState.active_target_repo);
    repoCountMap.set(rName, (repoCountMap.get(rName) || 0) + 1);
  }
  setText("sri-repos-count", Math.max(1, repoCountMap.size));

  // Render Multi-Repo Filter Bar (#sri-repo-filter-bar)
  const filterBarEl = document.getElementById("sri-repo-filter-bar");
  if (filterBarEl) {
    const pills = [
      `<button type="button" class="sri-repo-pill ${!selectedSubTaskRepoFilter ? "active" : ""}" onclick="setSubTaskRepoFilter('')">Tất cả Repos (${subTasks.length})</button>`,
    ];
    for (const [rName, cnt] of repoCountMap.entries()) {
      const isAct = selectedSubTaskRepoFilter === rName;
      pills.push(
        `<button type="button" class="sri-repo-pill ${isAct ? "active" : ""}" onclick="setSubTaskRepoFilter('${escapeAttr(rName)}')">📦 ${escapeHtml(rName)} (${cnt})</button>`
      );
    }
    filterBarEl.innerHTML = pills.join("");
  }

  // Populate inline repo selector for adding new sub-task
  const inlineRepoSel = document.getElementById("inp-inline-subtask-repo");
  if (inlineRepoSel) {
    const affected = dashboardState.active_affected_repos || [];
    const repoChoices = affected.length > 0 ? affected : [dashboardState.active_target_repo || ""];
    inlineRepoSel.innerHTML = repoChoices
      .filter(Boolean)
      .map((rp) => {
        const bName = formatRepoBasename(rp);
        return `<option value="${escapeAttr(rp)}">📦 ${escapeHtml(bName)}</option>`;
      })
      .join("");
  }

  // Group filtered sub-tasks by GroupTitle (Nhóm 1..5)
  const filteredSubs = selectedSubTaskRepoFilter
    ? subTasks.filter(
        (st) => (st.target_repo_name || formatRepoBasename(st.target_repo)) === selectedSubTaskRepoFilter
      )
    : subTasks;

  const groupedMap = new Map();
  for (const st of filteredSubs) {
    const gTitle = st.group_title || `Nhóm Wave ${(st.wave_index ?? st.wave ?? 0) + 1} (${st.workflow_step || "S4"})`;
    if (!groupedMap.has(gTitle)) groupedMap.set(gTitle, []);
    groupedMap.get(gTitle).push(st);
  }

  const subListEl = document.getElementById("sri-subtasks-list");
  if (subListEl) {
    const groupBlocks = [];
    const roleChoices = [
      { value: "Coder", label: "🛠️ MUSCLE · Backend Coder (Slot B)" },
      { value: "Frontend", label: "🎨 MUSCLE · Frontend Coder (Slot D)" },
      { value: "Data-Engine", label: "⚡ MUSCLE · Data Engine Coder (Slot E)" },
      { value: "Tester", label: "🧪 QA & E2E Auditor (Slot H)" },
      { value: "Planner", label: "🧠 BRAIN · Architect (Slot A)" },
      { value: "Security", label: "🛡️ Security Auditor (Slot I)" },
    ];
    const modelChoices = [
      "gemini-4-argon (antigravity)",
      "gemini-4-argon-thinking (antigravity)",
      "claude-opus-4-6",
      "claude-sonnet-4-6",
      "gemini-2.5-pro (antigravity)",
    ];

    const lsp = dashboardState.live_step_progress || {};
    for (const [grpTitle, items] of groupedMap.entries()) {
      const cardsHtml = items
        .map((st) => {
          const stId = st.sub_task_id || st.id || "";
          const isSelectedSub = stId === selectedSubTaskID;
          const stStep = st.workflow_step || st.phase || `Wave ${st.wave_index ?? st.wave ?? 0}`;
          const stStage = String(st.stage || st.status || "PENDING").toUpperCase();
          const stStatus = String(st.status || "PENDING").toUpperCase();
          const isDone = stStatus === "DONE" || stStage === "DONE";
          const isSubRunning = Boolean(
            stStage === "RUNNING" ||
              stStatus === "RUNNING" ||
              (lsp.active && lsp.task_key === activeKey && lsp.sub_task_id === stId) ||
              (isLifecycleActionRunning && runningActionTaskKey === activeKey && runningActionSubTaskID === stId)
          );
          const repoName = st.target_repo_name || formatRepoBasename(st.target_repo || dashboardState.active_target_repo);
          const ctxLine = st.context_file_line || st.target_files || repoName;
          const desc = st.description || "";
          const dod = st.test_dod || "";
          const roleLabel = st.role_label || "🛠️ MUSCLE · Backend Coder";
          const agentSlot = st.agent_slot || "Slot B";
          const skillBadge = st.skill_badge || "/muscle-execute";
          const stModel = st.model || "gemini-4-argon (antigravity)";
          const curAssignee = st.assignee || "Coder";
          const stBranch = st.branch_name || `task/${activeKey}/${stId.toLowerCase()}`;
          const deps = Array.isArray(st.depends_on) && st.depends_on.length > 0 ? `📥 Input từ: ${st.depends_on.join(", ")}` : "📥 Wave 0 Entry";
          const downIds = st.output_contract?.downstream_sub_task_ids || [];
          const downLabel = downIds.length > 0 ? `📤 Output -> ${downIds.join(", ")}` : "📤 Output -> Parent Task";

          const roleOpts = roleChoices
            .map((rc) => `<option value="${escapeAttr(rc.value)}" ${rc.value === curAssignee ? "selected" : ""}>${escapeHtml(rc.label)}</option>`)
            .join("");
          const mdlOpts = modelChoices
            .map((mc) => `<option value="${escapeAttr(mc)}" ${mc === stModel ? "selected" : ""}>${escapeHtml(mc)}</option>`)
            .join("");

          const stStageBadgeHtml = isSubRunning
            ? `<span class="running-pulse-pill"><span class="spin-inline">⟳</span> ĐANG CHẠY...</span>`
            : `<span class="wcard-stage ${escapeAttr(stStage)}">● ${escapeHtml(stStage)}</span>`;

          const stopOrRunBtnHtml = isSubRunning
            ? `<button type="button" class="sfi-btn stop-sub-btn" onclick="stopRunningProcess('${escapeAttr(activeKey)}', '${escapeAttr(stId)}', event)">⏹ Dừng ${escapeHtml(stId)}</button>`
            : `<button type="button" class="sfi-btn primary" onclick="runSubTaskLifecycle('${escapeAttr(stId)}', 'subtask_autopilot', event)">▶ Chạy ${escapeHtml(stId)} (Start ➔ Branch)</button>`;

          return `
            <div class="sri-subtask-card ${isDone ? "done" : ""} ${isSelectedSub ? "selected-subtask-card" : ""} ${isSubRunning ? "is-actively-running" : ""}"
                 onclick="selectSubTaskFractal('${escapeAttr(stId)}', event)">
              <div class="sri-st-top">
                <label class="sri-st-check-wrap" onclick="event.stopPropagation()">
                  <input type="checkbox" ${isDone ? "checked" : ""} onchange="toggleSubTaskStatus('${escapeAttr(stId)}')" />
                  <span class="sri-st-id">${escapeHtml(stId)}</span>
                  <strong class="sri-st-title">${escapeHtml(st.title)}</strong>
                </label>
                <div class="sri-st-badges">
                  <code class="sfi-branch-pill">⎇ ${escapeHtml(stBranch)}</code>
                  <span class="sri-repo-badge" title="Target Repo: ${escapeAttr(st.target_repo || repoName)}">📦 ${escapeHtml(repoName)}</span>
                  <span class="sri-role-badge" title="Role & Slot: ${escapeAttr(roleLabel)} (${escapeAttr(agentSlot)} · ${escapeAttr(skillBadge)})">${escapeHtml(roleLabel)} · ${escapeHtml(agentSlot)}</span>
                  <span class="sri-model-badge" title="Model thực thi Sub-Task này">🤖 ${escapeHtml(stModel)}</span>
                  <span class="sri-wave-badge">${escapeHtml(stStep)}</span>
                  ${stStageBadgeHtml}
                </div>
              </div>
              <div class="sri-st-meta">
                <div>
                  <span class="sri-meta-key">Context (file:line):</span> <code>${escapeHtml(ctxLine)}</code>
                  <span class="sri-skill-pill">${escapeHtml(skillBadge)}</span>
                  <span class="sri-dep-pill">${escapeHtml(deps)}</span>
                  <span class="sri-dep-pill">${escapeHtml(downLabel)}</span>
                </div>
                ${desc ? `<div><span class="sri-meta-key">Mô tả &amp; Plan riêng:</span> ${escapeHtml(st.plan_summary || desc)}</div>` : ""}
                ${dod ? `<div><span class="sri-meta-key">Test / DoD:</span> <span class="sri-dod-text">${escapeHtml(dod)}</span></div>` : ""}
                <div class="sri-st-role-editor" onclick="event.stopPropagation()">
                  <span class="sri-meta-key">Role &amp; Model (${escapeHtml(stId)}):</span>
                  <select class="sri-inline-select" onchange="updateSubTaskRoleOrModel('${escapeAttr(stId)}', this.value, '')">
                    ${roleOpts}
                  </select>
                  <select class="sri-inline-select" onchange="updateSubTaskRoleOrModel('${escapeAttr(stId)}', '', this.value)">
                    ${mdlOpts}
                  </select>
                  <button type="button" class="sfi-btn" onclick="inspectSubTaskDedicatedPlan('${escapeAttr(stId)}', 'plan', event)">📐 Xem Plan &amp; Demo Riêng</button>
                  ${stopOrRunBtnHtml}
                  <button type="button" class="sfi-btn del-sub-btn" onclick="deleteSingleSubTask('${escapeAttr(activeKey)}', '${escapeAttr(stId)}', event)" title="Xóa Sub-Task ${escapeAttr(stId)}">🗑 Xóa</button>
                </div>
              </div>
            </div>
          `;
        })
        .join("");

      groupBlocks.push(`
        <div class="sri-subtask-group">
          <div class="sri-group-hdr">${escapeHtml(grpTitle)} <span class="sri-group-cnt">${items.length} sub-tasks</span></div>
          ${cardsHtml}
        </div>
      `);
    }
    subListEl.innerHTML = groupBlocks.join("");
  }
}

async function updateSubTaskRoleOrModel(subTaskID, newAssignee, newModel) {
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-1";
  try {
    const res = await fetch("/api/tasks/subtasks", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        task_key: activeKey,
        action: "update_role_model",
        sub_task_id: subTaskID,
        assignee: newAssignee || "",
        model: newModel || "",
      }),
    });
    if (!res.ok) return;
    dashboardState = await res.json();
    renderAll();
  } catch (e) {
    console.error("updateSubTaskRoleOrModel error:", e);
  }
}

async function copyActiveCodeDemo() {
  const codeEl = document.getElementById("sri-code-demo-content");
  if (!codeEl) return;
  try {
    await navigator.clipboard.writeText(codeEl.textContent || "");
  } catch (e) {
    console.warn("Clipboard write failed:", e);
  }
}

async function regenerateSubTasksForActiveTask() {
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-1";
  try {
    const res = await fetch("/api/tasks/subtasks", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        task_key: activeKey,
        auto_split: true,
      }),
    });
    if (!res.ok) return;
    dashboardState = await res.json();
    renderAll();
  } catch (e) {
    console.error("regenerateSubTasksForActiveTask error:", e);
  }
}

async function addSubTaskToActiveTask(ev) {
  if (ev && ev.preventDefault) ev.preventDefault();
  const inp = document.getElementById("inp-inline-subtask-title");
  if (!inp || !inp.value.trim()) return;
  const newTitle = inp.value.trim();
  const repoVal = document.getElementById("inp-inline-subtask-repo")?.value || "";
  const ctxVal = document.getElementById("inp-inline-subtask-ctx")?.value.trim() || "";
  const dodVal = document.getElementById("inp-inline-subtask-dod")?.value.trim() || "";

  inp.value = "";
  const ctxEl = document.getElementById("inp-inline-subtask-ctx");
  if (ctxEl) ctxEl.value = "";
  const dodEl = document.getElementById("inp-inline-subtask-dod");
  if (dodEl) dodEl.value = "";

  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-1";
  try {
    const res = await fetch("/api/tasks/subtasks", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        task_key: activeKey,
        add_title: newTitle,
        target_repo: repoVal,
        context_file_line: ctxVal,
        description: newTitle,
        test_dod: dodVal,
      }),
    });
    if (!res.ok) return;
    dashboardState = await res.json();
    renderAll();
  } catch (e) {
    console.error("addSubTaskToActiveTask error:", e);
  }
}

/* ============================================================================
   LIVE EXECUTION & MULTI-REPO PROGRESS TELEMETRY HUD (LIVE-PROCESS-HUD-01)
   ============================================================================ */
function renderLiveTelemetryHUD() {
  if (!dashboardState) return;
  const le = dashboardState.live_execution || {};
  const lsp = dashboardState.live_step_progress || {};
  const activeKey = dashboardState.active_task_key || selectedChainKey || "";
  const isRunning = Boolean(
    le.is_running ||
      (lsp.active && (!lsp.task_key || lsp.task_key === activeKey)) ||
      (isLifecycleActionRunning && (!runningActionTaskKey || runningActionTaskKey === activeKey))
  );
  const wasStopped = Boolean(!isRunning && (le.was_stopped || (lsp.was_stopped && lsp.task_key === activeKey)));

  const pct = Number(isRunning && lsp.percent ? lsp.percent : le.overall_progress_pct ?? 100);
  const doneSubs = le.done_sub_tasks ?? 0;
  const totalSubs = le.total_sub_tasks ?? 0;
  const doneSteps = le.done_workflow_steps ?? 17;
  const totalSteps = le.total_workflow_steps ?? 17;
  const diffLines = le.total_diff_lines ?? 0;

  setText("hud-phase-label", le.active_phase_label || `Stage: ${dashboardState.active_task_stage || "READY"}`);
  setText("hud-subtasks-ratio", `${doneSubs}/${totalSubs} DONE`);
  setText("hud-steps-ratio", `${doneSteps}/${totalSteps} bước`);
  setText("hud-diff-lines", `${diffLines} lines`);
  setText("hud-overall-pct", `${pct}%`);

  const hudWrap = document.getElementById("live-telemetry-hud");
  if (hudWrap) {
    hudWrap.classList.toggle("hud-actively-running", isRunning);
  }

  const fillEl = document.getElementById("hud-overall-progress-fill");
  if (fillEl) {
    fillEl.style.width = `${Math.max(4, Math.min(100, pct))}%`;
    fillEl.classList.toggle("shimmer-active", isRunning);
  }

  const hudElapsedPill = document.getElementById("hud-elapsed-pill");
  if (hudElapsedPill) {
    hudElapsedPill.classList.toggle("hidden", !isRunning);
  }
  const hudElapsedSec = document.getElementById("hud-elapsed-sec");
  if (hudElapsedSec && isRunning) {
    hudElapsedSec.textContent = formatElapsedSeconds(lsp.elapsed_sec ?? le.elapsed_sec ?? 0);
  }

  const hudStopBtn = document.getElementById("hud-btn-stop-task");
  if (hudStopBtn) {
    hudStopBtn.classList.toggle("hidden", !isRunning);
    hudStopBtn.onclick = (ev) => stopRunningProcess(activeKey, lsp.sub_task_id || runningActionSubTaskID || "", ev);
  }

  const indEl = document.getElementById("hud-live-indicator");
  if (indEl) {
    const st = String(le.active_stage || dashboardState.active_task_stage || "").toUpperCase();
    if (isRunning) {
      indEl.innerHTML = `<span class="spin-inline">⟳</span> ⚡ ĐANG CHẠY TRỰC TIẾP (LIVE RUNNING · ${pct}%)`;
      indEl.className = "hud-live-badge running";
    } else if (wasStopped) {
      indEl.textContent = "⏹ ĐÃ DỪNG TIẾN TRÌNH (STOPPED @ NEEDS_YOU)";
      indEl.className = "hud-live-badge stopped";
    } else if (st === "NEEDS_YOU") {
      indEl.textContent = "⏸ WAITING FOR YOU (S3 GATE)";
      indEl.className = "hud-live-badge waiting";
    } else if (st === "DONE") {
      indEl.textContent = "✅ VERIFIED ON DISK (100%)";
      indEl.className = "hud-live-badge done";
    } else {
      indEl.textContent = "⏸ SẴN SÀNG CHẠY (IDLE — CHƯA CHẠY)";
      indEl.className = "hud-live-badge idle";
    }
  }

  const whatMain = document.getElementById("hud-what-main");
  if (whatMain) {
    const activeStepNum = (isRunning && lsp.step_index) ? lsp.step_index : (le.active_step_num || 17);
    const activeStepTitle = (isRunning && (lsp.stage_label || lsp.current_action))
      ? [lsp.stage_label, lsp.current_action].filter(Boolean).join(" — ")
      : (le.active_step_title || "Master Workflow");
    whatMain.innerHTML = `Bước <b>${escapeHtml(activeStepNum)}/${escapeHtml(totalSteps)}</b> · ${escapeHtml(activeStepTitle)}`;
  }
  const whatSub = document.getElementById("hud-what-sub");
  if (whatSub) {
    const activeSubId = (isRunning && lsp.sub_task_id) ? lsp.sub_task_id : (le.active_sub_task_id || "TASK-M01");
    const activeSubTitle = (isRunning && lsp.sub_task_title) ? lsp.sub_task_title : (le.active_sub_task_title || dashboardState.active_task_title || "");
    whatSub.innerHTML = `Sub-Task: <code>${escapeHtml(activeSubId)}</code> — ${escapeHtml(activeSubTitle)}`;
  }

  const whoMain = document.getElementById("hud-who-main");
  if (whoMain) {
    if (isRunning && lsp.active_role_slot) {
      whoMain.innerHTML = `<code>${escapeHtml(lsp.active_role_slot)}</code>`;
    } else {
      whoMain.innerHTML = `${escapeHtml(le.active_role || "🧠 BRAIN · Architect")} · <code>${escapeHtml(le.active_agent_slot || "Slot A")}</code>`;
    }
  }
  setText("hud-who-model", le.active_model || dashboardState.planner_model || "gemini-4-argon (antigravity)");
  setText("hud-who-skill", le.active_skill || "/brain-delegate");

  const whereMain = document.getElementById("hud-where-main");
  if (whereMain) {
    whereMain.innerHTML = `📦 <code>${escapeHtml(le.active_repo_name || "centralized-data-service")}</code> · ⎇ <code>${escapeHtml(le.active_branch || "main")}</code>`;
  }
  setText("hud-where-file", (isRunning && lsp.current_where) ? lsp.current_where : (le.active_file_line || le.active_worktree_path || ""));
  setText("hud-how-cmd", le.active_command || "go test -race ./...");

  const repoListEl = document.getElementById("hud-repo-progress-list");
  if (repoListEl) {
    const repos = le.repo_progress || [];
    repoListEl.innerHTML = repos
      .map((rp) => {
        const rpPct = Number(rp.progress_pct ?? 0);
        const stBadge = rp.verification_status === "VERIFIED" ? "✓ VERIFIED" : rp.verification_status || "READY";
        return `
          <div class="hud-repo-row" title="${escapeAttr(rp.repo_path || "")} | ${escapeAttr(rp.active_sub_task || "")} | ${escapeAttr(rp.diff_summary || "")}">
            <div class="hud-repo-top">
              <span>📦 <b>${escapeHtml(rp.repo_name)}</b> <small>(${escapeHtml(rp.active_role || "Muscle")} · ${escapeHtml(rp.active_model || "gemini-4-argon")})</small></span>
              <span><b>${rp.done_sub_tasks}/${rp.total_sub_tasks}</b> (${rpPct}%) · <code class="hud-ver-tag">${escapeHtml(stBadge)}</code></span>
            </div>
            <div class="hud-repo-mini-track">
              <div class="hud-repo-mini-fill" style="width:${Math.max(4, rpPct)}%"></div>
            </div>
          </div>
        `;
      })
      .join("");
  }
}

/* ============================================================================
   CO-ARCHITECT & SOLUTION PLANNING STUDIO (CO-ARCHITECT-CHAT-01)
   ============================================================================ */
function scrollToCoArchitectStudio() {
  if (currentScreenMode !== "autopilot") {
    switchScreen("autopilot");
  }
  const el = document.getElementById("co-architect-studio");
  if (el) {
    el.scrollIntoView({ behavior: "smooth", block: "center" });
    const ta = document.getElementById("inp-co-architect-msg");
    if (ta) setTimeout(() => ta.focus(), 200);
  }
}

function fillQuickSteeringPrompt(mode, text) {
  const modeSel = document.getElementById("cas-mode-select");
  if (modeSel && mode) modeSel.value = mode;
  const ta = document.getElementById("inp-co-architect-msg");
  if (ta) {
    ta.value = text;
    ta.focus();
  }
}

function renderCoArchitectStudio() {
  if (!dashboardState) return;
  const listEl = document.getElementById("co-architect-chat-messages");
  if (!listEl) return;
  const messages = dashboardState.active_task_chat || [];

  listEl.innerHTML = messages
    .map((m) => {
      const isUser = m.sender === "user";
      const modeBadge =
        m.action_mode === "revise_solution"
          ? "✍️ CẬP NHẬT 09_tasks_solution.md"
          : m.action_mode === "update_subtasks"
          ? "🧩 CẬP NHẬT 08_tasks.md"
          : "💬 THẢO LUẬN KIẾN TRÚC";
      const appliedHtml = m.applied_to_plan
        ? `<span class="cas-applied-pill">✓ Đã ghi vào ${escapeHtml(formatRepoBasename(m.target_file || "09_tasks_solution.md"))}</span>`
        : "";
      return `
        <div class="cas-msg-card ${isUser ? "user-msg" : "agent-msg"}">
          <div class="cas-msg-hdr">
            <div class="cas-msg-author">
              <strong>${escapeHtml(m.agent_role || (isUser ? "👤 You" : "🧠 BRAIN"))}</strong>
              <span class="cas-slot-pill">${escapeHtml(m.agent_slot || "Slot A")}</span>
              ${m.skill ? `<span class="cas-skill-pill">${escapeHtml(m.skill)}</span>` : ""}
              <span class="cas-model-pill">🤖 ${escapeHtml(m.model || "gemini-4-argon")}</span>
            </div>
            <div class="cas-msg-meta">
              <span class="cas-mode-pill">${modeBadge}</span>
              ${appliedHtml}
              <span class="cas-time">${escapeHtml(m.timestamp || "")}</span>
            </div>
          </div>
          <pre class="cas-msg-body">${escapeHtml(m.message || "")}</pre>
        </div>
      `;
    })
    .join("");
  listEl.scrollTop = listEl.scrollHeight;
}

async function sendCoArchitectChatMessage(ev, overrideMode) {
  if (ev && ev.preventDefault) ev.preventDefault();
  const ta = document.getElementById("inp-co-architect-msg");
  if (!ta || !ta.value.trim()) return;
  const msg = ta.value.trim();
  const role = document.getElementById("cas-role-select")?.value || "BRAIN";
  const model = document.getElementById("cas-model-select")?.value || "gemini-4-argon (antigravity)";
  const actionMode = overrideMode || document.getElementById("cas-mode-select")?.value || "discuss";
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-1";

  ta.value = "";
  try {
    const res = await fetch("/api/tasks/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        task_key: activeKey,
        message: msg,
        action_mode: actionMode,
        agent_role: role,
        model: model,
      }),
    });
    if (!res.ok) {
      alert("Không thể gửi tin nhắn Co-Architect: " + (await res.text()));
      return;
    }
    dashboardState = await res.json();
    if (actionMode === "revise_solution") {
      selectedReportTab = "S3";
      userLockedReportTab = true;
    }
    renderAll();
  } catch (e) {
    console.error("sendCoArchitectChatMessage error:", e);
  }
}

async function submitCoArchitectDirectRevise() {
  const ta = document.getElementById("inp-co-architect-msg");
  if (!ta || !ta.value.trim()) {
    fillQuickSteeringPrompt(
      "revise_solution",
      "Bổ sung cơ chế Retry Exponential Backoff + Dead-Letter Queue (DLQ) khi BulkWrite MongoDB gặp lỗi transient network timeout, giữ nguyên OCC _source_ts."
    );
  }
  await sendCoArchitectChatMessage(null, "revise_solution");
}

function renderAll() {
  if (!dashboardState) return;

  const brEl = document.getElementById("hdr-project-branch");
  if (brEl) {
    brEl.textContent = `${dashboardState.project_name} · ${dashboardState.branch}${dashboardState.git_commit ? " (" + dashboardState.git_commit + ")" : ""}`;
  }

  const gateStrip = document.getElementById("gate-badges-strip");
  if (gateStrip && Array.isArray(dashboardState.gates)) {
    gateStrip.innerHTML = dashboardState.gates
      .map((g) => {
        const isPass = String(g.Status || g.status || "").startsWith("PASS");
        return `<span class="gate-chip ${isPass ? "" : "warn"}">${escapeHtml(g.Name || g.name)}: ${escapeHtml(g.Status || g.status)}</span>`;
      })
      .join("");
  }

  renderCrossContextBar();
  renderGlobalRunningProgressBanner();
  renderSidebarTaskQueue();
  renderLiveTelemetryHUD();
  renderAgentKernelCard();
  renderIsolationInspector();
  renderAutopilot();
  renderStepReportInspector();
  renderCoArchitectStudio();
  renderWaveBoard();
  renderVibeStudio();
}

/* ============================================================================
   1. RENDER VIEW 1: AUTOPILOT & PER-TASK ISOLATION ENGINE
   ============================================================================ */
function mapAutopilotBoxIndexToReportTab(idx) {
  const grp = (dashboardState?.autopilot_stages || [])[idx];
  const gid = String(grp?.id || "").toLowerCase();
  if (gid === "start") return "S1";
  if (gid === "critique") return "DG0";
  if (gid === "plan" || gid === "you") return "S3";
  if (gid === "implement") return "S4";
  if (gid === "qa" || gid === "pr") return "S5";
  if (idx === 0) return "S1";
  if (idx === 1) return "DG0";
  if (idx === 2 || idx === 3) return "S3";
  if (idx === 4) return "S4";
  return "S5";
}

function selectAutopilotStageBox(idx) {
  selectedReportTab = mapAutopilotBoxIndexToReportTab(idx);
  userLockedReportTab = true;
  renderStepReportInspector();
}

function renderAutopilot() {
  if (!dashboardState) return;

  const routingEl = document.getElementById("autopilot-model-routing");
  if (routingEl) {
    const sh = dashboardState.active_self_heal_state || {};
    const retryCnt = sh.retry_count ?? 0;
    const maxRetries = sh.max_retries || 3;
    const qaMdl = dashboardState.qa_auditor_model || dashboardState.cli_settings?.qa_auditor_model || "claude-opus-4-6 (antigravity)";
    const layaMdl = dashboardState.laya_gate_model || dashboardState.cli_settings?.laya_gate_model || "convaiinnovations/laya (local :8200)";
    const secMdl = dashboardState.security_model || dashboardState.cli_settings?.security_model || "agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)";
    routingEl.innerHTML = `Planner: <b>${escapeHtml(dashboardState.planner_model || "gemini-4-argon")}</b> · Coder: <b>${escapeHtml(dashboardState.coder_high_model || "gemini-4-argon")}</b> · Cross-Model QA (Slot K): <b>${escapeHtml(qaMdl)}</b> · Laya Gate (Slot H · 0 Token): <b>${escapeHtml(layaMdl)}</b> · Security (Slot I): <b>${escapeHtml(secMdl)}</b> · Circuit Breaker: <b>Retry ${retryCnt}/${maxRetries}</b>`;
  }

  const filterPill = document.getElementById("autopilot-filter-pill");
  if (filterPill) {
    const activeKey = dashboardState.active_task_key || selectedChainKey || "TASK-1";
    const activeStage = dashboardState.active_task_stage || "READY";
    const repoShort = formatAffectedReposSummary(dashboardState.active_affected_repos, dashboardState.active_target_repo);
    filterPill.textContent = `Đang xem 100% ngữ cảnh của: ${activeKey} (${activeStage}) · Repos: ${repoShort} · Worktree Attached (Slot C): ${dashboardState.active_isolation?.branch_name || "task/" + activeKey}`;
  }

  const flowEl = document.getElementById("autopilot-flow");
  if (flowEl) {
    const groups = dashboardState.autopilot_stages || [];
    const activeStage = dashboardState.active_task_stage || "READY";
    const htmlParts = [];
    groups.forEach((grp, idx) => {
      const isSingle = (grp.cards || []).length === 1 && !grp.round_badge;
      const cardsHtml = (grp.cards || [])
        .map((c) => {
          const st = String(c.status || "done").toLowerCase();
          let icon = "✓";
          if (st === "running") icon = `<span class="spin-inline">⟳</span>`;
          else if (st === "waiting") icon = "●";
          else if (st === "pending") icon = "◌";

          const isApproveGateCard = c.title.startsWith("You") && activeStage === "NEEDS_YOU";
          const roleLine = c.role_label
            ? `<div class="ap-subcard-role">${escapeHtml(c.role_label)} · <code>${escapeHtml(c.agent_slot || "")}</code> · <span>${escapeHtml(c.model || "")}</span></div>`
            : "";
          return `
            <div class="ap-subcard status-${escapeAttr(st)} ${st === "running" ? "is-actively-running" : ""}" onclick="selectAutopilotStageBox(${idx})" style="cursor:pointer;" title="Click để mở Báo Cáo & Code Demo của bước này">
              <div class="ap-subcard-title"><span class="chk">${icon}</span> ${escapeHtml(c.title)}</div>
              ${c.subtitle ? `<div class="ap-subcard-sub">${escapeHtml(c.subtitle)}</div>` : ""}
              ${roleLine}
              ${c.meta ? `<div class="ap-subcard-meta">${escapeHtml(c.meta)}</div>` : ""}
              ${
                isApproveGateCard
                  ? `<button type="button" class="mini-run-btn" style="margin-top:5px;width:100%;" onclick="runTaskAction('', 'approve_and_run', event)">✓ APPROVE &amp; Run S4</button>`
                  : ""
              }
            </div>
          `;
        })
        .join("");

      htmlParts.push(`
        <div class="ap-group-shell ${isSingle ? "single" : ""}">
          ${cardsHtml}
          ${grp.round_badge ? `<div class="ap-round-foot">${escapeHtml(grp.round_badge)}</div>` : ""}
        </div>
      `);
      if (idx < groups.length - 1) {
        htmlParts.push(`<span class="ap-arrow">→</span>`);
      }
    });
    flowEl.innerHTML = htmlParts.join("");
  }

  const cyclesEl = document.getElementById("autopilot-cycles");
  if (cyclesEl) {
    const cycles = dashboardState.autopilot_cycles || [];
    const activeKey = dashboardState.active_task_key || selectedChainKey || "";
    cyclesEl.innerHTML = cycles
      .map((sec) => {
        const rows = sec.rows || [];
        const rowsHtml = rows
          .map((r) => {
            const isSelected = Boolean(activeKey && r.task_key === activeKey);
            const leftHtml = r.status_word
              ? `${escapeHtml(r.left_label)}: <span class="ap-status-word ${escapeAttr(r.status_tone)}">${escapeHtml(r.status_word)}</span>`
              : escapeHtml(r.left_label);
            return `
              <div class="ap-row clickable ${isSelected ? "selected-row" : ""}"
                   onclick="selectTaskFromAutopilot('${escapeAttr(r.task_key || "")}', '${escapeAttr(r.agent_slot || "")}')">
                <span>${leftHtml}</span>
                <span class="ap-row-actions">
                  <span>${escapeHtml(r.right_summary || "")}</span>
                </span>
              </div>
            `;
          })
          .join("");

        return `
          <div class="ap-cycle-block">
            <div class="ap-cycle-hdr">
              <span>${escapeHtml(sec.title)}</span>
              <span>${escapeHtml(sec.right_badge || "")}</span>
            </div>
            ${rowsHtml || `<div class="ap-row"><span>Chưa có vòng chạy nào cho Task này.</span></div>`}
          </div>
        `;
      })
      .join("");
  }

  const totalEl = document.getElementById("autopilot-total-val");
  if (totalEl) {
    const runs = dashboardState.total_runs ?? 0;
    const mins = dashboardState.total_minutes ?? 0;
    const cost = Number(dashboardState.total_cost_usd ?? 0).toFixed(2);
    totalEl.textContent = `${runs} runs · ${mins} min · $${cost}`;
  }
}

/* ============================================================================
   2. RENDER VIEW 2: 3-TIER HIERARCHICAL WAVE 0..N DEPENDENCY DAG (EPIC -> TASK -> SUB-TASK)
   ============================================================================ */
const EPIC_THEME_TITLES = {
  DATA: "Data Pipeline, CDC & Master MongoDB Sink",
  CORE: "Agentic Kernel, Memory & Governance Hooks",
  SRV: "Backend Microservices & API Gateway",
  UX: "Mission Control WebUI & Interactive Studio",
  QA: "Adversarial Audit S5 & Gates G1–G8 / DG3",
  CLI: "Multi-CLI Runners & Telegram Bridge",
  PM: "Architecture Planning & Task Decomposition",
};

function renderWaveBoard() {
  if (!dashboardState) return;
  const sc = dashboardState.status_counts || {};
  setText("cnt-needs-you", sc.needs_you ?? 0);
  setText("cnt-running", sc.running ?? 0);
  setText("cnt-in-review", sc.in_review ?? 0);
  setText("cnt-under-way", sc.under_way ?? 0);
  setText("cnt-ready", sc.ready ?? 0);
  setText("cnt-blocked", sc.blocked ?? 0);
  setText("cnt-done", sc.done ?? 0);

  const colsContainer = document.getElementById("wave-columns");
  if (!colsContainer) return;

  const activeKey = dashboardState.active_task_key || selectedChainKey || "TASK-1";
  const lsp = dashboardState.live_step_progress || {};
  const le = dashboardState.live_execution || {};
  const allNodes = getAllWaveNodes();

  const assigneeFilter = (document.getElementById("filter-assignee")?.value || "").toLowerCase();
  const msFilter = (document.getElementById("filter-milestone")?.value || "").toUpperCase();
  const dropdownEpic = (document.getElementById("filter-epic")?.value || "").toUpperCase();
  if (dropdownEpic && dropdownEpic !== selectedHierarchyEpic) {
    selectedHierarchyEpic = dropdownEpic;
  }
  const searchVal = (document.getElementById("wave-search")?.value || "").toLowerCase().trim();

  // ---------------------------------------------------------------------------
  // TIER 1: INDEPENDENT EPICS (CHỦ ĐỀ LỚN — KHÔNG NỐI DÂY GIỮA CÁC EPIC)
  // ---------------------------------------------------------------------------
  const epicMap = new Map();
  for (const n of allNodes) {
    const ep = String(n.epic || "DATA").toUpperCase();
    if (!epicMap.has(ep)) {
      epicMap.set(ep, {
        code: ep,
        theme: EPIC_THEME_TITLES[ep] || `Chủ đề ${ep}`,
        tasks: [],
        doneTasks: 0,
        needsYouTasks: 0,
        subTaskCount: 0,
        repos: new Set(),
      });
    }
    const g = epicMap.get(ep);
    g.tasks.push(n);
    if (n.stage === "DONE") g.doneTasks += 1;
    if (n.stage === "NEEDS_YOU") g.needsYouTasks += 1;
    g.subTaskCount += (n.sub_tasks || []).length;
    for (const rp of n.affected_repos || [n.target_repo]) {
      if (rp) g.repos.add(formatRepoBasename(rp));
    }
  }

  const tierEpicsEl = document.getElementById("wave-tier-epics");
  if (tierEpicsEl) {
    const totalTasksAll = allNodes.length;
    const totalDoneAll = allNodes.filter((x) => x.stage === "DONE").length;
    const allCardHtml = `
      <div class="wave-epic-card ${!selectedHierarchyEpic ? "active-epic" : ""}"
           onclick="selectHierarchyEpic('')">
        <div class="wec-top">
          <span class="wec-code">🌐 TẤT CẢ EPICS</span>
          <span class="wec-count">${epicMap.size} Epics · ${totalDoneAll}/${totalTasksAll} Tasks DONE</span>
        </div>
        <div class="wec-theme">Hiển thị toàn bộ chủ đề độc lập (Bấm chọn từng Epic bên cạnh để lọc riêng Tasks thuộc chủ đề đó)</div>
      </div>
    `;

    const epicCardsHtml = Array.from(epicMap.values())
      .map((eg) => {
        const isSelected = selectedHierarchyEpic === eg.code;
        const hasActiveTask = eg.tasks.some((t) => t.key === activeKey);
        const repoList = Array.from(eg.repos).slice(0, 3).join(", ") || "ai-orchestrator";
        return `
          <div class="wave-epic-card ${isSelected ? "active-epic" : ""} ${hasActiveTask ? "contains-active-task" : ""}"
               onclick="selectHierarchyEpic('${escapeAttr(eg.code)}')">
            <div class="wec-top">
              <span class="wec-code">🏛️ EPIC: ${escapeHtml(eg.code)}</span>
              <span class="wec-count">${eg.doneTasks}/${eg.tasks.length} Tasks · 🧩 ${eg.subTaskCount} Sub-Tasks</span>
            </div>
            <div class="wec-theme">${escapeHtml(eg.theme)}</div>
            <div class="wec-foot">
              <span>📦 ${escapeHtml(repoList)}</span>
              ${eg.needsYouTasks > 0 ? `<span class="wec-Gate">● ${eg.needsYouTasks} NEEDS YOU</span>` : ""}
            </div>
          </div>
        `;
      })
      .join("");

    tierEpicsEl.innerHTML = allCardHtml + epicCardsHtml;
  }

  setText(
    "wave-tier-epics-meta",
    selectedHierarchyEpic ? `Đang lọc theo Epic: ${selectedHierarchyEpic} (Bấm lại để bỏ lọc)` : `Đang hiển thị: TẤT CẢ EPICS (${epicMap.size} chủ đề độc lập)`
  );
  setText("wave-selected-epic-label", selectedHierarchyEpic || "TẤT CẢ EPICS");

  // ---------------------------------------------------------------------------
  // TIER 2: TASKS BELONGING TO THE SELECTED EPIC
  // ---------------------------------------------------------------------------
  const filteredTasks = allNodes.filter((n) => {
    if (selectedHierarchyEpic && String(n.epic || "").toUpperCase() !== selectedHierarchyEpic) return false;
    if (assigneeFilter && !String(n.assignee || "").toLowerCase().includes(assigneeFilter)) return false;
    if (msFilter && String(n.milestone || "").toUpperCase() !== msFilter) return false;
    if (searchVal) {
      const hay = `${n.key} ${n.title} ${n.milestone} ${n.epic} ${n.model} ${n.target_repo || ""} ${(n.affected_repos || []).join(" ")} ${n.agent_slot || ""}`.toLowerCase();
      if (!hay.includes(searchVal)) return false;
    }
    return true;
  });

  const tierTasksEl = document.getElementById("wave-tier-tasks");
  if (tierTasksEl) {
    tierTasksEl.innerHTML = filteredTasks
      .map((n) => {
        const isFocused = n.key === activeKey;
        const isTaskRunning = Boolean(
          (lsp.active && lsp.task_key === n.key) ||
            (isLifecycleActionRunning && runningActionTaskKey === n.key) ||
            n.stage === "RUNNING" ||
            (isFocused && le.is_running)
        );
        const runPct = isTaskRunning ? Math.max(8, Math.min(100, Number(lsp.percent || le.overall_progress_pct || 24))) : 0;
        const stageLabel = isTaskRunning
          ? `<span class="running-pulse-pill"><span class="spin-inline">⟳</span> ĐANG CHẠY ${runPct}%</span>`
          : `<span class="wcard-stage ${escapeAttr(n.stage)}">${formatStageBadge(n.stage)}</span>`;
        const repoShort = formatAffectedReposSummary(n.affected_repos, n.target_repo);
        const subCnt = (n.key === activeKey ? dashboardState.active_sub_tasks || n.sub_tasks || [] : n.sub_tasks || []).length;
        const runAction = n.stage === "NEEDS_YOU" ? "approve_and_run" : "autopilot";
        const runText = n.stage === "NEEDS_YOU" ? "✓ Approve & S4" : "▶ Run";

        const actionBtnsHtml = isTaskRunning
          ? `
            <button type="button" class="wcard-stop-btn" onclick="stopRunningProcess('${escapeAttr(n.key)}', '', event)">⏹ Dừng</button>
            <button type="button" class="wcard-del-btn" title="Xóa Task ${escapeAttr(n.key)}" onclick="deleteSingleTask('${escapeAttr(n.key)}', event)">🗑</button>
          `
          : `
            <button type="button" class="wcard-jump-btn" onclick="runTaskAction('${escapeAttr(n.key)}', '${runAction}', event)">${runText}</button>
            <button type="button" class="wcard-del-btn" title="Xóa Task ${escapeAttr(n.key)}" onclick="deleteSingleTask('${escapeAttr(n.key)}', event)">🗑</button>
          `;

        return `
          <div class="wave-task-card ${isFocused ? "active-task-card" : ""} ${isTaskRunning ? "is-actively-running" : ""}"
               id="wtcard-${escapeAttr(n.key)}"
               onclick="selectHierarchyTask('${escapeAttr(n.key)}')">
            <div class="wtc-top">
              <div class="wcard-id-group">
                <span class="wcard-key">${escapeHtml(n.key)}</span>
                <span class="wcard-ms">Epic ${escapeHtml(n.epic || "DATA")} · ${escapeHtml(n.milestone || "M1")}</span>
                <span class="wcard-model">${escapeHtml(n.model || "gemini-4-argon")}</span>
              </div>
              ${stageLabel}
            </div>
            <div class="wcard-title" title="${escapeAttr(n.title)}">${escapeHtml(n.title)}</div>
            <div class="wcard-cross-meta">
              <span>📦 ${escapeHtml(repoShort)} · 🧩 <b>${subCnt} Sub-Tasks</b></span>
              <span class="wcard-jump-group">
                ${actionBtnsHtml}
              </span>
            </div>
          </div>
        `;
      })
      .join("");
  }
  setText("wave-tier-tasks-meta", `${filteredTasks.length} Tasks hiển thị`);

  // ---------------------------------------------------------------------------
  // TIER 3: SUB-TASKS DEPENDENCY DAG (WAVE 0..N WITH SVG BEZIER LINES) OF SELECTED TASK
  // ---------------------------------------------------------------------------
  const activeNode = findWaveNodeByKey(activeKey) || filteredTasks[0] || allNodes[0];
  const selectedTaskKey = activeNode?.key || activeKey;
  const subTasks =
    selectedTaskKey === dashboardState.active_task_key && Array.isArray(dashboardState.active_sub_tasks) && dashboardState.active_sub_tasks.length > 0
      ? dashboardState.active_sub_tasks
      : activeNode?.sub_tasks || [];

  setText("wave-selected-task-label", `${selectedTaskKey} (${activeNode?.title || ""})`);
  setText("wave-tier-subtasks-meta", `${subTasks.length} Sub-Tasks chia theo cột Wave 0..N (Bấm chọn 1 Sub-Task để xem Đầu vào, Plan riêng & Luồng Start ➔ Isolated Branch ở dưới)`);

  if (subTasks.length > 0 && (!selectedSubTaskID || !subTasks.some((x) => (x.sub_task_id || x.id) === selectedSubTaskID))) {
    selectedSubTaskID = subTasks[0].sub_task_id || subTasks[0].id || "";
  }
  const activeSubTaskObj = subTasks.find((x) => (x.sub_task_id || x.id) === selectedSubTaskID) || subTasks[0] || null;

  const byWave = new Map([
    [0, []],
    [1, []],
    [2, []],
    [3, []],
    [4, []],
  ]);
  for (const st of subTasks) {
    if (searchVal) {
      const stHay = `${st.sub_task_id || st.id} ${st.title} ${st.target_repo_name || ""} ${st.context_file_line || ""} ${st.role_label || ""}`.toLowerCase();
      if (!stHay.includes(searchVal)) continue;
    }
    const w = Number(st.wave_index ?? st.wave ?? 0);
    if (!byWave.has(w)) byWave.set(w, []);
    byWave.get(w).push(st);
  }

  colsContainer.innerHTML = Array.from(byWave.entries())
    .filter(([wNum, list]) => list.length > 0 || wNum <= 3)
    .map(([wNum, list]) => {
      const doneCnt = list.filter((x) => String(x.status || "").toUpperCase() === "DONE" || String(x.stage || "").toUpperCase() === "DONE").length;
      const grpLabel = list[0]?.group_title ? list[0].group_title.split("(")[0].trim() : `Wave ${wNum}`;
      const cardsHtml = list
        .map((st) => {
          const stId = st.sub_task_id || st.id || "";
          const isSelectedSub = stId === selectedSubTaskID;
          const stRepoName = st.target_repo_name || formatRepoBasename(st.target_repo || dashboardState.active_target_repo);
          const stCtx = st.context_file_line || st.target_files || stRepoName;
          const stStage = String(st.stage || st.status || "PENDING").toUpperCase();
          const isSubRunning = Boolean(
            stStage === "RUNNING" ||
              (lsp.active && lsp.task_key === selectedTaskKey && lsp.sub_task_id === stId) ||
              (isLifecycleActionRunning && runningActionTaskKey === selectedTaskKey && runningActionSubTaskID === stId)
          );
          const stBranch = st.branch_name || `task/${selectedTaskKey}/${stId.toLowerCase()}`;
          const deps = Array.isArray(st.depends_on) && st.depends_on.length > 0 ? `📥 Input: ${st.depends_on.join(", ")}` : "📥 Wave 0 Entry";
          const downIds = st.output_contract?.downstream_sub_task_ids || [];
          const downStr = downIds.length > 0 ? `📤 Output ➔ ${downIds.join(", ")}` : "📤 Output ➔ Parent";
          const stBadgeHtml = isSubRunning
            ? `<span class="running-pulse-pill"><span class="spin-inline">⟳</span> ĐANG CHẠY</span>`
            : `<span class="wcard-stage ${escapeAttr(stStage)}">● ${escapeHtml(stStage)}</span>`;

          return `
            <div class="wave-card chain-active critical-card ${isSelectedSub ? "active-subtask-wave-card" : ""} ${isSubRunning ? "is-actively-running" : ""}"
                 id="wcard-${escapeAttr(stId)}"
                 data-key="${escapeAttr(stId)}"
                 onclick="selectSubTaskFractal('${escapeAttr(stId)}', event)">
              <div class="wcard-top">
                <div class="wcard-id-group">
                  <span class="wcard-key">${escapeHtml(stId)}</span>
                  <span class="wcard-ms">📦 ${escapeHtml(stRepoName)}</span>
                  <span class="wcard-model">${escapeHtml(st.model || "gemini-4-argon")}</span>
                </div>
                <div style="display:flex;align-items:center;gap:4px;">
                  ${stBadgeHtml}
                  <button type="button" class="wcard-del-btn" title="Xóa Sub-Task ${escapeAttr(stId)}" onclick="deleteSingleSubTask('${escapeAttr(selectedTaskKey)}', '${escapeAttr(stId)}', event)">🗑</button>
                </div>
              </div>
              <div class="wcard-title" title="${escapeAttr(st.title)}">${escapeHtml(st.title)}</div>
              <div class="wcard-cross-meta">
                <span title="${escapeAttr(stCtx)}">📄 ${escapeHtml(stCtx)}</span>
                <code class="sfi-branch-pill">⎇ ${escapeHtml(stBranch)}</code>
              </div>
              <div class="wcard-cross-meta" style="margin-top:4px;">
                <span class="sri-dep-pill">${escapeHtml(deps)}</span>
                <span class="sri-dep-pill">${escapeHtml(downStr)}</span>
              </div>
            </div>
          `;
        })
        .join("");

      return `
        <div class="wave-col" data-wave="${wNum}">
          <div class="wave-col-hdr">
            <span title="${escapeAttr(grpLabel)}">Wave ${wNum} · ${escapeHtml(grpLabel)}</span>
            <span>${doneCnt}/${list.length}</span>
          </div>
          ${cardsHtml}
        </div>
      `;
    })
    .join("");

  const waveSubInspectorEl = document.getElementById("wave-subtask-fractal-inspector");
  if (waveSubInspectorEl) {
    waveSubInspectorEl.innerHTML = buildSubTaskFractalInspectorHTML(activeSubTaskObj, selectedTaskKey);
  }

  setText(
    "wave-item-count",
    `${epicMap.size} Epics · ${filteredTasks.length}/${allNodes.length} Tasks · ${subTasks.length} Sub-Tasks (${selectedTaskKey})`
  );

  if (currentScreenMode === "wave") {
    requestAnimationFrame(() => drawWaveBezierEdges());
  }
}

function formatStageBadge(stage) {
  switch (stage) {
    case "READY":
      return "● READY";
    case "NEEDS_YOU":
      return "● NEEDS YOU";
    case "BLOCKED":
      return "◌ BLOCKED";
    case "RUNNING":
      return "● RUNNING";
    case "DONE":
      return "● DONE";
    default:
      return "● " + stage;
  }
}

function drawWaveBezierEdges() {
  const svg = document.getElementById("wave-svg");
  const canvas = document.getElementById("wave-canvas");
  if (!svg || !canvas || !dashboardState) return;

  const canvasRect = canvas.getBoundingClientRect();
  if (canvasRect.width === 0) return;

  svg.setAttribute("width", canvas.scrollWidth);
  svg.setAttribute("height", canvas.scrollHeight);
  svg.setAttribute("viewBox", `0 0 ${canvas.scrollWidth} ${canvas.scrollHeight}`);

  const activeKey = dashboardState.active_task_key || selectedChainKey || "TASK-1";
  const activeNode = findWaveNodeByKey(activeKey);
  const subs =
    Array.isArray(dashboardState.active_sub_tasks) && dashboardState.active_sub_tasks.length > 0
      ? dashboardState.active_sub_tasks
      : activeNode?.sub_tasks || [];

  const edges = [];
  for (let i = 0; i < subs.length; i++) {
    const curId = subs[i].sub_task_id || subs[i].id;
    const deps = subs[i].depends_on || [];
    if (deps.length > 0) {
      for (const d of deps) {
        edges.push({ from: d, to: curId, critical: true, active_chain: true });
      }
    } else if (i > 0 && Number(subs[i].wave_index ?? subs[i].wave ?? 0) > Number(subs[i - 1].wave_index ?? subs[i - 1].wave ?? 0)) {
      edges.push({
        from: subs[i - 1].sub_task_id || subs[i - 1].id,
        to: curId,
        critical: true,
        active_chain: true,
      });
    }
  }

  const paths = [];
  for (const e of edges) {
    const fromEl = document.getElementById(`wcard-${e.from}`);
    const toEl = document.getElementById(`wcard-${e.to}`);
    if (!fromEl || !toEl) continue;

    const r1 = fromEl.getBoundingClientRect();
    const r2 = toEl.getBoundingClientRect();
    if (r1.width === 0 || r2.width === 0) continue;

    const x1 = r1.right - canvasRect.left;
    const y1 = r1.top - canvasRect.top + r1.height / 2;
    const x2 = r2.left - canvasRect.left;
    const y2 = r2.top - canvasRect.top + r2.height / 2;

    const dx = Math.max(20, (x2 - x1) * 0.5);
    const d = `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;

    const stroke = e.critical ? "#eab308" : "#38bdf8";
    const width = e.critical ? "2.3" : "2.0";
    const zOrder = e.critical ? 3 : 2;

    paths.push({ d, stroke, width, zOrder });
  }

  paths.sort((a, b) => a.zOrder - b.zOrder);
  svg.innerHTML = paths
    .map((p) => `<path d="${p.d}" fill="none" stroke="${p.stroke}" stroke-width="${p.width}" stroke-linecap="round" />`)
    .join("");
}

window.addEventListener("resize", () => {
  if (currentScreenMode === "wave") {
    requestAnimationFrame(() => drawWaveBezierEdges());
  }
});

/* ============================================================================
   3. RENDER VIEW 3 & SIDEBAR: WORKFLOW STEPS, PROJECTS, & TERMINAL GRID A..K
   ============================================================================ */
function renderVibeStudio() {
  if (!dashboardState) return;
  const activeKey = dashboardState.active_task_key || selectedChainKey || "TASK-1";
  const repoShort = formatRepoShortName(dashboardState.active_target_repo);

  setText("vibe-proj-name", repoShort);
  setText("sidebar-wf-task-tag", `${activeKey} · ${dashboardState.active_task_stage || "READY"}`);
  setText("vibe-branch-tag", `⎇ ${dashboardState.active_isolation?.branch_name || dashboardState.branch || "HEAD"}`);
  setText("toast-branch", dashboardState.branch || "HEAD");
  if (dashboardState.workspace_name) {
    setText("vibe-ws-tab-name", `${activeKey} · ${repoShort}`);
  }
  const hintEl = document.getElementById("vibe-active-filter-hint");
  if (hintEl) {
    hintEl.textContent = `Đang gắn 100% các ô Agent A..K vào Task ${activeKey} (${dashboardState.active_task_title || ""}) trên Repo ${repoShort}`;
  }

  const gitSumEl = document.getElementById("toast-git-summary");
  if (gitSumEl && dashboardState.git_status_summary) {
    gitSumEl.textContent = `${dashboardState.project_name} · ${dashboardState.branch}: ${dashboardState.git_status_summary}`;
  }

  // Left Sidebar Section 3: Projects & Agent Slots A..K
  const projListEl = document.getElementById("vibe-projects-list");
  if (projListEl) {
    const projs = dashboardState.projects || [];
    projListEl.innerHTML = projs
      .map((p) => {
        const agentsHtml = (p.agents || [])
          .map((a) => {
            const isSlotSelected = selectedAgentSlot === a.slot;
            return `
            <div class="vibe-agent-item ${isSlotSelected ? "selected-slot" : ""}"
                 onclick="selectCrossAgent('${escapeAttr(a.slot)}', '${escapeAttr(activeKey)}')">
              <div class="vibe-agent-item-title">${escapeHtml(a.slot)} · ${escapeHtml(a.name)} (${escapeHtml(activeKey)})</div>
              <div class="vibe-agent-item-sub">${escapeHtml(repoShort)}</div>
              <div class="vibe-agent-item-status">${escapeHtml(a.status)}</div>
            </div>
          `;
          })
          .join("");

        return `
          <div class="vibe-proj-card ${p.active ? "active" : ""}">
            <div class="vibe-proj-hdr">
              <span><b>${escapeHtml(p.code)}</b> ${escapeHtml(repoShort)}</span>
              ${p.badge_count ? `<span class="vibe-proj-badge">${p.badge_count}</span>` : ""}
            </div>
            ${agentsHtml}
          </div>
        `;
      })
      .join("");
  }

  // Left Sidebar Section 2: 17-Step Workflow (Scoped to Active Task + Role/Slot/Skill/Model Attribution)
  const wfEl = document.getElementById("vibe-workflow-list");
  if (wfEl) {
    const steps = dashboardState.workflow_steps || [];
    wfEl.innerHTML = steps
      .map((s) => {
        const isStepSelected = selectedStepNum === s.num;
        const isStepRunning = String(s.status || "").toLowerCase() === "running";
        const roleMeta = s.role_label
          ? `<div class="wf-step-role-meta">
               <span class="wf-role-pill">${escapeHtml(s.role_label)}</span>
               ${s.agent_slot ? `<span class="wf-slot-pill">Slot ${escapeHtml(s.agent_slot)}</span>` : ""}
               ${s.model ? `<span class="wf-model-pill">🤖 ${escapeHtml(s.model)}</span>` : ""}
             </div>`
          : "";
        return `
        <div class="wf-step ${escapeAttr(s.status)} ${isStepSelected ? "selected-step" : ""} ${isStepRunning ? "is-actively-running" : ""}"
             title="Click để xem Báo Cáo & Code Demo của bước ${s.num} (${escapeAttr(s.role_label || "")} · ${escapeAttr(s.model || "")}) cho ${escapeAttr(activeKey)}"
             onclick="selectCrossStep(${s.num})">
          ${isStepRunning ? `<span class="spin-inline" style="color:#38bdf8;font-weight:800;">⟳</span>` : `<i class="wf-dot"></i>`}
          <div class="wf-step-body">
            <span>${s.num}. ${escapeHtml(s.title)} ${s.note ? `<small style="opacity:0.75">${escapeHtml(s.note)}</small>` : ""}</span>
            ${roleMeta}
          </div>
        </div>
      `;
      })
      .join("");
  }

  // View 3: Asymmetric 4 Collapsed Vertical Strips (C, A, I, G) + Bento Grid + Status Bar J
  const allPanes = dashboardState.agent_panes || [];
  const stripSlots = new Set(["A", "I", "G"]);
  const stripsEl = document.getElementById("vibe-collapsed-strips");
  if (stripsEl) {
    const stripPanes = allPanes.filter((p) => stripSlots.has(p.slot) || p.slot === "C");
    stripsEl.innerHTML = stripPanes
      .map((p) => {
        const isFocused = selectedAgentSlot === p.slot;
        return `
          <div class="collapsed-strip-card ${isFocused ? "active-strip" : ""}"
               title="Click để focus Slot ${escapeAttr(p.slot)}: ${escapeAttr(p.title)}"
               onclick="selectCrossAgent('${escapeAttr(p.slot)}', '${escapeAttr(activeKey)}')">
            <span class="strip-slot-badge">${escapeHtml(p.slot)}</span>
            <span class="strip-vertical-title">${escapeHtml(p.title)} · ${escapeHtml(activeKey)}</span>
            <span class="dot green"></span>
          </div>
        `;
      })
      .join("");
  }

  const barJEl = document.getElementById("vibe-status-bar-j");
  const paneJ = allPanes.find((p) => p.slot === "J");
  if (barJEl && paneJ) {
    const lastLine = (paneJ.lines || [])[paneJ.lines.length - 1] || "";
    barJEl.innerHTML = `
      <span><b>Slot J · ${escapeHtml(paneJ.title)}</b> (${escapeHtml(paneJ.model)}) — ${escapeHtml(lastLine)}</span>
      <span>${escapeHtml(paneJ.status_text)}</span>
    `;
  }

  const gridEl = document.getElementById("vibe-panes-grid");
  if (gridEl) {
    const gridPanes = allPanes.filter((p) => {
      if (p.slot === "J") return false;
      if (activeRunnerFilter) {
        return String(p.runner || "").toLowerCase() === activeRunnerFilter;
      }
      if (p.slot === "C" || p.slot === selectedAgentSlot) return true;
      return !stripSlots.has(p.slot);
    });

    gridEl.innerHTML = gridPanes
      .map((p) => {
        const logsHtml = (p.lines || [])
          .map((ln) => `<div class="vp-log-line">${escapeHtml(ln)}</div>`)
          .join("");
        const isTerminal = p.runner === "terminal";
        const highlight = p.slot === selectedAgentSlot || p.slot === "C";
        const taskKey = activeKey;

        return `
          <div class="vibe-pane ${highlight ? "highlight-border" : ""}" id="vp-slot-${escapeAttr(p.slot)}">
            <div class="vp-hdr">
              <div class="vp-hdr-left">
                <span class="vp-runner-tag ${isTerminal ? "terminal" : ""}">${escapeHtml(p.runner_badge)}</span>
                <span class="vp-slot-badge">${escapeHtml(p.slot)} · ${escapeHtml(p.title)}</span>
                <span class="vp-task-badge"
                      onclick="jumpToScreenWithTask(event, 'wave', '${escapeAttr(taskKey)}', '${escapeAttr(p.slot)}')">
                  ${escapeHtml(taskKey)} ↗
                </span>
              </div>
              <div class="vp-hdr-left">
                <span class="vp-model-tag">${escapeHtml(p.model)}</span>
                <span class="vp-status">${escapeHtml(p.status_text)}</span>
              </div>
            </div>
            <div class="vp-logs">${logsHtml}</div>
            <form class="vp-input-wrap" onsubmit="sendAgentMessage(event, '${escapeAttr(p.slot)}', '${escapeAttr(taskKey)}')">
              <input type="text"
                     id="vp-inp-${escapeAttr(p.slot)}"
                     class="vp-input"
                     placeholder="${isTerminal ? "Lệnh shell trong Worktree cô lập (VD: git status -sb)..." : "Truy vấn FTS5 BM25 hoặc gửi lệnh cho Agent..."}" />
              <div class="vp-foot">
                <span>🎯 ${escapeHtml(repoShort)} · ⎇ ${escapeHtml(dashboardState.active_isolation?.branch_name || "worktree")}</span>
                <button type="submit" class="vp-send-btn">Gửi</button>
              </div>
            </form>
          </div>
        `;
      })
      .join("");
  }
}

function filterPanes(runner) {
  activeRunnerFilter = runner;
  renderVibeStudio();
}

function toggleGitToast(show) {
  const el = document.getElementById("vibe-git-toast");
  if (el) el.classList.toggle("hidden", !show);
}

async function runGitStatusInTerminal() {
  const inp = document.getElementById("vp-inp-C");
  if (inp) {
    inp.value = "git status -sb && git log -n 2 --oneline";
    const fakeEv = { preventDefault: () => {} };
    await sendAgentMessage(fakeEv, "C", dashboardState?.active_task_key || "TASK-7");
  }
}

async function sendQuickWorktreeCommand(ev) {
  if (ev && ev.preventDefault) ev.preventDefault();
  const inp = document.getElementById("inp-quick-worktree-cmd");
  if (!inp || !inp.value.trim()) return;
  const cmd = inp.value.trim();
  inp.value = "";
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-1";
  try {
    const res = await fetch("/api/agents/message", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        slot: "C",
        message: cmd,
        task_key: activeKey,
      }),
    });
    if (!res.ok) {
      alert("Gatekeeper H5 chặn lệnh: " + (await res.text()));
      return;
    }
    dashboardState = await res.json();
    renderAll();
  } catch (e) {
    console.error("sendQuickWorktreeCommand error:", e);
  }
}

async function sendAgentMessage(ev, slot, taskKey) {
  if (ev && ev.preventDefault) ev.preventDefault();
  const inp = document.getElementById(`vp-inp-${slot}`);
  if (!inp || !inp.value.trim()) return;
  const msg = inp.value.trim();
  inp.value = "";
  try {
    const res = await fetch("/api/agents/message", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        slot,
        message: msg,
        task_key: taskKey || dashboardState?.active_task_key || selectedChainKey || "",
      }),
    });
    if (!res.ok) {
      alert("Gatekeeper chặn lệnh: " + (await res.text()));
      return;
    }
    dashboardState = await res.json();
    selectedAgentSlot = slot;
    renderAll();
  } catch (e) {
    console.error("sendAgentMessage error:", e);
  }
}

async function triggerPipelineCycle() {
  await triggerPrimaryTaskAction();
}

/* ============================================================================
   4. MODALS: + GIAO VIỆC MỚI (DUAL-MODE INTAKE: TRACKER SYNC VS MANUAL + BRD)
   ============================================================================ */
let currentTaskIntakeMode = "manual"; // "tracker" | "manual"
let cachedTrackerIssues = [];
let selectedTrackerIssueKey = "";
let uploadedBRDFileName = "";

function openDispatchModal() {
  const modal = document.getElementById("dispatch-modal");
  if (!modal) return;
  const nextNum = (dashboardState?.total_items || 9) + 1;
  const idInp = document.getElementById("inp-task-id");
  if (idInp && !idInp.value) {
    idInp.value = `TASK-${nextNum}`;
  }

  // Populate Target Repository dropdown & Multi-Repo Checkboxes from dashboardState.available_repos
  const repoSel = document.getElementById("inp-task-repo");
  const multiBox = document.getElementById("inp-task-affected-repos");
  const repos = dashboardState?.available_repos || [];
  const activeRepo = dashboardState?.active_target_repo || "";
  const activeAffected = new Set(dashboardState?.active_affected_repos || (activeRepo ? [activeRepo] : []));

  if (repoSel) {
    repoSel.innerHTML = repos
      .map((r) => {
        const sel = r.path === activeRepo ? "selected" : "";
        return `<option value="${escapeAttr(r.path)}" ${sel}>${escapeHtml(r.name)} (${escapeHtml(r.language)} · ⎇ ${escapeHtml(r.branch)}) — ${escapeHtml(r.path)}</option>`;
      })
      .join("");
  }

  if (multiBox) {
    multiBox.innerHTML = repos
      .map((r) => {
        const checked = activeAffected.has(r.path) ? "checked" : "";
        return `
          <label class="multi-repo-chk-item">
            <input type="checkbox" name="affected_repo_chk" value="${escapeAttr(r.path)}" ${checked} onchange="fillDefaultSubTasksInModal()" />
            <span>📦 <b>${escapeHtml(r.name)}</b> <small>(${escapeHtml(r.language)})</small></span>
          </label>
        `;
      })
      .join("");
  }

  renderTrackerConnectionsInModal();
  fillDefaultSubTasksInModal();
  modal.classList.remove("hidden");
}

function switchTaskIntakeMode(mode) {
  currentTaskIntakeMode = mode === "tracker" ? "tracker" : "manual";
  const btnTracker = document.getElementById("btn-intake-mode-tracker");
  const btnManual = document.getElementById("btn-intake-mode-manual");
  const panelTracker = document.getElementById("intake-tracker-panel");

  if (btnTracker) btnTracker.classList.toggle("active", currentTaskIntakeMode === "tracker");
  if (btnManual) btnManual.classList.toggle("active", currentTaskIntakeMode === "manual");
  if (panelTracker) panelTracker.classList.toggle("hidden", currentTaskIntakeMode !== "tracker");

  if (currentTaskIntakeMode === "tracker") {
    renderTrackerConnectionsInModal();
    loadTrackerIssuesForSelectedConn();
  }
}

function renderTrackerConnectionsInModal() {
  const sel = document.getElementById("inp-tracker-conn-select");
  if (!sel) return;
  const conns = dashboardState?.tracker_connections || [];
  if (conns.length === 0) {
    sel.innerHTML = `<option value="">-- Chưa có tài khoản kết nối — Bấm '+ Kết nối tài khoản mới' --</option>`;
    return;
  }
  const prev = sel.value;
  sel.innerHTML = conns
    .map((c) => {
      const selected = c.id === prev ? "selected" : "";
      return `<option value="${escapeAttr(c.id)}" ${selected}>[${escapeHtml(String(c.provider || "").toUpperCase())}] ${escapeHtml(c.name)} (${escapeHtml(c.project_key || "ALL")} · ${escapeHtml(c.masked_token || "***")})</option>`;
    })
    .join("");
}

function toggleNewTrackerAuthForm() {
  const box = document.getElementById("new-tracker-auth-box");
  if (box) box.classList.toggle("hidden");
}

async function testTrackerConnectionInModal() {
  const provider = document.getElementById("nta-provider")?.value || "jira";
  const name = document.getElementById("nta-name")?.value.trim() || "";
  const base_url = document.getElementById("nta-base-url")?.value.trim() || "";
  const project_key = document.getElementById("nta-project-key")?.value.trim() || "";
  const email = document.getElementById("nta-email")?.value.trim() || "";
  const token = document.getElementById("nta-token")?.value.trim() || "";
  const stEl = document.getElementById("nta-test-status");

  try {
    const res = await fetch("/api/trackers/test", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ provider, name, base_url, project_key, email, token }),
    });
    const data = await res.json();
    if (stEl) {
      stEl.textContent = data.message || (data.ok ? "✓ Kết nối hợp lệ!" : "⚠ Vui lòng kiểm tra thông tin.");
      stEl.style.color = data.ok ? "#4ade80" : "#fbbf24";
    }
  } catch (e) {
    if (stEl) {
      stEl.textContent = "Lỗi kiểm tra kết nối: " + e.message;
      stEl.style.color = "#f87171";
    }
  }
}

async function saveNewTrackerConnectionInModal() {
  const provider = document.getElementById("nta-provider")?.value || "jira";
  const name = document.getElementById("nta-name")?.value.trim() || "";
  const base_url = document.getElementById("nta-base-url")?.value.trim() || "";
  const project_key = document.getElementById("nta-project-key")?.value.trim() || "";
  const email = document.getElementById("nta-email")?.value.trim() || "";
  const token = document.getElementById("nta-token")?.value.trim() || "";
  const stEl = document.getElementById("nta-test-status");

  if (!name && !base_url) {
    alert("Vui lòng nhập Tên gợi nhớ tài khoản hoặc Base URL.");
    return;
  }

  try {
    const res = await fetch("/api/trackers", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ provider, name, base_url, project_key, email, token }),
    });
    if (!res.ok) {
      alert("Lỗi lưu tài khoản Tracker: " + (await res.text()));
      return;
    }
    const data = await res.json();
    if (dashboardState) {
      dashboardState.tracker_connections = data.connections || [];
    }
    renderTrackerConnectionsInModal();
    const sel = document.getElementById("inp-tracker-conn-select");
    if (sel && data.created?.id) {
      sel.value = data.created.id;
    }
    const tokenInp = document.getElementById("nta-token");
    if (tokenInp) tokenInp.value = "";
    if (stEl) {
      stEl.textContent = `✓ Đã lưu & xác thực: ${data.created?.name || name} (Token: ${data.created?.masked_token || "***"})`;
      stEl.style.color = "#4ade80";
    }
    document.getElementById("new-tracker-auth-box")?.classList.add("hidden");
    await loadTrackerIssuesForSelectedConn();
  } catch (e) {
    console.error("saveNewTrackerConnectionInModal error:", e);
  }
}

async function loadTrackerIssuesForSelectedConn() {
  const connId = document.getElementById("inp-tracker-conn-select")?.value || "";
  const query = document.getElementById("inp-tracker-issue-search")?.value.trim() || "";
  const listEl = document.getElementById("tracker-issues-list");
  if (!listEl) return;

  try {
    const url = `/api/trackers/issues?connection_id=${encodeURIComponent(connId)}&query=${encodeURIComponent(query)}`;
    const res = await fetch(url);
    if (!res.ok) return;
    const data = await res.json();
    cachedTrackerIssues = data.issues || [];
    if (cachedTrackerIssues.length === 0) {
      listEl.innerHTML = `<div class="tib-empty">Không tìm thấy ticket nào khớp bộ lọc.</div>`;
      return;
    }
    listEl.innerHTML = cachedTrackerIssues
      .map((iss) => {
        const iKey = iss.issue_key || iss.key || "";
        const isSelected = selectedTrackerIssueKey === iKey;
        const reposBadges = (iss.affected_repos || [])
          .map((rp) => {
            const shortRp = String(rp).split("/").filter(Boolean).pop() || rp;
            return `<span class="tib-repo-chip">📦 ${escapeHtml(shortRp)}</span>`;
          })
          .join("");
        return `
          <div class="tib-issue-card ${isSelected ? "selected" : ""}" onclick="selectTrackerIssueForTask('${escapeAttr(iKey)}')">
            <div class="tib-issue-top">
              <span class="tib-issue-key">${escapeHtml(iKey)}</span>
              <span class="tib-issue-prov">${escapeHtml(String(iss.provider || "").toUpperCase())}</span>
              <span class="tib-issue-prio">${escapeHtml(iss.priority || "High")}</span>
              <span class="tib-issue-epic">Epic: ${escapeHtml(iss.epic || "DATA")}</span>
              <span class="tib-issue-status">${escapeHtml(iss.status || "READY")}</span>
              <div class="tib-issue-repos">${reposBadges}</div>
            </div>
            <div class="tib-issue-title">${escapeHtml(iss.title)}</div>
            <div class="tib-issue-desc">${escapeHtml(iss.description)}</div>
          </div>
        `;
      })
      .join("");
  } catch (e) {
    console.error("loadTrackerIssuesForSelectedConn error:", e);
  }
}

function selectTrackerIssueForTask(issueKey) {
  const iss = cachedTrackerIssues.find((x) => (x.issue_key || x.key) === issueKey);
  if (!iss) return;
  const iKey = iss.issue_key || iss.key || issueKey;
  selectedTrackerIssueKey = iKey;

  const idInp = document.getElementById("inp-task-id");
  const titleInp = document.getElementById("inp-task-title");
  const descInp = document.getElementById("inp-task-desc");
  const brdInp = document.getElementById("inp-task-brd");
  const compSel = document.getElementById("inp-task-complexity");
  const epicSel = document.getElementById("inp-task-epic");

  if (idInp) idInp.value = iKey;
  if (titleInp) titleInp.value = iss.title || "";
  if (descInp) descInp.value = iss.description || "";
  if (brdInp) brdInp.value = iss.brd_markdown || "";
  if (compSel && iss.complexity) compSel.value = iss.complexity;
  if (epicSel && iss.epic) epicSel.value = iss.epic;

  // Sync Affected Repos checkboxes if ticket specifies affected_repos
  if (Array.isArray(iss.affected_repos) && iss.affected_repos.length > 0) {
    const repoSel = document.getElementById("inp-task-repo");
    if (repoSel) repoSel.value = iss.affected_repos[0];
    const targetSet = new Set(iss.affected_repos);
    document.querySelectorAll('input[name="affected_repo_chk"]').forEach((chk) => {
      chk.checked = targetSet.has(chk.value);
    });
  }

  fillDefaultSubTasksInModal();
  loadTrackerIssuesForSelectedConn();
}

function handleBRDFileUpload(ev) {
  const file = ev?.target?.files?.[0];
  if (!file) return;
  uploadedBRDFileName = file.name;
  const reader = new FileReader();
  reader.onload = (e) => {
    const content = String(e.target?.result || "");
    const brdInp = document.getElementById("inp-task-brd");
    if (brdInp) {
      brdInp.value = content;
    }
    const descInp = document.getElementById("inp-task-desc");
    if (descInp && !descInp.value.trim()) {
      descInp.value = `Tài liệu BRD đính kèm: ${file.name} (${content.length} bytes).`;
    }
  };
  reader.readAsText(file);
}

function loadSampleBRDIntoModal() {
  const titleInp = document.getElementById("inp-task-title");
  const descInp = document.getElementById("inp-task-desc");
  const brdInp = document.getElementById("inp-task-brd");

  if (titleInp && !titleInp.value.trim()) {
    titleInp.value = "tích hợp Master MongoDB Destination (Dynamic Schema & Ordered BulkWrite OCC)";
  }
  if (descInp && !descInp.value.trim()) {
    descInp.value =
      "Triển khai đồng bộ dữ liệu CDC sang đích Master MongoDB xuyên 3 repository (cdc-cms-service, cdc-cms-web, centralized-data-service) với Dynamic Document Mapping, Ordered BulkWrite OCC chống ghi đè bản ghi mới hơn và Watermark Continuity sau sự cố Outage.";
  }
  if (brdInp) {
    brdInp.value = `# BRD: Master MongoDB Destination (Multi-Repo Epic)
## 1. Mục tiêu Nghiệp vụ (Business Objective)
Hỗ trợ cấu hình và đồng bộ dữ liệu CDC thời gian thực từ Kafka sang đích **Master MongoDB** với khả năng:
- Định nghĩa **Dynamic Schema Mapping** (Root Fields, Nested Sub-Documents, Array Unwind/Upsert) trên giao diện quản trị \`cdc-cms-web\` và lưu trữ/validate qua \`cdc-cms-service\`.
- Thực thi **Ordered BulkWrite kèm Optimistic Concurrency Control (OCC)** theo \`_cdc_version / lsn\` tại \`centralized-data-service\` để đảm bảo tuyệt đối không ghi đè dữ liệu cũ lên dữ liệu mới khi chạy đa worker (\`WorkerCount > 1\`).

## 2. Phạm vi 3 Repository bị ảnh hưởng
1. \`cdc-cms-service\`: API tạo/cập nhật Destination MongoDB, kiểm tra kết nối \`Ping + WriteConcern majority\`, lưu cấu hình \`DynamicCollectionMapping\`.
2. \`cdc-cms-web\`: Form cấu hình Connection URI (masked secret theo Rule #20), trình soạn thảo Dynamic Field Mapping & chế độ Upsert/Ordered BulkWrite.
3. \`centralized-data-service\`: MongoDB Sink Worker Pool, gom lô \`BatchDebouncer\`, \`Ordered BulkWrite\` với filter OCC \`$lt\`, và quản lý \`Watermark/Cursor\` an toàn khi Rebalance/Outage.

## 3. Tiêu chí Nghiệm thu (Acceptance Criteria — Gates G1..G8)
- **G3 (Real Runtime Trace):** Toàn bộ unit, integration và runtime healthcheck có biên lai \`EV-...\` thật.
- **G4 (Concurrency & Edge Cases):** Đa worker chạy song song không xảy ra race condition hoặc tụt lùi Watermark.`;
  }
  autoPreviewSubTasksOnTitle();
}

function syncPrimaryRepoToMultiCheckboxes() {
  const primary = document.getElementById("inp-task-repo")?.value || "";
  if (!primary) return;
  document.querySelectorAll('input[name="affected_repo_chk"]').forEach((chk) => {
    if (chk.value === primary) chk.checked = true;
  });
  fillDefaultSubTasksInModal();
}

function getSelectedModalAffectedRepos() {
  const checked = Array.from(document.querySelectorAll('input[name="affected_repo_chk"]:checked')).map(
    (el) => el.value
  );
  const primary = document.getElementById("inp-task-repo")?.value || "";
  if (primary && !checked.includes(primary)) {
    checked.unshift(primary);
  }
  return checked;
}

function closeDispatchModal() {
  document.getElementById("dispatch-modal")?.classList.add("hidden");
}

function autoPreviewSubTasksOnTitle() {
  const title = document.getElementById("inp-task-title")?.value.trim() || "";
  if (!title) return;
  const lower = title.toLowerCase();
  if (lower.includes("mongo") || lower.includes("master")) {
    document.querySelectorAll('input[name="affected_repo_chk"]').forEach((chk) => {
      if (
        chk.value.includes("cdc-cms-service") ||
        chk.value.includes("cdc-cms-web") ||
        chk.value.includes("centralized-data-service")
      ) {
        chk.checked = true;
      }
    });
  }
  fillDefaultSubTasksInModal();
}

function fillDefaultSubTasksInModal() {
  const title = document.getElementById("inp-task-title")?.value.trim() || "";
  const ta = document.getElementById("inp-task-subtasks");
  if (!ta) return;

  const lower = title.toLowerCase();
  if (lower.includes("mongo") || lower.includes("destination")) {
    // Leave empty or show canonical 08_tasks markdown so backend syncs exact 10 Sub-Tasks (TASK-M01..M10) from /agent workspace
    ta.value = "";
    return;
  }
  const selectedRepos = getSelectedModalAffectedRepos();
  if (selectedRepos.length > 0 && title) {
    ta.value = "";
  }
}

async function submitNewTask(ev) {
  ev.preventDefault();
  const external_id = document.getElementById("inp-task-id").value.trim();
  const title = document.getElementById("inp-task-title").value.trim();
  const description = document.getElementById("inp-task-desc")?.value.trim() || title;
  const brd_content = document.getElementById("inp-task-brd")?.value.trim() || "";
  const tracker_connection_id =
    currentTaskIntakeMode === "tracker"
      ? document.getElementById("inp-tracker-conn-select")?.value || ""
      : "";
  const tracker_issue_key = currentTaskIntakeMode === "tracker" ? selectedTrackerIssueKey || external_id : "";
  const target_repo = document.getElementById("inp-task-repo")?.value || "";
  const target_repos = getSelectedModalAffectedRepos();
  const complexity = document.getElementById("inp-task-complexity").value;
  const milestone = document.getElementById("inp-task-milestone").value;
  const epic = document.getElementById("inp-task-epic").value;
  const assignee = document.getElementById("inp-task-assignee")?.value || "Coder";
  const runner = document.getElementById("inp-task-runner")?.value || getSelectedRunner();
  const model = document.getElementById("inp-task-model")?.value || "gemini-4-argon (antigravity)";
  const auto_autopilot = Boolean(document.getElementById("inp-task-auto-autopilot")?.checked);
  const depsRaw = document.getElementById("inp-task-deps").value;
  const critical = document.getElementById("inp-task-critical").checked;
  const depends_on = depsRaw
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);

  const subTaskLines = (document.getElementById("inp-task-subtasks")?.value || "")
    .split("\n")
    .map((s) => s.trim())
    .filter(Boolean);

  try {
    const res = await fetch("/api/tasks", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        external_id,
        title,
        description,
        intake_mode: currentTaskIntakeMode,
        tracker_connection_id,
        tracker_issue_key,
        brd_content,
        brd_file_name: uploadedBRDFileName,
        target_repo,
        target_repos,
        model,
        sub_task_titles: subTaskLines,
        complexity,
        milestone,
        epic,
        assignee,
        runner,
        auto_autopilot,
        depends_on,
        critical,
      }),
    });
    if (!res.ok) {
      alert("Lỗi tạo task: " + (await res.text()));
      return;
    }
    dashboardState = await res.json();
    closeDispatchModal();
    document.getElementById("inp-task-id").value = "";
    document.getElementById("inp-task-title").value = "";
    const descEl = document.getElementById("inp-task-desc");
    if (descEl) descEl.value = "";
    const brdEl = document.getElementById("inp-task-brd");
    if (brdEl) brdEl.value = "";
    uploadedBRDFileName = "";
    selectedChainKey = dashboardState.active_task_key || external_id;
    selectedSubTaskRepoFilter = "";
    selectedReportTab = "S3";
    userLockedReportTab = true;
    switchScreen("autopilot");
    renderAll();
  } catch (e) {
    console.error("submitNewTask error:", e);
  }
}

function openCLISettingsModal() {
  const modal = document.getElementById("cli-settings-modal");
  if (!modal) return;
  const cfg = dashboardState?.cli_settings || {};
  const setVal = (id, v) => {
    const el = document.getElementById(id);
    if (!el || v === undefined || v === null || v === "") return;
    if (el.tagName === "SELECT" && !Array.from(el.options).some((o) => o.value === v)) {
      const opt = document.createElement("option");
      opt.value = v;
      opt.textContent = v;
      el.insertBefore(opt, el.firstChild);
    }
    el.value = String(v);
  };
  setVal("cfg-default-runner", cfg.default_runner || dashboardState?.active_runner || "antigravity");
  setVal("cfg-planner-model", cfg.planner_model || dashboardState?.planner_model || "gemini-4-argon (antigravity)");
  setVal("cfg-coder-high-model", cfg.coder_high_model || dashboardState?.coder_high_model || "gemini-4-argon (antigravity)");
  setVal("cfg-coder-std-model", cfg.coder_std_model || dashboardState?.coder_std_model || "gemini-2.5-flash (antigravity)");
  setVal("cfg-qa-model", cfg.qa_auditor_model || cfg.qa_model || "claude-opus-4-6 (antigravity)");
  setVal("cfg-laya-gate-model", cfg.laya_gate_model || "convaiinnovations/laya (local :8200)");
  setVal("cfg-security-model", cfg.security_model || "agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)");
  setVal("cfg-max-self-heal-retries", cfg.max_self_heal_retries || 3);

  const box = document.getElementById("cfg-detected-runners-table");
  if (box) {
    const runners = dashboardState?.available_runners || [];
    box.innerHTML = runners
      .map(
        (r) => `
        <div class="runner-detect-row">
          <span><b>${escapeHtml(r.name)}</b> (${escapeHtml(r.id)})</span>
          <span style="color:${r.installed ? "#4ade80" : "#94a3b8"}">
            ${r.installed ? `✓ Installed: ${escapeHtml(r.binary_path || r.id)} (${escapeHtml(r.version || "ready")})` : "◌ Not in PATH"}
          </span>
        </div>
      `
      )
      .join("");
  }

  modal.classList.remove("hidden");
}

function closeCLISettingsModal() {
  document.getElementById("cli-settings-modal")?.classList.add("hidden");
}

async function submitCLISettings(ev) {
  ev.preventDefault();
  const maxRetriesRaw = parseInt(document.getElementById("cfg-max-self-heal-retries")?.value || "3", 10);
  const payload = {
    default_runner: document.getElementById("cfg-default-runner")?.value || "antigravity",
    planner_model: document.getElementById("cfg-planner-model")?.value.trim() || "gemini-4-argon (antigravity)",
    coder_high_model: document.getElementById("cfg-coder-high-model")?.value.trim() || "gemini-4-argon (antigravity)",
    coder_std_model: document.getElementById("cfg-coder-std-model")?.value.trim() || "gemini-2.5-flash (antigravity)",
    qa_auditor_model: document.getElementById("cfg-qa-model")?.value.trim() || "claude-opus-4-6 (antigravity)",
    laya_gate_model: document.getElementById("cfg-laya-gate-model")?.value.trim() || "convaiinnovations/laya (local :8200)",
    security_model: document.getElementById("cfg-security-model")?.value.trim() || "agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)",
    max_self_heal_retries: Number.isFinite(maxRetriesRaw) && maxRetriesRaw > 0 ? maxRetriesRaw : 3,
  };
  try {
    const res = await fetch("/api/settings/cli", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      alert("Lỗi lưu cấu hình CLI & Model: " + (await res.text()));
      return;
    }
    dashboardState = await res.json();
    closeCLISettingsModal();
    renderAll();
  } catch (e) {
    console.error("submitCLISettings error:", e);
  }
}

/* ============================================================================
   5. MODAL: SETUP TELEGRAM (S3 PLAN NOTIFICATION & REMOTE APPROVE AUTO-RUN)
   ============================================================================ */
function collectTelegramFormPayload(activeKey) {
  const notifyVal = Boolean(document.getElementById("tg-notify-s3")?.checked ?? true);
  const payload = {
    enabled: Boolean(document.getElementById("tg-enabled")?.checked ?? true),
    bot_token: document.getElementById("tg-bot-token")?.value.trim() || "",
    chat_id: document.getElementById("tg-chat-id")?.value.trim() || "",
    bot_username: document.getElementById("tg-bot-username")?.value.trim() || "@tk001sea_bot",
    notify_on_s3: notifyVal,
    notify_on_s3_plan: notifyVal,
    auto_run_on_approve: Boolean(document.getElementById("tg-auto-run")?.checked ?? true),
    polling_enabled: Boolean(document.getElementById("tg-polling")?.checked ?? true),
  };
  if (activeKey) {
    payload.task_key = activeKey;
  }
  return payload;
}

function openTelegramSettingsModal() {
  const modal = document.getElementById("telegram-settings-modal");
  if (!modal) return;
  const tg = dashboardState?.telegram_settings || {};

  const setChk = (id, val) => {
    const el = document.getElementById(id);
    if (el) el.checked = Boolean(val);
  };
  setChk("tg-enabled", tg.enabled ?? true);
  setChk("tg-notify-s3", tg.notify_on_s3 ?? tg.notify_on_s3_plan ?? true);
  setChk("tg-auto-run", tg.auto_run_on_approve ?? true);
  setChk("tg-polling", tg.polling_enabled ?? true);

  const chatInp = document.getElementById("tg-chat-id");
  if (chatInp) chatInp.value = tg.chat_id || "";
  const userInp = document.getElementById("tg-bot-username");
  if (userInp) userInp.value = tg.bot_username || "@tk001sea_bot";
  const tokenInp = document.getElementById("tg-bot-token");
  if (tokenInp) tokenInp.value = "";

  const hintEl = document.getElementById("tg-masked-token-hint");
  if (hintEl) {
    hintEl.textContent = `Token đang lưu (Rule #20 Masked): ${tg.masked_bot_token || "Chưa cấu hình"}`;
  }
  const stEl = document.getElementById("tg-test-status");
  if (stEl) {
    stEl.textContent = tg.last_status ? `Trạng thái gần nhất: ${tg.last_status}` : "";
  }
  renderTelegramEventsInModal(tg.recent_events || []);
  modal.classList.remove("hidden");
}

function closeTelegramSettingsModal() {
  document.getElementById("telegram-settings-modal")?.classList.add("hidden");
}

function renderTelegramEventsInModal(events) {
  const box = document.getElementById("tg-recent-events-box");
  if (!box) return;
  if (!Array.isArray(events) || events.length === 0) {
    box.innerHTML = `<div class="runner-detect-row"><span>Chưa có sự kiện Telegram nào.</span></div>`;
    return;
  }
  box.innerHTML = events
    .map(
      (ev) => `
      <div class="runner-detect-row">
        <span><b>[${escapeHtml(ev.direction || "OUT")}]</b> <code>${escapeHtml(ev.task_key || "GLOBAL")}</code> — ${escapeHtml(ev.summary || "")}</span>
        <span style="color:#67e8f9">${escapeHtml(ev.status || "")} · ${escapeHtml(ev.timestamp || "")}</span>
      </div>
    `
    )
    .join("");
}

async function submitTelegramSettings(ev) {
  if (ev && ev.preventDefault) ev.preventDefault();
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-11";
  const payload = collectTelegramFormPayload(activeKey);
  const stEl = document.getElementById("tg-test-status");
  if (stEl) {
    stEl.textContent = `⏳ Đang lưu cấu hình & bắn thông báo S3 Plan (${activeKey}) sang Telegram...`;
  }
  try {
    const res = await fetch("/api/settings/telegram", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      alert("Lỗi lưu cấu hình Telegram: " + (await res.text()));
      return;
    }
    const tokenInp = document.getElementById("tg-bot-token");
    if (tokenInp) tokenInp.value = "";
    dashboardState = await res.json();
    const tg = dashboardState?.telegram_settings || {};
    const hintEl = document.getElementById("tg-masked-token-hint");
    if (hintEl) {
      hintEl.textContent = `Token đang lưu (Rule #20 Masked): ${tg.masked_bot_token || "Chưa cấu hình"}`;
    }
    if (stEl) {
      stEl.textContent = `✓ Đã lưu cấu hình & gửi thông báo Telegram: ${tg.last_status || "SENT_TELEGRAM_API_OK"}`;
    }
    renderTelegramEventsInModal(tg.recent_events || []);
    renderAll();
  } catch (e) {
    console.error("submitTelegramSettings error:", e);
    if (stEl) stEl.textContent = "Lỗi lưu cấu hình Telegram: " + e.message;
  }
}

async function testTelegramS3Notification() {
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-11";
  const payload = collectTelegramFormPayload(activeKey);
  const stEl = document.getElementById("tg-test-status");
  if (stEl) stEl.textContent = `⏳ Đang tự lưu cấu hình & gửi thông báo S3 Plan của ${activeKey} qua Telegram Bridge...`;
  try {
    const res = await fetch("/api/settings/telegram/test", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      if (stEl) stEl.textContent = "Lỗi gửi thử Telegram: " + (await res.text());
      return;
    }
    const tokenInp = document.getElementById("tg-bot-token");
    if (tokenInp) tokenInp.value = "";
    const data = await res.json();
    const tg = data.telegram_settings || data.settings || {};
    const statusStr = data.delivery_status || tg.last_status || "OK";
    const hintEl = document.getElementById("tg-masked-token-hint");
    if (hintEl && tg.masked_bot_token) {
      hintEl.textContent = `Token đang lưu (Rule #20 Masked): ${tg.masked_bot_token}`;
    }
    if (stEl) {
      stEl.textContent = `✓ Đã gửi thông báo S3 cho ${data.task_key || activeKey}: ${statusStr}`;
    }
    if (dashboardState) {
      dashboardState.telegram_settings = tg;
    }
    renderTelegramEventsInModal(tg.recent_events || []);
    await fetchDashboard();
  } catch (e) {
    if (stEl) stEl.textContent = "Lỗi gửi thử Telegram: " + e.message;
  }
}

async function sendTelegramTaskMenuNow(btnEl) {
  const btn = btnEl || document.getElementById("btn-push-tg-task-menu");
  const origHtml = btn ? btn.innerHTML : "✈️ /task";
  if (btn) {
    btn.disabled = true;
    btn.innerHTML = "⏳ Đang gửi...";
  }
  const stEl = document.getElementById("tg-test-status");
  if (stEl) stEl.textContent = "📋 Đang đẩy danh sách Hàng Đợi Task (/task) kèm nút Click-to-Run & Duyệt Từng Sub-Task lên Telegram...";
  try {
    const res = await fetch("/api/telegram/webhook", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        text: "/task",
        step_delay_ms: 0,
      }),
    });
    const data = await res.json();
    if (stEl) {
      stEl.textContent = `✓ Đã đẩy Menu /task lên Telegram (Active: ${data.task_key || "OK"}). Anh có thể bấm nút Task trên Telegram để chạy ngay!`;
    }
    if (btn) {
      btn.classList.add("sent-ok");
      btn.innerHTML = "✓ Đã đẩy Tele";
      setTimeout(() => {
        btn.disabled = false;
        btn.classList.remove("sent-ok");
        btn.innerHTML = origHtml;
      }, 2200);
    }
    await fetchDashboard();
    if (dashboardState?.telegram_settings) {
      renderTelegramEventsInModal(dashboardState.telegram_settings.recent_events || []);
    }
  } catch (e) {
    if (stEl) stEl.textContent = "Lỗi gửi Menu /task lên Telegram: " + e.message;
    if (btn) {
      btn.disabled = false;
      btn.innerHTML = "❌ Lỗi gửi";
      setTimeout(() => {
        btn.innerHTML = origHtml;
      }, 2200);
    }
  }
}

async function simulateTelegramRemoteApprove() {
  const activeKey = dashboardState?.active_task_key || selectedChainKey || "TASK-11";
  const stEl = document.getElementById("tg-test-status");
  if (stEl) stEl.textContent = `⚡ Đang giả lập tin nhắn "APPROVE ${activeKey}" từ Telegram để tự chạy S4–S5...`;
  try {
    const res = await fetch("/api/telegram/webhook", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        task_key: activeKey,
        text: `APPROVE ${activeKey}`,
        step_delay_ms: 0,
      }),
    });
    const data = await res.json();
    if (stEl) {
      stEl.textContent = `✓ Telegram Remote APPROVE hoàn tất: ${data.task_key} -> Stage ${data.new_stage}`;
    }
    await fetchDashboard();
    if (dashboardState?.telegram_settings) {
      renderTelegramEventsInModal(dashboardState.telegram_settings.recent_events || []);
    }
  } catch (e) {
    if (stEl) stEl.textContent = "Lỗi giả lập APPROVE Telegram: " + e.message;
  }
}

function setText(id, val) {
  const el = document.getElementById(id);
  if (el) el.textContent = String(val);
}

function escapeHtml(str) {
  return String(str ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function escapeAttr(str) {
  return escapeHtml(str).replace(/'/g, "&#39;");
}

window.addEventListener("DOMContentLoaded", () => {
  const scrollArea = document.getElementById("wave-scroll-area");
  if (scrollArea) {
    scrollArea.addEventListener("scroll", () => {
      requestAnimationFrame(() => drawWaveBezierEdges());
    });
  }
  fetchDashboard();
  connectSSE();
});

