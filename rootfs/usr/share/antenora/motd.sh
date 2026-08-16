#!/usr/bin/env bash
# Antenora MOTD — the crest, the machine, and a line from the Inferno.
set -u

CREST_FILE="/usr/share/antenora/crest.txt"
QUOTES="/usr/share/antenora/quotes.txt"
RED=$'\033[1;31m'
DIM=$'\033[1;30m'
RST=$'\033[0m'

if [ -f "$CREST_FILE" ]; then
    cat "$CREST_FILE"
fi

echo "${RED}Antenora Linux 1.0.0 \"Inferno\" — The Ninth Circle of Speed${RST}"
echo
echo "${DIM}Kernel :${RST} $(uname -r)"
echo "${DIM}Uptime :${RST} $(uptime -p 2>/dev/null | sed 's/^up //')"
echo "${DIM}Memory :${RST} $(free -h | awk '/^Mem:/{print $3 " / " $2}')"
echo "${DIM}Init   :${RST} s6 (systemd-free)"

if [ -f "$QUOTES" ]; then
    quote=$(shuf -n1 "$QUOTES" 2>/dev/null)
    [ -n "$quote" ] && echo && echo "${RED}\"${quote}\"${RST}"
fi
echo
