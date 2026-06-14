import { attachDragDrop } from "./utils/dragDrop.js";
import { t } from "./utils/i18n.js";
import { renderProgress } from "./components/progressBar.js";

const backend = window.go?.main?.App;
const runtime = window.runtime;
let selectedPaths = [];
let configPayload = null;

const $ = (selector) => document.querySelector(selector);
const statusText = $("#statusText");
const fileList = $("#fileList");

document.querySelectorAll(".tab").forEach((button) => {
  button.addEventListener("click", () => {
    document.querySelectorAll(".tab").forEach((tab) => tab.classList.toggle("active", tab === button));
    document.querySelectorAll(".view").forEach((view) => view.classList.toggle("active", view.id === `view-${button.dataset.view}`));
  });
});

attachDragDrop($("#dropzone"), (paths) => {
  selectedPaths = paths;
  renderSelected(paths);
});

$("#selectFiles").addEventListener("click", async () => {
  try {
    selectedPaths = await backend?.SelectTextFiles() || [];
    renderSelected(selectedPaths);
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

$("#selectFolder").addEventListener("click", async () => {
  try {
    const dir = await backend?.SelectInputDirectory();
    selectedPaths = dir ? [dir] : selectedPaths;
    renderSelected(selectedPaths);
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

$("#startBatch").addEventListener("click", async () => {
  try {
    await backend?.StartBatch(selectedPaths);
    setStatus(t("running"));
  } catch (error) {
    setStatus(error.message || String(error));
  }
});

$("#pauseBatch").addEventListener("click", () => backend?.PauseBatch());
$("#resumeBatch").addEventListener("click", () => backend?.ResumeBatch());
$("#cancelBatch").addEventListener("click", () => backend?.CancelBatch());

$("#convertText").addEventListener("click", async () => {
  try {
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
    setStatus(t("saved"));
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

runtime?.EventsOn?.("batch:progress", (progress) => updateProgress(progress));
runtime?.EventsOn?.("app:error", (message) => setStatus(message));
runtime?.EventsOn?.("app:state", (state) => {
  if (state?.files) {
    fileList.innerHTML = "";
    state.files.forEach(updateProgress);
  }
  setStatus(state?.running ? (state.paused ? t("paused") : t("running")) : t("ready"));
});

async function init() {
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
  $("#splitThreshold").value = cfg.SPLIT_THRESHOLD || 1200;
  $("#concurrency").value = cfg.CONCURRENCY || 2;
  $("#outputDir").value = cfg.OUTPUT_DIR || "";
  $("#configJSON").value = payload.json || JSON.stringify(cfg, null, 2);
}

function buildConfigPayload() {
  const cfg = configPayload?.data || {};
  return {
    ...cfg,
    API_BASE_URL: $("#apiBaseURL").value.trim(),
    API_TOKEN: $("#apiToken").value.trim(),
    MODEL: $("#model").value.trim(),
    VOICE_JSON: $("#voiceInput").value.trim(),
    SPLIT_THRESHOLD: Number($("#splitThreshold").value),
    CONCURRENCY: Number($("#concurrency").value),
    OUTPUT_DIR: $("#outputDir").value.trim(),
  };
}

function renderSelected(paths) {
  fileList.innerHTML = "";
  paths.forEach((path, index) => updateProgress({
    fileIndex: index,
    fileName: path.split(/[\\/]/).pop(),
    percent: 0,
    status: "queued",
  }));
}

function updateProgress(progress) {
  let row = document.querySelector(`[data-file-index="${progress.fileIndex}"]`);
  if (!row) {
    row = document.createElement("article");
    row.className = "fileRow";
    row.dataset.fileIndex = progress.fileIndex;
    fileList.appendChild(row);
  }
  row.innerHTML = `
    <div>
      <strong>${escapeHTML(progress.fileName || "Text input")}</strong>
      <span>${escapeHTML(progress.status || "queued")} ${progress.message ? `· ${escapeHTML(progress.message)}` : ""}</span>
    </div>
    ${renderProgress(progress.percent || 0)}
  `;
}

function setStatus(message) {
  statusText.textContent = message || t("ready");
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
