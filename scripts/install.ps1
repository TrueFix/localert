<#
.SYNOPSIS
Installs Localert for the current Windows user and enables desktop-login autostart.
.EXAMPLE
.\install.ps1 -BinaryPath .\localert-windows-amd64.exe
.EXAMPLE
.\install.ps1 -Uninstall
#>
[CmdletBinding()]
param(
    [string]$BinaryPath,
    [string]$ReleaseVersion,
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'Programs\Localert'),
    [string]$DataDir = (Join-Path $env:APPDATA 'localert'),
    [switch]$NoStart,
    [switch]$Uninstall
)

$ErrorActionPreference = 'Stop'
$InstallDir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($InstallDir)
$DataDir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($DataDir)
$Target = Join-Path $InstallDir 'localert.exe'
$Launcher = Join-Path $InstallDir 'launch.ps1'
$LogDir = Join-Path $env:LOCALAPPDATA 'Localert\logs'
$PidFile = Join-Path $LogDir 'localert.pid'
$Startup = [Environment]::GetFolderPath('Startup')
$Programs = [Environment]::GetFolderPath('Programs')
$Shortcut = Join-Path $Startup 'Localert.lnk'
$DashboardShortcut = Join-Path $Programs 'Localert.url'

function Stop-InstalledLocalert {
    Get-Process -Name localert -ErrorAction SilentlyContinue | ForEach-Object {
        if ($_.Path -and ([IO.Path]::GetFullPath($_.Path) -eq $Target)) {
            $_.Kill()
            if (-not $_.WaitForExit(10000)) { throw 'Localert is still stopping. Try again shortly.' }
        }
    }
}

if ($Uninstall) {
    Stop-InstalledLocalert
    foreach ($File in @($Shortcut, $DashboardShortcut, $Launcher, $Target, $PidFile)) {
        if (Test-Path -LiteralPath $File) { Remove-Item -LiteralPath $File -Force }
    }
    Write-Host 'Localert and its login autostart entry were removed.'
    Write-Host "Database, icons, and logs were kept. Default data location: $DataDir"
    exit 0
}

if (-not [Environment]::Is64BitOperatingSystem) { throw 'Localert requires 64-bit Windows.' }
if (-not $BinaryPath -and $PSScriptRoot -and -not $ReleaseVersion) {
    $Candidates = @(
        (Join-Path $PSScriptRoot 'localert-windows-amd64.exe'),
        (Join-Path $PSScriptRoot '..\bin\localert-windows-amd64.exe'),
        (Join-Path $PSScriptRoot 'localert.exe')
    )
    $BinaryPath = $Candidates | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -First 1
}
if (-not $BinaryPath) {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    if (-not $ReleaseVersion) { $ReleaseVersion = (Invoke-RestMethod -Uri 'https://api.github.com/repos/TrueFix/localert/releases/latest').tag_name }
    if ($ReleaseVersion -notmatch '^v[0-9][0-9A-Za-z.+-]*$') { throw 'Invalid release tag, or no GitHub release is available yet.' }
    $DownloadDir = Join-Path ([IO.Path]::GetTempPath()) ('localert-install-' + [guid]::NewGuid())
    [IO.Directory]::CreateDirectory($DownloadDir) | Out-Null
    $Base = "https://github.com/TrueFix/localert/releases/download/$ReleaseVersion"
    $BinaryPath = Join-Path $DownloadDir 'localert-windows-amd64.exe'
    try {
        Invoke-WebRequest -UseBasicParsing -Uri "$Base/localert-windows-amd64.exe" -OutFile $BinaryPath
        $Checksums = (Invoke-WebRequest -UseBasicParsing -Uri "$Base/SHA256SUMS").Content
        $Match = [regex]::Match($Checksums, '(?m)^([a-fA-F0-9]{64})\s+localert-windows-amd64\.exe\s*$')
        if (-not $Match.Success -or (Get-FileHash -LiteralPath $BinaryPath -Algorithm SHA256).Hash -ne $Match.Groups[1].Value) { throw 'Checksum mismatch; existing installation left unchanged.' }
        # Keep the downloaded file until the copy below; never execute an unverified file.
    } catch { Remove-Item -LiteralPath $DownloadDir -Recurse -Force; throw }
}
if (-not $BinaryPath -or -not (Test-Path -LiteralPath $BinaryPath -PathType Leaf)) {
    throw 'Binary not found. Supply -BinaryPath C:\path\to\localert-windows-amd64.exe.'
}
$BinaryPath = (Resolve-Path -LiteralPath $BinaryPath).Path
$Version = & $BinaryPath --version
if ($LASTEXITCODE -ne 0 -or $Version -notlike 'Localert *') { throw 'The selected file is not a compatible Localert executable.' }

