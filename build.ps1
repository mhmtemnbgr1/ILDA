# Cross-compiles system-critters for common platforms into ./dist
# Usage:  ./build.ps1

$ErrorActionPreference = "Stop"
$name = "system-critters"
New-Item -ItemType Directory -Force -Path dist | Out-Null

$targets = @(
    @{ os = "windows"; arch = "amd64"; out = "$name-windows-amd64.exe" },
    @{ os = "linux";   arch = "amd64"; out = "$name-linux-amd64" },
    @{ os = "linux";   arch = "arm64"; out = "$name-linux-arm64" },
    @{ os = "darwin";  arch = "amd64"; out = "$name-macos-amd64" },
    @{ os = "darwin";  arch = "arm64"; out = "$name-macos-arm64" }
)

foreach ($t in $targets) {
    Write-Host "building $($t.out)..."
    $env:GOOS = $t.os
    $env:GOARCH = $t.arch
    go build -trimpath -ldflags "-s -w" -o "dist/$($t.out)" .
}

Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
Write-Host "done -> ./dist"
