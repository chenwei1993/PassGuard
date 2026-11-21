# 守密 PassGuard（MyGo 原生界面版）

本地密码管理器。界面由 Go 的 `github.com/egoist/mygo/ui` 直接绘制，不使用 WebView，也没有前端构建。

> 官网 / 宣传页：https://chenwei1993.github.io/PassGuard/

## 功能

- 加密保险库：scrypt 派生密钥 + AES-256-GCM，与 Electron 版使用同一文件格式
- 密码条目：新增、编辑、删除、收藏、搜索、按标签/文件夹/强度筛选、列表与网格视图
- 密码生成器：随机密码、易记密码、PIN，强度评分、破解时间与历史
- 安全检查：弱密码、重复密码、已泄露、已过期，及改进建议
- 导入/导出：加密备份（.bin）与 CSV，支持合并或覆盖导入
- 本地备份：立即备份、自动备份间隔与保留数量、历史恢复
- 设置：主题（浅色/深色/跟随系统）、字体大小、语言（简体中文/English）、自动锁定、剪贴板清理、通知、快捷键、修改主密码、数据修复与删除
- 系统集成：托盘菜单（显示/锁定/退出）、关闭到托盘、单实例、最小化锁定、系统通知、系统凭据存储记住密钥
- 自动更新：启动时检查 GitHub Releases，发现新版本可下载并安装；只访问 GitHub，不发送密码或保险库数据，可在“设置 → 关于”中关闭

## 开发与构建

```sh
npm install
npm run dev      # 开发模式（原生窗口）
npm run build    # 生成 build/windows-amd64/PassGuard.exe 与安装包
```

构建产物：

- `build/windows-amd64/PassGuard.exe`
- `build/windows-amd64/passguard-setup-2.4.2-amd64.exe`

## 数据位置

- 保险库：`%APPDATA%\PassGuard\PasswordVault\vault.bin`
- 备份：`%APPDATA%\PassGuard\Backups\`
- 应用设置：`%APPDATA%\PassGuard\settings.json`

与 Electron 版的数据目录和保险库格式兼容。首次运行会自动合并 `Password Vault` / `localpwd` 旧目录中的数据；原有主密码可直接解锁。

> Electron 版用系统 DPAPI 把“记住的密钥”保存在保险库文件中。原生版改用系统凭据存储（Windows Credential Manager）保存该密钥，因此首次需要手动输入一次主密码；勾选“启动时自动解锁”后，之后可自动解锁。

调试时可用 `PASSGUARD_DATA` 指定独立的数据目录：

```sh
$env:PASSGUARD_DATA='C:\temp\passguard-test'; npm run dev
```

## 支持的系统

Windows 10（1809 及以上）/ Windows 11，64 位。不支持 Windows 7。

## 许可证

MIT，详见 [LICENSE](LICENSE)。
