#!/bin/bash

startUpdateAptUbuntu () { #For distros that use apt. *only* needed for the package repos - *DO NOT USE FOR CD REPOS*

	#Assumes $1 is the distribution name (as used for temporary and main index) and $2 is the mirror source
	LOCKOUT=/home/plug/mirror-locks/$1
	EXECUTESTRING="touch $LOCKOUT; rsync --archive --verbose --times --links --hard-links --exclude "Packages*" --exclude "Contents*" --exclude "Sources*" --exclude "Release*" --exclude "InRelease" --delay-updates --log-file /home/plug/rsync-logs/$1-stage1.log --safe-links $2 /mirror/mirror/$1/; rsync --archive --verbose --times --links --hard-links --delete --delete-after --log-file /home/plug/rsync-logs/$1-stage2.log --safe-links $2 /mirror/mirror/$1/; date -u > /mirror/mirror/$1/project/trace/$(hostname -f); rm $LOCKOUT"
	
	if [ -f "$LOCKOUT" ]; then
		echo "LOCKOUT FILE FOUND FOR $1 ON $(date), NOT UPDATING" >> /home/plug/mirror-errorlog
	else
		rm /home/plug/rsync-logs/$1-stage1.log
		rm /home/plug/rsync-logs/$1-stage2.log
		screen -dmS $1 -m bash -c "$EXECUTESTRING"
		echo "Ran update for $1 ON $(date)" >> /home/plug/mirror-updatelog
		sleep 21600
	fi

 }


startUpdateHTTP () { #Mirring iso only files

	#Assumes $1 is the distribution name (as used for temporary and main index) and $2 is the mirror source
	LOCKOUT=/home/plug/mirror-locks/$1
	EXECUTESTRING="touch $LOCKOUT; cd /mirror/wget-sync/; wget -m -np -w 1 -c -e robots=off -R "index.html*" -R "robots.txt" $2; rm $LOCKOUT"
	
	if [ -f "$LOCKOUT" ]; then
		echo "LOCKOUT FILE FOUND FOR $1 ON $(date), NOT UPDATING" >> /home/plug/mirror-errorlog
	else
		screen -dmS $1 -m bash -c "$EXECUTESTRING"
		echo "Ran update for $1 ON $(date)" >> /home/plug/mirror-updatelog
		sleep 21600
	fi

}

startUpdate () {

	#Assumes $1 is the distribution name (as used for temporary and main index) and $2 is the mirror source
	LOCKOUT=/home/plug/mirror-locks/$1
	EXECUTESTRING="touch $LOCKOUT; rsync --archive --verbose --times --links --hard-links --delete --delete-after --delay-updates --log-file /home/plug/rsync-logs/$1.log --safe-links $2 /mirror/mirror/$1/; rm $LOCKOUT" 
	
	if [ -f "$LOCKOUT" ]; then
		echo "LOCKOUT FILE FOUND FOR $1 ON $(date), NOT UPDATING" >> /home/plug/mirror-errorlog
	else
		rm /home/plug/rsync-logs/$1.log
		screen -dmS $1 -m bash -c "$EXECUTESTRING"
		echo "Ran update for $1 ON $(date)" >> /home/plug/mirror-updatelog
		sleep 21600
	fi

}


#Raspbian/Noobs images
startUpdateHTTP "noobs_lite" "http://downloads.raspberrypi.org/NOOBS_lite/images/"
startUpdateHTTP "noobs" "http://downloads.raspberrypi.org/NOOBS/images/"
startUpdateHTTP "raspbian_images" "http://downloads.raspberrypi.org/raspbian/images/"
startUpdateHTTP "raspbian_full" "http://downloads.raspberrypi.org/raspbian_full/images/"
startUpdateHTTP "raspbian_lite" "http://downloads.raspberrypi.org/raspbian_lite/images/"

#Debian Archive
startUpdate "debian-archive" "archive.debian.org::debian-archive"
startUpdate "debian-cd-archive" "cdimage.debian.org::cdimage/archive/"

#Ubuntu Archive
startUpdateAptUbuntu "ubuntu-archive" "archive.ubuntu.com::ubuntu"
