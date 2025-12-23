# dashboard-login.ps1
#Requires -Version 5.1

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
try {
  Add-Type -TypeDefinition @"
  using System;
  using System.Runtime.InteropServices;
  public static class WinApi {
      [DllImport("user32.dll")] public static extern bool SetCursorPos(int X, int Y);
      [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, uint dx, uint dy, uint dwData, UIntPtr dwExtraInfo);
      public const uint MOUSEEVENTF_LEFTDOWN = 0x0002;
      public const uint MOUSEEVENTF_LEFTUP = 0x0004;
  }
"@ -ErrorAction Stop
} catch {}

# =========================
# 사용자 설정
# =========================


$LoginUrl = " "


$DashboardUrls = @(

)


$ChromePath = ""


$UsernamePlain = ""
$PasswordPlain = ""

$EnableCertBypass = $true

# CDP 설정
$CdpHost = "127.0.0.1"
$CdpPort = 9222
$CdpStatePath = Join-Path $PSScriptRoot "dashboard-cdp.json"

$UseSlowTyping = $true          # true면 한 글자씩 천천히 전송
$SlowTypingDelayMs = 90         # 글자 사이 간격(ms)


$UsernameClickRelativeX = 0.5
$UsernameClickRelativeY = 0.4
$PasswordClickRelativeX = 0.5
$PasswordClickRelativeY = 0.48


$UsernameTabPressCount = 0
$PasswordTabPressCount = 0

# 클릭/포커스 후 입력까지 잠시 대기(ms)
$InputFocusDelayMs = 250

# 페이지 로딩 대기
$AfterLaunchWaitSeconds = 30       # Chrome 창이 완전히 뜰 때까지
$AfterLoginWaitSeconds = 8        # 로그인 제출 후 대시보드가 뜰 때까지
$AfterOpenTabWaitSeconds = 4      # 각 탭 URL 로딩 대기


$UserDataDir = Join-Path $env:LOCALAPPDATA "dashboard-roller-chrome-profile"

# 문제 발생 시 프로필 초기화(쿠키/세션 포함 전부 초기화됨)
$ResetProfileOnStart = $false

# =========================
# 내부 함수(가급적 수정 금지)
# =========================

function Resolve-ChromePath {
  param([string]$Preferred)


  $candidates = @(
    $Preferred,
    (Join-Path $env:ProgramFiles "Google\Chrome\Application\chrome.exe"),
    (Join-Path ${env:ProgramFiles(x86)} "Google\Chrome\Application\chrome.exe")
  ) | Where-Object { $_ -and $_.Trim() -ne "" } | Select-Object -Unique

  foreach ($path in $candidates) {
    if (Test-Path -LiteralPath $path) { return $path }
  }

  throw "Chrome not found. Set `$ChromePath explicitly."
}

function Escape-SendKeysText {
  param([Parameter(Mandatory)] [string]$Text)


  $sb = New-Object System.Text.StringBuilder
  foreach ($ch in $Text.ToCharArray()) {
    switch ($ch) {
      '+' { [void]$sb.Append('{+}') }
      '^' { [void]$sb.Append('{^}') }
      '%' { [void]$sb.Append('{%}') }
      '~' { [void]$sb.Append('{~}') }
      '(' { [void]$sb.Append('{(}') }
      ')' { [void]$sb.Append('{)}') }
      '{' { [void]$sb.Append('{{}') }  # '{' 리터럴
      '}' { [void]$sb.Append('{}}') }  # '}' 리터럴
      default { [void]$sb.Append($ch) }
    }
  }
  return $sb.ToString()
}

function Click-PagePoint {
  param(
    [double]$RelativeX,
    [double]$RelativeY
  )


  $screen = [System.Windows.Forms.Screen]::PrimaryScreen
  $x = [int]([math]::Round($screen.Bounds.Width * $RelativeX))
  $y = [int]([math]::Round($screen.Bounds.Height * $RelativeY))

  [WinApi]::SetCursorPos($x, $y) | Out-Null
  Start-Sleep -Milliseconds 100
  [WinApi]::mouse_event([WinApi]::MOUSEEVENTF_LEFTDOWN, 0, 0, 0, [UIntPtr]::Zero)
  [WinApi]::mouse_event([WinApi]::MOUSEEVENTF_LEFTUP, 0, 0, 0, [UIntPtr]::Zero)
  Start-Sleep -Milliseconds 150
}

