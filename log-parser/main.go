package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

//Using the nginx log file, generate prometheus-readable output that can be used to track downloads of differing distributions.
//Also has some functionality to track rsync state strapped on

var (
	listen_addr   string = ""                          //Where we should listen. Leave blank for everything.
	listen_port   string = "9101"                      //What port we should listen on.
	logpath       string = "/var/log/nginx/access.log" //Log file path we are reading from for nginx
	rsynclogdir   string = "/home/plug/rsync-logs/"    //Log file path for rsync logs
	lockfilesdir  string = "/home/plug/mirror-locks/"  //Lock file path
	oldlogseconds int    = 777600                      //How many seconds it needs to be before we consider a log file stale and mark it as erroring. Set to 9 days due to length of some distributions.
	delay         int    = 300                         //Delay for how long it will be before the daemon runs again. Can be used as the "resolution" of data. This should match what prometheus is configured to query at.
	distributions        = map[string]bool{            //A map of the distributions we actually mirror. If it isn't here, it will be discarded as background traffic.
		"adelie":            true,
		"alpine":            true,
		"almalinux":         true,
		"archlinux":         true,
		"blender":           true,
		"cbsd":              true,
		"cbsd-cloud":        true,
		"cbsd-iso":          true,
		"centos":            true,
		"debian":            true,
		"debian-archive":    true,
		"debian-cd":         true,
		"debian-cd-archive": true,
		"fdroid":            true,
		"info.html":         true,
		"manjaro":           true,
		"mint":              true,
		"mint-images":       true,
		"openbsd":           true,
		"opensuse":          true,
		"osdn":              true,
		"pub":               true, //OpenBSD, just again, because historical reasons. Could probably write an alias exception but I don't think we push enough traffic to justify it right now.
		"qubes":             true,
		"raspbian":          true,
		"raspbian-images":   true,
		"rocky":             true,
		"slackware":         true,
		"tails":             true,
		"termux":            true,
		"ubuntu":            true,
		"ubuntu-archive":    true,
		"ubuntu-cd":         true,
		"ubuntu-cloud":      true,
		"ubuntu-ports":      true,
		"ubuntu-releases":   true,
		"vlc":               true,
	}

	ignoredrsyncerrors = map[string]bool{
		"some files vanished before they could be transferred": true,
		"some files/attrs were not transferred":                true,
	}

	downloadbytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mirror_downloaded_bytes",
		Help: "How many bytes a given distribution has served",
	},
		[]string{"distro"},
	)

	downloadcount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mirror_downloaded_count",
		Help: "How many downloads a given distro has had",
	},
		[]string{"distro"},
	)

	lockfilesg = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mirror_distro_locked",
		Help: "Whether or not a given distribution is locked from receiving updates",
	},
		[]string{"distro"},
	)

	rsynclogsg = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mirror_distro_rsync_errors",
		Help: "Whether or not a given distribution is showing rsync errors",
	},
		[]string{"distro"},
	)

	rsynclogsabnornal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mirror_distro_rsync_abonormal",
		Help: "Whether or not a given distribution is showing abnormal rsync behaivor",
	},
		[]string{"distro"},
	)
)

func herr(err error, bodydata interface{}) {
	if err != nil {
		logrus.Error(bodydata, err)
	}
}

type logsevent struct {
	//A bunch of items are commented out. Left here in case they are needed, but not run in the meanwhile to decrease memory usage.
	// remote_addr string
	// remote_user     string
	time_local time.Time
	request    string
	project    string
	// status          int
	body_bytes_sent float64
	// http_referrer   string
	http_user_agent string
}

