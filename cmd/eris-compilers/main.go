package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/eris-ltd/eris-compilers"
	"github.com/eris-ltd/eris-compilers/version"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/eris-ltd/common/go/common"
	//logger "github.com/eris-ltd/common/go/log"
)

// simple eris-compilers and cli
func main() {

	app := cli.NewApp()
	app.Name = "eris-compilers"
	app.Usage = ""
	app.Version = version.VERSION
	app.Author = "Eris Industries"
	app.Email = "support@erisindustries.com"

	app.Action = cliServer
	app.Before = before
	app.After = after

	app.Flags = []cli.Flag{
		securePortFlag,
		unsecurePortFlag,
		unsecureOnlyFlag,
		secureOnlyFlag,
		certFlag,
		keyFlag,
		internalFlag,
		verboseFlag,
		debugFlag,
		hostFlag,
	}

	app.Commands = []cli.Command{
		{
			Name:   "compile",
			Usage:  "compile a contract",
			Action: cliClient,
			Flags: []cli.Flag{
				hostFlag,
				localFlag,
				langFlag,
				//logFlag,
			},
		},
		{
			Name:   "proxy",
			Usage:  "run a proxy server for out of process access",
			Action: cliProxy,
			Flags: []cli.Flag{
				portFlag,
			},
		},
	}

	run(app)
}

func before(c *cli.Context) error {
	//log.SetFormatter(logger.ErisFormatter{})

	if c.Bool("verbose") {
		log.SetLevel(log.InfoLevel)
	} else if c.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}
	return nil
}

func after(c *cli.Context) error {
	return nil
}

func cliClient(c *cli.Context) {
	if len(c.Args()) == 0 {
		ifExit(fmt.Errorf("Specify a contract to compile"))
	}
	tocompile := c.Args()[0]

	var err error
	lang := c.String("language")
	if lang == "" {
		lang, err = compilers.LangFromFile(tocompile)
		ifExit(err)
	}

	host := c.String("host")
	if host != "" {
		url := host + "/" + "compile"
		compilers.SetLanguageURL(lang, url)
	}
	log.Debugln("language config:", compilers.Languages[lang])

	libraries := c.String("libraries")

	common.InitDataDir(compilers.ClientCache)
	log.Infoln("compiling", tocompile)
	if c.Bool("local") {
		compilers.SetLanguageNet(lang, false)
		//b, err := compilers.CompileWrapper(tocompile, lang)
		// force it through the compile pipeline so we get caching
		resp := compilers.Compile(tocompile, libraries)
		if resp.Error != "" {
			log.Errorln(resp.Error)
			os.Exit(0)
		}
		for _, r := range resp.Objects {
			log.Infoln("objectname:", r.Objectname)
			log.Infoln("bytecode:", hex.EncodeToString(r.Bytecode))
			log.Infoln("abi:", r.ABI)
		}
	} else {
		resp := compilers.Compile(tocompile, libraries)
		if resp.Error != "" {
			fmt.Println(err)
		}
		for _, r := range resp.Objects {
			log.Infoln("objectname:", r.Objectname)
			log.Infoln("bytecode:", hex.EncodeToString(r.Bytecode))
			log.Infoln("abi:", r.ABI)
			log.Infoln("abi:", r.ABI)
		}
	}
}

func cliProxy(c *cli.Context) {
	addr := "localhost:" + strconv.Itoa(c.Int("port"))
	compilers.StartProxy(addr)
}

func cliServer(c *cli.Context) {

	common.InitDataDir(compilers.ServerCache)
	addrUnsecure := ""
	addrSecure := ""

	if c.Bool("internal") {
		addrUnsecure = "localhost"
		addrSecure = "localhost"
	}

	addrUnsecure += ":" + strconv.Itoa(c.Int("unsecure-port"))
	addrSecure += ":" + strconv.Itoa(c.Int("secure-port"))

	if c.Bool("secure-only") {
		addrUnsecure = ""
	}
	if c.Bool("no-ssl") {
		addrSecure = ""
	}

	key := c.String("key")
	cert := c.String("cert")

	if !c.Bool("no-ssl") {

		if _, err := os.Stat(key); os.IsNotExist(err) {
			common.Exit(fmt.Errorf("Can't find ssl key %s. Use --no-ssl flag to disable", key))
		}
		if _, err := os.Stat(cert); os.IsNotExist(err) {
			common.Exit(fmt.Errorf("Can't find ssl cert %s. Use --no-ssl flag to disable", cert))
		}

	}

	compilers.StartServer(addrUnsecure, addrSecure, key, cert)
}

// so we can catch panics
func run(app *cli.App) {
	defer func() {
		if r := recover(); r != nil {
			trace := make([]byte, 2048)
			count := runtime.Stack(trace, true)
			fmt.Println("Panic: ", r)
			fmt.Printf("Stack of %d bytes: %s", count, trace)
		}
	}()

	app.Run(os.Args)
}

var (
	localFlag = cli.BoolFlag{
		Name:  "local",
		Usage: "use local compilers",
	}

	langFlag = cli.StringFlag{
		Name:  "language, l",
		Usage: "language the script is written in",
	}

	verboseFlag = cli.BoolFlag{
		Name:  "verbose",
		Usage: "verbose output",
	}

	debugFlag = cli.BoolFlag{
		Name:  "debug",
		Usage: "debug output",
	}

	portFlag = cli.IntFlag{
		Name:  "port",
		Usage: "set the proxy port",
		Value: 9097,
	}

	unsecurePortFlag = cli.IntFlag{
		Name:  "unsecure-port, p",
		Usage: "set the listening port",
		Value: 9099,
	}

	securePortFlag = cli.IntFlag{
		Name:  "secure-port, P",
		Usage: "set the listening port",
		Value: 9098,
	}

	secureOnlyFlag = cli.BoolFlag{
		Name:  "secure-only, s",
		Usage: "only use https",
	}

	unsecureOnlyFlag = cli.BoolFlag{
		Name:   "no-ssl",
		Usage:  "do not use ssl",
		EnvVar: "NO_SSL",
	}

	certFlag = cli.StringFlag{
		Name:  "cert",
		Usage: "set the https certificate",
		Value: "",
	}

	keyFlag = cli.StringFlag{
		Name:  "key",
		Usage: "set the https certificate",
		Value: "",
	}

	internalFlag = cli.BoolFlag{
		Name:  "internal, i",
		Usage: "only bind localhost (don't expose to internet)",
	}

	hostFlag = cli.StringFlag{
		Name:   "host",
		Usage:  "set the server host (include http(s)://)",
		Value:  "",
		EnvVar: "HOST",
	}
)

func ifExit(err error) {
	if err != nil {
		log.Errorln(err)
		os.Exit(0)
	}
}
