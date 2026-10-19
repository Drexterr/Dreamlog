param(
    [string]$Sdk = 'C:\Users\bharat.jain\AppData\Local\Android\Sdk',
    [int]$MemoryMB = 3072,
    [ValidateSet('auto', 'host', 'software', 'swiftshader', 'swangle', 'lavapipe')]
    [string]$Gpu = 'host',
    [switch]$Launch
)
$ErrorActionPreference = 'Stop'
if ($MemoryMB -lt 2048 -or $MemoryMB -gt 4096) { throw 'QA emulator memory must be between 2048 and 4096 MB.' }
$repo = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$manager = Join-Path $Sdk 'cmdline-tools/latest/bin/avdmanager.bat'
$image = Join-Path $Sdk 'system-images/android-35/google_apis/x86_64/system.img'
if (-not (Test-Path -LiteralPath $manager) -or -not (Test-Path -LiteralPath $image)) { throw 'Verified command-line tools and API 35 image are required.' }
$env:JAVA_HOME = 'C:\Program Files\Android\Android Studio\jbr'
$env:JAVA_TOOL_OPTIONS = "-Djdk.net.unixdomain.tmpdir=$($repo.Replace('\','/'))/.qa-tools/disabled-unix-sockets"
$env:ANDROID_HOME = $Sdk
$env:ANDROID_SDK_ROOT = $Sdk
$env:ANDROID_AVD_HOME = Join-Path $repo '.qa-tools/avd'
$env:ANDROID_USER_HOME = Join-Path $repo '.qa-tools/android-user'
New-Item -ItemType Directory -Force -Path $env:ANDROID_AVD_HOME, $env:ANDROID_USER_HOME | Out-Null
$avd = Join-Path $env:ANDROID_AVD_HOME 'Ode_QA_API35.avd'
if (-not (Test-Path -LiteralPath (Join-Path $avd 'config.ini'))) {
    'no' | & $manager create avd --name Ode_QA_API35 --package 'system-images;android-35;google_apis;x86_64' --path $avd --device pixel_7
    if ($LASTEXITCODE -ne 0) { throw 'Could not create the isolated API 35 AVD.' }
}
if ($Launch) {
    $connected = & adb devices
    if ($connected -match 'emulator-5556\s+device') { throw 'Port 5556 already has a device; inspect it before launching another.' }
    $log = Join-Path $repo 'test-results/stable-emulator'
    New-Item -ItemType Directory -Force -Path $log | Out-Null
    $stamp = Get-Date -Format 'yyyy-MM-ddTHH-mm-ss'
    $process = Start-Process -FilePath (Join-Path $Sdk 'emulator/emulator.exe') -ArgumentList @('-avd', 'Ode_QA_API35', '-port', '5556', '-memory', "$MemoryMB", '-cores', '2', '-gpu', $Gpu, '-no-window', '-no-audio', '-no-snapshot') -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $log "stdout-$stamp.log") -RedirectStandardError (Join-Path $log "stderr-$stamp.log")
    Write-Host "Launched isolated API 35 emulator PID $($process.Id); serial emulator-5556. Wait for sys.boot_completed before installing."
} else { Write-Host 'Isolated API 35 AVD prepared; launch after builds finish.' }
