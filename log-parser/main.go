package main

import (
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

var (
	logpath       string = "/var/log/nginx/access.log" //Log file path we are reading from
	delay         int    = 300                         //Delay for how long it will be before the daemon runs again. Can be used as the "resolution" of data. This should match what prometheus is configured to query at.
	distributions        = map[string]bool{            //A map of the distributions we actually mirror. If it isn't here, it will be discarded as background traffic.
		"adelie":            true,
		"alpine":            true,
		"archlinux":         true,
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

	downloadbytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "downloaded_bytes",
		Help: "How many bytes a given distribution has served",
	},
		[]string{"distro"},
	)

	downloadcount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "downloaded_count",
		Help: "How many downloads a given distro has had",
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
	// const layout = "Jan 2, 2006 at 3:04pm (MST)"
	const layout = "02/Jan/2006:15:04:05 -0700"
	//Probably need to put a time filter on this
	logfile, err := os.ReadFile(logpath)
	herr(err, logfile)
	logentries := strings.Split(string(logfile), "\n") //Split the log file into individual lines
	// events := make([]logsevent, 0) //This isn't needed right now but might be handy in the future
	totalbyproject := make(map[string]float64)
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
			downloadcount.WithLabelValues(k.project).Inc()
			// events = append(events, k) //This isn't needed right now but might be handy in the future
		}
	}
	for i, k := range totalbyproject { //Add what we just learned to the metrics instance
		downloadbytes.WithLabelValues(i).Add(k) //Not an issue so long as we refresh the log based on our own internal timer
	}
	// fmt.Println("Running")
	// fmt.Println(totalbyproject)
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
	http.ListenAndServe(":9101", nil)
}