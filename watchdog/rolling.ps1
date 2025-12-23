# dashboard-roll.ps1
# Requires -Version 5.1

$ErrorActionPreference = 'Stop'

# ---------------------------------------------------------
# [설정]
# ---------------------------------------------------------
$ProcessName = "chrome"                 # 브라우저 프로세스명
$WindowTitleLike = "*Chrome*"         

$InitialDelaySeconds = 2                # F11 감지 후 대기(초)
$IterationIntervalSeconds = 30          # 탭 전환 간격(초) = 페이지 체류 시간
$TabCount = 4                           # 전체 탭 개수(요구사항: 4)

$AlwaysRefreshTabs = @()                
$FullRefreshIntervalMinutes = 180       # 전체 탭 새로고침 주기(분) = 3시간
# ---------------------------------------------------------

# ---------------------------------------------------------
# [라이브러리 및 WinAPI]
# ---------------------------------------------------------
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

try {
    Add-Type -TypeDefinition @"
    using System;
    using System.Runtime.InteropServices;
    public static class WinApi {
        [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
        [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
        [DllImport("user32.dll")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
        [DllImport("user32.dll")] public static extern bool IsIconic(IntPtr hWnd);
        [DllImport("user32.dll")] public static extern short GetAsyncKeyState(int vKey);
        [DllImport("user32.dll")] public static extern int ShowCursor(bool bShow);
    }
"@ -ErrorAction Stop
} catch {
    
}

$VK_F11 = 0x7A
$script:CursorHidden = $false
$script:CursorHideCount = 0

# ---------------------------------------------------------
# [헬퍼 함수]
# ---------------------------------------------------------

function Get-TargetWindow {
    Get-Process -Name $ProcessName -ErrorAction SilentlyContinue |
        Where-Object { $_.MainWindowHandle -ne 0 -and $_.MainWindowTitle -like $WindowTitleLike } |
        Sort-Object StartTime -Descending |
        Select-Object -First 1
}

function Hide-MouseCursor {
    if (-not $script:CursorHidden) {
        [WinApi]::ShowCursor($false) | Out-Null
        $script:CursorHidden = $true
        $script:CursorHideCount++
    }
}

function Restore-MouseCursor {
    while ($script:CursorHideCount -gt 0) {
        [WinApi]::ShowCursor($true) | Out-Null
        $script:CursorHideCount--
    }

    $script:CursorHidden = $false

    $screen = [System.Windows.Forms.Screen]::PrimaryScreen
    [System.Windows.Forms.Cursor]::Position = New-Object System.Drawing.Point($screen.Bounds.Width / 2, $screen.Bounds.Height / 2)
}

function Focus-Window {
    param($Proc)

    if (-not $Proc) { return $false }

    $handle = $Proc.MainWindowHandle


    if ([WinApi]::IsIconic($handle)) {
        [WinApi]::ShowWindowAsync($handle, 9) | Out-Null
        Start-Sleep -Milliseconds 200
    }

    [WinApi]::SetForegroundWindow($handle) | Out-Null
    return $true
}

function Test-KeyPressed {
    param($VirtualKeyCode)
    return ([WinApi]::GetAsyncKeyState($VirtualKeyCode) -band 0x8000) -ne 0
}

function Wait-ForF11 {
    Write-Host ">>> 준비 완료." -ForegroundColor Green
    Write-Host ">>> 브라우저(Chrome)를 클릭하고 [F11] 키를 눌러 시작하세요."

    while ($true) {
        if (Test-KeyPressed -VirtualKeyCode $VK_F11) {
            Write-Host ""
            Write-Host "[시작] ${InitialDelaySeconds}초 후 루프가 시작됩니다." -ForegroundColor Green
            Start-Sleep -Seconds $InitialDelaySeconds
            break
        }
        Start-Sleep -Milliseconds 100
    }
}

function Write-ProgressStatus {
    param($CurrentTab, $TotalTabs, $RemainingSeconds, $ModeText)

    $bar = ("█" * $CurrentTab) + ("░" * ($TotalTabs - $CurrentTab))
    Write-Host "`r[$bar] 탭 $CurrentTab/$TotalTabs | 다음: ${RemainingSeconds}초 | $ModeText    " -NoNewline -ForegroundColor White
}

function Send-Action {
    param($LogMessage, $Keys)

    $target = Get-TargetWindow
    if (-not $target) {
        Write-Host ""
        Write-Host "[WARN] 대상 Chrome 창을 찾지 못했습니다. (ProcessName=$ProcessName, TitleLike=$WindowTitleLike)" -ForegroundColor Yellow
        Write-Host "[WARN] Chrome 창을 한 개만 띄우고, 창을 활성화한 뒤 다시 시도하세요." -ForegroundColor Yellow
        return $false
    }

    if (-not (Focus-Window -Proc $target)) {
        Write-Host ""
        Write-Host "[WARN] Chrome 창 포커싱에 실패했습니다. 다시 시도합니다." -ForegroundColor Yellow
        return $false
    }

    Start-Sleep -Milliseconds 200

    $timestamp = Get-Date -Format "HH:mm:ss"
    Write-Host ""
    Write-Host "[$timestamp] $LogMessage" -ForegroundColor Cyan

    foreach ($k in $Keys) {
        [System.Windows.Forms.SendKeys]::SendWait($k)
        Start-Sleep -Milliseconds 100
    }

    return $true
}

function Wait-WithProgress {
    param($Seconds, $CurrentTab, $TotalTabs, $ModeText)

    for ($i = $Seconds; $i -gt 0; $i--) {
        Write-ProgressStatus -CurrentTab $CurrentTab -TotalTabs $TotalTabs -RemainingSeconds $i -ModeText $ModeText
        Start-Sleep -Seconds 1
    }
}

$null = Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { try { Restore-MouseCursor } catch {} }
trap { try { Restore-MouseCursor } catch {} ; Write-Error $_ ; return }

# ---------------------------------------------------------
# [메인 로직]
# ---------------------------------------------------------

Wait-ForF11
Hide-MouseCursor
$LastFullRefreshTime = (Get-Date).AddMinutes(-($FullRefreshIntervalMinutes + 1))
$CycleCount = 0

while ($true) {
    $CycleCount++
    $Now = Get-Date
$IsFullRefreshTurn = ($Now - $LastFullRefreshTime).TotalMinutes -ge $FullRefreshIntervalMinutes
    $StatusTitle = if ($IsFullRefreshTurn) { "전체 탭 새로고침" } else { "순환 모드" }

    Write-Host ""
    Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Yellow
    Write-Host " 사이클 #$CycleCount : $StatusTitle" -ForegroundColor Yellow
    Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Yellow

    for ($tabNum = 1; $tabNum -le $TabCount; $tabNum++) {
        $ShouldRefresh = $IsFullRefreshTurn -or ($AlwaysRefreshTabs -contains $tabNum)

        if ($ShouldRefresh) {
            $keys = @("^$tabNum", "{F5}")
            $reason = if ($IsFullRefreshTurn) { "주기" } else { "상시" }
            $logMsg = "탭 $tabNum/$TabCount : 새로고침 수행 ($reason)"
            $modeText = "새로고침(F5)"
        } else {
            $keys = @("^$tabNum")
            $logMsg = "탭 $tabNum/$TabCount"
            $modeText = "모니터링"
        }

        $ok = Send-Action -LogMessage $logMsg -Keys $keys
        if (-not $ok) {
          
            Start-Sleep -Seconds 2
            continue
        }

       
        Wait-WithProgress -Seconds $IterationIntervalSeconds -CurrentTab $tabNum -TotalTabs $TabCount -ModeText $modeText
    }

    if ($IsFullRefreshTurn) {
        $LastFullRefreshTime = Get-Date
        Write-Host ""
        Write-Host ">> 전체 탭 새로고침 완료." -ForegroundColor Green
    }
}
