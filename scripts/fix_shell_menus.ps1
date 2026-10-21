# fix_shell_menus.ps1 - Automated Administrator Fix for Rsync Context Menus & Service
# Uses the exact Microsoft ExtendedSubCommandsKey pattern verified from PowerShell 7.

$ErrorActionPreference = "SilentlyContinue"

# 1. Elevate to Administrator if not already elevated
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "[INFO] Elevation required to configure system-wide Explorer context menus." -ForegroundColor Yellow
    Write-Host "[INFO] Prompting Windows UAC..." -ForegroundColor Cyan
    Start-Process powershell.exe -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs
    exit
}

$baseDir = "E:\Backup\pgwiz\rsync"
$uiExe = "$baseDir\rsync-ui.exe"
$svcExe = "$baseDir\rsync-service.exe"
$iconPath = "$baseDir\assets\icon.ico"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host " Rsync Controller - System-Wide Explorer Context Menu Fix " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 2. Clean up any stale/conflicting keys in HKCU and HKLM
Write-Host "[STEP 1/4] Purging old/conflicting Rsync registry entries..." -ForegroundColor Yellow
$targets = @(
    "Directory\shell\Rsync",
    "Directory\Background\shell\Rsync",
    "Folder\shell\Rsync",
    "Drive\shell\Rsync",
    "DesktopBackground\Shell\Rsync",
    "*\shell\Rsync",
    "Directory\ContextMenus\Rsync"
)

foreach ($t in $targets) {
    reg delete "HKCU\Software\Classes\$t" /f *>$null
    reg delete "HKLM\Software\Classes\$t" /f *>$null
}

# 3. Create ContextMenus Pool under HKLM (System-Wide) & HKCU
Write-Host "[STEP 2/4] Registering ContextMenus submenu pool..." -ForegroundColor Yellow

$roots = @("HKLM:\Software\Classes", "HKCU:\Software\Classes")

