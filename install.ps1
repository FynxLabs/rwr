#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$REPO = "FynxLabs/rwr"
$BINARY_PATH = "$env:ProgramFiles\rwr"
$LICENSE_PATH = "$env:ProgramFiles\rwr\doc"
$README_PATH = "$env:ProgramFiles\rwr\doc"

# Installing into Program Files and editing the machine PATH both need
# elevation. Checking up front gives a clear message instead of failing
# part-installed further down.
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($identity)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "This installer writes to $BINARY_PATH and the machine PATH, so it needs an elevated PowerShell."
    Write-Host "Re-run it from a PowerShell started with 'Run as administrator'."
    exit 1
}

$OS = "Windows"

# Match the architecture names goreleaser publishes. Is64BitOperatingSystem only
# answers 64-bit yes/no, so it reported x86_64 on ARM64 machines — which
# installed the amd64 build to run under emulation while the native arm64 build
# we ship went unused — and i386 on 32-bit, which has never been published at all.
switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    'X64'   { $ARCH = "x86_64" }
    'Arm64' { $ARCH = "arm64" }
    default {
        $detected = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
        Write-Host "Unsupported architecture: $detected. RWR publishes Windows builds for x86_64 and arm64."
        exit 1
    }
}

Write-Host "Installing RWR for $OS $ARCH"

$headers = @{ 'User-Agent' = 'rwr-installer' }
try {
    $latest_release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest" -Headers $headers
} catch {
    Write-Host "Failed to query the latest release: $_"
    exit 1
}

$asset_name = "rwr_${OS}_${ARCH}.zip"
$download_url = $latest_release.assets |
    Where-Object { $_.name -eq $asset_name } |
    Select-Object -First 1 -ExpandProperty browser_download_url

if (-not $download_url) {
    Write-Host "Could not find $asset_name in the latest release. Exiting."
    exit 1
}

# goreleaser publishes checksums.txt alongside the archives; without verifying
# it, a corrupted or substituted download installs silently.
$checksums_url = $latest_release.assets |
    Where-Object { $_.name -eq "checksums.txt" } |
    Select-Object -First 1 -ExpandProperty browser_download_url

if (-not $checksums_url) {
    Write-Host "The release does not publish checksums.txt, so the download cannot be verified. Exiting."
    exit 1
}

# A fresh directory per run: a fixed path under TEMP can be pre-created by
# another process, and this script later copies out of it with elevation.
$tmp_dir = Join-Path ([System.IO.Path]::GetTempPath()) ("rwr_" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmp_dir -Force | Out-Null

try {
    $tmp_file = Join-Path $tmp_dir "rwr.zip"
    $tmp_sums = Join-Path $tmp_dir "checksums.txt"

    Write-Host "Downloading $asset_name"
    Invoke-WebRequest -Uri $download_url -OutFile $tmp_file -Headers $headers -UseBasicParsing
    Invoke-WebRequest -Uri $checksums_url -OutFile $tmp_sums -Headers $headers -UseBasicParsing

    $expected = (Get-Content $tmp_sums |
        Where-Object { $_ -match "\s\*?$([regex]::Escape($asset_name))$" } |
        Select-Object -First 1) -split '\s+' | Select-Object -First 1

    if (-not $expected) {
        Write-Host "checksums.txt has no entry for $asset_name. Exiting."
        exit 1
    }

    $actual = (Get-FileHash -Path $tmp_file -Algorithm SHA256).Hash
    if ($actual -ne $expected.ToUpper()) {
        Write-Host "Checksum mismatch for ${asset_name}:"
        Write-Host "  expected $expected"
        Write-Host "  actual   $actual"
        Write-Host "Refusing to install."
        exit 1
    }
    Write-Host "Checksum verified"

    $tmp_extract = Join-Path $tmp_dir "extracted"
    Expand-Archive -Path $tmp_file -DestinationPath $tmp_extract -Force

    $binary = Join-Path $tmp_extract "rwr.exe"
    if (-not (Test-Path $binary)) {
        Write-Host "Binary 'rwr.exe' not found in the downloaded archive. Exiting."
        exit 1
    }

    New-Item -ItemType Directory -Force -Path $BINARY_PATH | Out-Null
    New-Item -ItemType Directory -Force -Path $LICENSE_PATH | Out-Null
    New-Item -ItemType Directory -Force -Path $README_PATH | Out-Null

    Copy-Item -Path $binary -Destination $BINARY_PATH -Force

    foreach ($doc in @("LICENSE", "README.adoc", "README", "README.md")) {
        $path = Join-Path $tmp_extract $doc
        if (Test-Path $path) {
            Copy-Item -Path $path -Destination $README_PATH -Force
        }
    }

    # Read the machine PATH from the registry rather than $env:Path, which is the
    # expanded per-process copy — writing that back would flatten other entries.
    $current_path = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ([string]::IsNullOrEmpty($current_path)) {
        [Environment]::SetEnvironmentVariable("Path", $BINARY_PATH, "Machine")
    } elseif ($current_path.Split(';') -notcontains $BINARY_PATH) {
        [Environment]::SetEnvironmentVariable("Path", ($current_path.TrimEnd(';') + ";" + $BINARY_PATH), "Machine")
    }

    Write-Host "rwr has been installed to $BINARY_PATH for $OS $ARCH."
    Write-Host "Open a new terminal for the PATH change to take effect."
} finally {
    Remove-Item -Path $tmp_dir -Recurse -Force -ErrorAction SilentlyContinue
}
