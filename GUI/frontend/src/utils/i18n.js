const messages = {
  en: {
    ready: "Ready",
    running: "Running",
    paused: "Paused",
    saved: "Settings saved",
  },
  zh: {
    ready: "就绪",
    running: "转换中",
    paused: "已暂停",
    saved: "设置已保存",
  },
};

const lang = navigator.language?.toLowerCase().startsWith("zh") ? "zh" : "en";

export function t(key) {
  return messages[lang][key] || messages.en[key] || key;
}