foreach ($Directory in @($InstallDir, $DataDir, $LogDir, $Startup, $Programs)) {
    [IO.Directory]::CreateDirectory($Directory) | Out-Null
}
Stop-InstalledLocalert
if ($BinaryPath -ne $Target) { [IO.File]::Copy($BinaryPath, $Target, $true) }
if ($DownloadDir) { Remove-Item -LiteralPath $DownloadDir -Recurse -Force }

function ConvertTo-ScriptLiteral([string]$Value) { return "'" + $Value.Replace("'", "''") + "'" }

# Values are single-quoted PowerShell literals, never interpolated as commands.
$LaunchTemplate = @'
$ErrorActionPreference = 'Stop'
$Target = __TARGET__
$DataDir = __DATA__
$LogDir = __LOG__
$PidFile = Join-Path $LogDir 'localert.pid'
if (Test-Path -LiteralPath $PidFile) {
    $OldPid = Get-Content -LiteralPath $PidFile -ErrorAction SilentlyContinue | Select-Object -First 1
    $ParsedPid = 0
    if ([int]::TryParse([string]$OldPid, [ref]$ParsedPid)) {
        $Existing = Get-Process -Id $ParsedPid -ErrorAction SilentlyContinue
        if ($Existing -and $Existing.Path -eq $Target) { exit 0 }
    }
}
# Start-Process joins ArgumentList into one Windows command line. Quote the data
# path and double any trailing backslashes before the closing quote.
$QuotedData = '"' + ($DataDir -replace '(\\+)$', '$1$1') + '"'
$QuotedPid = '"' + $PidFile + '"'
$Process = Start-Process -FilePath $Target -ArgumentList @('--data-dir', $QuotedData, '--pid-file', $QuotedPid) -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $LogDir 'localert.log') -RedirectStandardError (Join-Path $LogDir 'localert-error.log')
Set-Content -LiteralPath $PidFile -Value $Process.Id -Encoding ASCII
'@
$Replacements = @{ '__TARGET__' = $Target; '__DATA__' = $DataDir; '__LOG__' = $LogDir }
$LaunchText = [regex]::Replace($LaunchTemplate, '__(TARGET|DATA|LOG)__', {
    param($Match)
    ConvertTo-ScriptLiteral $Replacements[$Match.Value]
})
Set-Content -LiteralPath $Launcher -Value $LaunchText -Encoding UTF8

$PowerShell = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
$Arguments = '-NoProfile -NonInteractive -WindowStyle Hidden -ExecutionPolicy Bypass -File "' + $Launcher + '"'
$Shell = New-Object -ComObject WScript.Shell
$Link = $Shell.CreateShortcut($Shortcut)
$Link.TargetPath = $PowerShell
$Link.Arguments = $Arguments
$Link.WorkingDirectory = $InstallDir
$Link.Description = 'Localert desktop notification service'
$Link.WindowStyle = 7
$Link.Save()
Set-Content -LiteralPath $DashboardShortcut -Encoding ASCII -Value "[InternetShortcut]`r`nURL=http://127.0.0.1:8787/`r`n"

if (-not $NoStart) { Start-Process -FilePath $PowerShell -ArgumentList $Arguments -WindowStyle Hidden }
Write-Host "Installed $Version"
Write-Host "Binary:    $Target"
Write-Host "Database:  $(Join-Path $DataDir 'localert.db')"
Write-Host "Autostart: $Shortcut"
Write-Host "Logs:      $LogDir"
Write-Host 'Dashboard: http://127.0.0.1:8787 (also available in the Start menu)'
if ($NoStart) { Write-Host 'Localert will start the next time you sign in.' }
else { Write-Host 'Localert is starting. The browser opens automatically if password setup is needed.' }
