$sdk = "$env:LOCALAPPDATA\Android\Sdk"
$escaped = $sdk -replace '\\', '\\'
Set-Content -Path "mobile\android\local.properties" -Value "sdk.dir=$escaped"
Write-Host "local.properties written: sdk.dir=$escaped"

# Patch gradle.properties: reduce heap, limit workers, build arm64 only (faster + less RAM)
$gradleProps = "mobile\android\gradle.properties"
$content = Get-Content $gradleProps -Raw
$content = $content -replace 'org\.gradle\.jvmargs=.*', 'org.gradle.jvmargs=-Xmx2048m -XX:MaxMetaspaceSize=512m'
$content = $content -replace 'reactNativeArchitectures=.*', 'reactNativeArchitectures=arm64-v8a'
if ($content -notmatch 'org\.gradle\.workers\.max') {
    $content = $content -replace '(org\.gradle\.jvmargs=.*)', "`$1`norg.gradle.workers.max=1"
} else {
    $content = $content -replace 'org\.gradle\.workers\.max=\d+', 'org.gradle.workers.max=1'
}
Set-Content -Path $gradleProps -Value $content
Write-Host "gradle.properties patched: 2048m heap, arm64-v8a only, max 1 worker"
