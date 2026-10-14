param([Parameter(Mandatory=$true)][string]$Assets, [Parameter(Mandatory=$true)][string]$Tag)
$ErrorActionPreference = 'Stop'
$version = $Tag.Substring(1)
$native = ((& go run ./cmd/packaging --mode versions --tag $Tag) | ConvertFrom-Json).native_version
if ($LASTEXITCODE -ne 0) { throw 'Version conversion failed' }
$assetsPath = (Resolve-Path $Assets).Path
$stage = Join-Path $env:RUNNER_TEMP 'msi-public'
Expand-Archive -LiteralPath (Join-Path $assetsPath "tunnelctl_${version}_windows_amd64.zip") -DestinationPath $stage
$info = (& (Join-Path $stage 'tunnelctl.exe') version --json) | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $info.data.tag -ne $Tag -or $info.data.application -ne $version) { throw 'Binary/tag mismatch' }
$toolDir = Join-Path $env:RUNNER_TEMP 'wix-tool'
& dotnet tool install wix --version 5.0.2 --tool-path $toolDir
if ($LASTEXITCODE -ne 0) { throw 'Pinned WiX installation failed' }
& (Join-Path $toolDir 'wix.exe') build packaging/windows/Product.wxs -arch x64 -d "SourceDir=$stage" -d "NativeVersion=$native" -o (Join-Path $assetsPath "tunnelctl_${version}_windows_amd64.msi")
if ($LASTEXITCODE -ne 0) { throw 'MSI build failed' }
# WiX debug symbols are development artifacts, not release assets.
Get-ChildItem -LiteralPath $assetsPath -Filter '*.wixpdb' | Remove-Item
