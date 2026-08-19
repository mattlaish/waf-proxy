# git-save-push.ps1
# Commit and push current project changes

Write-Host "Checking repository..." -ForegroundColor Cyan

git status

Write-Host ""
$confirm = Read-Host "Continue with git add, commit, and push? (y/n)"

if ($confirm -ne "y") {
    Write-Host "Cancelled."
    exit
}

$message = Read-Host "Commit message"

if ([string]::IsNullOrWhiteSpace($message)) {
    $message = "Update project files"
}

Write-Host ""
Write-Host "Adding files..." -ForegroundColor Cyan
git add .

Write-Host ""
Write-Host "Committing..." -ForegroundColor Cyan
git commit -m "$message"

if ($LASTEXITCODE -ne 0) {
    Write-Host "Nothing committed or commit failed."
    exit
}

Write-Host ""
Write-Host "Pushing to GitHub..." -ForegroundColor Cyan
git push

Write-Host ""
Write-Host "Done." -ForegroundColor Green

git status