export function renderProgress(percent) {
  const value = Math.max(0, Math.min(100, Number(percent) || 0));
  return `<div class="progress" aria-label="Progress"><span style="width:${value}%"></span><em>${value}%</em></div>`;
}
