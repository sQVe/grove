# shellcheck shell=sh
# Grove shell integration for POSIX sh
# Wraps grove to enable 'grove switch' and 'grove add --switch' to change directories
grove() {
  case "$1" in
    switch)
      shift
      _grove_target="$(command grove switch "$@")"
      _grove_exit=$?
      if [ "${_grove_exit}" -eq 0 ] && [ -d "${_grove_target}" ]; then
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
        _grove_target="$(command grove add "$@")"
        _grove_exit=$?
        if [ "${_grove_exit}" -eq 0 ] && [ -d "${_grove_target}" ]; then
          cd "${_grove_target}" || return 1
        else
          [ -n "${_grove_target}" ] && printf '%s\n' "${_grove_target}"
          return "${_grove_exit}"
        fi
      else
        command grove "$@"
      fi
      ;;
    *)
      command grove "$@"
      ;;
  esac
}
