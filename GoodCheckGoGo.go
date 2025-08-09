// by Ori
package main

import (
	"flag"
	"fmt"
	"goodcheckgogo/checklist"
	"goodcheckgogo/lookup"
	"goodcheckgogo/options"
	"goodcheckgogo/requestscurl"
	"goodcheckgogo/requestsnative"
	"goodcheckgogo/strategy"
	"goodcheckgogo/utils"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	gochoice "github.com/TwiN/go-choice"
)

const (
	VERSION     = "0.8.5"
	PROGRAMNAME = "GoodCheckGoGo"

	CONFIGFILE      = "config.ini"
	CHECKLISTFOLDER = "Checklists"
	LOGSFOLDER      = "Logs"
	STRATEGYFOLDER  = "StrategiesGoGo"
)

var (
	allStrategies    []strategy.Strategy
	allWebsites      []checklist.Website
	programToUse     options.OptionFoolingProgram
	testMode         int    = -1 // 1 for native, 2 for curl
	resolverOfChoice string = ""
	passes           int    = 0
	testBegun        bool   = false
	ggc              string = ""
	ggcURL           string = ""
	stratlist        string = ""
	checklistfile    string = ""

	flagHelp           *bool
	flagIsQuiet        *bool
	flagFoolingProgram *string
	flagMode           *string
	flagChecklist      *string
	flagStrategyList   *string
	flagPasses         *int
	flagSkipTaskKill   *bool
	flagSkipSvcKill    *bool

	errInterrupt error = fmt.Errorf("interrupt")
)

func init() {

	// reading args
	flagHelp = flag.Bool("?", false, "display help")
	flagIsQuiet = flag.Bool("q", false, "activates quiet mode")
	flagFoolingProgram = flag.String("f", "", "fooling program to use; can be either 'gdpi', 'zapret' or 'ciadpi'")
	flagMode = flag.String("m", "", "mode for requests; can be either 'native' of 'curl'; testing with proxies requires 'curl'")
	flagChecklist = flag.String("c", "", "checklist filename; must be enclosed in quotes")
	flagStrategyList = flag.String("s", "", "strategies list filename; must be enclosed in quotes")
	flagPasses = flag.Int("p", -1, "number of passes; must be greater than 0")
	flagSkipTaskKill = flag.Bool("skiptaskkill", false, "allow to skip automatic gdpi/zapret/ciadpi tasks termination")
	flagSkipSvcKill = flag.Bool("skipsvckill", false, "allow to skip automatic gdpi/zapret/ciadpi/windivert services termination (hightly not recommended!)")
	flag.Parse()
	if *flagHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// restrarting as admin if not
	if !utils.AmAdmin(*flagIsQuiet) {
		err := utils.RunMeElevated(*flagIsQuiet)
		if err != nil {
			check(fmt.Errorf("can't elevate privilegies: %v", err))
		}
		os.Exit(0)
	}

	// creating logfile
	t := time.Now().Format("2006-01-02_15-04-05")
	n := "logfile_" + PROGRAMNAME + "_" + t + ".log"
	err := utils.CreateLog(filepath.Join(LOGSFOLDER, n), *flagIsQuiet)
	if err != nil {
		check(fmt.Errorf("can't initialize log: %v", err))
	}

	// writing down some basic info into the log
	log.Printf("%s v%s\n", PROGRAMNAME, VERSION)
	log.Printf("\nOS: %s\nArchitecture: %s\n", utils.ReturnWindowsVersion(), utils.ReturnArchitecture())
	log.Printf("\nCommand-line arguments: %q", os.Args[1:])

	// setting window title
	err = utils.SetTitle(fmt.Sprintf("%s v%s", PROGRAMNAME, VERSION))
	if err != nil {
		check(fmt.Errorf("can't set title: %v", err))
	}

	// clear console window
	if !*flagIsQuiet {
		err = utils.CLS()
		if err != nil {
			check(fmt.Errorf("can't clear console window: %v", err))
		}
	}

	// parsing config
	log.Printf("\nReading config...\n")
	err = options.ParseConfig(CONFIGFILE)
	if err != nil {
		check(fmt.Errorf("can't parse config: %v", err))
	}

	log.Printf("\nInit completed\n")
}

