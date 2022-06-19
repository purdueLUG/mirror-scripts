#!/bin/bash

startUpdate () {

	#Assumes $1 is the distribution name (as used for temporary and main index) and $2 is the mirror source
	LOCKOUT=/home/plug/mirror-locks/$1
	EXECUTESTRING="touch $LOCKOUT; rsync --archive --verbose --times --links --hard-links --delete --delete-after --delay-updates --log-file /home/plug/rsync-logs/$1.log --temp-dir /mirror/tmp/$1/ --safe-links $2 /mirror/mirror/$1/; rm $LOCKOUT" 
	
	if [ -f "$LOCKOUT" ]; then
		echo "LOCKOUT FILE FOUND FOR $1 ON $(date), NOT UPDATING" >> /home/plug/mirror-errorlog
	else
		rm /home/plug/rsync-logs/$1.log
		screen -dmS $1 -m bash -c "$EXECUTESTRING"
		echo "Ran update for $1 ON $(date)" >> /home/plug/mirror-updatelog
		sleep 1200
	fi

}

startUpdateFDROID () {

        #Assumes $1 is the distribution name (as used for temporary and main index) and $2 is the mirror source
        LOCKOUT=/home/plug/mirror-locks/$1
        #EXECUTESTRING="touch $LOCKOUT; RSYNC_PASSWORD=<redacted> rsync --archive --verbose --times --links --hard-links --delete --delete-after --delay-updates --log-file /home/plug/rsync-logs/$1.log --temp-dir /mirror/tmp/$1/ --safe-links $2 /mirror/mirror/fdroid/$1/; rm $LOCKOUT" 
        EXECUTESTRING="touch $LOCKOUT; rsync --archive --verbose --times --links --hard-links --delete --delete-after --delay-updates --log-file /home/plug/rsync-logs/$1.log --temp-dir /mirror/tmp/$1/ --safe-links $2 /mirror/mirror/$1/; rm $LOCKOUT" 

        if [ -f "$LOCKOUT" ]; then
                echo "LOCKOUT FILE FOUND FOR $1 ON $(date), NOT UPDATING" >> /home/plug/mirror-errorlog
        else
                rm /home/plug/rsync-logs/$1.log
                screen -dmS $1 -m bash -c "$EXECUTESTRING"
                echo "Ran update for $1 ON $(date)" >> /home/plug/mirror-updatelog
                sleep 1200
        fi

}


startUpdateApt () { #For distros that use apt. *only* needed for the package repos - *DO NOT USE FOR CD REPOS*

	#Assumes $1 is the distribution name (as used for temporary and main index) and $2 is the mirror source
	LOCKOUT=/home/plug/mirror-locks/$1
	EXECUTESTRING="touch $LOCKOUT; rsync --archive --verbose --times --links --hard-links --exclude "Packages*" --exclude "Contents*" --exclude "Sources*" --exclude "Release*" --exclude "InRelease" --temp-dir /mirror/tmp/$1/ --delay-updates --log-file /home/plug/rsync-logs/$1-stage1.log --safe-links $2 /mirror/mirror/$1/; rsync --archive --verbose --times --links --hard-links --delete --delete-after --temp-dir /mirror/tmp/$1/ --log-file /home/plug/rsync-logs/$1-stage2.log --safe-links $2 /mirror/mirror/$1/; rm $LOCKOUT"
	
	if [ -f "$LOCKOUT" ]; then
		echo "LOCKOUT FILE FOUND FOR $1 ON $(date), NOT UPDATING" >> /home/plug/mirror-errorlog
	else
		rm /home/plug/rsync-logs/$1-stage1.log
		rm /home/plug/rsync-logs/$1-stage2.log
		screen -dmS $1 -m bash -c "$EXECUTESTRING"
		echo "Ran update for $1 ON $(date)" >> /home/plug/mirror-updatelog
		sleep 1200
	fi

}

startUpdateAptUbuntu () { #For distros that use apt. *only* needed for the package repos - *DO NOT USE FOR CD REPOS*

	#Assumes $1 is the distribution name (as used for temporary and main index) and $2 is the mirror source
	LOCKOUT=/home/plug/mirror-locks/$1
	EXECUTESTRING="touch $LOCKOUT; rsync --archive --verbose --times --links --hard-links --exclude "Packages*" --exclude "Contents*" --exclude "Sources*" --exclude "Release*" --exclude "InRelease" --delay-updates --log-file /home/plug/rsync-logs/$1-stage1.log --temp-dir /mirror/tmp/$1/ --safe-links $2 /mirror/mirror/$1/; rsync --archive --verbose --times --links --temp-dir /mirror/tmp/$1/ --hard-links --delete --delete-after --log-file /home/plug/rsync-logs/$1-stage2.log --safe-links $2 /mirror/mirror/$1/; date -u > /mirror/mirror/$1/project/trace/$(hostname -f); rm $LOCKOUT"
	
	if [ -f "$LOCKOUT" ]; then
		echo "LOCKOUT FILE FOUND FOR $1 ON $(date), NOT UPDATING" >> /home/plug/mirror-errorlog
	else
		rm /home/plug/rsync-logs/$1-stage1.log
		rm /home/plug/rsync-logs/$1-stage2.log
		screen -dmS $1 -m bash -c "$EXECUTESTRING"
		echo "Ran update for $1 ON $(date)" >> /home/plug/mirror-updatelog
		sleep 1200
	fi

}

