import { attachDragDrop } from "./utils/dragDrop.js";
import { getLanguage, setLanguage, t } from "./utils/i18n.js";
import { renderProgress } from "./components/progressBar.js";

const backend = window.go?.main?.App;
const runtime = window.runtime;
let selectedPaths = [];
let configPayload = null;
let textProgress = null;
let activeMode = null;
const batchProgress = new Map();

const $ = (selector) => document.querySelector(selector);
const statusText = $("#statusText");
const fileList = $("#fileList");
const textProgressNode = $("#textProgress");
const dropzone = $("#dropzone");
const pendingList = $("#pendingList");
const dropzoneHint = $("#dropzoneHint");
const languageSelect = $("#languageSelect");
const retryFailedButton = $("#retryFailed");
const clearTasksButton = $("#clearTasks");
const pauseBatchButton = $("#pauseBatch");
const resumeBatchButton = $("#resumeBatch");
const cancelBatchButton = $("#cancelBatch");
let currentStatusKey = "ready";
let appRunning = false;
let appPaused = false;

document.querySelectorAll(".tab").forEach((button) => {
  button.addEventListener("click", () => {
    document.querySelectorAll(".tab").forEach((tab) => tab.classList.toggle("active", tab === button));
    document.querySelectorAll(".view").forEach((view) => view.classList.toggle("active", view.id === `view-${button.dataset.view}`));
  });
});

languageSelect.addEventListener("change", () => {
  setLanguage(languageSelect.value);
  applyTranslations();
});

attachDragDrop(dropzone, (paths) => {
  selectedPaths = mergePaths(selectedPaths, paths);
  renderPendingFiles(selectedPaths);
});

