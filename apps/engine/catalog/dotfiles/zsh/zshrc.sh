# value: the colours below are the Catppuccin Macchiato palette the Zed and Ghostty
# themes use: ANSI 1=red(#ed8796) 2=green(#a6da95) 3=yellow(#eed49f)
# 4=blue(#8aadf4) 5=magenta(#f5bde6) 6=cyan(#8bd5ca) 7=white(#a5adcb).

# invariant: Homebrew's bins are on PATH before anything below looks for a tool, since
# only a login shell's ~/.zprofile runs `brew shellenv`, and an IDE's or ssh's zsh -ic
# would otherwise find neither the Brew-installed mise nor starship.
if [[ -d /opt/homebrew/bin && ":$PATH:" != *":/opt/homebrew/bin:"* ]]; then
    export PATH="/opt/homebrew/bin:/opt/homebrew/sbin:$PATH"
fi

# invariant: each shell start makes this terminal's id, which `station use` keeps its
# machine under, so a new terminal starts on This Mac.
export STATION_TERMINAL="${$}x${RANDOM}${RANDOM}"

# invariant: a fresh machine without starship still gets a usable, path-only prompt.
if command -v starship &>/dev/null; then
    export STARSHIP_CONFIG="$HOME/.config/starship.toml"
    eval "$(starship init zsh)"
else
    PROMPT="%(?:%{$fg_bold[green]%}➜ :%{$fg_bold[red]%}➜ ) %{$fg[blue]%}%c%{$reset_color%} "
fi

# value: BSD ls colours, with dirs=blue, links=cyan, sockets=red, pipes=yellow, exec=green.
export CLICOLOR=1
export LSCOLORS="ExGxBxDxCxegedabagacad"

# value: the completion menu re-uses the ls colours.
zstyle ':completion:*' list-colors 'di=34:ln=36:so=31:pi=33:ex=32:bd=34;46:cd=34;43:su=30;41:sg=30;46:tw=30;42:ow=30;43'

# value: grep --color, with match=red bold, filename=magenta, line number=green.
export GREP_COLORS='ms=01;31:mc=01;31:sl=:cx=:fn=35:ln=32:bn=32:se=36'

# value: man pages through less, with headings=blue bold, search=yellow bg, emphasis=green.
export LESS_TERMCAP_mb=$'\e[1;31m'
export LESS_TERMCAP_md=$'\e[1;34m'
export LESS_TERMCAP_me=$'\e[0m'
export LESS_TERMCAP_so=$'\e[30;43m'
export LESS_TERMCAP_se=$'\e[0m'
export LESS_TERMCAP_us=$'\e[1;32m'
export LESS_TERMCAP_ue=$'\e[0m'
export LESS=-R
export MANPAGER='less -R --use-color -Dd+r -Du+b'

export FZF_DEFAULT_OPTS="\
--color=bg+:#363a4f,bg:#24273a,spinner:#f4dbd6,hl:#ed8796 \
--color=fg:#cad3f5,header:#ed8796,info:#c6a0f6,pointer:#f4dbd6 \
--color=marker:#f4dbd6,fg+:#cad3f5,prompt:#c6a0f6,hl+:#ed8796"

export BAT_THEME="Catppuccin Macchiato"

export PATH="$HOME/.local/share/bun/bin:$PATH"

export PATH="$HOME/.local/share/go/bin:$PATH"

if [[ -o interactive && "$PWD" == "$HOME" && -d "$HOME/Developer" ]]; then
    cd "$HOME/Developer"
fi

jj() {
    if ! command -v tmux &>/dev/null; then
        echo "tmux not found"
        return 1
    fi

    if ! tmux has-session -t main 2>/dev/null; then
        tmux new-session -ds main
    fi

    if [[ -n "$TMUX" ]]; then
        tmux switch-client -t main
    else
        tmux attach-session -t main
    fi
}

_tmux_tool() {
    local window_name="$1"
    local tool_cmd="$2"

    if ! command -v tmux &>/dev/null; then
        echo "tmux not found"
        return 1
    fi
    if ! command -v "$tool_cmd" &>/dev/null; then
        echo "$tool_cmd not found"
        return 1
    fi

    if [[ -n "$TMUX" ]]; then
        if tmux has-session -t main 2>/dev/null; then
            tmux new-window -t main -n "$window_name" "$tool_cmd"
            tmux switch-client -t main
        else
            tmux new-session -ds main -n "$window_name" "$tool_cmd"
            tmux switch-client -t main
        fi
        return
    fi

    if tmux has-session -t main 2>/dev/null; then
        tmux new-session -t main \; set destroy-unattached on \; new-window -n "$window_name" "$tool_cmd"
        return
    fi

    tmux new-session -s main -n "$window_name" "$tool_cmd"
}

jc() { _tmux_tool "claude" "claude"; }
jo() { _tmux_tool "codex" "codex"; }
jg() { _tmux_tool "gemini" "gemini"; }
