package main

import (
	"log"
	"fmt"
	"flag"
//	"time"
//	"net"
//	"net/http"
	"os"
	"os/signal"
	"os/exec"
//	"syscall"
//	"runtime"
//	"sync"
	"strings"
	"gopkg.in/ini.v1"
	"github.com/zserge/lorca"
)

/*
// Go types that are bound to the UI must be thread-safe, because each binding
// is executed in its own goroutine. In this simple case we may use atomic
// operations, but for more complex cases one should use proper synchronization.
type counter struct {
	sync.Mutex
	count int
}

func (c *counter) Add(n int) {
	c.Lock()
	defer c.Unlock()
	c.count = c.count + n
}

func (c *counter) Value() int {
	c.Lock()
	defer c.Unlock()
	return c.count
}
*/

var	builddate string
var	codetag string
var version = flag.Bool("version", false, "version info")
var bg = flag.Bool("B", false, "backgound")

func main() {
	flag.Parse()
	if *version {
		if codetag!="" {
			fmt.Printf("version %s\n",codetag)
		}
		fmt.Printf("builddate %s\n",builddate)
		return
	}

	//log.Println("start...",len(flag.Args()), flag.Args())
	//log.Printf("bg=%v\n",*bg)

	domain := "timur.mobi"
	calleeId := ""
	calleeUrl := ""

	if separator=="/callee/" {
		// callee mode: read config from local ini file
		configFileName := "webcall.ini"
		log.Printf("try local config file %s\n",configFileName)
		configIni, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true,},configFileName)
		if err != nil {
			// read config from home ini file
			dirname, err := os.UserHomeDir()
			if err != nil {
				log.Fatal(err)
			}
			configFileName = "~/.webcall.ini"
			configFile := strings.Replace(configFileName,"~",dirname,1)
			log.Printf("try home config file %s\n",configFile)
			configIni, err = ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true,},configFile)
			if err != nil {
				configIni = nil
				log.Printf("no config file found err=%v\n", configFile, err)
			}
		}
		// read cfg from selected file
		if configIni!=nil {
			log.Printf("read config file...\n")
			config := readIniString(configIni, "arg", "", "")
			if config!="" {
				toks := strings.Split(config, " ")

				if strings.HasPrefix(toks[0],"https://") {
					calleeUrl = toks[0]
					log.Printf("cfg calleeUrl=%s\n",calleeUrl)
				} else {
					if len(toks)>1 {
						domain = toks[0]
						calleeId = toks[1]
						log.Printf("cfg domain=%s calleeId=%s\n",domain,calleeId)
					} else {
						calleeId = toks[0]
						log.Printf("cfg calleeId=%s\n",calleeId)
					}
					if domain!="" && calleeId!="" {
						calleeUrl = "https://"+domain+separator+calleeId
						//log.Printf("cfg calleeUrl=%s\n",calleeUrl)
					}
				}
			}
		}
	}

	// override cfg via commandline args
	if len(flag.Args())>0 {
		if strings.HasPrefix(flag.Arg(0),"https://") {
			calleeUrl = flag.Arg(0)
			log.Printf("calleeUrl=%s\n",calleeUrl)
		} else {
			if len(flag.Args())>1 {
				domain = flag.Arg(0)
				calleeId = flag.Arg(1)
				log.Printf("domain=%s calleeId=%s\n",domain,calleeId)
			} else {
// TODO support "calleeId@domain"
				idxAt := strings.Index(flag.Arg(0),"@")
				if idxAt>=0 {
					calleeId = flag.Arg(0)[:idxAt]
					domain = flag.Arg(0)[idxAt+1:]
				} else {
					calleeId = flag.Arg(0)
				}
				log.Printf("calleeId=%s\n",calleeId)
			}
			if domain!="" && calleeId!="" {
				calleeUrl = "https://"+domain+separator+calleeId
				//log.Printf("calleeUrl=%s\n",calleeUrl)
			}
		}
	}

	if calleeUrl=="" {
		if calleeId!="" {
			if separator=="/user/" {
// TODO for webcall (separator=="/user/") there should be a way to enter the target calleeId (and the domain) via UI
			}
		}
		if calleeId!="" {
			log.Fatalf("# missing calleeId\n")
		}
		if domain!="" {
			log.Fatalf("# missing domain name\n")
		}
	}
	log.Printf("calleeUrl=%s\n",calleeUrl)

	// daemonize this process (by spawning a bg-child)
	if !*bg {
		launch := os.Args[0]+" -B "+calleeUrl
		log.Printf("launch=%s\n",launch)
	    cmd := exec.Command(os.Args[0], "-B", calleeUrl)
		err := cmd.Start()
		if err!=nil {
			log.Printf("# exec err=%v\n",err)
		}
	    return
	}

	// this is the daemonized process
	args := []string{} // "--start-fullscreen"
	log.Printf("lorca.New args=%v\n",args)
	ui, err := lorca.New("", "", 480, 520)
	if err != nil {
		// most likely: cannot find /usr/lib/chromium/chromium
		log.Fatalf("# lorca.New ui=%v err=%v\n",ui,err)
	}
	defer ui.Close()

	log.Println("ui.Bind('start')...")
	// A simple way to know when UI is ready (uses body.onload event in JS)
	ui.Bind("start", func() {
		log.Println("UI is ready")
	})

	// Create and bind Go object to the UI
//	c := &counter{}
//	ui.Bind("counterAdd", c.Add)
//	ui.Bind("counterValue", c.Value)

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

	log.Println("exiting...")
}
