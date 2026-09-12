# shellcheck shell=sh
# Grove shell integration for POSIX sh
# Changes directories for switch, add --switch, and removal of the current worktree.
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
    remove)
      shift
      _grove_original="${PWD}"
      for _grove_arg in "$@"; do
        case "${_grove_arg}" in
          -*) continue ;;
        esac

        if _grove_target="$(GROVE_SHELL=1 command grove switch "${_grove_arg}" 2>/dev/null)"; then
          case "${PWD}" in
            "${_grove_target}" | "${_grove_target}"/*)
              cd "$(dirname "${_grove_target}")" || return 1
              ;;
          esac
        fi
      done

      if GROVE_SHELL=1 command grove remove "$@"; then
        _grove_exit=0
      else
        _grove_exit=$?
        if [ "${PWD}" != "${_grove_original}" ] && [ -d "${_grove_original}" ]; then
          cd "${_grove_original}" || :
        fi
      fi

      return "${_grove_exit}"
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