function Get-ChromeMainWindowProcessAfterLaunch {
  param(
    [Parameter(Mandatory)] [DateTime]$LaunchTime,
    [int]$TimeoutSeconds = 45
  )


  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)

  while ((Get-Date) -lt $deadline) {
    $candidates = @()

    foreach ($p in (Get-Process chrome -ErrorAction SilentlyContinue)) {
      try {
        if ($p.MainWindowHandle -ne 0 -and $p.StartTime -ge $LaunchTime.AddSeconds(-2)) {
          $candidates += $p
        }
      } catch {

      }
    }

    if ($candidates.Count -gt 0) {
      return ($candidates | Sort-Object StartTime -Descending | Select-Object -First 1)
    }

    Start-Sleep -Milliseconds 200
  }

  throw "Chrome main window not detected. Chrome 창이 완전히 뜰 때까지 더 오래 기다리거나, 다른 Chrome 창을 모두 닫은 뒤 다시 실행하세요."
}

function Activate-ChromeOrThrow {
  param(
    [Parameter(Mandatory)] $Shell,
    [Parameter(Mandatory)] [int]$TargetPid
  )


  if (-not $Shell.AppActivate($TargetPid)) {
    throw "Failed to activate Chrome window. Focus must remain on the dedicated dashboard Chrome window."
  }
}

function Send-ActiveKeys {
  param(
    [Parameter(Mandatory)] $Shell,
    [Parameter(Mandatory)] [int]$TargetPid,
    [Parameter(Mandatory)] [string]$Keys,
    [int]$DelayMs = 80
  )

  Activate-ChromeOrThrow -Shell $Shell -TargetPid $TargetPid
  $Shell.SendKeys($Keys)
  if ($DelayMs -gt 0) { Start-Sleep -Milliseconds $DelayMs }
}

function Type-ActiveText {
  param(
    [Parameter(Mandatory)] $Shell,
    [Parameter(Mandatory)] [int]$TargetPid,
    [Parameter(Mandatory)] [string]$Text
  )

  Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys (Escape-SendKeysText -Text $Text) -DelayMs 30
}

function Type-SlowText {

  param(
    [Parameter(Mandatory)] $Shell,
    [Parameter(Mandatory)] [int]$TargetPid,
    [Parameter(Mandatory)] [string]$Text
  )

  foreach ($ch in $Text.ToCharArray()) {
    $piece = Escape-SendKeysText -Text ([string]$ch)
    Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys $piece -DelayMs $SlowTypingDelayMs
  }
}

function Clear-ActiveInput {

  param(
    [Parameter(Mandatory)] $Shell,
    [Parameter(Mandatory)] [int]$TargetPid
  )


  Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys "{HOME}" -DelayMs 80
  Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys "+{END}" -DelayMs 80
  Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys "{BACKSPACE}" -DelayMs 80
}

function FocusAndType {
  param(
    [Parameter(Mandatory)] $Shell,
    [Parameter(Mandatory)] [int]$TargetPid,
    [Parameter(Mandatory)] [double]$RelativeX,
    [Parameter(Mandatory)] [double]$RelativeY,
    [Parameter(Mandatory)] [int]$TabPressCount,
    [Parameter(Mandatory)] [string]$Text
  )

  Click-PagePoint -RelativeX $RelativeX -RelativeY $RelativeY
  for ($i = 0; $i -lt $TabPressCount; $i++) {
    Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys "{TAB}" -DelayMs 120
  }
  Start-Sleep -Milliseconds $InputFocusDelayMs
  Clear-ActiveInput -Shell $Shell -TargetPid $TargetPid
  if ($UseSlowTyping) {
    Type-SlowText -Shell $Shell -TargetPid $TargetPid -Text $Text
  } else {
    Type-ActiveText -Shell $Shell -TargetPid $TargetPid -Text $Text
  }
}

