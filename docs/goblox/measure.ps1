param([string]$Exe, [string]$ArgLine, [string]$Dir, [string]$Log)
# Runs $Exe $ArgLine in $Dir, polls the peak working set every 20 ms, prints wall time and peak MB.
$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $Exe
$psi.Arguments = $ArgLine
$psi.WorkingDirectory = $Dir
$psi.UseShellExecute = $false
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$sw = [System.Diagnostics.Stopwatch]::StartNew()
$p = [System.Diagnostics.Process]::Start($psi)
$out = $p.StandardOutput.ReadToEndAsync()
$err = $p.StandardError.ReadToEndAsync()
$peak = 0
while (-not $p.HasExited) {
	try { $p.Refresh(); if ($p.PeakWorkingSet64 -gt $peak) { $peak = $p.PeakWorkingSet64 } } catch {}
	Start-Sleep -Milliseconds 20
}
$p.WaitForExit()
$sw.Stop()
if ($Log) { ($out.Result + $err.Result) | Out-File -Encoding utf8 $Log }
"wall {0:N2} s  peak {1:N0} MB  exit {2}" -f $sw.Elapsed.TotalSeconds, ($peak / 1MB), $p.ExitCode
