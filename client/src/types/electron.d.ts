export {};

declare global {
  interface Window {
    electronAPI?: {
      chooseDirectory: (options?: { recursive?: boolean }) => Promise<{ paths: string[] } | undefined>;
      chooseFile: () => Promise<string | undefined>;
    };
  }
}
