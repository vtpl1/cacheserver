# Application name
$AppName = "cacheserver"

# Version and build info
$Version = git describe --tags --always
$Build = Get-Date -Format "yyyy-MM-ddTHH:mm"

# Output directory
$OutputDir = "binwin"

# Platforms to build for
$Platforms = @(
    "windows/amd64",
    "linux/386",
    "linux/amd64",
    "linux/arm/7",
    "linux/arm64",
    "darwin/amd64"
)

# Build flags
$LdFlags = "-s -w -X main.GitCommit=$Version -X main.BuildTime=$Build"

# Clean up the output directory
Write-Host "Cleaning up..."
if (Test-Path $OutputDir) {
    Remove-Item -Recurse -Force $OutputDir
}
New-Item -ItemType Directory -Path $OutputDir | Out-Null

# Build for each platform
foreach ($platform in $Platforms) {
    $parts = $platform -split "/"
    $os = $parts[0]
    $arch = $parts[1]
    $arm = if ($parts.Length -gt 2) { $parts[2] } else { $null }
    
    $output = "$OutputDir/$AppName"
    $output += "_$os"
    $output += "_$arch"
    if ($arm) {
        $output += "v$arm"
    }
    if ($os -eq "windows") {
        $output += ".exe"
    }

    Write-Host "Building for $os/$arch$(if ($arm) { "v$arm" })..."
    $env:GOOS = $os
    $env:GOARCH = $arch
    if ($arm) {
        $env:GOARM = $arm
    } else {
        Remove-Item Env:GOARM -ErrorAction SilentlyContinue
    }

    go build -ldflags $LdFlags -o $output
    Write-Host "Built: $output"
}
