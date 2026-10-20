# Deploys a custom vikunja.exe over an NSSM-managed Vikunja in C:\vikunja.
# Run in an elevated PowerShell from the folder that holds the new vikunja.exe:
#   powershell -ExecutionPolicy Bypass -File .\deploy.ps1
# Rollback: powershell -ExecutionPolicy Bypass -File .\deploy.ps1 -Rollback
param(
	[string]$VikunjaDir = 'C:\vikunja',
	[switch]$Rollback
)
$ErrorActionPreference = 'Stop'

$exe = Join-Path $VikunjaDir 'vikunja.exe'
$previous = Join-Path $VikunjaDir 'vikunja.previous.exe'
$nssm = Join-Path $VikunjaDir 'nssm.exe'

$service = Get-CimInstance Win32_Service |
	Where-Object { $_.PathName -like "*$nssm*" } |
	Select-Object -First 1
if (-not $service) { throw "No NSSM service found for $VikunjaDir" }
Write-Host "Service: $($service.Name), runs as $($service.StartName)"

# Without database.path in config.yml, Vikunja keeps SQLite in %LOCALAPPDATA%\Vikunja of the account
# running it, so the backup must see the service account's environment, not ours.
function Use-ServiceEnvironment {
	$account = $service.StartName
	$env:LOCALAPPDATA = switch -Regex ($account) {
		'^LocalSystem$' { "$env:SystemRoot\System32\config\systemprofile\AppData\Local" }
		'LocalService$' { "$env:SystemRoot\ServiceProfiles\LocalService\AppData\Local" }
		'NetworkService$' { "$env:SystemRoot\ServiceProfiles\NetworkService\AppData\Local" }
		default {
			$name = $account -replace '^\.\\', ''
			$sid = (New-Object System.Security.Principal.NTAccount($name)).Translate(
				[System.Security.Principal.SecurityIdentifier]).Value
			(Get-CimInstance Win32_UserProfile -Filter "SID='$sid'").LocalPath + '\AppData\Local'
		}
	}
	# Variables NSSM passes to vikunja.exe (AppEnvironmentExtra), such as VIKUNJA_DATABASE_PATH.
	foreach ($line in (& $nssm get $service.Name AppEnvironmentExtra 2>$null)) {
		$line = $line -replace "`0", ''
		if ($line -match '^([^=]+)=(.*)$') { Set-Item "env:$($Matches[1])" $Matches[2] }
	}
}

function Stop-Vikunja {
	Stop-Service $service.Name
	# NSSM can report "stopped" while vikunja.exe is still shutting down.
	$deadline = (Get-Date).AddSeconds(30)
	do {
		$running = Get-Process vikunja -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $exe }
		if (-not $running) { return }
		Start-Sleep -Milliseconds 500
	} while ((Get-Date) -lt $deadline)
	throw "vikunja.exe is still running (PID $($running.Id -join ', ')), probably started outside the service. " +
		'Close it and run the script again.'
}

if ($Rollback) {
	if (-not (Test-Path $previous)) { throw "No $previous to roll back to" }
	try {
		Stop-Vikunja
		Copy-Item $previous $exe -Force
		Write-Host 'Rolled back to the previous vikunja.exe'
	} finally {
		Start-Service $service.Name
	}
	return
}

$new = Join-Path $PSScriptRoot 'vikunja.exe'
if (-not (Test-Path $new)) { throw "Put the new vikunja.exe next to this script ($new)" }
if ((Get-FileHash $new).Hash -eq (Get-FileHash $exe).Hash) {
	Write-Host 'This vikunja.exe is already deployed.'
	return
}

# Full backup (database, files, config) with Vikunja's own dump command, before anything changes.
Use-ServiceEnvironment
$started = Get-Date
Push-Location $VikunjaDir
try {
	# Vikunja logs to stderr; Windows PowerShell would turn those lines into errors under 'Stop'.
	$ErrorActionPreference = 'Continue'
	$dumpLog = & $exe dump -p $VikunjaDir 2>&1 | ForEach-Object { "$_" }
	$dumpExit = $LASTEXITCODE
	$ErrorActionPreference = 'Stop'
	$dumpLog | Write-Host
	if ($dumpExit -ne 0) { throw "vikunja dump failed with exit code $dumpExit" }
} finally {
	Pop-Location
}

# A database file created by this very dump means config.yml points somewhere new (for example a "C:\..." path in
# double quotes, where YAML turns \v into a control character), so the service would start on an empty database.
$dbLine = $dumpLog | Select-String 'Using SQLite database at: (.+?)"' | Select-Object -First 1
if ($dbLine) {
	$dbPath = $dbLine.Matches[0].Groups[1].Value -replace '\\\\', '\'
	try { $db = Get-Item -LiteralPath $dbPath -ErrorAction Stop } catch { $db = $null }
	if (-not $db -or $db.CreationTime -ge $started) {
		throw "Vikunja would use a new, empty database at '$dbPath'. Nothing was deployed. " +
			"Set database.path in config.yml to the absolute path of your database, in single quotes."
	}
	Write-Host "Database: $dbPath ($([math]::Round($db.Length / 1KB)) KB)"
}
Write-Host "Backup written to $VikunjaDir (vikunja-dump_*.zip)."

try {
	Stop-Vikunja
	Copy-Item $exe $previous -Force
	Copy-Item $new $exe -Force
	Write-Host 'New vikunja.exe in place.'
} finally {
	Start-Service $service.Name
}

& $exe version
Write-Host 'Deployed. Open Vikunja, press Ctrl+F5, check the filters. Run with -Rollback if anything is wrong.'
