# Grove shell integration for fish
# Wraps grove to enable 'grove switch' and 'grove add --switch' to change directories
function grove
    if test (count $argv) -gt 0 -a "$argv[1]" = "switch"
        set -l target
        set -l exit_code 0
        if test (count $argv) -gt 1; and test "$argv[2]" = "-"
            if not set -q GROVE_PREV_WORKTREE; or not test -d "$GROVE_PREV_WORKTREE"
                printf '%s\n' 'no previous worktree' >&2
                return 1
            end

            set target "$GROVE_PREV_WORKTREE"
        else
            set target (GROVE_SHELL=1 command grove switch $argv[2..])
            set exit_code $status
        end
        if test $exit_code -eq 0 -a -d "$target"
            set -gx GROVE_PREV_WORKTREE "$PWD"
            cd "$target"
        else
            test -n "$target" && printf '%s\n' "$target"
            return $exit_code
        end
    else if test (count $argv) -gt 0 -a "$argv[1]" = "add"
        if contains -- -s $argv[2..]; or contains -- --switch $argv[2..]
            set -l target (GROVE_SHELL=1 command grove add $argv[2..])
            set -l exit_code $status
            if test $exit_code -eq 0 -a -d "$target"
                set -gx GROVE_PREV_WORKTREE "$PWD"
                cd "$target"
            else
                test -n "$target" && printf '%s\n' "$target"
                return $exit_code
            end
        else
            GROVE_SHELL=1 command grove $argv
        end
    else
        GROVE_SHELL=1 command grove $argv
    end
end
