#!/bin/bash

startUpdate () {

	#Assumes $1 is the distribution name (as used for temporary and main index) and $2 is the mirror source
	LOCKOUT=/home/plug/mirror-locks/$1
	EXECUTESTRING="touch $LOCKOUT; rsync --archive --temp-dir /mirror/tmp/$1/ --verbose --times --links --hard-links --delete --delete-after --delay-updates --log-file /home/plug/rsync-logs/$1.log --safe-links $2 /mirror/mirror/$1/; rm $LOCKOUT" 
	
	if [ -f "$LOCKOUT" ]; then
		echo "LOCKOUT FILE FOUND FOR $1 ON $(date), NOT UPDATING" >> /home/plug/mirror-errorlog
	else
		rm /home/plug/rsync-logs/$1.log
		screen -dmS $1 -m bash -c "$EXECUTESTRING"
		echo "Ran update for $1 ON $(date)" >> /home/plug/mirror-updatelog
		sleep 5
	fi

}

#ArchLinux
startUpdate "archlinux" "mirrors.kernel.org::archlinux"
