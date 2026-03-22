param(
  [ValidateSet("current", "linux")]
  [string]$Target = "current",
  [string]$OutputDir = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$BackendRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
if ([string]::IsNullOrWhiteSpace($OutputDir)) {
  $OutputDir = Join-Path $BackendRoot "bin"
}

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

$binaries = @(
  @{ Name = "server"; Package = "./cmd/server" },
  @{ Name = "worker"; Package = "./cmd/worker" },
  @{ Name = "initdb"; Package = "./cmd/initdb" }
)

Push-Location $BackendRoot
try {
  $oldGOOS = $env:GOOS
  $oldCGO = $env:CGO_ENABLED

  if ($Target -eq "linux") {
    $env:GOOS = "linux"
    $env:CGO_ENABLED = "0"
  } else {
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
  }

  $goos = (& go env GOOS).Trim()
  $ext = if ($goos -eq "windows") { ".exe" } else { "" }

  foreach ($binary in $binaries) {
    $outputPath = Join-Path $OutputDir ($binary.Name + $ext)
    Write-Host "Building $($binary.Name) -> $outputPath"
    & go build -ldflags "-s -w" -o $outputPath $binary.Package
    if ($LASTEXITCODE -ne 0) {
      throw "go build failed for $($binary.Package)"
    }
  }

  Write-Host ""
  Write-Host "Build completed. Binaries are in: $OutputDir"
}
finally {
  if ($null -eq $oldGOOS) {
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
  } else {
    $env:GOOS = $oldGOOS
  }

  if ($null -eq $oldCGO) {
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
  } else {
    $env:CGO_ENABLED = $oldCGO
  }

  Pop-Location
}
