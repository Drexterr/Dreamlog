param([string]$Architecture = 'x86_64', [string]$ApiUrl)
$ErrorActionPreference = 'Stop'
if ($Architecture -notin @('x86_64', 'arm64-v8a')) { throw 'Unsupported regression architecture.' }
$repo = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
if (-not $env:JAVA_HOME) { $env:JAVA_HOME = 'C:\Program Files\Android\Android Studio\jbr' }
if (-not (Test-Path -LiteralPath (Join-Path $env:JAVA_HOME 'bin/java.exe'))) { throw 'JAVA_HOME must point to an installed JDK.' }
$unavailableSocketPath = Join-Path $repo '.qa-tools/disabled-unix-sockets'
if (Test-Path -LiteralPath $unavailableSocketPath) { throw 'The regression Unix-socket fallback path must not exist.' }
# The local Windows runtime accepts Unix-socket binds but fails to connect.
# An unavailable bind path makes Java use its supported TCP fallback instead.
$env:JAVA_TOOL_OPTIONS = "$env:JAVA_TOOL_OPTIONS -Djdk.net.unixdomain.tmpdir=$($unavailableSocketPath.Replace('\','/'))".Trim()
$env:SENTRY_DISABLE_AUTO_UPLOAD = 'true'
if ($ApiUrl) { $env:EXPO_PUBLIC_API_URL = $ApiUrl }
Push-Location (Join-Path $repo 'mobile/android')
try {
    & './gradlew.bat' ':app:assembleRelease' "-PreactNativeArchitectures=$Architecture" '--no-daemon'
    if ($LASTEXITCODE -ne 0) { throw "Android build failed ($LASTEXITCODE)." }
} finally { Pop-Location }
