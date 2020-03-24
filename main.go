package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mdzio/go-mqtt/auth"

	"github.com/gorilla/handlers"

	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/ccu-jack/vmodel"
	"github.com/mdzio/go-lib/hmccu/itf"
	"github.com/mdzio/go-lib/hmccu/script"
	"github.com/mdzio/go-lib/util/httputil"
	"github.com/mdzio/go-lib/veap"
	"github.com/mdzio/go-lib/veap/model"
	"github.com/mdzio/go-logging"
)

const (
	appDisplayName = "CCU-Jack"
	appName        = "ccu-jack"
	appDescription = "REST/MQTT-Server for the HomeMatic CCU"
	appVendor      = "info@ccu-historian.de"

	webUIDir       = "webui"
	caCertFile     = "cacert.pem"
	caKeyFile      = "cacert.key"
	serverCertFile = "svrcert.pem"
	serverKeyFile  = "svrcert.key"
)

var (
	appVersion = "-dev-" // overwritten during build process

	log     = logging.Get("main")
	logFile *os.File

	logLevel     = logging.InfoLevel
	logFilePath  = flag.String("logfile", "", "write log messages to `file` instead of stderr")
	serverHost   = flag.String("host", "", "host `name` for certificate generation (normally autodetected)")
	serverAddr   = flag.String("addr", "127.0.0.1", "`address` of the host")
	httpPort     = flag.Int("http", 2121, "`port` for serving HTTP")
	httpPortTLS  = flag.Int("https", 2122, "`port` for serving HTTPS")
	mqttPort     = flag.Int("mqtt", 1883, "`port` for serving MQTT")
	mqttPortTLS  = flag.Int("mqtts", 8883, "`port` for serving Secure MQTT")
	initID       = flag.String("id", "CCU-Jack", "additional `identifier` for the XMLRPC init method")
	ccuAddress   = flag.String("ccu", "127.0.0.1", "`address` of the CCU")
	ccuItfs      = itf.Types{itf.BidCosRF}
	authUser     = flag.String("user", "", "user `name` for HTTP Basic Authentication/MQTT (disabled by default)")
	authPassword = flag.String("password", "", "`password` for HTTP Basic Authentication/MQTT, q.v. -user")
	corsOrigin   = flag.String("cors", "*", "set `host` as allowed origin for CORS requests")
)

func init() {
	flag.Var(
		&logLevel,
		"log",
		"specifies the minimum `severity` of printed log messages: off, error, warning, info, debug or trace",
	)
	flag.Var(
		&ccuItfs,
		"interfaces",
		"`types` of the CCU communication interfaces (comma separated): BidCosWired, BidCosRF, System, HmIPRF, VirtualDevices",
	)
}

func logFatal(v interface{}) {
	log.Error(v)
	if logFile != nil {
		logFile.Close()
	}
	os.Exit(1)
}

func configure() error {
	// parse command line
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage of "+appName+":")
		flag.PrintDefaults()
	}
	// flag.Parse calls os.Exit(2) on error
	flag.Parse()

	// set log options
	logging.SetLevel(logLevel)
	if *logFilePath != "" {
		var err error
		logFile, err = os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Opening log file failed: %w", err)
		}
		// switch to file log
		logging.SetWriter(logFile)
	}

	// configure hostname
	if *serverHost == "" {
		name, err := os.Hostname()
		if err != nil {
			return err
		}
		*serverHost = name
	}
	return nil
}

func certificates() error {
	// certificate already present?
	_, errCert := os.Stat(serverCertFile)
	_, errKey := os.Stat(serverKeyFile)
	if !os.IsNotExist(errCert) && !os.IsNotExist(errKey) {
		return nil
	}

	// generate certificates
	log.Info("Generating certificates")
	now := time.Now()
	gen := &httputil.CertGenerator{
		Hosts:          []string{*serverHost},
		Organization:   appDisplayName,
		NotBefore:      now,
		NotAfter:       now.Add(10 * 365 * 24 * time.Hour),
		CACertFile:     caCertFile,
		CAKeyFile:      caKeyFile,
		ServerCertFile: serverCertFile,
		ServerKeyFile:  serverKeyFile,
	}
	if err := gen.Generate(); err != nil {
		return err
	}
	log.Debugf("Created certificate files: %s, %s, %s, %s", caCertFile, caKeyFile, serverCertFile, serverKeyFile)
	return nil
}

func newRoot(handlerStats *veap.HandlerStats) *model.Root {
	// root domain
	r := new(model.Root)
	r.Identifier = "root"
	r.Title = "Root"
	r.Description = "Root of the CCU-Jack VEAP server"
	r.ItemRole = "domain"

	// vendor domain
	vendor := model.NewVendor(&model.VendorCfg{
		ServerName:        appDisplayName,
		ServerVersion:     appVersion,
		ServerDescription: appDescription,
		VendorName:        appVendor,
		Collection:        r,
	})
	model.NewHandlerStats(vendor, handlerStats)

	return r
}

