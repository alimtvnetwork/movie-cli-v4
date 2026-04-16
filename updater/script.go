package updater

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// executeUpdateWindows writes a temp PowerShell script and runs it.
func executeUpdateWindows(repoPath string) error {
	scriptPath, err := writeUpdateScript(repoPath)
	if err != nil {
		return apperror.Wrap("cannot write update script", err)
	}
	defer os.Remove(scriptPath)

	return runPowerShellScript(scriptPath)
}

// executeUpdateUnix runs the update via pwsh (if available) or direct commands.
func executeUpdateUnix(repoPath string) error {
	if !hasPwshWithRunPS1(repoPath) {
		return executeUpdateDirect(repoPath)
	}
	scriptPath, err := writeUpdateScript(repoPath)
	if err != nil {
		return apperror.Wrap("cannot write update script", err)
	}
	defer os.Remove(scriptPath)
	return runPowerShellScript(scriptPath)
}

func hasPwshWithRunPS1(repoPath string) bool {
	if _, err := exec.LookPath("pwsh"); err != nil {
		return false
	}
	runPS1 := filepath.Join(repoPath, "run.ps1")
	_, statErr := os.Stat(runPS1)
	return statErr == nil
}

// executeUpdateDirect runs the update pipeline directly without PowerShell.
func executeUpdateDirect(repoPath string) error {
	// Pull
	fmt.Println("📥 Pulling latest changes...")
	pullOut, err := gitOutput(repoPath, "pull", "--ff-only")
	if err != nil {
		return apperror.Wrap("git pull failed", err)
	}

	if pullOut == "Already up to date." {
		fmt.Println("✔ Already up to date")
		return nil
	}
	fmt.Printf("  %s\n", pullOut)

	// Build
	fmt.Println("🔨 Building...")
	buildCmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", binaryOutputPath(repoPath), ".")
	buildCmd.Dir = repoPath
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return apperror.Wrap("build failed", err)
	}

	fmt.Println("✅ Build complete")
	return nil
}

// binaryOutputPath returns where the binary should be built.
func binaryOutputPath(repoPath string) string {
	binDir := filepath.Join(repoPath, "bin")
	_ = os.MkdirAll(binDir, 0o755)
	name := "movie"
	if runtime.GOOS == "windows" {
		name = "movie.exe"
	}
	return filepath.Join(binDir, name)
}

// writeUpdateScript generates a temp PowerShell script for the update.
func writeUpdateScript(repoPath string) (string, error) {
	script := buildUpdateScriptContent(repoPath)

	tmpFile, err := os.CreateTemp(os.TempDir(), "movie-update-script-*.ps1")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// UTF-8 BOM for PowerShell compatibility
	bom := []byte{0xEF, 0xBB, 0xBF}
	if _, err := tmpFile.Write(bom); err != nil {
		return "", err
	}
	if _, err := tmpFile.WriteString(script); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

// buildUpdateScriptContent generates the PowerShell script content.
func buildUpdateScriptContent(repoPath string) string {
	return fmt.Sprintf(`$ErrorActionPreference = "Stop"
$repoPath = "%s"

# Capture current version
$oldVersion = "unknown"
$movieBin = Get-Command movie -ErrorAction SilentlyContinue
if ($movieBin -and $movieBin.Source -and (Test-Path $movieBin.Source)) {
    $oldVersion = (& $movieBin.Source version 2>&1) -join " "
}
Write-Host "  Version before: $oldVersion" -ForegroundColor Gray

# Pull latest
Set-Location $repoPath
$pullOutput = git pull --ff-only 2>&1
$pullText = ($pullOutput | ForEach-Object { "$_" }) -join [char]10

if ($pullText -match "Already up to date") {
    Write-Host ""
    Write-Host "  Already up to date ($oldVersion)" -ForegroundColor Green
    exit 0
}

Write-Host "  Pulled new changes" -ForegroundColor Cyan
foreach ($line in $pullOutput) {
    $text = "$line".Trim()
    if ($text.Length -gt 0) { Write-Host "    $text" -ForegroundColor Gray }
}

# Wait for parent to release file handles
Start-Sleep -Seconds 1.2

# Build and deploy (skip pull — already done)
$runScript = Join-Path $repoPath "run.ps1"
if (Test-Path $runScript) {
    & $runScript -NoPull
} else {
    Write-Host "  run.ps1 not found at $runScript" -ForegroundColor Red
    exit 1
}

# Compare versions
$newVersion = "unknown"
$movieBin = Get-Command movie -ErrorAction SilentlyContinue
if ($movieBin -and $movieBin.Source -and (Test-Path $movieBin.Source)) {
    $newVersion = (& $movieBin.Source version 2>&1) -join " "
}

Write-Host ""
if ($oldVersion -eq $newVersion) {
    Write-Host "  WARNING: Version unchanged after update" -ForegroundColor Yellow
    Write-Host "  Was version/info.go bumped?" -ForegroundColor Yellow
} else {
    Write-Host "  Updated: $oldVersion -> $newVersion" -ForegroundColor Green
}

# Show changelog
if ($movieBin -and $movieBin.Source -and (Test-Path $movieBin.Source)) {
    Write-Host ""
    $clOutput = & $movieBin.Source changelog --latest 2>&1
    foreach ($cl in $clOutput) { Write-Host "  $cl" }
}

# Auto-cleanup
if ($movieBin -and $movieBin.Source -and (Test-Path $movieBin.Source)) {
    & $movieBin.Source update-cleanup 2>&1 | Out-Null
}
`, repoPath)
}

// runPowerShellScript executes a PowerShell script with output piped to terminal.
func runPowerShellScript(scriptPath string) error {
	psBin := "powershell"
	if runtime.GOOS != "windows" {
		psBin = "pwsh"
	}

	cmd := exec.Command(psBin, "-ExecutionPolicy", "Bypass", "-NoProfile", "-NoLogo", "-File", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
