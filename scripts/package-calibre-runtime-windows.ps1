param(
    [Parameter(Mandatory = $true)]
    [string] $DistDir
)

$ErrorActionPreference = 'Stop'

$calibreDir = if ($env:KIDNEY_CALIBRE_DIR) {
    $env:KIDNEY_CALIBRE_DIR
} else {
    Join-Path $env:ProgramFiles 'Calibre2'
}

$ebookConvert = Join-Path $calibreDir 'ebook-convert.exe'
if (-not (Test-Path $ebookConvert)) {
    Write-Error "Calibre ebook-convert not found at $ebookConvert"
}

$toolsDir = Join-Path $DistDir 'tools'
$runtimeDir = Join-Path $toolsDir 'calibre'

if (Test-Path $runtimeDir) {
    Remove-Item -Recurse -Force $runtimeDir
}

New-Item -ItemType Directory -Force $toolsDir | Out-Null
Copy-Item -Recurse -Force $calibreDir $runtimeDir

& (Join-Path $runtimeDir 'ebook-convert.exe') --version
Write-Host "Packaged Calibre runtime at $runtimeDir"
