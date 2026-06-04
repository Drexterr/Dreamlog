param([string]$Task = "assembleRelease")
$env:JAVA_HOME = "C:\Program Files\Android\Android Studio\jbr"
Set-Location "$PSScriptRoot\..\mobile\android"
& ".\gradlew.bat" $Task "--no-daemon"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
