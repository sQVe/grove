# shellcheck shell=sh
# Grove shell integration for POSIX sh
# Wraps grove to enable 'grove switch' and 'grove add --switch' to change directories
grove() {
  case "$1" in
    switch)
      shift
      if [ "${1-}" = "-" ]; then
        if [ ! -d "${GROVE_PREV_WORKTREE-}" ]; then
          printf '%s\n' 'no previous worktree' >&2
          return 1
        fi

        _grove_target="${GROVE_PREV_WORKTREE}"
        _grove_exit=0
      else
        _grove_target="$(GROVE_SHELL=1 command grove switch "$@")"
        _grove_exit=$?
      fi
      if [ "${_grove_exit}" -eq 0 ] && [ -d "${_grove_target}" ]; then
        export GROVE_PREV_WORKTREE="${PWD}"
        cd "${_grove_target}" || return 1
      else
        [ -n "${_grove_target}" ] && printf '%s\n' "${_grove_target}"
        return "${_grove_exit}"
      fi
      ;;
    add)
      # Check if -s or --switch is in the arguments
      _grove_has_switch=0
      for _grove_arg in "$@"; do
        case "${_grove_arg}" in
          -s | --switch)
            _grove_has_switch=1
            break
            ;;
        esac
      done
      if [ "${_grove_has_switch}" -eq 1 ]; then
        shift
        _grove_target="$(GROVE_SHELL=1 command grove add "$@")"
        _grove_exit=$?
        if [ "${_grove_exit}" -eq 0 ] && [ -d "${_grove_target}" ]; then
          export GROVE_PREV_WORKTREE="${PWD}"
          cd "${_grove_target}" || return 1
        else
          [ -n "${_grove_target}" ] && printf '%s\n' "${_grove_target}"
          return "${_grove_exit}"
        fi
      else
        GROVE_SHELL=1 command grove "$@"
      fi
      ;;
    *)
      GROVE_SHELL=1 command grove "$@"
      ;;
  esac
}
