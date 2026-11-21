import { defineConfig } from "mygo-cli";

export default defineConfig({
  name: "PassGuard",
  identifier: "com.passguard.app",
  version: "2.4.1",
  copyright: "© 2026 PassGuard",
  // The window shows the native interface Go draws, not a web page, so
  // there is no frontend to develop or build.
  out: "build",
  windows: {
    displayName: "守密",
    installDirName: "PassGuard",
    // Refreshes the shortcuts' icons after an update: Windows caches them
    // by path and can keep a blank one from an earlier build.
    installerInclude: "installer.nsh",
  },
  linux: {
    categories: ["Utility", "Security"],
    comment: "守密 - 本地密码管理器",
    webview: false,
  },
});
