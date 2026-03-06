#!/usr/bin/env zsh
# terminal-track zsh hook
# This file is sourced by .zshrc to transparently record commands.

# Avoid double-loading
[[ -n "$_TT_LOADED" ]] && return
export _TT_LOADED=1

# Generate a unique session ID for this shell instance (UUID-like)
if command -v uuidgen &>/dev/null; then
    export _TT_SESSION_ID="$(uuidgen)"
else
    export _TT_SESSION_ID="$(date +%s)-$$-${RANDOM}${RANDOM}${RANDOM}"
fi

# Path to the tt binary — resolve once
_TT_BIN="${_TT_BIN:-$(command -v tt 2>/dev/null)}"
if [[ -z "$_TT_BIN" ]]; then
    return
fi

# Capture terminal context once per session (these don't change within a shell)
typeset -g _tt_tty="$(tty 2>/dev/null)"
typeset -g _tt_terminal="${TERM_PROGRAM:-unknown}"
typeset -g _tt_tmux_pane="${TMUX_PANE:-}"

# Temporary state for capturing command and timing
typeset -g _tt_cmd=""
typeset -g _tt_dir=""
typeset -g _tt_ts=""

# preexec: runs just before a command is executed
_tt_preexec() {
    _tt_cmd="$1"
    _tt_dir="$PWD"
    _tt_ts="$(date -u +%Y-%m-%dT%H:%M:%S.%NZ 2>/dev/null || date -u +%Y-%m-%dT%H:%M:%SZ)"
}

# precmd: runs after a command finishes, before the next prompt
_tt_precmd() {
    local exit_code=$?

    # Nothing to record if no command was captured
    [[ -z "$_tt_cmd" ]] && return

    # Record in background to avoid slowing down the prompt
    "$_TT_BIN" record \
        --cmd "$_tt_cmd" \
        --dir "$_tt_dir" \
        --exit-code "$exit_code" \
        --session "$_TT_SESSION_ID" \
        --timestamp "$_tt_ts" \
        --tty "$_tt_tty" \
        --terminal "$_tt_terminal" \
        --tmux-pane "$_tt_tmux_pane" \
        &>/dev/null &!

    # Reset state
    _tt_cmd=""
    _tt_dir=""
    _tt_ts=""
}

# Register hooks
autoload -Uz add-zsh-hook
add-zsh-hook preexec _tt_preexec
add-zsh-hook precmd  _tt_precmd