func main() {
	var err error

	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan,
			os.Interrupt,
			syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
			syscall.SIGINT,  // Ctrl+C
			syscall.SIGQUIT, // Ctrl-\
			//syscall.SIGKILL, // "always fatal", "SIGKILL and SIGSTOP may not be caught by a program"
			syscall.SIGHUP, // "terminal is disconnected"
		)
		<-sigchan
		check(errInterrupt)
	}()

	// program choice
	log.Printf("\nChoosing fooling program...\n")
	switch *flagFoolingProgram {
	case "":
		programToUse, err = userChooseProgram()
		if err != nil {
			check(fmt.Errorf("can't choose fooling program: %v", err))
		}
	case "gdpi":
		log.Printf("Proceeding with '%s' (from args)\n", options.MyOptions.Gdpi.ProgramName)
		if !options.MyOptions.Gdpi.IsExist {
			check(fmt.Errorf("flag forcing the use of '%s', but it wasn't found", options.MyOptions.Gdpi.ProgramName))
		}
		programToUse = options.MyOptions.Gdpi
	case "zapret":
		log.Printf("Proceeding with '%s' (from args)\n", options.MyOptions.Zapret.ProgramName)
		if !options.MyOptions.Zapret.IsExist {
			check(fmt.Errorf("flag forcing the use of '%s', but it wasn't found", options.MyOptions.Zapret.ProgramName))
		}
		programToUse = options.MyOptions.Zapret
	case "ciadpi":
		log.Printf("Proceeding with '%s' (from args)\n", options.MyOptions.Ciadpi.ProgramName)
		if !options.MyOptions.Ciadpi.IsExist {
			check(fmt.Errorf("flag forcing the use of '%s', but it wasn't found", options.MyOptions.Ciadpi.ProgramName))
		}
		programToUse = options.MyOptions.Ciadpi
	default:
		check(fmt.Errorf("flag -f has the wrong value '%s'", *flagFoolingProgram))
	}

	// stopping programs and services
	log.Printf("\nStopping active fooling programs and services...\n")
	err = stopFoolingProgramsAndServices(*flagSkipTaskKill, *flagSkipSvcKill)
	if err != nil {
		check(fmt.Errorf("can't stop all fooling programs and services: %v", err))
	}

	// strategy list choice
	log.Printf("\nChoosing strategy list...\n")
	switch *flagStrategyList {
	case "":
		stratlist, err = userChooseStrategyList()
		if err != nil {
			check(fmt.Errorf("can't choose strategy list: %v", err))
		}
	default:
		stratlist = *flagStrategyList
		log.Printf("Proceeding with '%s' (from args)\n", *flagStrategyList)
	}

	// strategy list processing
	log.Printf("\nParsing strategy list...\n")
	allStrategies, err = strategy.ReadStrategies(filepath.Join(STRATEGYFOLDER, programToUse.ProgramName, stratlist))
	if err != nil {
		check(fmt.Errorf("can't parse strategy list: %v", err))
	}
	if programToUse.WorksAsProxy && strategy.Proxy == "noproxy" {
		check(fmt.Errorf("choosen program works as proxy, but proxy itself is unset"))
	}

	// test mode choice
	log.Printf("\nChoosing requests mode...\n")
	switch *flagMode {
	case "":
		testMode, err = userChooseTestMode()
		if err != nil {
			check(fmt.Errorf("can't choose requests mode: %v", err))
		}
	case "native":
		testMode = 1
		log.Println("Proceeding with 'Native' (from args)")
		if strategy.Proxy != "noproxy" {
			check(fmt.Errorf("testing with proxy requires 'Curl', but flag is forcing 'Native' mode"))
		}
	case "curl":
		testMode = 2
		log.Println("Proceeding with 'Curl' (from args)")
	default:
		check(fmt.Errorf("flag -m has the wrong value '%s'", *flagMode))
	}

	// connectivity check
	if options.MyOptions.NetConnTest.Value {
		//log.Printf("\nChecking '%s' connectivity...\n", strategy.ProtoFull)
		log.Printf("\nChecking '%s' connectivity...\n", fmt.Sprintf("tcp%d", strategy.IPV))
		switch testMode {
		case 1:
			// native
			requestsnative.SetTransport(5, 2)
			err = requestsnative.CheckConnectivityNative()
			if err == nil {
				break
			}
			if options.MyOptions.SkipCertVerify.Value {
				check(fmt.Errorf("connectivity test failed: %v", err))
			}
			options.MyOptions.SkipCertVerify.Value = true
			requestsnative.SetTransport(5, 2)
			log.Println("Normal connectivity test failed, switching mode to insecure")
			err = requestsnative.CheckConnectivityNative()
			if err != nil {
				check(fmt.Errorf("connectivity test failed: %v", err))
			}
			if !*flagIsQuiet {
				err := userChooseContinueInsecure()
				if err != nil {
					check(fmt.Errorf("can't choose whether to continue or not: %v", err))
				}
			} else {
				log.Println("Auto-accept continue insecure in quiet mode")
			}
		case 2:
			// curl
			c, err := requestscurl.CheckConnectivityCurl()
			if err != nil {
				check(fmt.Errorf("connectivity test failed: %v", err))
			}
			if c {
				break
			}
			if options.MyOptions.SkipCertVerify.Value {
				check(fmt.Errorf("connectivity test failed"))
			}
			options.MyOptions.SkipCertVerify.Value = true
			log.Println("Normal connectivity test failed, switching mode to insecure")
			c, err = requestscurl.CheckConnectivityCurl()
			if err != nil {
				check(fmt.Errorf("connectivity test failed: %v", err))
			}
			if !c {
				check(fmt.Errorf("connectivity test failed"))
			}
			if !*flagIsQuiet {
				err := userChooseContinueInsecure()
				if err != nil {
					check(fmt.Errorf("can't choose whether to continue or not: %v", err))
				}
			} else {
				log.Println("Auto-accept continue insecure in quiet mode")
			}
		default:
			check(fmt.Errorf("schrodinger's cat: 'testMode' value is out of bounds: '%d'", testMode))
		}
	} else {
		log.Printf("\nSkipping connectivity test...\n")
	}

	// resolver connectivity test
	if options.MyOptions.UseDoH.Value && strategy.Proxy == "noproxy" {
		domainOnly := utils.InsensitiveReplace(options.MyOptions.NetConnTestURL.Value, "https://", "")
		switch testMode {
		case 1:
			// native
			log.Printf("\nChecking DNS resolvers availability (Native)...\n")
			for _, resolver := range options.MyOptions.DoHResolvers.Value {
				log.Printf("Testing '%s' resolver, looking up ipv%d for '%s'...\n", resolver, strategy.IPV, domainOnly)
				dnsResult, err := lookup.DnsLookup(resolver, domainOnly, strategy.IPV, options.MyOptions.ResolverNativeTimeout.Value, options.MyOptions.ResolverNativeRetries.Value, options.MyOptions.SkipCertVerify.Value)
				if err != nil || !dnsResult.Response {
					log.Println("No proper response from DNS, trying next one...")
					continue
				}
				if dnsResult.Zero {
					log.Println("Zero response from DNS, trying next one...")
					continue
				}
				var _ip string
				for _, answer := range dnsResult.Answer {
					if strategy.IPV == 4 && answer.A != "" {
						_ip = answer.A
						break
					} else if strategy.IPV == 6 && answer.AAAA != "" {
						_ip = answer.AAAA
						break
					}
				}
				if _ip == "" {
					log.Println("No valid IP was found, trying next one...")
					continue
				}
				resolverOfChoice = resolver
				log.Printf("Resolver seems ok: '%s' -> %s\n", domainOnly, _ip)
				break
			}
			if resolverOfChoice == "" {
				check(fmt.Errorf("all resolvers failed"))
			}
		case 2:
			// curl
			log.Printf("\nChecking DNS resolvers availability (Curl)...\n")
			for _, resolver := range options.MyOptions.DoHResolvers.Value {
				log.Printf("Testing '%s' resolver, looking up ipv%d for '%s'...\n", resolver, strategy.IPV, domainOnly)
				dnsResult := requestscurl.DnsLookupCurl(resolver, domainOnly)
				if dnsResult == "" {
					log.Println("Can't resolve IP, trying next one...")
					continue
				}
				resolverOfChoice = resolver
				log.Printf("Resolver seems ok: '%s' -> %s\n", domainOnly, dnsResult)
				break
			}
			if resolverOfChoice == "" {
				check(fmt.Errorf("all resolvers failed"))
			}
		}
	} else {
		log.Printf("\nCustom resolver disabled or proxy in effect, skipping DNS availability test...\n")
	}

	// checklist choice
	log.Printf("\nChoosing checklist...\n")
	switch *flagChecklist {
	case "":
		checklistfile, err = userChooseChecklist()
		if err != nil {
			check(fmt.Errorf("can't choose checklist: %v", err))
		}
	default:
		checklistfile = *flagChecklist
		log.Printf("Proceeding with '%s' (from args)\n", *flagChecklist)
	}

	// checklist processing
	log.Printf("\nReading checklist...\n")
	allWebsites, err = checklist.ReadChecklist(filepath.Join(CHECKLISTFOLDER, checklistfile))
	if err != nil {
		check(fmt.Errorf("can't read checklist: %v", err))
	}

	// auto GGC
	if options.MyOptions.AutoGGC.Value {
		log.Printf("\nLooking for Google Cache Server URL...\n")
		switch testMode {
		case 1:
			//native
			for _, url := range options.MyOptions.MappingURLs.Value {
				ggc = requestsnative.ExtractClusterNative(url)
				if ggc != "" {
					break
				}
			}
			if ggc != "" {
				ggcURL = checklist.ConvertClusterToURL(ggc)
				log.Println("------------------")
				log.Println("Your googlevideo cluster:", ggc)
				log.Println("Your googlevideo URL:", ggcURL)
				log.Println("------------------")
				allWebsites = append(allWebsites, checklist.NewWebsite(ggcURL))
			} else {
				log.Println("Can't find googlevideo cluster")
			}
		case 2:
			//curl
			for _, url := range options.MyOptions.MappingURLs.Value {
				ggc = requestscurl.ExtractClusterCurl(url)
				if ggc != "" {
					break
				}
			}
			if ggc != "" {
				ggcURL = checklist.ConvertClusterToURL(ggc)
				log.Println("------------------")
				log.Println("Your googlevideo cluster:", ggc)
				log.Println("Your googlevideo URL:", ggcURL)
				log.Println("------------------")
				allWebsites = append(allWebsites, checklist.NewWebsite(ggcURL))
			} else {
				log.Println("Can't find googlevideo cluster")
			}
		}
	} else {
		log.Printf("\nAuto-looking for Google Cache Server disabled, skipping...\n")
	}
	if len(allWebsites) == 0 {
		check(fmt.Errorf("no URLs to check"))
	}

	// resolving
	if strategy.Proxy == "noproxy" {
		log.Printf("\nResolving IP addresses...\n")
		switch testMode {
		case 1:
			//native
			for i := 0; i < len(allWebsites); i++ {
				err = utils.SetTitle(fmt.Sprintf("%s v%s - Resolving %d/%d", PROGRAMNAME, VERSION, (i + 1), len(allWebsites)))
				if err != nil {
					check(fmt.Errorf("can't set title: %v", err))
				}

				domainOnly := utils.InsensitiveReplace(allWebsites[i].Address, "https://", "")
				dnsResult, err := lookup.DnsLookup(resolverOfChoice, domainOnly, strategy.IPV, options.MyOptions.ResolverNativeTimeout.Value, options.MyOptions.ResolverNativeRetries.Value, options.MyOptions.SkipCertVerify.Value)
				if err != nil {
					check(fmt.Errorf("can't finish DNS lookup: %v", err))
				}
				if !dnsResult.Response {
					log.Printf("No response from DNS for '%s'; removing URL from the checklist...\n", domainOnly)
					continue
				}
				if dnsResult.Zero {
					log.Printf("No valid IPv%d was found for '%s'; removing URL from the checklist...\n", strategy.IPV, domainOnly)
					continue
				}
				if strategy.IPV == 4 {
					var _ip string
					for _, answer := range dnsResult.Answer {
						if answer.A != "" {
							_ip = answer.A
							break
						}
					}
					allWebsites[i].IP = _ip
				} else {
					var _ip string
					for _, answer := range dnsResult.Answer {
						if answer.AAAA != "" {
							_ip = answer.AAAA
							break
						}
					}
					allWebsites[i].IP = _ip
				}
				if allWebsites[i].IP != "" {
					allWebsites[i].IsResolved = true
					log.Printf("IPv%d for '%s' was found: %s", strategy.IPV, domainOnly, allWebsites[i].IP)
				} else {
					log.Printf("No valid IPv%d was found for '%s'; removing URL from the checklist...\n", strategy.IPV, domainOnly)
				}
			}
		case 2:
			//curl
			for i := 0; i < len(allWebsites); i++ {
				domainOnly := utils.InsensitiveReplace(allWebsites[i].Address, "https://", "")
				dnsResult := requestscurl.DnsLookupCurl(resolverOfChoice, domainOnly)
				if dnsResult == "" {
					log.Printf("No valid IPv%d was found for '%s'; removing URL from the checklist...\n", strategy.IPV, domainOnly)
					continue
				}
				allWebsites[i].IsResolved = true
				allWebsites[i].IP = dnsResult
				log.Printf("IPv%d for '%s' was found: %s", strategy.IPV, domainOnly, allWebsites[i].IP)
			}
		}
		var w []checklist.Website
		for _, site := range allWebsites {
			if site.IsResolved {
				w = append(w, site)
			}
		}
		n := len(allWebsites)
		allWebsites = w
		n = n - len(allWebsites)
		log.Printf("URLs removed: %d\nURLs left: %d\n", n, len(allWebsites))
		if len(allWebsites) == 0 {
			check(fmt.Errorf("no URLs left"))
		}
		err = utils.SetTitle(fmt.Sprintf("%s v%s", PROGRAMNAME, VERSION))
		if err != nil {
			check(fmt.Errorf("can't set title: %v", err))
		}
	}

	// passes choice
	log.Printf("\nChoosing number of passes...\n")
	if *flagPasses <= 0 {
		passes, err = userChoosePasses()
		if err != nil {
			check(fmt.Errorf("can't choose number of passes: %v", err))
		}
	} else {
		passes = *flagPasses
		log.Printf("Proceeding with '%d' pass(es) (from args)\n", *flagPasses)
	}

	// time estimation etc
	log.Printf("\nTime estimation...\n")
	if !*flagIsQuiet {
		err = utils.CLS()
		if err != nil {
			check(fmt.Errorf("can't clear console window: %v", err))
		}
	}
	log.Println("Program:", programToUse.ProgramName)
	if testMode == 1 {
		log.Println("Requests mode: Native")
	} else {
		log.Println("Requests mode: Curl")
	}
	log.Println("Protocol:", strategy.Protocol)
	log.Println("IP version:", strategy.IPV)
	log.Println("Proxy:", strategy.Proxy)
	log.Println("Strategies list:", stratlist)
	log.Println("Total strategies:", len(allStrategies))
	log.Println("Checklist:", checklistfile)
	log.Println("Total URLs:", len(allWebsites))
	log.Println("Number of passes:", passes)
	log.Println("Timeout:", options.MyOptions.ConnTimeout.Value, "sec")
	estimStepMilliseconds := options.MyOptions.ConnTimeout.Value*1000 + options.MyOptions.InternalTimeoutMs.Value*2 + 100
	estimTMilliseconds := len(allStrategies) * passes * estimStepMilliseconds
	estimT := utils.ConvertMillisecondsSecondsToMinutesSeconds(estimTMilliseconds)
	log.Println("\nEstimated time for a test:", estimT)
	if !*flagIsQuiet {
		fmt.Println("\nPreparations complete, press [ENTER] to begin...")
		fmt.Scanln()
	}

	// main loop
	startT := time.Now()
	log.Printf("\nTesting started at %s...\n", startT.String())
	totalStrategies := len(allStrategies)
	totalURLs := len(allWebsites)

	if testMode == 1 {
		//requestsnative.SetThreads(len(allWebsites))
		requestsnative.SetTransport(len(allWebsites)*2, options.MyOptions.ConnTimeout.Value)
	}
	var keysCurl []string
	if testMode == 2 {
		keysCurl = requestscurl.FormRequestsKeys(resolverOfChoice, allWebsites)
		log.Println("Curl request line formed:", keysCurl)
	}

	// if testMode == 1 && strategy.Proxy != "noproxy" {
	// 	requestsnative.SetProxy()
	// }

	if !*flagIsQuiet {
		err = utils.CLS()
		if err != nil {
			check(fmt.Errorf("can't clear console window: %v", err))
		}
	}

	testBegun = true

	for i := 0; i < totalStrategies; i++ {
		log.Printf("\nLaunching '%s', strategy %d/%d: %s\n", programToUse.ProgramName, (i + 1), totalStrategies, allStrategies[i].Keys)
		prog, err := utils.StartProgramWithArguments(programToUse.ExecutableFullPath, allStrategies[i].Keys)
		if err != nil {
			check(fmt.Errorf("can't launch fooling program with arguments: %v", err))
		}
		for j := 1; j <= passes; j++ {
			utils.SetTitle(fmt.Sprintf("%s v%s - Testing - Strategy %d/%d, Pass %d/%d - Time left: %s", PROGRAMNAME, VERSION, (i + 1), totalStrategies, j, passes, estimT))
			estimTMilliseconds = estimTMilliseconds - estimStepMilliseconds
			estimT = utils.ConvertMillisecondsSecondsToMinutesSeconds(estimTMilliseconds)

			log.Printf("Making requests, pass %d/%d...\n", j, passes)
			time.Sleep(time.Duration(options.MyOptions.InternalTimeoutMs.Value) * time.Millisecond)
			switch testMode {
			case 1:
				// native
				wg := sync.WaitGroup{}
				for p := 0; p < totalURLs; p++ {
					wg.Add(1)
					go requestsnative.SendRequest(&wg, &allWebsites[p], allWebsites)
				}
				wg.Wait()
				requestsnative.CloseIdle()
			case 2:
				//curl
				err = requestscurl.SendRequestsAndParse(keysCurl, &allWebsites)
				if err != nil {
					check(fmt.Errorf("can't finish curl requests: %v", err))
				}
			}

			log.Printf("Displaying results...\n")
			totalS := 0
			for n := 0; n < len(allWebsites); n++ {
				var s string
				if allWebsites[n].LastResponseCode == 0 {
					s = "[CODE: 000] FAILURE"
				} else if allWebsites[n].LastResponseCode == -1 {
					s = "[CODE: ERR] ERROR  "
				} else {
					totalS++
					s = fmt.Sprintf("[CODE: %d] SUCCESS", allWebsites[n].LastResponseCode)
				}
				log.Printf("%s\t%s\n", s, allWebsites[n].Address)
				//allWebsites[n].LastResponseCode = -1
			}
			log.Printf("Successes: %d/%d\n", totalS, totalURLs)
			if !allStrategies[i].IsTested || allStrategies[i].Successes > totalS {
				allStrategies[i].Successes = totalS
				allStrategies[i].IsTested = true
				log.Printf("Writing it down; worst result for this strategy: %d/%d\n\n", allStrategies[i].Successes, totalURLs)
			} else {
				log.Printf("Skipping it; worst result for this strategy: %d/%d\n\n", allStrategies[i].Successes, totalURLs)
			}
		}
		if allStrategies[i].Successes > 0 {
			allStrategies[i].HasSuccesses = true
			for k := 0; k < totalURLs; k++ {
				if allWebsites[k].LastResponseCode != -1 && allWebsites[k].LastResponseCode != 0 && allWebsites[k].MostSuccessfulStrategySuccesses < allStrategies[i].Successes {
					allWebsites[k].MostSuccessfulStrategySuccesses = allStrategies[i].Successes
					allWebsites[k].MostSuccessfulStrategyNum = i
				}
			}
		} else {
			log.Println("This strategy has no successes")
		}
		// for k := 0; k < totalURLs; k++ {
		// 	allWebsites[k].LastResponseCode = -1
		// }
		log.Printf("Terminating program...\n")
		err = utils.StopProgram(prog)
		if err != nil {
			check(fmt.Errorf("can't terminate fooling program: %v", err))
		}
		time.Sleep(time.Duration(options.MyOptions.InternalTimeoutMs.Value) * time.Millisecond)
	}

	utils.SetTitle(fmt.Sprintf("%s v%s - Test completed", PROGRAMNAME, VERSION))
	log.Printf("\nTest ended at %s\n", time.Now().String())
	log.Printf("Total time taken: %s\n", time.Since(startT))

	// stopping programs and services
	log.Printf("\nStopping still active fooling program and WinDivert services...\n")
	err = stopFoolingProgramsAndServicesMini(*flagSkipTaskKill, *flagSkipSvcKill)
	if err != nil {
		check(fmt.Errorf("can't stop all fooling programs and services: %v", err))
	}

	// final results showcase
	log.Printf("\nDisplaying summary...\n")
	finalResultsShowcase()

	log.Printf("\nAll Done\n")
	if !*flagIsQuiet {
		fmt.Printf("\nPress [ENTER] to exit...\n")
		fmt.Scanln()
	}
	os.Exit(0)
}

