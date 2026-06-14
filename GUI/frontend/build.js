const fs = require("fs");
const path = require("path");

const root = __dirname;
const src = path.join(root, "src");
const dist = path.join(root, "dist");

function copyDir(from, to) {
  fs.mkdirSync(to, { recursive: true });
  for (const entry of fs.readdirSync(from, { withFileTypes: true })) {
    const source = path.join(from, entry.name);
    const target = path.join(to, entry.name);
    if (entry.isDirectory()) {
      copyDir(source, target);
    } else {
      fs.copyFileSync(source, target);
    }
  }
}

function build() {
  fs.rmSync(dist, { recursive: true, force: true });
  copyDir(src, dist);
  console.log("frontend built to", dist);
}

build();

if (process.argv.includes("--watch")) {
  fs.watch(src, { recursive: true }, build);
  setInterval(() => {}, 1000);
}
