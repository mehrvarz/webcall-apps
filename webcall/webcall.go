// WebCall Apps Copyright 2023 timur.mobi. All rights reserved.
package main

import (
	"log"
	"fmt"
	"flag"
	"time"
	"os"
	"os/signal"
	"os/exec"
	"strings"
	"encoding/json"
	"gopkg.in/ini.v1"
	"github.com/zserge/lorca"
	"github.com/hashicorp/logutils"
)

// Cookie struct
type Cookie struct {
	Domain  string  `json:"domain"`
	Name    string  `json:"name"`
	Value   string  `json:"value"`
	Expires float64 `json:"expires"`
}

var	builddate string
var	codetag string
var version = flag.Bool("version", false, "version info")
var bg = flag.Bool("B", false, "backgound")
var logflag = flag.Bool("L", false, "logflag")

func main() {
	flag.Parse()
	if *version {
		if codetag!="" {
			fmt.Printf("version %s\n",codetag)
		}
		fmt.Printf("builddate %s\n",builddate)
		return
	}

	logfilter := &logutils.LevelFilter{
		Levels: []logutils.LogLevel{"INFO", "ERROR"},
		MinLevel: logutils.LogLevel("ERROR"),
		Writer: os.Stderr,
	}
	if *logflag {
		logfilter = &logutils.LevelFilter{
			Levels: []logutils.LogLevel{"INFO", "ERROR"},
			MinLevel: logutils.LogLevel("INFO"),
			Writer: os.Stderr,
		}
	}
	log.SetOutput(logfilter)

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("[ERROR] UserHomeDir err=%v\n",err)
	}

	domain := "timur.mobi"
	calleeId := ""
	calleeUrl := ""

	if apptype=="callee" {
		// callee mode: read config from local ini file
		configFileName := "webcall.ini"
		log.Printf("[INFO] try local config file %s\n",configFileName)
		configIni, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true,},configFileName)
		if err != nil {
			// read config from home ini file
			configFileName = "~/.webcall.ini"
			configFile := strings.Replace(configFileName,"~",homedir,1)
			log.Printf("[INFO] try home config file %s\n",configFile)
			configIni, err = ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true,},configFile)
			if err != nil {
				configIni = nil
				log.Printf("[INFO] no config file found err=%v\n", configFile, err)
			}
		}
		// read cfg from selected file
		if configIni!=nil {
			log.Printf("[INFO] read config file...\n")
			config := readIniString(configIni, "arg", "", "")
			if config!="" {
				toks := strings.Split(config, " ")

				if strings.HasPrefix(toks[0],"https://") {
					calleeUrl = toks[0]
					log.Printf("[INFO] cfg calleeUrl=%s\n",calleeUrl)
				} else {
					if len(toks)>1 {
						domain = toks[0]
						calleeId = toks[1]
						log.Printf("[INFO] cfg domain=%s calleeId=%s\n",domain,calleeId)
					} else {
						calleeId = toks[0]
						log.Printf("[INFO] cfg calleeId=%s\n",calleeId)
					}
					if domain!="" && calleeId!="" {
						calleeUrl = "https://"+domain+"/"+apptype+"/"+calleeId
					}
				}
			}
		}
	}

	// override cfg via commandline args
	if len(flag.Args())>0 {
		if strings.HasPrefix(flag.Arg(0),"https://") {
			calleeUrl = flag.Arg(0)
			log.Printf("[INFO] calleeUrl=%s\n",calleeUrl)
		} else {
			if len(flag.Args())>1 {
				domain = flag.Arg(0)
				calleeId = flag.Arg(1)
				log.Printf("[INFO] domain=%s calleeId=%s\n",domain,calleeId)
			} else {
				// support "calleeId@domain"
				idxAt := strings.Index(flag.Arg(0),"@")
				if idxAt>=0 {
					calleeId = flag.Arg(0)[:idxAt]
					domain = flag.Arg(0)[idxAt+1:]
				} else {
					calleeId = flag.Arg(0)
				}
				log.Printf("[INFO] calleeId=%s\n",calleeId)
			}
			if domain!="" /*&& calleeId!=""*/ {
				calleeUrl = "https://"+domain+"/"+apptype+"/"+calleeId
			}
		}
	}

	if calleeId=="" && apptype=="callee" {
		log.Fatalf("[ERROR] missing calleeId (apptype==callee)\n")
	}
	if domain=="" {
		log.Fatalf("[ERROR] missing domain name (calleeId=%s)\n",calleeId)
	}

	// daemonize this process (by spawning a bg-child)
	if !*bg {
		launch := os.Args[0]+" -B "+calleeUrl
		if *logflag	{
			launch = os.Args[0]+" -B -L "+calleeUrl
		}
		log.Printf("[INFO] launch %s\n",launch)
	    cmd := exec.Command(os.Args[0], "-B", calleeUrl)
		err := cmd.Start()
		if err!=nil {
			log.Fatalf("[ERROR] exec err=%v\n",err)
		}
	    return
	}

	// this is the daemonized process
	args := []string{} // "--start-fullscreen"
	log.Printf("[INFO] lorca.New args=%v\n",args)
	//webcalldir := ""
	webcalldir := homedir+"/.webcall/"+apptype+"/"
	ui, err := lorca.New("", webcalldir, 480, 520)
	if err != nil {
		// most likely: cannot find /usr/lib/chromium/chromium
		log.Fatalf("[ERROR] lorca.New ui=%v err=%v\n",ui,err)
		// TODO empty chrome window may stay open
	}
	defer ui.Close()

	if apptype=="user" {
		cookieHuman := Cookie{"timur.mobi","webcalluser","human",float64(time.Now().Unix()+500000)}
		res := map[string]interface{}{}
		raw, _ := json.Marshal(cookieHuman)
		//log.Printf("[INFO] cookie raw (%v)\n",raw)
		_ = json.Unmarshal(raw, &res)
		//log.Printf("[INFO] cookie res (%v)\n",res)
		ui.Send("Network.setCookie", res)
	}

	log.Println("[INFO] ui.Bind('start')...")
	// A simple way to know when UI is ready (uses body.onload event in JS)
	ui.Bind("start", func() {
		log.Println("[INFO] UI is ready")
	})

	// Create and bind Go object to the UI
	//c := &counter{}
	//ui.Bind("counterAdd", c.Add)
	//ui.Bind("counterValue", c.Value)

	// Load HTML.
	// You may also use `data:text/html,<base64>` approach to load initial HTML,
	// e.g: ui.Load("data:text/html," + url.PathEscape(html))
	/*
	log.Println("net.Listen...")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	go http.Serve(ln, http.FileServer(FS))
	*/

	log.Printf("[INFO] ui.Load...")
	ui.Load(calleeUrl)

	/*
	// You may use console.log to debug your JS code, it will be printed via
	// log.Println(). Also exceptions are printed in a similar manner.
	ui.Eval(`
		console.log("Hello, world!");
		console.log('Multiple values:', [1, false, {"x":5}]);
	`)
	*/

	// Wait until the interrupt signal arrives or browser window is closed
	sigc := make(chan os.Signal)
	signal.Notify(sigc, os.Interrupt)
	select {
	case <-sigc:
	case <-ui.Done():
	}

	log.Println("[INFO] exiting...")
}