func stopFoolingProgramsAndServices(skiptaskkill bool, skipsvckill bool) error {
	if !skiptaskkill {
		err := utils.TaskKill(options.MyOptions.Gdpi.ExecutableName, options.MyOptions.Zapret.ExecutableName, options.MyOptions.Ciadpi.ExecutableName)
		if err != nil {
			return fmt.Errorf("can't properly terminate fooling programs: %v", err)
		}
	}
	if !skipsvckill {
		var snames []string
		snames = append(snames, options.MyOptions.Gdpi.ServiceNames...)
		snames = append(snames, options.MyOptions.Zapret.ServiceNames...)
		snames = append(snames, options.MyOptions.Ciadpi.ServiceNames...)
		snames = append(snames, options.MyOptions.WinDivert.Value...)
		err := utils.StopAndDeleteServices(snames...)
		if err != nil {
			return fmt.Errorf("can't properly stop fooling services: %v", err)
		}
	}
	return nil
}

func stopFoolingProgramsAndServicesMini(skiptaskkill bool, skipsvckill bool) error {
	if !skiptaskkill {
		err := utils.TaskKill(programToUse.ExecutableName)
		if err != nil {
			return fmt.Errorf("can't properly terminate fooling programs: %v", err)
		}
	}
	if !skipsvckill {
		var snames []string
		snames = append(snames, options.MyOptions.WinDivert.Value...)
		err := utils.StopAndDeleteServices(snames...)
		if err != nil {
			return fmt.Errorf("can't properly stop fooling services: %v", err)
		}
	}
	return nil
}

