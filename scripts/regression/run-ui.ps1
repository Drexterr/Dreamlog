param(
    [string]$Serial = 'emulator-5554',
    [string[]]$Flows = @('01-app-launch', '02-onboarding', '04-navigation'),
    [switch]$UseSeededAccount
)
$ErrorActionPreference = 'Stop'
if ($Serial -notmatch '^emulator-\d+$') { throw 'Default Maestro evidence includes device-wide logs. This runner is restricted to dedicated emulators; use app-scoped checks for physical phones.' }
$repo = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$runner = Join-Path $repo '.qa-tools/maestro/maestro/bin/maestro.bat'
if (-not (Test-Path -LiteralPath $runner)) { throw 'Install the official Maestro CLI in .qa-tools/maestro first.' }
if (-not $env:JAVA_HOME) { $env:JAVA_HOME = 'C:\Program Files\Android\Android Studio\jbr' }
$env:JAVA_TOOL_OPTIONS = "$env:JAVA_TOOL_OPTIONS -Djdk.net.unixdomain.tmpdir=$($repo.Replace('\','/'))/.qa-tools/disabled-unix-sockets".Trim()
$env:MAESTRO_CLI_NO_ANALYTICS = 'true'
$previousPassword = $env:ODE_QA_PASSWORD
$previousEmail = $env:ODE_QA_EMAIL
$accountArgs = @()
try {
if ($UseSeededAccount) {
    $fixture = & node -e "const fs=require('fs'),path=require('path');const root=process.argv[1];const yaml=require(path.join(root,'mobile/node_modules/yaml'));console.log(JSON.stringify(yaml.parseAllDocuments(fs.readFileSync(path.join(root,'maestro/flows/03-authentication.yaml'),'utf8'))[0].toJSON().env));" $repo | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0) { throw 'Could not read the existing QA account fixture.' }
    if (-not $env:ODE_QA_EMAIL) { $env:ODE_QA_EMAIL = $fixture.EMAIL_EXISTING }
    if (-not $env:ODE_QA_PASSWORD) { $env:ODE_QA_PASSWORD = $fixture.PASSWORD }
    if (-not $env:ODE_QA_EMAIL -or -not $env:ODE_QA_PASSWORD) { throw 'QA email/password must be configured.' }
    $accountArgs = @('-e', "EMAIL_EXISTING=$($env:ODE_QA_EMAIL)", '-e', "EMAIL_EXISTING_REGEX=$([regex]::Escape($env:ODE_QA_EMAIL))", '-e', "PASSWORD=$($env:ODE_QA_PASSWORD)")
}
$installed = & adb -s $Serial shell pm path com.talktoode.app
if ($LASTEXITCODE -ne 0 -or -not ($installed -match 'package:')) { throw 'Install the current-source com.talktoode.app build first.' }
$stamp = Get-Date -Format 'yyyy-MM-ddTHH-mm-ss'
$output = Join-Path $repo "test-results/ui-$stamp"
New-Item -ItemType Directory -Path $output -Force | Out-Null
$results = @()
foreach ($flow in $Flows) {
    if ($flow -notmatch '^\d{2}-[a-z-]+$') { throw "Invalid flow name: $flow" }
    $file = Join-Path $repo "maestro/flows/$flow.yaml"
    if (-not (Test-Path -LiteralPath $file)) { throw "Missing flow: $file" }
    $log = Join-Path $output "$flow.log"
    $reportPath = Join-Path $output "$flow.xml"
    & $runner test --device $Serial --format JUNIT --output $reportPath --debug-output (Join-Path $output "$flow-artifacts") --test-output-dir (Join-Path $output "$flow-artifacts") @accountArgs $file 2>&1 | ForEach-Object {
        $line = $_.ToString()
        if ($UseSeededAccount) { $line = $line.Replace($env:ODE_QA_PASSWORD, '[REDACTED]') }
        $line
    } | Tee-Object -FilePath $log
    $code = $LASTEXITCODE
    if ($UseSeededAccount) {
        & node (Join-Path $PSScriptRoot 'redact-ui.mjs') $output
        if ($LASTEXITCODE -ne 0) { throw 'Could not redact QA credentials from artifacts.' }
    }
    $reason = 'Mandatory flow assertions passed'
    if ($code -ne 0) {
        $reason = "UI runner exited $code; inspect log and artifacts."
        if (Test-Path -LiteralPath $reportPath) {
            [xml]$report = Get-Content -LiteralPath $reportPath -Raw
            $failures = @($report.SelectNodes('//failure') | ForEach-Object InnerText)
            if ($failures.Count) { $reason = $failures -join '; ' }
        }
    }
    $results += [pscustomobject]@{ feature = $flow; status = $(if ($code -eq 0) { 'passed' } else { 'failed' }); exitCode = $code; reason = $reason; log = $log; report = $reportPath }
}
ConvertTo-Json -InputObject @($results) -Depth 5 | Set-Content -LiteralPath (Join-Path $output 'results.json')
& node (Join-Path $PSScriptRoot 'summarize-ui.mjs') $output
if ($LASTEXITCODE -ne 0) { throw 'Could not summarize device command evidence.' }
Write-Host "UI results: $output"
if ($results.status -contains 'failed') { exit 1 }
} finally {
    $env:ODE_QA_PASSWORD = $previousPassword
    $env:ODE_QA_EMAIL = $previousEmail
}
