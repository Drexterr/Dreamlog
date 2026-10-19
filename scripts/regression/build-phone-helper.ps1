$ErrorActionPreference = 'Stop'
$root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$sdk = Join-Path $env:LOCALAPPDATA 'Android\Sdk'
$tools = Join-Path $sdk 'build-tools\36.0.0'
$java = Join-Path $root '.qa-tools\temurin17\jdk-17.0.20.1+1\bin'
$source = Join-Path $PSScriptRoot 'android-helper'
$out = Join-Path $root 'test-results\phone-helper-build'
$classes = Join-Path $out 'classes'
$dex = Join-Path $out 'dex'
New-Item -ItemType Directory -Force -Path $classes, $dex | Out-Null
$platform = Join-Path $sdk 'platforms\android-36\android.jar'
function Invoke-Checked($exe, $arguments) {
    & $exe @arguments
    if ($LASTEXITCODE -ne 0) { throw "Helper build command failed: $exe" }
}
Invoke-Checked (Join-Path $java 'javac.exe') @('-source', '8', '-target', '8', '-classpath', $platform, '-d', $classes, (Join-Path $source 'Hierarchy.java'))
$env:JAVA_HOME = Split-Path $java -Parent
$env:PATH = "$java;$env:PATH"
Invoke-Checked (Join-Path $java 'jar.exe') @('cf', (Join-Path $out 'helper.jar'), '-C', $classes, '.')
Invoke-Checked (Join-Path $tools 'd8.bat') @('--lib', $platform, '--min-api', '24', '--output', $dex, (Join-Path $out 'helper.jar'))
$unsigned = Join-Path $out 'unsigned.apk'
Invoke-Checked (Join-Path $tools 'aapt.exe') @('package', '-f', '-M', (Join-Path $source 'AndroidManifest.xml'), '-I', $platform, '-F', $unsigned)
Push-Location $dex
try { Invoke-Checked (Join-Path $tools 'aapt.exe') @('add', $unsigned, 'classes.dex') } finally { Pop-Location }
Invoke-Checked (Join-Path $tools 'zipalign.exe') @('-f', '4', $unsigned, (Join-Path $out 'aligned.apk'))
Invoke-Checked (Join-Path $tools 'apksigner.bat') @('sign', '--ks', (Join-Path $root 'mobile\android\app\debug.keystore'), '--ks-pass', 'pass:android', '--key-pass', 'pass:android', '--out', (Join-Path $out 'ode-qa-helper.apk'), (Join-Path $out 'aligned.apk'))
Write-Output "Built Ode-only test helper: $out\ode-qa-helper.apk"
