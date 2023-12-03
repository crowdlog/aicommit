/* eslint-disable @typescript-eslint/no-explicit-any */
import { watch } from "chokidar";

const log = (...args: any[]) =>
  console.log("\x1b[36m%s\x1b[0m", "[watcher]", ...args);
const logError = (...args: any[]) => console.log("\x1b[31m%s\x1b[0m", ...args);

const compile = async ({
  path,
  ready,
  init,
}: {
  path: string;
  ready: boolean;
  init: boolean;
}) => {
  if (!ready) {
    return;
  }
  log("compiling", path);
  try {
    log("building go");
    const buildResult = Bun.spawnSync("go build".split(" "), {
      cwd: import.meta.dir,
    });
    log(buildResult.stderr.toString());
    log(buildResult.stdout.toString());
  } catch (e) {
    log(e);
  }
};

async function main() {
  const isWatch = process.execArgv.find((arg) => arg.includes("watch"));
  if (!isWatch) {
    log("not in watch mode, compiling once");
    await compile({
      init: true,
      ready: true,
      path: "",
    });
    return;
  }

  const watcher = watch("**/*.{ts,go}", {
    ignored: "(node_modules|target)/**/*",
  });
  let ready = false;

  watcher.on("ready", async () => {
    log("ready");
    ready = true;
    await compile({
      init: true,
      ready,
      path: "",
    });
  });

  watcher.on("change", async (path) => {
    await compile({
      init: false,
      path,
      ready,
    });
  });
  watcher.on("add", async (path) => {
    await compile({
      path,
      ready,
      init: false,
    });
  });
}

main();
