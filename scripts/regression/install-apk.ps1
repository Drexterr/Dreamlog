param(
    [string]$Apk,
    [string]$Serial = 'emulator-5554',
    [switch]$NoStreaming
)
$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
if (-not $Apk) {
    $candidate = Get-ChildItem -LiteralPath (Join-Path $repo 'mobile/android/app/build/outputs/apk') -Filter '*.apk' -Recurse |
        Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if (-not $candidate) { throw 'No local APK found. Build the intended variant first.' }
    $Apk = $candidate.FullName
}
$Apk = (Resolve-Path -LiteralPath $Apk).Path
[string]$abi = & adb -s $Serial shell getprop ro.product.cpu.abi
if ($LASTEXITCODE -ne 0 -or -not $abi) { throw 'Android target is unavailable or unauthorized. Reconnect the selected test device.' }
$abi = $abi.Trim()
Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [System.IO.Compression.ZipFile]::OpenRead($Apk)
try {
    $native = @($archive.Entries | Where-Object { $_.FullName -match '^lib/[^/]+/libreactnative\.so$' } | ForEach-Object { $_.FullName.Split('/')[1] })
    if ($abi -notin $native) { throw "APK architectures [$($native -join ', ')] do not match device $abi. Build with -PreactNativeArchitectures=$abi." }
} finally { $archive.Dispose() }
Write-Host "Installing $Apk on $Serial ($abi)"
$installArgs = @('-s', $Serial, 'install', '-r')
if ($NoStreaming) { $installArgs += '--no-streaming' }
$installArgs += $Apk
& adb @installArgs
if ($LASTEXITCODE -ne 0) { throw 'APK installation failed.' }