func run() error {
	defer func() {
		log.Info("Shutting down CCU-Jack")
	}()

	// react on INT or TERM signal (to ensure that no signal is missed, the
	// buffer size must be 1)
	termSig := make(chan os.Signal, 1)
	signal.Notify(termSig, os.Interrupt, syscall.SIGTERM)

	// file handler for static files
	http.Handle("/ui/", http.StripPrefix("/ui", http.FileServer(http.Dir(webUIDir))))

	// setup and start http(s) server
	httpServeErr := make(chan error)
	httpServer := &httputil.Server{
		Addr:     ":" + strconv.Itoa(*httpPort),
		AddrTLS:  ":" + strconv.Itoa(*httpPortTLS),
		CertFile: serverCertFile,
		KeyFile:  serverKeyFile,
		ServeErr: httpServeErr,
	}
	httpServer.Startup()
	defer httpServer.Shutdown()

	// veap handler and model
	veapHandler := &veap.Handler{}
	root := newRoot(&veapHandler.Stats)
	modelService := &model.Service{Root: root}
	veapHandler.Service = modelService

	// create device collection
	deviceCol := vmodel.NewDeviceCol(root)

	// configure HM script client
	scriptClient := &script.Client{
		Addr: *ccuAddress,
	}

	// create system variable collection
	sysVarCol := vmodel.NewSysVarCol(root)
	sysVarCol.ScriptClient = scriptClient
	sysVarCol.Start()
	defer sysVarCol.Stop()

	// create programs collection
	prgCol := vmodel.NewProgramCol(root)
	prgCol.ScriptClient = scriptClient
	prgCol.Start()
	defer prgCol.Stop()

	// MQTT authentication handler
	var mqttAuth string
	if *authUser != "" || *authPassword != "" {
		const handlerID = "singleAuthHandler"
		auth.Register(handlerID, &mqtt.SingleAuthHandler{
			User:     *authUser,
			Password: *authPassword,
		})
		mqttAuth = handlerID
	} else {
		mqttAuth = "mockSuccess"
	}

	// setup and start MQTT server
	mqttServeErr := make(chan error)
	mqttServer := &mqtt.Broker{
		Addr:          "tcp://:" + strconv.Itoa(*mqttPort),
		AddrTLS:       "tcp://:" + strconv.Itoa(*mqttPortTLS),
		CertFile:      serverCertFile,
		KeyFile:       serverKeyFile,
		Authenticator: mqttAuth,
		ServeErr:      mqttServeErr,
		Service:       modelService,
	}
	mqttServer.Start()
	defer mqttServer.Stop()

	// event receiver for MQTT
	mqttReceiver := &mqtt.EventReceiver{
		Broker: mqttServer,
		// forward events
		Next: deviceCol,
	}

	// configure interconnector
	intercon := &itf.Interconnector{
		CCUAddr:  *ccuAddress,
		Types:    ccuItfs,
		IDPrefix: *initID + "-",
		Receiver: mqttReceiver,
		// full URL of the DefaultServeMux for callbacks
		ServerURL: "http://" + *serverAddr + ":" + strconv.Itoa(*httpPort),
	}

	// start ReGa DOM explorer
	reGaDOM := script.NewReGaDOM(scriptClient)
	reGaDOM.Start()
	defer reGaDOM.Stop()

	// create room and function collections
	vmodel.NewRoomCol(root, reGaDOM, modelService)
	vmodel.NewFunctionCol(root, reGaDOM, modelService)

	// startup device domain (starts handling of events)
	deviceCol.Interconnector = intercon
	deviceCol.ReGaDOM = reGaDOM
	deviceCol.ModelService = modelService
	deviceCol.Start()
	defer deviceCol.Stop()

	// startup interconnector
	// (an additional handler for XMLRPC is registered at the DefaultServeMux.)
	intercon.Start()
	defer intercon.Stop()

	// authentication for VEAP
	var handler http.Handler
	if *authUser != "" || *authPassword != "" {
		log.Info("Forcing HTTP Basic Authentication")
		authHandler := &httputil.SingleAuthHandler{
			Handler:  veapHandler,
			User:     *authUser,
			Password: *authPassword,
			Realm:    "CCU-Jack VEAP-Server",
		}
		handler = authHandler
	} else {
		handler = veapHandler
	}

	// CORS handler for VEAP
	allowedMethods := handlers.AllowedMethods([]string{http.MethodGet, http.MethodPut})
	if *corsOrigin == "" || *corsOrigin == "*" {
		handler = handlers.CORS(allowedMethods)(handler)
	} else {
		allowedOrigins := handlers.AllowedOrigins([]string{*corsOrigin})
		// only if origin is specified, credentials are allowed (CORS spec)
		allowCredentials := handlers.AllowCredentials()
		handler = handlers.CORS(allowedMethods, allowedOrigins, allowCredentials)(handler)
	}

	// register VEAP handler
	http.Handle(veapHandler.URLPrefix+"/", handler)

	// wait for start up to complete (do not call Close() on the servers before
	// the start up is finished)
	time.Sleep(1 * time.Second)

	// wait for shutdown or error
	select {
	case err := <-httpServeErr:
		return err
	case err := <-mqttServeErr:
		return err
	case <-termSig:
		return nil
	}
}

func main() {
	// setup configuration
	if err := configure(); err != nil {
		logFatal(err)
	}

	// startup message
	log.Info(appDisplayName, " V", appVersion)
	log.Info("(C)MDZ, info@ccu-historian.de")
	log.Info("Configuration:")
	log.Info("  Log level: ", logLevel.String())
	log.Info("  Log file: ", *logFilePath)
	log.Info("  Server host name: ", *serverHost)
	log.Info("  Server address: ", *serverAddr)
	log.Info("  HTTP port: ", *httpPort)
	log.Info("  HTTPS port: ", *httpPortTLS)
	log.Info("  MQTT port: ", *mqttPort)
	log.Info("  Secure MQTT port: ", *mqttPortTLS)
	log.Info("  CORS origin: ", *corsOrigin)
	log.Info("  CCU address: ", *ccuAddress)
	log.Info("  Interfaces: ", ccuItfs.String())
	log.Info("  Init ID: ", *initID)

	// other setups
	if err := certificates(); err != nil {
		logFatal(err)
	}

	// run
	if err := run(); err != nil {
		logFatal(err)
	}
	if logFile != nil {
		logFile.Close()
	}
	os.Exit(0)
}