function Navigate-CurrentTabUrl {
  param(
    [Parameter(Mandatory)] $Shell,
    [Parameter(Mandatory)] [int]$TargetPid,
    [Parameter(Mandatory)] [string]$Url,
    [int]$WaitSeconds = 2
  )

  # 주소창으로 이동 -> URL 입력 -> Enter
  Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys "^{l}" -DelayMs 120
  Type-ActiveText -Shell $Shell -TargetPid $TargetPid -Text $Url
  Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys "{ENTER}" -DelayMs 0
  Start-Sleep -Seconds $WaitSeconds
}

function Open-NewTabUrl {
  param(
    [Parameter(Mandatory)] $Shell,
    [Parameter(Mandatory)] [int]$TargetPid,
    [Parameter(Mandatory)] [string]$Url,
    [int]$WaitSeconds = 2
  )

  # 새 탭 -> URL 입력 -> Enter
  Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys "^{t}" -DelayMs 200

  Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys "^{l}" -DelayMs 120
  Type-ActiveText -Shell $Shell -TargetPid $TargetPid -Text $Url
  Send-ActiveKeys -Shell $Shell -TargetPid $TargetPid -Keys "{ENTER}" -DelayMs 0
  Start-Sleep -Seconds $WaitSeconds
}

# =========================
# CDP 상태 파일 생성
# =========================

function Get-CdpTargets {
  param(
    [Parameter(Mandatory)] [string]$Host,
    [Parameter(Mandatory)] [int]$Port,
    [int]$TimeoutSeconds = 5
  )

  $uri = "http://$Host`:$Port/json"
  return Invoke-RestMethod -Method Get -Uri $uri -TimeoutSec $TimeoutSeconds
}

function Resolve-CdpTargetsForUrls {
  param(
    [Parameter(Mandatory)] [string[]]$Urls,
    [Parameter(Mandatory)] [string]$Host,
    [Parameter(Mandatory)] [int]$Port,
    [int]$RetryCount = 10,
    [int]$DelaySeconds = 1
  )

  $lastError = $null
  $lastMissing = @()

  for ($attempt = 1; $attempt -le $RetryCount; $attempt++) {
    try {
      $targets = Get-CdpTargets -Host $Host -Port $Port -TimeoutSeconds 5
      $lastError = $null
    } catch {
      $lastError = $_
      Start-Sleep -Seconds $DelaySeconds
      continue
    }

    $pages = $targets | Where-Object { $_.type -eq "page" -and $_.webSocketDebuggerUrl }
    $resolved = @()
    $missing = @()

    for ($i = 0; $i -lt $Urls.Count; $i++) {
      $url = $Urls[$i]
      $match = $pages | Where-Object { $_.url -eq $url } | Select-Object -First 1
      if (-not $match) {
        $match = $pages | Where-Object { $_.url -like "$url*" } | Select-Object -First 1
      }

      if ($match) {
        $resolved += [PSCustomObject]@{
          tabIndex = $i + 1
          url = $url
          targetId = $match.id
          webSocketDebuggerUrl = $match.webSocketDebuggerUrl
        }
      } else {
        $missing += $url
      }
    }

    if ($missing.Count -eq 0) {
      return $resolved
    }

    $lastMissing = $missing
    Start-Sleep -Seconds $DelaySeconds
  }

  if ($lastError) {
    throw "CDP 접속 실패: $($lastError.Exception.Message)"
  }

  throw "CDP 대상 URL 매칭 실패: $([string]::Join(', ', $lastMissing))"
}

# =========================
# 실행 로직
# =========================

if ($DashboardUrls.Count -ne 4) {
  throw "DashboardUrls must contain exactly 4 URLs."
}


if ([string]::IsNullOrWhiteSpace($UsernamePlain) -or [string]::IsNullOrWhiteSpace($PasswordPlain)) {
  throw "Username/Password must be set in script (PUT_ID_HERE / PUT_PW_HERE 교체 필요)."
}

$chromeExe = Resolve-ChromePath -Preferred $ChromePath

