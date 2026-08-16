# Antenora default Zsh — Powerlevel10k, minimal but mighty.

if [[ -r "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh" ]]; then
  source "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh"
fi

export ZSH="${HOME}/.oh-my-zsh"
ZSH_THEME="powerlevel10k/powerlevel10k"

# History — never lose a command in the pit.
HISTFILE=~/.zsh_history
HISTSIZE=100000
SAVEHIST=100000
setopt SHARE_HISTORY HIST_IGNORE_DUPS INC_APPEND_HISTORY

# Completion
autoload -Uz compinit && compinit

# Editors
export EDITOR=nvim
export VISUAL=nvim

# Antenora aliases live in /etc/profile.d, but keep Dante close.
alias d='dante'
alias di='dante install'
alias ds='dante search'
alias du='dante update'

# Antenora MOTD on every login shell.
[[ -f /usr/share/antenora/motd.sh ]] && bash /usr/share/antenora/motd.sh

if [[ -r ~/.p10k.zsh ]]; then
  source ~/.p10k.zsh
fi
