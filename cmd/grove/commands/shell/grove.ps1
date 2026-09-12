# Grove shell integration for PowerShell
# Changes directories for switch, add --switch, and removal of the current worktree.
function grove {
    $previousGroveShell = $env:GROVE_SHELL
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
        } elseif ($args.Count -gt 0 -and $args[0] -eq "remove") {
            $original = (Get-Location).Path
            $removeArguments = @($args | Select-Object -Skip 1)
            $comparison = [StringComparison]::Ordinal
            if ([IO.Path]::DirectorySeparatorChar -eq '\') {
                $comparison = [StringComparison]::OrdinalIgnoreCase
            }

            $afterTerminator = $false
            foreach ($argument in $removeArguments) {
                if (-not $afterTerminator) {
                    if ("$argument" -eq "--") { $afterTerminator = $true; continue }
                    if ("$argument".StartsWith("-")) { continue }
                }

                $target = & grove.exe switch -- $argument 2>$null
                if ($LASTEXITCODE -eq 0 -and $target) {
                    $target = [IO.Path]::GetFullPath("$target")
                    $current = [IO.Path]::GetFullPath((Get-Location).Path)
                    $prefix = $target + [IO.Path]::DirectorySeparatorChar
                    if ($current.Equals($target, $comparison) -or $current.StartsWith($prefix, $comparison)) {
                        Set-Location -LiteralPath (Split-Path -Parent $target) -ErrorAction Stop
                    }
                }
            }

            & grove.exe remove @removeArguments
            $exitCode = $LASTEXITCODE
            if ((Get-Location).Path -ne $original -and (Test-Path -LiteralPath $original -PathType Container)) {
                $env:GROVE_PREV_WORKTREE = (Get-Location).Path
                Set-Location -LiteralPath $original
            }

            $global:LASTEXITCODE = $exitCode
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
        if ($null -eq $previousGroveShell) {
            Remove-Item Env:GROVE_SHELL -ErrorAction SilentlyContinue
        } else {
            $env:GROVE_SHELL = $previousGroveShell
        }
    }
}
