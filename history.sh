export HISTORY_SESSION=$(histowe session)
export PS0='$(HISTTIMEFORMAT='' history 1 | cut -f 4- -d" " | histowe track "${HISTORY_SESSION}")'
unset HISTFILE
history -c
