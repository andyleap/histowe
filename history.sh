export HISTORY_SESSION=$(mktemp -q)
export PS0='$(HISTTIMEFORMAT='' history 1 | histowe track)'
unset HISTFILE
history -c
