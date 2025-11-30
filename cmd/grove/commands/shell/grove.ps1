# Grove shell integration for PowerShell
# Wraps grove to enable 'grove switch' and 'grove add --switch' to change directories
function grove {
    if ($args.Count -gt 0 -and $args[0] -eq "switch") {
        $target = & grove.exe switch @($args | Select-Object -Skip 1)
        if ($LASTEXITCODE -eq 0 -and $target -and (Test-Path $target -PathType Container)) {
            Set-Location $target
        } else {
            if ($target) { Write-Output $target }
            return $LASTEXITCODE
        }
    } elseif ($args.Count -gt 0 -and $args[0] -eq "add" -and ($args -contains "-s" -or $args -contains "--switch")) {
        $target = & grove.exe add @($args | Select-Object -Skip 1)
        if ($LASTEXITCODE -eq 0 -and $target -and (Test-Path $target -PathType Container)) {
            Set-Location $target
        } else {
            if ($target) { Write-Output $target }
            return $LASTEXITCODE
        }
    } else {
        & grove.exe @args
    }
}
