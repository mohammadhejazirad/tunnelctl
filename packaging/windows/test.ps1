param([Parameter(Mandatory=$true)][string]$Assets, [Parameter(Mandatory=$true)][string]$Tag)
$ErrorActionPreference = 'Stop'
$version = $Tag.Substring(1)
$assetsPath = (Resolve-Path $Assets).Path
$installer = Join-Path $assetsPath "tunnelctl_${version}_windows_amd64.msi"
$installDir = Join-Path $env:ProgramFiles 'tunnelctl'
$exe = Join-Path $installDir 'tunnelctl.exe'
$originalPath = [Environment]::GetEnvironmentVariable('Path','Machine')
$canaryDir = Join-Path $env:LOCALAPPDATA 'tunnelctl-package-test'
New-Item -ItemType Directory -Path $canaryDir | Out-Null
$canary = Join-Path $canaryDir 'preserved-user-data.txt'
Set-Content -LiteralPath $canary -Value 'synthetic-preserved-data'
function Invoke-Msi([string[]]$Arguments, [bool]$ExpectFailure=$false) {
  $log = Join-Path $env:RUNNER_TEMP ('msi-' + [guid]::NewGuid().ToString() + '.log')
  $proc = Start-Process msiexec.exe -ArgumentList ($Arguments + @('/qn','/norestart','/l*v', ('"' + $log + '"'))) -Wait -PassThru
  if ($ExpectFailure) { if ($proc.ExitCode -eq 0 -or $proc.ExitCode -eq 3010) { throw 'Expected MSI refusal' } }
  elseif ($proc.ExitCode -ne 0 -and $proc.ExitCode -ne 3010) { Get-Content -LiteralPath $log; throw "MSI failed: $($proc.ExitCode)" }
}
# Do not claim/remove an independently configured pre-existing PATH entry.
$unownedPath = $originalPath + ';' + $installDir
[Environment]::SetEnvironmentVariable('Path',$unownedPath,'Machine')
try {
  Invoke-Msi @('/i', ('"' + $installer + '"')) $true
  if ([Environment]::GetEnvironmentVariable('Path','Machine') -ne $unownedPath) { throw 'Unmanaged PATH entry changed' }
} finally { [Environment]::SetEnvironmentVariable('Path',$originalPath,'Machine') }
# Refuse an unmanaged binary instead of overwriting an unrelated installation.
New-Item -ItemType Directory -Path $installDir | Out-Null
Set-Content -LiteralPath $exe -Value 'synthetic-unowned-collision'
Invoke-Msi @('/i', ('"' + $installer + '"')) $true
if ((Get-Content -LiteralPath $exe -Raw).Trim() -ne 'synthetic-unowned-collision') { throw 'Collision overwritten' }
Remove-Item -LiteralPath $exe
Remove-Item -LiteralPath $installDir

# Build a lower-version fixture with identical payload to exercise the actual
# Windows Installer upgrade engine. This is not a claim of a prior public release.
$native = ((& go run ./cmd/packaging --mode versions --tag $Tag) | ConvertFrom-Json).native_version
$parts = $native.Split('.')
if ([int]$parts[2] -lt 1) { throw 'No lower fixture version available' }
$lower = "$($parts[0]).$($parts[1]).$([int]$parts[2]-1)"
$fixture = Join-Path $env:RUNNER_TEMP 'older-installer-fixture.msi'
& (Join-Path $env:RUNNER_TEMP 'wix-tool/wix.exe') build packaging/windows/Product.wxs -arch x64 -d "SourceDir=$(Join-Path $env:RUNNER_TEMP 'msi-public')" -d "NativeVersion=$lower" -o $fixture
if ($LASTEXITCODE -ne 0) { throw 'Upgrade fixture failed' }
Invoke-Msi @('/i', ('"' + $fixture + '"'))
Invoke-Msi @('/i', ('"' + $installer + '"'))
$machine = [Environment]::GetEnvironmentVariable('Path','Machine')
$owned = @($machine.Split(';') | Where-Object { $_.TrimEnd('\') -ieq $installDir })
if ($owned.Count -ne 1) { throw 'Missing or duplicate machine PATH entry' }
$env:Path = $machine + ';' + [Environment]::GetEnvironmentVariable('Path','User')
Push-Location $env:TEMP
try {
  $info = (& tunnelctl version --json) | ConvertFrom-Json
  if ($LASTEXITCODE -ne 0 -or $info.data.tag -ne $Tag -or $info.data.application -ne $version) { throw 'Installed version mismatch' }
  & tunnelctl --help
  if ($LASTEXITCODE -ne 0) { throw 'Installed help failed' }
} finally { Pop-Location }
Remove-Item -LiteralPath $exe
Invoke-Msi @('/fa', ('"' + $installer + '"'))
if (-not (Test-Path -LiteralPath $exe)) { throw 'Repair failed to restore binary' }
Invoke-Msi @('/i', ('"' + $fixture + '"')) $true
Invoke-Msi @('/x', ('"' + $installer + '"'))
if (Test-Path -LiteralPath $exe) { throw 'Executable left after uninstall' }
if ([Environment]::GetEnvironmentVariable('Path','Machine') -ne $originalPath) { throw 'Uninstall did not preserve original machine PATH' }
if ((Get-Content -LiteralPath $canary -Raw).Trim() -ne 'synthetic-preserved-data') { throw 'User data removed' }
Write-Output 'Native MSI collision/install/upgrade/repair/downgrade-refusal/PATH/uninstall passed. Proxy runtime remains unsupported.'
