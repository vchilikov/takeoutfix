# TakeoutFix（简体中文）

无需丢失重要元数据，即可迁移你的 Google Photos 照片库。
TakeoutFix 会处理 Google Takeout ZIP 压缩包，并恢复拍摄日期、位置和描述等信息。

[English](../README.md) · [Русский](README.ru.md) · [中文](README.zh-CN.md) · [हिन्दी](README.hi.md) · [Español](README.es.md) · [Français](README.fr.md) · [العربية](README.ar.md) · [Deutsch](README.de.md)

## 为什么选择 TakeoutFix

- 迁移时尽量保留照片和视频的关键元数据。
- 直接处理 Google Takeout 标准 ZIP 导出文件。
- 终端流程清晰，步骤明确。
- 面向普通用户，无需编写脚本。

## 快速开始（3 步）

1. 在 Google Takeout 中导出 Google Photos（ZIP 格式）。
2. 将所有 `*.zip` 放到同一个本地文件夹。
3. 在该文件夹中打开终端，并运行下面的推荐命令。

## 运行（推荐）

请在包含 Takeout ZIP 文件的文件夹中直接运行。

macOS/Linux:

```bash
curl -fsSL https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr -useb https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1 | iex
```

## 手动运行（可选）

如果你不想使用一键安装命令，可使用此方式。

1. 从 [GitHub Releases](https://github.com/vchilikov/takeout-fix/releases) 下载适用于你系统的最新二进制文件。
2. 在包含 Takeout ZIP 文件的文件夹中运行该程序。

macOS/Linux:

```bash
./takeoutfix
./takeoutfix --workdir /path/to/folder
```

Windows (PowerShell):

```powershell
.\takeoutfix.exe
.\takeoutfix.exe --workdir C:\path\to\folder
```

## 处理结果

成功运行后：

- 处理后的媒体文件位于 `./takeoutfix-extracted/Takeout`。
- 支持的照片和视频会写入元数据。
- JSON 中的 `Tags` 会写入 `Keywords` 和 `Subject`。
- 如果 JSON 中的拍摄时间戳缺失或无效，且文件名以 `YYYY-MM-DD HH.MM.SS` 开头，会从文件名恢复日期。
- 每次运行的详细报告会保存到 `./.takeoutfix/reports/report-YYYYMMDD-HHMMSS.json`。
- 你可以将 `./takeoutfix-extracted/Takeout` 上传到新的云存储。

## 常见问题

- `No ZIP files or extracted Takeout data found in this folder.`
  - 请将所有 Takeout ZIP 分卷放到工作目录顶层，或在已包含解压后 Takeout 内容的目录中运行工具。
- `Some ZIP files are corrupted. Please re-download them and run again.`
  - 请在 Google Takeout 重新下载损坏的分卷后重试。
- `Step 1/3: Checking dependencies... missing`
  - 请先使用上方推荐的一键命令，或手动安装 `exiftool` 后再运行。
- `Not enough free disk space to continue.`
  - 请释放更多磁盘空间后重试。
- macOS 提示应用未验证
  - 去除隔离属性后再运行：
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