func finalResultsShowcase() {
	if !testBegun {
		return
	}
	totalURLs := len(allWebsites)
	if totalURLs == 0 {
		return
	}
	totalStrategies := len(allStrategies)
	if totalStrategies == 0 {
		return
	}
	log.Printf("\n--------------------RESULTS BY URL---------------------\n")
	var urlsNoSuccess []int
	for i := 0; i < totalURLs; i++ {
		if !allWebsites[i].HasSuccesses {
			urlsNoSuccess = append(urlsNoSuccess, i)
		}
	}
	if len(urlsNoSuccess) > 0 {
		log.Println("\nURLs with NO successes:")
		for i := 0; i < len(urlsNoSuccess); i++ {
			log.Printf("%s | IP: %s\n", allWebsites[urlsNoSuccess[i]].Address, allWebsites[urlsNoSuccess[i]].IP)
		}
	}
	if len(urlsNoSuccess) != totalURLs {
		log.Println("\nURLs with successes:")
		for i := 0; i < totalURLs; i++ {
			if allWebsites[i].HasSuccesses {
				log.Printf("%s | IP: %s | Best strategy: %s", allWebsites[i].Address, allWebsites[i].IP, allStrategies[allWebsites[i].MostSuccessfulStrategyNum].Keys)
			}
		}
	}
	log.Printf("\n------------------RESULTS BY STRATEGY------------------\n")
	for i := 0; i <= totalURLs; i++ {
		var lines []strategy.Strategy
		for _, strat := range allStrategies {
			if strat.Successes == i {
				lines = append(lines, strat)
			}
		}
		if len(lines) > 0 {
			log.Printf("\nStrategies with %d/%d successes:\n", i, totalURLs)
			for _, line := range lines {
				log.Println(line.Keys)
			}
		}
	}
	log.Printf("\n----------------------INFORMATION----------------------\n\n")
	log.Println("Program:", programToUse.ProgramName)
	mode := "Native"
	if testMode == 2 {
		mode = "Curl"
	}
	log.Println("Requests mode:", mode)
	log.Println("Protocol:", strategy.Protocol)
	log.Println("IP version:", strategy.IPV)
	if strategy.Proxy != "noproxy" {
		log.Println("Proxy:", strategy.Proxy)
	}
	log.Println("Strategies list:", stratlist)
	log.Println("Checklist:", checklistfile)
	log.Println("Number of passes:", passes)
	log.Println("Timeout:", options.MyOptions.ConnTimeout.Value, "sec")
	if resolverOfChoice != "" {
		log.Println("Resolver:", resolverOfChoice)
	} else {
		log.Println("Resolver: System")
	}
	if options.MyOptions.AutoGGC.Value && ggcURL != "" {
		log.Println("Google Video Cluster:", ggc)
		log.Println("Google Video URL:", ggcURL)
	}
	testBegun = false
}

