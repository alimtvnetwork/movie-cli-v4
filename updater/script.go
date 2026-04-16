package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/apperror"
)

// executeUpdateWindows writes a temp PowerShell script and runs it.
func executeUpdateWindows(repoPath, targetBinary string) error {
	scriptPath, err := writeUpdateScript(repoPath, targetBinary)
	if err != nil {
		return apperror.Wrap("cannot write update script", err)
	}
	defer os.Remove(scriptPath)

	return runPowerShellScript(scriptPath)
}

// executeUpdateUnix runs the update via pwsh (if available) or direct commands.
func executeUpdateUnix(repoPath, targetBinary string) error {
	if !hasPwshWithRunPS1(repoPath) {
		return executeUpdateDirect(repoPath, targetBinary)
	}
	scriptPath, err := writeUpdateScript(repoPath, targetBinary)
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
func executeUpdateDirect(repoPath, targetBinary string) error {
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

	fmt.Println("🔨 Building...")
	outputPath := binaryOutputPath(repoPath)
	if targetBinary != "" {
		outputPath = targetBinary
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return apperror.Wrap("cannot create target directory", err)
	}

	buildCmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", outputPath, ".")
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
func writeUpdateScript(repoPath, targetBinary string) (string, error) {
	script := buildUpdateScriptContent(repoPath, targetBinary)

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
func buildUpdateScriptContent(repoPath, targetBinary string) string {
	repoPath = powerShellString(repoPath)
	targetBinary = powerShellString(targetBinary)

	return fmt.Sprintf(`$ErrorActionPreference = "Stop"
$repoPath = "%s"
$targetBinary = "%s"

function Resolve-VersionBinary {
    if ($targetBinary) {
        return $targetBinary
    }

    $movieBin = Get-Command movie -ErrorAction SilentlyContinue
    if ($movieBin -and $movieBin.Source -and (Test-Path $movieBin.Source)) {
        return $movieBin.Source
    }

    return $null
}

# Capture current version
$versionBinary = Resolve-VersionBinary
$oldVersion = "unknown"
if ($versionBinary -and (Test-Path $versionBinary)) {
    $oldVersion = (& $versionBinary version 2>&1) -join " "
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

# Build and deploy from repo root via run.ps1
$runScript = Join-Path $repoPath "run.ps1"
if (-not (Test-Path $runScript)) {
    Write-Host "  run.ps1 not found at $runScript" -ForegroundColor Red
    exit 1
}

$deployArgs = @("-NoPull", "-Update")
if ($targetBinary) {
    $deployDir = Split-Path -Parent $targetBinary
    $targetName = Split-Path -Leaf $targetBinary
    if ($deployDir) {
        $deployArgs += @("-DeployPath", $deployDir)
    }
    if ($targetName) {
        $deployArgs += @("-BinaryNameOverride", $targetName)
    }
    Write-Host "  Deploy target: $targetBinary" -ForegroundColor Gray
}
& $runScript @deployArgs

# Compare versions from the original target binary
$versionBinary = Resolve-VersionBinary
$newVersion = "unknown"
if ($versionBinary -and (Test-Path $versionBinary)) {
    $newVersion = (& $versionBinary version 2>&1) -join " "
}

Write-Host ""
if ($oldVersion -eq $newVersion) {
    Write-Host "  WARNING: Version unchanged after update" -ForegroundColor Yellow
    Write-Host "  Was version/info.go bumped?" -ForegroundColor Yellow
} else {
    Write-Host "  Updated: $oldVersion -> $newVersion" -ForegroundColor Green
}

# Show changelog from the updated target binary
if ($versionBinary -and (Test-Path $versionBinary)) {
    Write-Host ""
    $clOutput = & $versionBinary changelog --latest 2>&1
    foreach ($cl in $clOutput) { Write-Host "  $cl" }
}

# Auto-cleanup
if ($versionBinary -and (Test-Path $versionBinary)) {
    & $versionBinary update-cleanup 2>&1 | Out-Null
}
`, repoPath, targetBinary)
}

func powerShellString(value string) string {
	value = strings.ReplaceAll(value, "`", "``")
	value = strings.ReplaceAll(value, "\"", "`\"")
	return value
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
