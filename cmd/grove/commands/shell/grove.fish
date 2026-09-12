# Grove shell integration for fish
# Changes directories for switch, add --switch, and removal of the current worktree.
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
    else if test (count $argv) -gt 0 -a "$argv[1]" = "remove"
        set -l original "$PWD"
        for argument in $argv[2..]
            if string match -q -- '-*' "$argument"
                continue
            end

            if set -l target (GROVE_SHELL=1 command grove switch "$argument" 2>/dev/null)
                set -l physical (pwd -P)
                if test "$physical" = "$target"; or string match -qr -- '^'(string escape --style=regex -- "$target/") "$physical"
                    cd (dirname "$target"); or return 1
                end
            end
        end

        GROVE_SHELL=1 command grove remove $argv[2..]
        set -l exit_code $status
        if test $exit_code -ne 0; and test "$PWD" != "$original"; and test -d "$original"
            set -gx GROVE_PREV_WORKTREE "$PWD"
            cd "$original"
        end

        return $exit_code
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