func userChooseContinueInsecure() error {
	_, _choiceN, err := gochoice.Pick(
		fmt.Sprintln("\nATTENTION: Normal connectivity test failed, but insecure connectivity test succeeded\nEither root certificates are corrupted or your firewall/antivirus software is interfering\n\nContinue anyway? Test results may be compromised if your ISP performs a certificate substitution:"),
		[]string{"Continue in insecure mode", "Exit"},
	)
	if err != nil {
		return fmt.Errorf("problem with a gochoice: %v", err)
	}
	if _choiceN == 1 {
		exitByChoice()
	}
	log.Println("Continuing in insecure mode by user's choice")
	return nil
}

func userChooseProgram() (options.OptionFoolingProgram, error) {
	var programList []string
	if options.MyOptions.Gdpi.IsExist {
		programList = append(programList, options.MyOptions.Gdpi.ProgramName)
	}
	if options.MyOptions.Zapret.IsExist {
		programList = append(programList, options.MyOptions.Zapret.ProgramName)
	}
	if options.MyOptions.Ciadpi.IsExist {
		programList = append(programList, options.MyOptions.Ciadpi.ProgramName)
	}
	programList = append(programList, "Exit")

	choice, _, err := gochoice.Pick(
		fmt.Sprintf("\nATTENTION: '%s' '%s' and '%s' will be closed, their services will be removed\n\nChoose the fooling program to use:", options.MyOptions.Gdpi.ProgramName, options.MyOptions.Zapret.ProgramName, options.MyOptions.Ciadpi.ProgramName),
		programList,
	)
	if err != nil {
		var p options.OptionFoolingProgram
		return p, fmt.Errorf("problem with a gochoice: %v", err)
	}
	switch choice {
	case options.MyOptions.Gdpi.ProgramName:
		log.Printf("Proceeding with '%s'\n", choice)
		return options.MyOptions.Gdpi, nil
	case options.MyOptions.Zapret.ProgramName:
		log.Printf("Proceeding with '%s'\n", choice)
		return options.MyOptions.Zapret, nil
	case options.MyOptions.Ciadpi.ProgramName:
		log.Printf("Proceeding with '%s'\n", choice)
		return options.MyOptions.Ciadpi, nil
	case "Exit":
		exitByChoice()
	}
	var p options.OptionFoolingProgram
	return p, fmt.Errorf("schrodinger's cat: choice out of bounds: '%s'", choice)
}

