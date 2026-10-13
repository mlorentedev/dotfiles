# shellcheck shell=sh
# path.sh: the head of PATH every shell gets, interactive or not (#2013 W2b).
# Sourced by ~/.zshenv (every zsh, so `zsh -c` from an agent or an ssh command),
# the top of ~/.bashrc (bash run by sshd) and the end of ~/.profile (login
# shells). It puts mise's shims first and ~/.local/bin second: the shims run the
# pinned CLIs, and a copy left in ~/.local/bin must not shadow them (doctor's
# "Pinned CLIs (mise)"). Each entry is moved to the front rather than added
# again, so a nested shell neither grows PATH nor keeps a wrong order it
# inherited. POSIX sh: bash, zsh and dash all source it.

_dotf_path_front() {
    _dotf_rest=":$PATH:"
    while :; do
        case "$_dotf_rest" in
            *":$1:"*) _dotf_rest="${_dotf_rest%%:"$1":*}:${_dotf_rest#*:"$1":}" ;;
            *) break ;;
        esac
    done
    _dotf_rest="${_dotf_rest#:}"
    _dotf_rest="${_dotf_rest%:}"
    PATH="$1${_dotf_rest:+:$_dotf_rest}"
}

_dotf_path_front "$HOME/.local/bin"
_dotf_path_front "${MISE_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/mise}/shims"
export PATH
unset _dotf_rest
unset -f _dotf_path_front