startUpdateUbuntuCD () {

	#Assumes $1 is the distribution name (as used for temporary and main index) and $2 is the mirror source
	LOCKOUT=/home/plug/mirror-locks/$1
	EXECUTESTRING="touch $LOCKOUT; rsync --archive --verbose --times --links --hard-links --delete --delete-after --delay-updates --temp-dir /mirror/tmp/$1/ --log-file /home/plug/rsync-logs/$1.log --safe-links $2 /mirror/mirror/$1/; date -u > /mirror/mirror/$1/.trace/$(hostname -f); rm $LOCKOUT" 
	
	if [ -f "$LOCKOUT" ]; then
		echo "LOCKOUT FILE FOUND FOR $1 ON $(date), NOT UPDATING" >> /home/plug/mirror-errorlog
	else
		rm /home/plug/rsync-logs/$1.log
		screen -dmS $1 -m bash -c "$EXECUTESTRING"
		echo "Ran update for $1 ON $(date)" >> /home/plug/mirror-updatelog
		sleep 1200
	fi

}

#Raspbian
startUpdateApt "raspbian" "archive.raspbian.org::archive/raspbian/"

#Debian - may not be top site
screen -dmS debian /home/plug/ftpsync/bin/ftpsync
startUpdate "debian-cd" "cdimage.debian.org::debian-cd"

#Linux Mint
startUpdateApt "mint" "rsync-packages.linuxmint.com::packages"
startUpdate "mint-images" "pub.linuxmint.com::pub"

#ArchLinux
#startUpdate "archlinux" "mirrors.kernel.org::archlinux"

#Ubuntu
#startUpdateAptUbuntu "ubuntu" "archive.ubuntu.com::ubuntu"
screen -dmS ubuntu /home/plug/ftpsync-ubuntu/bin/ftpsync
startUpdateUbuntuCD "ubuntu-cd" "cdimage.ubuntu.com::cdimage"
startUpdate "ubuntu-cloud" "cloud-images.ubuntu.com::cloud-images"
startUpdateApt "ubuntu-ports" "ports.ubuntu.com::ubuntu-ports"
startUpdateUbuntuCD "ubuntu-releases" "releases.ubuntu.com::releases"

#CentOS - NOT TOP SITE
#startUpdate "centos" "mirror.team-cymru.com::CentOS"

#FreeBSD - NOT TOP SITE
#startUpdate "freebsd" "ftp6.us.freebsd.org::FreeBSD"

#Kali - NOT TOP SITE, UNABLE TO ACCESS
#startUpdateApt "kali" "archive.linux.duke.edu::kali"
#startUpdate "kali-images" "archive.linux.duke.edu::kali-images"

#OpenBSD
startUpdate "openbsd" "ftp.usa.openbsd.org::ftp"

#Gentoo
#startUpdate "gentoo" "masterdistfiles.gentoo.org::gentoo"

#OpenSUSE
startUpdate "opensuse" "stage.opensuse.org::opensuse-full-really-everything-including-repositories/opensuse/"

#Tails - NOT TOP SITE?
startUpdate "tails" "mirrors.rsync.tails.boum.org::amnesia-archive/tails/"

#CygWin
#startUpdate "cygwin" "cygwin.com::cygwin-ftp"

#Manjaro - NOT TOP SITE
#startUpdate "manjaro" "repo.manjaro.org::repos"
startUpdate "manjaro" "ftp.halifax.rwth-aachen.de::manjaro"

#Adelie
startUpdate "adelie" "mirrormaster.adelielinux.org::distfiles"

#Void
#startUpdate "void" "alpha.us.repo.voidlinux.org::repo"

#Qubes
startUpdate "qubes" "ftp.qubes-os.org::qubes-mirror"

#Alpine - NOT TOP SITE
startUpdate "alpine" "mirrors.kernel.org::alpine"

#Slackware - NOT TOP SITE
startUpdate "slackware" "slackware.cs.utah.edu::slackware"

#OSDN - whitelisted for top site
startUpdate "osdn" "master.dl.osdn.net::download"
#startUpdate "osdn" "plug-mirror.rcac.purdue.edu::osdn"

#VLC - is top site
startUpdate "vlc" "rsync.videolan.org::videolan-ftp"

#Fdroid
#startUpdateFDROID "repo" "fdroid-mirror@mirror.f-droid.org::repo"
#startUpdateFDROID "archive" "fdroid-mirror@mirror.f-droid.org::archive"
##startUpdateFDROID "fdroid" "ftp.fau.de::fdroid"

#OpenWRT
#startUpdate "openwrt" "downloads.openwrt.org::downloads"

#TDF (LibreOffice)
#startUpdate "tdf" "rsync.documentfoundation.org::tdf-pub"

#CentOS - NOT TOP SITE
startUpdate "centos" "bay.uchicago.edu::CentOS"

#Rocky - NOT TOP SITE
startUpdate "rocky" "mirrors.rit.edu::rocky"

#Termux (not tmux) Android terminal emulator thing
export RSYNC_PASSWORD="<removed>"
startUpdate "termux" "rsync@grimler.se::termux"
unset RSYNC_PASSWORD
