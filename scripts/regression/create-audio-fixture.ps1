$ErrorActionPreference = 'Stop'
$root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$out = Join-Path $root 'test-results\audio-fixtures'
New-Item -ItemType Directory -Force -Path $out | Out-Null
$text = 'Ode automated QA test. This is synthetic test content, not a personal journal. Today I felt stressed about a work deadline. I took a short walk and asked a colleague for help. Tomorrow I plan to break the task into three smaller steps.'
Add-Type -AssemblyName System.Speech
$voice = New-Object System.Speech.Synthesis.SpeechSynthesizer
try {
    $voice.SetOutputToWaveFile((Join-Path $out 'qa-english-processing.wav'))
    $voice.Speak($text)
} finally { $voice.Dispose() }
& ffmpeg -hide_banner -loglevel error -y -i (Join-Path $out 'qa-english-processing.wav') -ac 1 -ar 44100 -c:a aac -b:a 64k (Join-Path $out 'qa-english-processing.m4a')
if ($LASTEXITCODE -ne 0) { throw 'QA audio conversion failed' }
$duration = & ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 (Join-Path $out 'qa-english-processing.m4a')
if ($LASTEXITCODE -ne 0) { throw 'QA duration verification failed' }
@{ fixture='qa-english-processing'; reference=$text; durationSec=[double]::Parse($duration, [Globalization.CultureInfo]::InvariantCulture); format='AAC mono 44100Hz'; synthetic=$true } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $out 'qa-english-processing.json') -Encoding utf8
Write-Output "Controlled QA audio ready: $out"
