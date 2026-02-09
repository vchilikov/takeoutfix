Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Repo = "vchilikov/takeout-fix"
$Cwd = (Get-Location).Path
$TmpDir = $null
$RunExe = $null
$ExitCode = 0

function Write-Step {
    param([string]$Message)
    Write-Host $Message
}

function Resolve-Arch {
    $archRaw = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    switch ($archRaw) {
        "X64" { return "amd64" }
        "Arm64" { return "arm64" }
        default { throw "Unsupported Windows architecture: $archRaw" }
    }
}

function Ensure-Exiftool {
    if (Get-Command exiftool -ErrorAction SilentlyContinue) {
        return
    }

    Write-Step "exiftool not found in PATH. Installing via winget..."
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        throw "exiftool is missing and winget is not available. Install exiftool manually and rerun."
    }

    & winget install --id OliverBetz.ExifTool --exact --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) {
        throw "winget failed to install exiftool (exit code $LASTEXITCODE)."
    }

    if (-not (Get-Command exiftool -ErrorAction SilentlyContinue)) {
        throw "exiftool installation completed but exiftool is still not available in PATH."
    }
}

function Get-LatestTag {
    $headers = @{
        "User-Agent" = "takeoutfix-installer"
        "Accept"     = "application/vnd.github+json"
    }
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers $headers
    $tag = [string]$release.tag_name
    if ([string]::IsNullOrWhiteSpace($tag)) {
        throw "Failed to resolve latest release tag from GitHub API."
    }
    return $tag
}

function Get-ExpectedChecksum {
    param(
        [string]$ChecksumsPath,
        [string]$AssetName
    )

    foreach ($line in Get-Content -LiteralPath $ChecksumsPath) {
        $trimmed = $line.Trim()
        if ([string]::IsNullOrWhiteSpace($trimmed)) {
            continue
        }
        $parts = $trimmed -split "\s+", 2
        if ($parts.Count -lt 2) {
            continue
        }

        $name = $parts[1].Trim()
        if ($name.StartsWith("*")) {
            $name = $name.Substring(1)
        }
        if ($name -eq $AssetName) {
            return $parts[0].ToLowerInvariant()
        }
    }

    throw "Checksum entry for $AssetName not found."
}

try {
    $arch = Resolve-Arch
    Ensure-Exiftool

    $tag = Get-LatestTag
    $assetName = "takeoutfix_${tag}_windows_${arch}.zip"
    $checksumsName = "checksums.txt"
    $baseUrl = "https://github.com/$Repo/releases/download/$tag"

    $TmpDir = Join-Path $Cwd (".takeoutfix-installer-{0}" -f [Guid]::NewGuid().ToString("N"))
    New-Item -Path $TmpDir -ItemType Directory -Force | Out-Null

    $archivePath = Join-Path $TmpDir $assetName
    $checksumsPath = Join-Path $TmpDir $checksumsName

    Write-Step "Downloading $assetName..."
    Invoke-WebRequest -Uri "$baseUrl/$assetName" -OutFile $archivePath
    Invoke-WebRequest -Uri "$baseUrl/$checksumsName" -OutFile $checksumsPath

    $expected = Get-ExpectedChecksum -ChecksumsPath $checksumsPath -AssetName $assetName
    $actual = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "Checksum mismatch for $assetName."
    }

    Expand-Archive -LiteralPath $archivePath -DestinationPath $TmpDir -Force

    $sourceExe = Join-Path $TmpDir "takeoutfix.exe"
    if (-not (Test-Path -LiteralPath $sourceExe -PathType Leaf)) {
        $candidate = Get-ChildItem -Path $TmpDir -File -Recurse -Filter "takeoutfix.exe" | Select-Object -First 1
        if ($null -eq $candidate) {
            throw "takeoutfix.exe was not found in downloaded archive."
        }
        $sourceExe = $candidate.FullName
    }

    $RunExe = Join-Path $Cwd (".takeoutfix-run-$PID-{0}.exe" -f [DateTimeOffset]::UtcNow.ToUnixTimeSeconds())
    Copy-Item -LiteralPath $sourceExe -Destination $RunExe -Force

    Write-Step "Running TakeoutFix in: $Cwd"
    & $RunExe --workdir $Cwd
    $ExitCode = if ($null -eq $LASTEXITCODE) { 0 } else { [int]$LASTEXITCODE }
    if ($ExitCode -ne 0) {
        throw "takeoutfix exited with code $ExitCode."
    }
}
catch {
    if ($ExitCode -eq 0) {
        $ExitCode = 1
    }
    $global:LASTEXITCODE = $ExitCode
    throw
}
finally {
    if ($RunExe -and (Test-Path -LiteralPath $RunExe)) {
        Remove-Item -LiteralPath $RunExe -Force -ErrorAction SilentlyContinue
    }
    if ($TmpDir -and (Test-Path -LiteralPath $TmpDir)) {
        Remove-Item -LiteralPath $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

$global:LASTEXITCODE = $ExitCode
