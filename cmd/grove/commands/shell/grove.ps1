# Grove shell integration for PowerShell
# Wraps grove to enable 'grove switch' and 'grove add --switch' to change directories
function grove {
    try {
        $env:GROVE_SHELL = "1"

        if ($args.Count -gt 0 -and $args[0] -eq "switch") {
            if ($args.Count -gt 1 -and $args[1] -eq "-") {
                if (-not $env:GROVE_PREV_WORKTREE -or -not (Test-Path -LiteralPath $env:GROVE_PREV_WORKTREE -PathType Container)) {
                    $global:LASTEXITCODE = 1
                    Write-Error "no previous worktree"
                    return
                }

                $target = $env:GROVE_PREV_WORKTREE
                $global:LASTEXITCODE = 0
            } else {
                $target = & grove.exe switch @($args | Select-Object -Skip 1)
            }

            if ($LASTEXITCODE -eq 0 -and $target -and (Test-Path -LiteralPath $target -PathType Container)) {
                $env:GROVE_PREV_WORKTREE = (Get-Location).Path
                Set-Location -LiteralPath $target
            } else {
                if ($target) { Write-Output $target }
                return $LASTEXITCODE
            }
        } elseif ($args.Count -gt 0 -and $args[0] -eq "add" -and ($args -contains "-s" -or $args -contains "--switch")) {
            $target = & grove.exe add @($args | Select-Object -Skip 1)
            if ($LASTEXITCODE -eq 0 -and $target -and (Test-Path -LiteralPath $target -PathType Container)) {
                $env:GROVE_PREV_WORKTREE = (Get-Location).Path
                Set-Location -LiteralPath $target
            } else {
                if ($target) { Write-Output $target }
                return $LASTEXITCODE
            }
        } else {
            & grove.exe @args
        }
    } finally {
        Remove-Item Env:GROVE_SHELL
    }
}