func gatherMetricsFromLog() {
	//Start by gathering logs from nginx
	// const layout = "Jan 2, 2006 at 3:04pm (MST)"
	const layout = "02/Jan/2006:15:04:05 -0700"
	//Probably need to put a time filter on this
	logfile, err := os.ReadFile(logpath)
	herr(err, logfile)
	logentries := strings.Split(string(logfile), "\n") //Split the log file into individual lines
	// events := make([]logsevent, 0) //This isn't needed right now but might be handy in the future
	totalbyproject := make(map[string]float64)
	totaldownloads := make(map[string]float64)
	currenttime := time.Now()
	starttime := currenttime.Add(time.Second * time.Duration(-delay))
	for i := range logentries { //Go grab the individual log lines
		t := strings.Split(logentries[i], " ")
		if len(t) >= 10 {
			var (
				k logsevent
				z int
			)
			// k.remote_addr = t[z]
			//1 will always be a -
			z = z + 2
			// k.remote_user = t[z]
			z++
			// k.time_local = t[z] + " " + t[z+1]
			k.time_local, _ = time.Parse(layout, t[z][1:]+" "+t[z+1][:len(t[z+1])-1])
			if !k.time_local.After(starttime) || k.time_local.After(currenttime) { //If the log entry is outside of our interest time, discard this entry
				continue
			}
			z = z + 2
			k.request = t[z+1] //This would be just a z if we wanted the whole thing
			l := strings.Split(k.request, "/")
			if len(l) < 2 { //If the request itself isn't long enough to be an actual request
				continue
			}
			if !distributions[l[1]] { //Break off if the request is some garbage that isn't actually a distribution
				continue
			}
			k.project = l[1]
			z++
			if (t[z-1][(len(t[z-1]) - 1):]) != `"` { //Normally this will always match. If someone is doing something nasty (ex a request that is just garbage data), it will fail.
			url:
				for z < 100 { //Set a maximum after which we will break out to avoid some weird user
					// k.request = k.request + " " + t[z]     Taken out as we don't want the whole thing, we just want the URL
					z++
					if (t[z-1][(len(t[z-1]) - 1):]) == `"` {
						break url
					}
				}
			}
			// k.status, _ = strconv.Atoi(t[z])
			z++
			k.body_bytes_sent, _ = strconv.ParseFloat(t[z], 64)
			z++
			// k.http_referrer = t[z]
			z++
			if (t[z-1][(len(t[z-1]) - 1):]) != `"` {
			host:
				for z < 100 {
					// k.http_referrer = k.http_referrer + " " + t[z]
					z++
					if (t[z-1][(len(t[z-1]) - 1):]) == `"` {
						break host
					}
				}
			}
			k.http_user_agent = t[z]
			z++
			if (t[z-1][(len(t[z-1]) - 1):]) != `"` {
			agent:
				for z < 100 {
					k.http_user_agent = k.http_user_agent + " " + t[z]
					z++
					if (t[z-1][(len(t[z-1]) - 1):]) == `"` {
						break agent
					}
				}
			}
			totalbyproject[k.project] = totalbyproject[k.project] + k.body_bytes_sent
			totaldownloads[k.project]++
			// events = append(events, k) //This isn't needed right now but might be handy in the future
		}
	}
	for i, k := range totalbyproject { //Add what we just learned to the metrics instance
		downloadbytes.WithLabelValues(i).Set(k) //Not an issue so long as we refresh the log based on our own internal timer
		downloadcount.WithLabelValues(i).Set(totaldownloads[i])
	}
	// fmt.Println("Running")
	// fmt.Println(totalbyproject)

	//Proceed to log data from rsync
	rsynclogs, err := ioutil.ReadDir(rsynclogdir)
	herr(err, lockfilesdir)
	lockfiles, err := ioutil.ReadDir(lockfilesdir)
	herr(err, lockfilesdir)
	oldestlogfile := time.Now().Add(time.Second * time.Duration(-oldlogseconds))
	lockfilesg.Reset()
	rsynclogsg.Reset()
	rsynclogsabnornal.Reset()

	rsyncerrorline := 4 //How many lines back we will check for an rsync error

	for i := range rsynclogs {
		f, err := os.Open(rsynclogdir + rsynclogs[i].Name())
		herr(err, rsynclogdir+rsynclogs[i].Name())
		defer f.Close()
		line := ""
		linebyte := make([]byte, 0)
		var position int64 = 0
		size := int64(rsynclogs[i].Size())
		char := make([]byte, 1)
		linesback := 0
		if size != 0 { //Hunt for a rsync error on the second to last line of the file
		errorinlog:
			for {
				position -= 1
				f.Seek(position, io.SeekEnd)
				f.Read(char)
				if char[0] == 10 {
					if linesback == rsyncerrorline { //If we have gone back more than rsyncerrorline number of lines
						break
					} else { //If we are still looking for an error, go look for said error
						line = string(linebyte)
						if strings.Contains(line, "rsync error") { //If the log file contains the evil words
							actuallyanerror := true
							for j, _ := range ignoredrsyncerrors {
								if strings.Contains(line, j) {
									actuallyanerror = false
								}
							}
							if actuallyanerror {
								logrus.Info("Log for ", rsynclogs[i].Name(), " contains what looks to be an error, erroring")
								rsynclogsg.WithLabelValues(rsynclogs[i].Name()).Set(1)
								break errorinlog
							}
						}
						linebyte = nil
						linesback++
					}
				}
				linebyte = append(char, linebyte...)
				if position == size { //If we hit the end of the file
					break
				}
			}
		}

		if size == 0 { //If the log file is of size 0
			logrus.Info("Log size of ", rsynclogs[i].Name(), " is 0, marking anomoly")
			rsynclogsabnornal.WithLabelValues(rsynclogs[i].Name()).Set(1)
		} else if !rsynclogs[i].ModTime().After(oldestlogfile) { //If the log file is older than the oldest log file
			logrus.Info("Log for ", rsynclogs[i].Name(), " is older than max age, marking anomoly")
			rsynclogsabnornal.WithLabelValues(rsynclogs[i].Name()).Set(1)
		}
	}
	for i := range lockfiles { //Ensure distributions with a lock file are represented
		lockfilesg.WithLabelValues(lockfiles[i].Name()).Set(1)
	}

}

func updateLogData() { //Update log data every tick rate. While we could do this on-demand, the CPU overhead is negligible and allows us to track one less state.
	for {
		gatherMetricsFromLog()
		time.Sleep(time.Duration(delay) * time.Second)
	}
}

func main() {
	go updateLogData()
	http.Handle("/metrics", promhttp.Handler())
	listen := listen_addr + ":" + listen_port
	http.ListenAndServe(listen, nil)
}