func userChooseTestMode() (int, error) {
	var modes []string
	if strategy.Proxy != "noproxy" {
		modes = []string{"Use Curl (only this mode is available when proxy is in use)"}
	} else {
		modes = []string{"Use Native (faster)", "Use Curl (reliable)"}
	}
	modes = append(modes, "Exit")
	choice, _, err := gochoice.Pick(
		"\nChoose requests mode:",
		modes,
	)
	if err != nil {
		return 0, fmt.Errorf("problem with a gochoice: %v", err)
	}
	switch choice {
	case "Exit":
		exitByChoice()
	case "Use Native (faster)":
		log.Println("Proceeding with 'Native'")
		return 1, nil
	case "Use Curl (reliable)":
		log.Println("Proceeding with 'Curl'")
		return 2, nil
	case "Use Curl (only this mode is available when proxy is in use)":
		log.Println("Proceeding with 'Curl' (forced by the use of proxy)")
		return 2, nil
	}
	return 0, fmt.Errorf("schrodinger's cat: choice out of bounds: '%s'", choice)
}

func userChoosePasses() (int, error) {
	var passesVariants = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}

	passesVariants = append(passesVariants, "Exit")
	choice, choiceN, err := gochoice.Pick(
		"\nNumber of passes:",
		passesVariants,
	)
	if err != nil {
		return 0, fmt.Errorf("problem with a gochoice: %v", err)
	}
	if choice == "Exit" {
		exitByChoice()
	}
	log.Printf("Proceeding with '%d' pass(es)\n", (choiceN + 1))
	return (choiceN + 1), nil
}