try {
  if ($ResetProfileOnStart -and (Test-Path -LiteralPath $UserDataDir)) {

    Remove-Item -LiteralPath $UserDataDir -Recurse -Force
  }
  New-Item -ItemType Directory -Path $UserDataDir -Force | Out-Null

  $launchTime = Get-Date


  $chromeArgs = @(
    "--new-window",
    "--guest",                     # 자동완성/저장 비밀번호 간섭 최소화(게스트 프로필)
    "--no-first-run",
    "--disable-session-crashed-bubble",
    "--disable-save-password-bubble", # 비밀번호 저장 팝업 간섭 방지
    "--test-type",                 # unsupported command-line flag 배너 완화
    "--remote-debugging-address=$CdpHost",
    "--remote-debugging-port=$CdpPort",
    "--user-data-dir=`"$UserDataDir`""
  )

  if ($EnableCertBypass) {
    $chromeArgs += @(
      "--ignore-certificate-errors",
      "--allow-running-insecure-content"
    )
  }

  $chromeArgs += $LoginUrl


  Start-Process -FilePath $chromeExe -ArgumentList $chromeArgs | Out-Null


  $chromeMain = Get-ChromeMainWindowProcessAfterLaunch -LaunchTime $launchTime -TimeoutSeconds 45
  $chromeWindowPid = $chromeMain.Id

  $wshell = New-Object -ComObject WScript.Shell
  Activate-ChromeOrThrow -Shell $wshell -TargetPid $chromeWindowPid

  Start-Sleep -Seconds $AfterLaunchWaitSeconds

  Send-ActiveKeys -Shell $wshell -TargetPid $chromeWindowPid -Keys "^{1}" -DelayMs 150

  if ($EnableCertBypass) {

    Type-ActiveText -Shell $wshell -TargetPid $chromeWindowPid -Text "thisisunsafe"
    Start-Sleep -Seconds 2
  }


  FocusAndType -Shell $wshell -TargetPid $chromeWindowPid -RelativeX $UsernameClickRelativeX -RelativeY $UsernameClickRelativeY -TabPressCount $UsernameTabPressCount -Text $UsernamePlain
  Send-ActiveKeys -Shell $wshell -TargetPid $chromeWindowPid -Keys "{TAB}" -DelayMs 150

  Start-Sleep -Milliseconds $InputFocusDelayMs
  Clear-ActiveInput -Shell $wshell -TargetPid $chromeWindowPid
  if ($UseSlowTyping) {
    Type-SlowText -Shell $wshell -TargetPid $chromeWindowPid -Text $PasswordPlain
  } else {
    Type-ActiveText -Shell $wshell -TargetPid $chromeWindowPid -Text $PasswordPlain
  }
  Send-ActiveKeys -Shell $wshell -TargetPid $chromeWindowPid -Keys "{ENTER}" -DelayMs 0


  Start-Sleep -Seconds $AfterLoginWaitSeconds


  Navigate-CurrentTabUrl -Shell $wshell -TargetPid $chromeWindowPid -Url $DashboardUrls[0] -WaitSeconds $AfterOpenTabWaitSeconds

  for ($i = 1; $i -lt $DashboardUrls.Count; $i++) {
    Open-NewTabUrl -Shell $wshell -TargetPid $chromeWindowPid -Url $DashboardUrls[$i] -WaitSeconds $AfterOpenTabWaitSeconds
  }

  $cdpTargets = Resolve-CdpTargetsForUrls -Urls $DashboardUrls -Host $CdpHost -Port $CdpPort -RetryCount 10 -DelaySeconds 1
  $cdpState = [PSCustomObject]@{
    generatedAt = (Get-Date).ToString("o")
    host = $CdpHost
    port = $CdpPort
    targets = $cdpTargets
  }
  $cdpState | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $CdpStatePath -Encoding UTF8

  Write-Host ""
  Write-Host "CDP 상태 파일 생성 완료: $CdpStatePath"


  Send-ActiveKeys -Shell $wshell -TargetPid $chromeWindowPid -Keys "^{1}" -DelayMs 150

  Write-Host ""
  Write-Host "로그인 및 탭 오픈 완료."
  Write-Host "이제 롤링 스크립트로 전환합니다."
  Write-Host "브라우저(Chrome)를 클릭한 뒤 [F11] 키를 눌러 롤링을 시작하세요."
  Write-Host ""


  $rollScriptPath = Join-Path $PSScriptRoot "rolling.ps1"
  Start-Process -FilePath "powershell.exe" -ArgumentList @(
    "-NoProfile",
    "-STA",
    "-ExecutionPolicy", "Bypass",
    "-File", $rollScriptPath
  )
}
finally {

}