foreach ($r in $roots) {
    $pool = "$r\Directory\ContextMenus\Rsync\shell"

    # 01: Local Copy
    New-Item -Path "$pool\01_local_copy\command" -Force | Out-Null
    Set-ItemProperty -Path "$pool\01_local_copy" -Name "MUIVerb" -Value "rs local -> &Copy to..."
    Set-ItemProperty -Path "$pool\01_local_copy" -Name "Icon" -Value $iconPath
    Set-ItemProperty -Path "$pool\01_local_copy\command" -Name "(Default)" -Value "`"$uiExe`" --mode=copy --src=`"%1`" --bg=`"%V`""

    # 02: Local Pull
    New-Item -Path "$pool\02_local_pull\command" -Force | Out-Null
    Set-ItemProperty -Path "$pool\02_local_pull" -Name "MUIVerb" -Value "rs local -> &Pull from..."
    Set-ItemProperty -Path "$pool\02_local_pull" -Name "Icon" -Value $iconPath
    Set-ItemProperty -Path "$pool\02_local_pull\command" -Name "(Default)" -Value "`"$uiExe`" --mode=pull --dest=`"%1`" --bg=`"%V`""

    # 03: Alpha Server Push
    New-Item -Path "$pool\03_alpha_copy\command" -Force | Out-Null
    Set-ItemProperty -Path "$pool\03_alpha_copy" -Name "MUIVerb" -Value "rs server Alpha -> P&ush to server..."
    Set-ItemProperty -Path "$pool\03_alpha_copy" -Name "Icon" -Value $iconPath
    Set-ItemProperty -Path "$pool\03_alpha_copy" -Name "SeparatorBefore" -Value ""
    Set-ItemProperty -Path "$pool\03_alpha_copy" -Name "CommandFlags" -Value 0x20 -Type DWord
    Set-ItemProperty -Path "$pool\03_alpha_copy\command" -Name "(Default)" -Value "`"$uiExe`" --mode=copy --server=`"alpha`" --src=`"%1`" --bg=`"%V`""

    # 04: Alpha Server Pull
    New-Item -Path "$pool\04_alpha_pull\command" -Force | Out-Null
    Set-ItemProperty -Path "$pool\04_alpha_pull" -Name "MUIVerb" -Value "rs server Alpha -> Pul&l from server..."
    Set-ItemProperty -Path "$pool\04_alpha_pull" -Name "Icon" -Value $iconPath
    Set-ItemProperty -Path "$pool\04_alpha_pull\command" -Name "(Default)" -Value "`"$uiExe`" --mode=pull --server=`"alpha`" --dest=`"%1`" --bg=`"%V`""

    # 05: Open UI
    New-Item -Path "$pool\05_open_ui\command" -Force | Out-Null
    Set-ItemProperty -Path "$pool\05_open_ui" -Name "MUIVerb" -Value "&Open Rsync Controller..."
    Set-ItemProperty -Path "$pool\05_open_ui" -Name "Icon" -Value $iconPath
    Set-ItemProperty -Path "$pool\05_open_ui" -Name "SeparatorBefore" -Value ""
    Set-ItemProperty -Path "$pool\05_open_ui" -Name "CommandFlags" -Value 0x20 -Type DWord
    Set-ItemProperty -Path "$pool\05_open_ui\command" -Name "(Default)" -Value "`"$uiExe`""

    # 4. Create Parent Trigger Verbs (using ExtendedSubCommandsKey pattern)
    $parentPaths = @(
        "$r\Directory\shell\Rsync",
        "$r\Directory\Background\shell\Rsync",
        "$r\Folder\shell\Rsync",
        "$r\Drive\shell\Rsync",
        "$r\DesktopBackground\Shell\Rsync",
        "$r\*\shell\Rsync"
    )

    foreach ($p in $parentPaths) {
        New-Item -Path $p -Force | Out-Null
        Set-ItemProperty -Path $p -Name "MUIVerb" -Value "Rsync"
        Set-ItemProperty -Path $p -Name "Icon" -Value $iconPath
        Set-ItemProperty -Path $p -Name "ExtendedSubCommandsKey" -Value "Directory\ContextMenus\Rsync"
        # Explicitly ensure NO (Default) value and NO SubCommands value
        Remove-ItemProperty -Path $p -Name "(Default)" -ErrorAction SilentlyContinue
        Remove-ItemProperty -Path $p -Name "SubCommands" -ErrorAction SilentlyContinue
        Remove-Item -Path "$p\shell" -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "[STEP 3/4] ContextMenus & trigger verbs successfully registered in HKLM & HKCU." -ForegroundColor Green

# 5. Service installation & startup
Write-Host "[STEP 4/4] Checking Windows Service [RsyncControllerDaemon]..." -ForegroundColor Yellow
$svc = Get-Service -Name "RsyncControllerDaemon" -ErrorAction SilentlyContinue
if (-not $svc) {
    Write-Host "[INFO] Installing Windows Service..." -ForegroundColor Cyan
    & $svcExe --install
    Start-Sleep -Milliseconds 500
}

$svc = Get-Service -Name "RsyncControllerDaemon" -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -ne "Running") {
    Write-Host "[INFO] Starting Windows Service..." -ForegroundColor Cyan
    Start-Service RsyncControllerDaemon -ErrorAction SilentlyContinue
}

$finalSvc = Get-Service -Name "RsyncControllerDaemon" -ErrorAction SilentlyContinue
if ($finalSvc -and $finalSvc.Status -eq "Running") {
    Write-Host "[OK] RsyncControllerDaemon is RUNNING." -ForegroundColor Green
} else {
    Write-Host "[WARN] Service status: $(if ($finalSvc) { $finalSvc.Status } else { 'Not installed' })" -ForegroundColor Yellow
}

# 6. Shell change notification and Explorer restart prompt
Write-Host "`n--- Explorer Refresh ---" -ForegroundColor Cyan
Write-Host "Broadcasting shell change notification..."
$code = @"
using System;
using System.Runtime.InteropServices;
public class ShellNotifier {
    [DllImport("shell32.dll", CharSet = CharSet.Auto, SetLastError = true)]
    public static extern void SHChangeNotify(int wEventId, uint uFlags, IntPtr dwItem1, IntPtr dwItem2);
}
"@
try {
    Add-Type -TypeDefinition $code
    [ShellNotifier]::SHChangeNotify(0x08000000, 0x1000, [IntPtr]::Zero, [IntPtr]::Zero)
    Write-Host "[OK] Shell notification sent." -ForegroundColor Green
} catch {
    Write-Host "[WARN] Shell notification Add-Type skipped."
}

Write-Host "`n[COMPLETE] Fix script finished successfully!" -ForegroundColor Green
Write-Host "You can now right-click any folder or background to see the Rsync submenu flyout." -ForegroundColor White
Write-Host "Press any key to exit..."
$null = $host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
