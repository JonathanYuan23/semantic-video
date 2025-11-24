const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("electronAPI", {
  chooseDirectory: (options) => ipcRenderer.invoke("choose-directory", options),
  chooseFile: () => ipcRenderer.invoke("choose-file"),
});