$("#selectFiles").addEventListener("click", async () => {
  try {
    selectedPaths = mergePaths(selectedPaths, await backend?.SelectTextFiles() || []);
    renderPendingFiles(selectedPaths);
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

$("#selectFilesAlt").addEventListener("click", async () => {
  try {
    selectedPaths = mergePaths(selectedPaths, await backend?.SelectTextFiles() || []);
    renderPendingFiles(selectedPaths);
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

$("#selectFolder").addEventListener("click", async () => {
  try {
    const dir = await backend?.SelectInputDirectory();
    selectedPaths = dir ? mergePaths(selectedPaths, [dir]) : selectedPaths;
    renderPendingFiles(selectedPaths);
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

dropzone.addEventListener("dblclick", async () => {
  if (selectedPaths.length > 0) return;
  try {
    selectedPaths = await backend?.SelectTextFiles() || [];
    renderPendingFiles(selectedPaths);
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

pendingList.addEventListener("click", (event) => {
  const button = event.target.closest("[data-remove-pending]");
  if (!button) return;
  selectedPaths.splice(Number(button.dataset.removePending), 1);
  renderPendingFiles(selectedPaths);
});

$("#startBatch").addEventListener("click", async () => {
  try {
    activeMode = "batch";
    batchProgress.clear();
    fileList.innerHTML = "";
    updateBatchControls();
    selectedPaths.forEach((path, index) => updateProgress({
      fileIndex: index,
      fileName: path.split(/[\\/]/).pop(),
      percent: 0,
      status: "queued",
    }));
    await backend?.StartBatch(selectedPaths);
    selectedPaths = [];
    renderPendingFiles(selectedPaths);
    setStatusKey("running");
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

$("#clearPending").addEventListener("click", () => {
  selectedPaths = [];
  renderPendingFiles(selectedPaths);
});

clearTasksButton.addEventListener("click", async () => {
  try {
    await backend?.ClearFinishedTasks();
  } catch (error) {
    setStatus(error.message || String(error));
    return;
  }
  Array.from(batchProgress.entries()).forEach(([fileIndex, progress]) => {
    if (isTerminal(progress.status)) {
      batchProgress.delete(fileIndex);
      document.querySelector(`[data-file-index="${fileIndex}"]`)?.remove();
    }
  });
  updateBatchControls();
});

pauseBatchButton.addEventListener("click", () => backend?.PauseBatch());
resumeBatchButton.addEventListener("click", () => backend?.ResumeBatch());
cancelBatchButton.addEventListener("click", () => backend?.CancelBatch());
$("#retryFailed").addEventListener("click", async () => {
  try {
    activeMode = "batch";
    await backend?.RetryFailed();
    updateBatchControls();
    setStatusKey("running");
  } catch (error) {
    setStatus(error.message || String(error));
  }
});
$("#pauseText").addEventListener("click", () => backend?.PauseBatch());
$("#resumeText").addEventListener("click", () => backend?.ResumeBatch());
$("#cancelText").addEventListener("click", () => backend?.CancelBatch());
fileList.addEventListener("click", async (event) => {
  const retryButton = event.target.closest("[data-retry-file]");
  if (retryButton) {
    try {
      await backend?.RetryFile(Number(retryButton.dataset.retryFile));
    } catch (error) {
      setStatus(error.message || String(error));
    }
    return;
  }
  const button = event.target.closest("[data-cancel-file]");
  if (!button) return;
  await backend?.CancelFile(Number(button.dataset.cancelFile));
});

$("#convertText").addEventListener("click", async () => {
  try {
    activeMode = "text";
    textProgress = {
      fileIndex: 0,
      fileName: $("#textOutputName").value.trim() || "text.mp3",
      percent: 0,
      status: "queued",
    };
    renderTextProgress(textProgress);
    await backend?.ConvertText($("#textInput").value, $("#textOutputName").value.trim() || "text.mp3");
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

$("#saveConfig").addEventListener("click", async () => {
  try {
    const payload = buildConfigPayload();
    configPayload = await backend?.SaveConfig(JSON.stringify(payload, null, 2));
    hydrateConfig(configPayload);
    setStatusKey("saved");
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

$("#selectOutputDir").addEventListener("click", async () => {
  const dir = await backend?.SelectOutputDir();
  if (dir) {
    $("#outputDir").value = dir;
  }
});

$("#formatVoiceJSON").addEventListener("click", () => {
  try {
    formatVoiceJSON();
    setStatusKey("jsonValid");
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

runtime?.EventsOn?.("batch:progress", (progress) => {
  if (activeMode === "text") {
    textProgress = progress;
    renderTextProgress(progress);
  } else {
    updateProgress(progress);
    updateBatchControls();
  }
});
runtime?.EventsOn?.("app:error", (message) => setStatus(message));
runtime?.EventsOn?.("app:state", (state) => {
  appRunning = !!state?.running;
  appPaused = !!state?.paused;
  if (state?.files && activeMode === "batch") {
    batchProgress.clear();
    fileList.innerHTML = "";
    state.files.forEach(updateProgress);
    updateBatchControls();
  } else if (state?.files && activeMode === "text" && state.files[0]) {
    textProgress = state.files[0];
    renderTextProgress(textProgress);
  }
  updateBatchControls();
  setStatusKey(state?.running ? (state.paused ? "paused" : "running") : "ready");
});

async function init() {
  applyTranslations();
  updateBatchControls();
  try {
    configPayload = await backend?.LoadConfig();
    hydrateConfig(configPayload);
  } catch (error) {
    setStatus(error.message || String(error));
  }
}

function hydrateConfig(payload) {
  if (!payload?.data) return;
  const cfg = payload.data;
  $("#apiBaseURL").value = cfg.API_BASE_URL || "";
  $("#apiToken").value = cfg.API_TOKEN || "";
  $("#model").value = cfg.MODEL || "";
  $("#voiceInput").value = cfg.VOICE_JSON || "";
  $("#splitThreshold").value = cfg.SPLIT_THRESHOLD || 300;
  $("#concurrency").value = cfg.CONCURRENCY || 2;
  $("#requestTimeout").value = cfg.REQUEST_TIMEOUT || 120;
  $("#outputBitrate").value = cfg.OUTPUT_BITRATE_KB || 128;
  $("#outputDir").value = cfg.OUTPUT_DIR || "";
}

function buildConfigPayload() {
  const cfg = configPayload?.data || {};
  const voiceJSON = formatVoiceJSON();
  return {
    ...cfg,
    API_BASE_URL: $("#apiBaseURL").value.trim(),
    API_TOKEN: $("#apiToken").value.trim(),
    MODEL: $("#model").value.trim(),
    VOICE_JSON: voiceJSON,
    SPLIT_THRESHOLD: Number($("#splitThreshold").value),
    CONCURRENCY: Number($("#concurrency").value),
    REQUEST_TIMEOUT: Number($("#requestTimeout").value),
    OUTPUT_BITRATE_KB: Number($("#outputBitrate").value),
    OUTPUT_DIR: $("#outputDir").value.trim(),
  };
}

function formatVoiceJSON() {
  const raw = $("#voiceInput").value.trim();
  if (!raw) {
    throw new Error("Request Options JSON is required");
  }
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    throw new Error(`Invalid Request Options JSON: ${error.message}`);
  }
  if (parsed === null || Array.isArray(parsed) || typeof parsed !== "object") {
    throw new Error("Request Options JSON must be an object");
  }
  const formatted = JSON.stringify(parsed, null, 2);
  $("#voiceInput").value = formatted;
  return formatted;
}

function renderPendingFiles(paths) {
  activeMode = "batch";
  dropzoneHint.hidden = paths.length > 0;
  pendingList.innerHTML = "";
  paths.forEach((path, index) => {
    const item = document.createElement("div");
    item.className = "pendingItem";
    item.title = path;
    item.innerHTML = `
      <span>${escapeHTML(path.split(/[\\/]/).pop())}</span>
      <button class="iconButton pendingRemove" data-remove-pending="${index}" title="${escapeHTML(t("remove"))}" aria-label="${escapeHTML(t("remove"))}">${iconSVG("x")}</button>
    `;
    pendingList.appendChild(item);
  });
  updateBatchControls();
}

function mergePaths(existing, next) {
  const seen = new Set(existing);
  const merged = existing.slice();
  next.forEach((path) => {
    if (!seen.has(path)) {
      seen.add(path);
      merged.push(path);
    }
  });
  return merged;
}

function updateProgress(progress) {
  batchProgress.set(progress.fileIndex, progress);
  let row = document.querySelector(`[data-file-index="${progress.fileIndex}"]`);
  if (!row) {
    row = document.createElement("article");
    row.className = "fileRow";
    row.dataset.fileIndex = progress.fileIndex;
    fileList.appendChild(row);
  }
  row.innerHTML = `
    <div>
      <strong>${escapeHTML(progress.fileName || t("textInput"))}</strong>
      <span>${escapeHTML(statusLabel(progress.status || "queued"))} ${progress.message ? `· ${escapeHTML(progress.message)}` : ""}</span>
    </div>
    ${renderProgress(progress.percent || 0)}
    ${renderRowActions(progress)}
  `;
  updateBatchControls();
}

function renderRowActions(progress) {
  const status = progress.status || "queued";
  if (status === "done") {
    return `
      <div class="rowActions">
        <span class="iconButton rowDone" title="${escapeHTML(t("done"))}" aria-label="${escapeHTML(t("done"))}" role="img">${iconSVG("check")}</span>
      </div>
    `;
  }
  const retryButton = status === "error" || status === "canceled"
    ? `<button class="iconButton rowRetry" data-retry-file="${progress.fileIndex}" title="${escapeHTML(t("retry"))}" aria-label="${escapeHTML(t("retry"))}">${iconSVG("retry")}</button>`
    : "";
  const cancelButton = isTerminal(status)
    ? ""
    : `<button class="iconButton rowCancel" data-cancel-file="${progress.fileIndex}" title="${escapeHTML(t("cancel"))}" aria-label="${escapeHTML(t("cancel"))}">${iconSVG("x")}</button>`;
  return `
    <div class="rowActions">
      ${retryButton}
      ${cancelButton}
    </div>
  `;
}

function iconSVG(name) {
  if (name === "check") {
    return `<svg viewBox="0 0 1024 1024" aria-hidden="true" focusable="false"><path d="M858.215186 365.296777c-19.841907-45.544289-46.667879-85.226057-80.476893-119.03507-33.813107-33.813107-73.629951-60.639079-119.455649-80.476893-45.828768-19.838837-94.727455-29.759791-146.704247-29.759791-51.972699 0-100.732216 9.920954-146.280598 29.759791-45.545312 19.837814-85.226057 46.663786-119.03507 80.476893-33.813107 33.809013-60.639079 73.490781-80.477916 119.03507-19.838837 45.549406-29.759791 94.308923-29.759791 146.281621 0 51.976792 9.920954 100.875478 29.759791 146.704247 19.838837 45.824675 46.664809 85.641519 80.477916 119.454626 33.809013 33.809013 73.489758 60.634985 119.03507 80.477916 45.548382 19.838837 94.307899 29.759791 146.280598 29.759791 51.976792 0 100.875478-9.920954 146.704247-29.759791 45.825698-19.84293 85.642542-46.668902 119.455649-80.477916 33.809013-33.813107 60.634985-73.629951 80.476893-119.454626 19.838837-45.828768 29.759791-94.727455 29.759791-146.704247C887.974977 459.605699 878.054023 410.846182 858.215186 365.296777zM687.075411 424.396803 477.3222 636.43301c-0.052189 0.052189-0.159636 0.079818-0.211824 0.156566-0.075725 0.051165-0.075725 0.159636-0.156566 0.211824-1.678222 1.622964-3.745301 2.617617-5.683443 3.717671-0.971118 0.551562-1.75497 1.390673-2.778276 1.782599-3.14155 1.254573-6.471388 1.910513-9.797134 1.910513-3.354398 0-6.731308-0.655939-9.901511-1.962701-1.047866-0.448208-1.886977-1.335415-2.882654-1.886977-1.938142-1.099031-3.957125-2.070148-5.631254-3.721765-0.052189-0.051165-0.080841-0.155543-0.132006-0.207731-0.052189-0.079818-0.156566-0.079818-0.207731-0.155543L336.777233 530.257829c-10.088776-10.373255-9.853415-26.953885 0.523933-37.043684 10.372232-10.057053 26.929326-9.872858 37.038568 0.523933l84.538395 86.869486 190.974519-193.041598c10.161431-10.296507 26.770713-10.376325 37.039591-0.211824C697.136557 397.547295 697.240935 414.127925 687.075411 424.396803z"></path></svg>`;
  }
  if (name === "retry") {
    return `<svg viewBox="0 0 1024 1024" aria-hidden="true" focusable="false"><path d="M512.47132701 9.25119901C235.48815945 9.25119901 10.82228901 233.91706944 10.82228901 510.90023698s224.66587044 501.64903801 501.649038 501.64903801 501.64903801-224.50876145 501.64903799-501.64903801S789.45449454 9.25119901 512.47132701 9.25119901z m287.35236156 638.64808627c-45.4045011 114.53246123-153.9668203 188.53080036-276.66894954 188.53080036-164.17890533 0-297.72155559-134.32819526-297.72155561-299.29264559s133.54265026-299.2926456 297.72155561-299.29264559c43.5191931 0 86.25284116 9.58364903 125.53009124 27.96540204l-11.62606602-32.67867204c-3.14217999-8.79810401-1.885308-19.63862503 3.613507-28.75094706 5.02748801-8.48388603 12.88293803-13.98270102 21.36682405-15.23957305 14.45402802-2.19952601 33.30710806 4.71327001 42.57653909 30.79336408l39.59146805 110.76184522c2.82796201 8.16966803 2.51374401 17.28199003-1.09976298 24.98033103s-10.21208501 13.82559203-18.06753503 16.81066304l-111.54739023 49.4893351-0.31421801 0.157109c-5.18459701 1.885308-13.66848304 1.885308-13.82559202 1.88530802-13.35426503 0-29.69360106-11.46895703-34.24976207-24.03767706-6.127251-16.81066303 4.556161-40.84834009 20.58127904-46.66137309l36.29217908-13.19715603c-29.85071007-14.61113702-62.52938211-22.15236905-95.9935992-22.15236903-121.28814825 0-220.10970944 99.29288819-220.10970944 221.36658141s98.66445221 221.36658144 220.10970944 221.36658145c90.65189318 0 170.93459234-54.67393211 204.55591841-139.35568328 2.82796201-7.38412301 9.74075802-13.51137403 19.16729804-17.59620804 10.84052103-4.71327001 22.30947806-5.34170603 30.79336406-2.04241699 16.65355405 6.75568702 26.08009405 29.22227406 19.32440704 46.1900461z"></path></svg>`;
  }
  return `<svg viewBox="0 0 1024 1024" aria-hidden="true" focusable="false"><path d="M512 938.666667c235.648 0 426.666667-191.018667 426.666667-426.666667S747.648 85.333333 512 85.333333 85.333333 276.352 85.333333 512s191.018667 426.666667 426.666667 426.666667zM318.72 363.946667l45.226667-45.226667L512 466.773333l148.053333-148.053333 45.226667 45.226667L557.226667 512l148.053333 148.053333-45.226667 45.226667L512 557.226667l-148.053333 148.053333-45.226667-45.226667L466.773333 512 318.72 363.946667z"></path></svg>`;
}

function renderTextProgress(progress) {
  if (!progress) {
    textProgressNode.innerHTML = "";
    return;
  }
  textProgressNode.innerHTML = `
    <article class="fileRow textProgressRow">
      <div>
        <strong>${escapeHTML(progress.output || progress.fileName || t("textConversion"))}</strong>
        <span>${escapeHTML(statusLabel(progress.status || "queued"))} ${progress.message ? `· ${escapeHTML(progress.message)}` : ""}</span>
      </div>
      ${renderProgress(progress.percent || 0)}
    </article>
  `;
}

function setStatus(message) {
  currentStatusKey = null;
  statusText.textContent = message || t("ready");
}

function setStatusKey(key) {
  currentStatusKey = key;
  statusText.textContent = t(key);
}

function applyTranslations() {
  document.documentElement.lang = getLanguage();
  document.querySelectorAll("[data-i18n]").forEach((node) => {
    node.textContent = t(node.dataset.i18n);
  });
  document.querySelectorAll("[data-i18n-placeholder]").forEach((node) => {
    node.placeholder = t(node.dataset.i18nPlaceholder);
  });
  document.querySelectorAll("[data-i18n-title]").forEach((node) => {
    const label = t(node.dataset.i18nTitle);
    node.title = label;
    node.setAttribute("aria-label", label);
  });
  languageSelect.value = getLanguage();
  if (currentStatusKey) {
    statusText.textContent = t(currentStatusKey);
  }
  if (batchProgress.size > 0) {
    fileList.innerHTML = "";
    Array.from(batchProgress.values())
      .sort((a, b) => a.fileIndex - b.fileIndex)
      .forEach(updateProgress);
    updateBatchControls();
  }
  if (textProgress) {
    renderTextProgress(textProgress);
  }
}

function statusLabel(status) {
  return t(status || "queued");
}

function isTerminal(status) {
  return status === "done" || status === "canceled" || status === "error";
}

function updateBatchControls() {
  const tasks = Array.from(batchProgress.values());
  const hasFailedTask = tasks.some((progress) => progress.status === "error");
  const hasTerminalTask = tasks.some((progress) => isTerminal(progress.status));
  const hasActiveTask = tasks.some((progress) => !isTerminal(progress.status));

  clearTasksButton.hidden = !hasTerminalTask;
  pauseBatchButton.hidden = !appRunning || appPaused || !hasActiveTask;
  resumeBatchButton.hidden = !appRunning || !appPaused || !hasActiveTask;
  cancelBatchButton.hidden = !appRunning || !hasActiveTask;
  retryFailedButton.hidden = !hasFailedTask;
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, (char) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#039;",
  }[char]));
}

init();
