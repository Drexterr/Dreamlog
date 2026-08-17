param([string]$Task = "assembleRelease")
$env:JAVA_HOME = "C:\Program Files\Android\Android Studio\jbr"

# Release variants run Sentry's source-map upload as part of the build
# (see mobile/android/sentry.properties + node_modules/@sentry/react-native/sentry.gradle).
# That step needs SENTRY_AUTH_TOKEN and fails the whole Gradle build late (after several
# minutes) if it's missing. Check for it up front so we either have it before starting,
# or explicitly skip the upload - never fail cold at the last step.
$isReleaseTask = $Task -match "Release"
$tokenCacheFile = Join-Path $PSScriptRoot ".sentry-token.local"
$sentryProps = Join-Path $PSScriptRoot "..\mobile\android\sentry.properties"

if ($isReleaseTask -and -not $env:SENTRY_AUTH_TOKEN -and -not $env:SENTRY_DISABLE_AUTO_UPLOAD) {
    $hasPropsToken = (Test-Path $sentryProps) -and (Select-String -Path $sentryProps -Pattern "^\s*auth\.token\s*=" -Quiet)

    if (-not $hasPropsToken) {
        if (Test-Path $tokenCacheFile) {
            $cachedToken = (Get-Content $tokenCacheFile -Raw).Trim()
            if ($cachedToken) {
                $env:SENTRY_AUTH_TOKEN = $cachedToken
                Write-Host "Using cached Sentry auth token ($tokenCacheFile)"
            }
        }

        if (-not $env:SENTRY_AUTH_TOKEN) {
            Write-Host ""
            Write-Host "This release build uploads source maps to Sentry, which needs an auth token." -ForegroundColor Yellow
            Write-Host "Create one at https://bharat-jain-i5.sentry.io -> Settings -> Auth Tokens (scope: project:releases)." -ForegroundColor Yellow
            $secureToken = Read-Host -Prompt "Paste Sentry auth token (leave blank to skip Sentry upload for this build)" -AsSecureString
            $plainToken = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
                [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureToken)
            )

            if ([string]::IsNullOrWhiteSpace($plainToken)) {
                Write-Host "No token entered - skipping Sentry source-map upload for this build." -ForegroundColor Yellow
                $env:SENTRY_DISABLE_AUTO_UPLOAD = "true"
            } else {
                $env:SENTRY_AUTH_TOKEN = $plainToken
                Set-Content -Path $tokenCacheFile -Value $plainToken -NoNewline
                Write-Host "Token saved to $tokenCacheFile (gitignored) - future builds won't ask again."
            }
        }
    }
}

Set-Location (Join-Path $PSScriptRoot "..\mobile\android")
& ".\gradlew.bat" $Task "--no-daemon"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
