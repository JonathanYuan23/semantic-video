import { app, BrowserWindow, ipcMain, dialog } from "electron";
import path from "node:path";
import { fileURLToPath } from "node:url";

// Suppress dconf/GSettings warnings in sandboxed environments (e.g., WSLg)
process.env.GSETTINGS_BACKEND = process.env.GSETTINGS_BACKEND || "memory";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const isDev = process.env.ELECTRON_START_URL || process.env.NODE_ENV === "development";

const createWindow = () => {
  const win = new BrowserWindow({
    width: 1280,
    height: 800,
    webPreferences: {
      preload: path.join(__dirname, "preload.cjs"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
    },
  });

  const startUrl =
    process.env.ELECTRON_START_URL ||
    (isDev
      ? "http://localhost:5173"
      : `file://${path.join(__dirname, "../dist/index.html")}`);

  win.loadURL(startUrl);
};

app.whenReady().then(() => {
  createWindow();

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});

ipcMain.handle("choose-directory", async (_event, _options) => {
  const win = BrowserWindow.getFocusedWindow();
  const result = await dialog.showOpenDialog(win, {
    properties: ["openDirectory", "multiSelections"],
  });
  if (result.canceled) return;
  return { paths: result.filePaths };
});

ipcMain.handle("choose-file", async () => {
  const win = BrowserWindow.getFocusedWindow();
  const result = await dialog.showOpenDialog(win, {
    properties: ["openFile"],
    filters: [
      { name: "Videos", extensions: ["mp4", "mov", "mkv", "avi", "m4v", "webm"] },
      { name: "All Files", extensions: ["*"] },
    ],
  });
  if (result.canceled || result.filePaths.length === 0) return;
  return result.filePaths[0];
});
