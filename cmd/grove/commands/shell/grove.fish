# Grove shell integration for fish
# Wraps grove to enable 'grove switch' and 'grove add --switch' to change directories
function grove
    if test (count $argv) -gt 0 -a "$argv[1]" = "switch"
        set -l target (command grove switch $argv[2..])
        set -l exit_code $status
        if test $exit_code -eq 0 -a -d "$target"
            cd "$target"
        else
            test -n "$target" && printf '%s\n' "$target"
            return $exit_code
        end
    else if test (count $argv) -gt 0 -a "$argv[1]" = "add"
        if contains -- -s $argv[2..]; or contains -- --switch $argv[2..]
            set -l target (command grove add $argv[2..])
            set -l exit_code $status
            if test $exit_code -eq 0 -a -d "$target"
                cd "$target"
            else
                test -n "$target" && printf '%s\n' "$target"
                return $exit_code
            end
        else
            command grove $argv
        end
    else
        command grove $argv
    end
end