func userChooseChecklist() (string, error) {
	log.Printf("Reading folder '%s'...\n", CHECKLISTFOLDER)
	entries, err := os.ReadDir(CHECKLISTFOLDER)
	if err != nil {
		return "", fmt.Errorf("can't read folder '%s': %v", CHECKLISTFOLDER, err)
	}

	var checklistsInFolder []string
	for _, e := range entries {
		if !e.IsDir() {
			checklistsInFolder = append(checklistsInFolder, e.Name())
		}
	}
	if len(checklistsInFolder) == 0 {
		return "", fmt.Errorf("can't find any checklists in folder '%s'", CHECKLISTFOLDER)
	}

	checklistsInFolder = append(checklistsInFolder, "Exit")
	choice, _, err := gochoice.Pick(
		"\nChoose the checklist:",
		checklistsInFolder,
	)
	if err != nil {
		return "", fmt.Errorf("problem with a gochoice: %v", err)
	}
	if choice == "Exit" {
		exitByChoice()
	}
	log.Printf("Proceeding with '%s'\n", choice)

	return choice, nil
}

func userChooseStrategyList() (string, error) {
	fullpath := filepath.Join(STRATEGYFOLDER, programToUse.ProgramName)
	log.Printf("Reading folder '%s'...\n", fullpath)
	entries, err := os.ReadDir(fullpath)
	if err != nil {
		return "", fmt.Errorf("can't read folder '%s': %v", fullpath, err)
	}

	var strategiesListsInFolder []string
	for _, e := range entries {
		if !e.IsDir() {
			strategiesListsInFolder = append(strategiesListsInFolder, e.Name())
		}
	}
	if len(strategiesListsInFolder) == 0 {
		return "", fmt.Errorf("can't find any strategy lists in folder '%s'", fullpath)
	}

	strategiesListsInFolder = append(strategiesListsInFolder, "Exit")
	choice, _, err := gochoice.Pick(
		"\nChoose the strategy list:",
		strategiesListsInFolder,
	)
	if err != nil {
		return "", fmt.Errorf("problem with a gochoice: %v", err)
	}
	if choice == "Exit" {
		exitByChoice()
	}
	log.Printf("Proceeding with '%s'\n", choice)

	return choice, nil
}

func exitByChoice() {
	log.Printf("\nExiting by users choice...\n")
	os.Exit(0)
}

func check(err error) {
	switch err {
	case nil:
		return

	case errInterrupt:
		log.Printf("\nSignal catched: %v\n", err)

		if testBegun {
			// stopping programs and services
			log.Printf("\nTest interrupted, stopping still active fooling programs and services, further errors will be silently ignored at this point...\n")
			stopFoolingProgramsAndServicesMini(*flagSkipTaskKill, *flagSkipSvcKill)

			finalResultsShowcase()
		}

		log.Printf("\nExiting with an interrupt...\n")
		os.Exit(1)

	default:
		log.Printf("\nCritical error: %v\n", err)

		if testBegun {
			// stopping programs and services
			log.Printf("\nTest failed, stopping still active fooling programs and services, further errors will be silently ignored at this point...\n")
			stopFoolingProgramsAndServicesMini(*flagSkipTaskKill, *flagSkipSvcKill)

			finalResultsShowcase()
		} else {
			if !*flagIsQuiet {
				fmt.Printf("\nPress [ENTER] to exit...\n")
				fmt.Scanln()
			}
		}

		log.Printf("\nExiting with an error...\n")
		os.Exit(1)
	}
}
