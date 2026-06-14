export function attachDragDrop(node, onPaths) {
  if (!node) return;
  node.addEventListener("dragover", (event) => {
    event.preventDefault();
    node.classList.add("dragging");
  });
  node.addEventListener("dragleave", () => node.classList.remove("dragging"));
  node.addEventListener("drop", (event) => {
    event.preventDefault();
    node.classList.remove("dragging");
    const paths = Array.from(event.dataTransfer?.files || []).map((file) => file.path || file.name);
    onPaths(paths);
  });
}
