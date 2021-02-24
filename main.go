package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/mdzio/ccu-jack/mqtt"
	"github.com/mdzio/ccu-jack/rtcfg"
	"github.com/mdzio/ccu-jack/vmodel"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/script"
	"github.com/mdzio/go-lib/httputil"
	"github.com/mdzio/go-logging"
	"github.com/mdzio/go-mqtt/auth"
	"github.com/mdzio/go-mqtt/service"
	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
)

const (
	appDisplayName = "CCU-Jack"
	appName        = "ccu-jack"
	appDescription = "REST/MQTT-Server for the HomeMatic CCU"
	appCopyright   = "(C)2020-2021"
	appVendor      = "info@ccu-historian.de"
)

var (
	appVersion = "-dev-" // overwritten during build process

	// command line options
	configFile = flag.String("config", "ccu-jack.cfg", "configuration `file`")

	// base services
	log          = logging.Get("main")
	logFile      *os.File
	store        rtcfg.Store
	httpServer   *httputil.Server
	modelRoot    *model.Root
	modelService *model.Service

	// application services
	sysVarCol    *vmodel.SysVarCol
	prgCol       *vmodel.ProgramCol
	mqttServer   *mqtt.Broker
	sysVarReader *mqtt.SysVarReader
	reGaDOM      *script.ReGaDOM
	deviceCol    *vmodel.DeviceCol
	intercon     *itf.Interconnector
)

func configure() error {
	// initial log level
	logging.SetLevel(logging.ErrorLevel)

	// parse command line
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage of "+appName+":")
		flag.PrintDefaults()
	}
	// flag.Parse calls os.Exit(2) on error
	flag.Parse()

	// read config file
	store.FileName = *configFile
	if err := store.Read(); err != nil {
		return err
	}

	// lock config for reading
	store.RLock()
	defer store.RUnlock()
	logCfg := &store.Config.Logging

	// configure logging
	logging.SetLevel(logCfg.Level)
	if logCfg.FilePath != "" {
		var err error
		logFile, err = os.OpenFile(logCfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Opening log file failed: %w", err)
		}
		// switch to file log
		logging.SetWriter(logFile)
	}
	return nil
}

func message() {
	// log startup message
	log.Info(appDisplayName, " V", appVersion)
	log.Info(appCopyright, " ", appVendor)

	// lock config for reading
	store.RLock()
	defer store.RUnlock()
	cfg := store.Config

	// log configuration
	log.Info("Configuration:")
	log.Info("  Log level: ", cfg.Logging.Level.String())
	log.Info("  Log file: ", cfg.Logging.FilePath)
	log.Info("  Server host name: ", cfg.Host.Name)
	log.Info("  Server address: ", cfg.Host.Address)
	log.Info("  HTTP port: ", cfg.HTTP.Port)
	log.Info("  HTTPS port: ", cfg.HTTP.PortTLS)
	log.Info("  CORS origins: ", strings.Join(cfg.HTTP.CORSOrigins, ","))
	log.Info("  Web UI dir: ", cfg.HTTP.WebUIDir)
	log.Info("  MQTT port: ", cfg.MQTT.Port)
	log.Info("  Secure MQTT port: ", cfg.MQTT.PortTLS)
	log.Info("  MQTT web socket path: ", cfg.MQTT.WebSocketPath)
	log.Info("  Generate certificates: ", cfg.Certificates.AutoGenerate)
	log.Infof("  Certificate files: %s, %s, %s, %s", cfg.Certificates.CACertFile, cfg.Certificates.CAKeyFile,
		cfg.Certificates.ServerCertFile, cfg.Certificates.ServerKeyFile)
	log.Info("  CCU address: ", cfg.CCU.Address)
	log.Info("  Interfaces: ", cfg.CCU.Interfaces.String())
	log.Info("  Init ID: ", cfg.CCU.InitID)
}

func certificates() error {
	// lock config for reading
	store.RLock()
	defer store.RUnlock()
	cert := store.Config.Certificates

	// exist certificates?
	_, errCert := os.Stat(cert.ServerCertFile)
	if errCert != nil && !os.IsNotExist(errCert) {
		return fmt.Errorf("Accessing file %s failed: %w", cert.ServerCertFile, errCert)
	}
	_, errKey := os.Stat(cert.ServerKeyFile)
	if errKey != nil && !os.IsNotExist(errKey) {
		return fmt.Errorf("Accessing file %s failed: %w", cert.ServerKeyFile, errKey)
	}
	if (errCert != nil) != (errKey != nil) {
		if errCert != nil {
			return fmt.Errorf("Missing certificate file: %s", cert.ServerCertFile)
		}
		return fmt.Errorf("Missing certificate file: %s", cert.ServerKeyFile)
	}
	if errCert == nil {
		// both certificate files exist
		return nil
	}

	// auto generation not enabled?
	if !cert.AutoGenerate {
		return errors.New("No certificate files found and auto generation is disabled")
	}

	// generate certificates
	log.Info("Generating certificates")
	now := time.Now()
	gen := &httputil.CertGenerator{
		Hosts:          []string{store.Config.Host.Name},
		Organization:   appDisplayName,
		NotBefore:      now,
		NotAfter:       now.Add(10 * 365 * 24 * time.Hour),
		CACertFile:     cert.CACertFile,
		CAKeyFile:      cert.CAKeyFile,
		ServerCertFile: cert.ServerCertFile,
		ServerKeyFile:  cert.ServerKeyFile,
	}
	if err := gen.Generate(); err != nil {
		return err
	}
	log.Debugf("Created certificate files: %s, %s, %s, %s", cert.CACertFile, cert.CAKeyFile,
		cert.ServerCertFile, cert.ServerKeyFile)
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
	vmodel.NewConfig(vendor, &store)
	model.NewHandlerStats(vendor, handlerStats)
	return r
}

func startupBase(serveErr chan<- error) {
	// lock config for reading
	store.RLock()
	defer store.RUnlock()
	cfg := store.Config

	// file handler for static files
	http.Handle("/ui/", http.StripPrefix("/ui", http.FileServer(http.Dir(cfg.HTTP.WebUIDir))))

	// setup and start http(s) server
	httpServer = &httputil.Server{
		Addr:     ":" + strconv.Itoa(cfg.HTTP.Port),
		AddrTLS:  ":" + strconv.Itoa(cfg.HTTP.PortTLS),
		CertFile: cfg.Certificates.ServerCertFile,
		KeyFile:  cfg.Certificates.ServerKeyFile,
		ServeErr: serveErr,
	}
	httpServer.Startup()

	// veap handler and model
	veapHandler := &veap.Handler{}
	modelRoot = newRoot(&veapHandler.Stats)
	modelService = &model.Service{Root: modelRoot}
	veapHandler.Service = modelService

	// authentication for VEAP
	var handler http.Handler
	handler = &HTTPAuthHandler{
		Handler: veapHandler,
		Store:   &store,
		Realm:   "CCU-Jack VEAP-Server",
	}

	// CORS handler for VEAP
	allowedMethods := handlers.AllowedMethods([]string{http.MethodGet, http.MethodPut})
	if len(cfg.HTTP.CORSOrigins) == 0 {
		handler = handlers.CORS(allowedMethods)(handler)
	} else {
		allowedOrigins := handlers.AllowedOrigins(cfg.HTTP.CORSOrigins)
		// only if origin is specified, credentials are allowed (CORS spec)
		allowCredentials := handlers.AllowCredentials()
		handler = handlers.CORS(allowedMethods, allowedOrigins, allowCredentials)(handler)
	}

	// register VEAP handler
	http.Handle(veapHandler.URLPrefix+"/", handler)
}

func shutdownBase() {
	httpServer.Shutdown()
}

func startupApp(serveErr chan<- error) {
	// lock config for reading
	store.RLock()
	defer store.RUnlock()
	cfg := store.Config

	// create device collection
	deviceCol = vmodel.NewDeviceCol(modelRoot)

	// configure HM script client
	scriptClient := &script.Client{
		Addr: cfg.CCU.Address,
	}

	// create system variable collection
	sysVarCol = vmodel.NewSysVarCol(modelRoot)
	sysVarCol.ScriptClient = scriptClient
	sysVarCol.Start()

	// create programs collection
	prgCol = vmodel.NewProgramCol(modelRoot)
	prgCol.ScriptClient = scriptClient
	prgCol.Start()

	// MQTT authentication handler
	mqttAuth := "configAuthHandler"
	auth.Register(mqttAuth, &mqtt.AuthHandler{Store: &store})

	// setup and start MQTT server
	mqttServer = &mqtt.Broker{
		Addr:          "tcp://:" + strconv.Itoa(cfg.MQTT.Port),
		AddrTLS:       "tcp://:" + strconv.Itoa(cfg.MQTT.PortTLS),
		CertFile:      cfg.Certificates.ServerCertFile,
		KeyFile:       cfg.Certificates.ServerKeyFile,
		Authenticator: mqttAuth,
		ServeErr:      serveErr,
		Service:       modelService,
	}
	mqttServer.Start()

	// register websocket proxy for MQTT
	log.Infof("MQTT websocket path: " + cfg.MQTT.WebSocketPath)
	mqttWs := &service.WebsocketHandler{
		Addr: ":" + strconv.Itoa(cfg.MQTT.Port),
	}
	http.Handle(cfg.MQTT.WebSocketPath, mqttWs)

	// event receiver for MQTT
	mqttReceiver := &mqtt.EventReceiver{
		Broker: mqttServer,
		// forward events
		Next: deviceCol,
	}

	// system variable reader for MQTT
	sysVarReader = &mqtt.SysVarReader{
		Service:      modelService,
		ScriptClient: scriptClient,
		Broker:       mqttServer,
	}
	sysVarReader.Start()

	// configure interconnector
	intercon = &itf.Interconnector{
		CCUAddr:             cfg.CCU.Address,
		Types:               cfg.CCU.Interfaces,
		IDPrefix:            cfg.CCU.InitID + "-",
		NotificationHandler: mqttReceiver,
		ServeErr:            serveErr,
		// for callbacks from CCU
		HostAddr:   cfg.Host.Address,
		XMLRPCPort: cfg.HTTP.Port,
		BINRPCPort: cfg.BINRPC.Port,
	}

	// start ReGa DOM explorer
	reGaDOM = script.NewReGaDOM(scriptClient)
	reGaDOM.Start()

	// create room and function collections
	vmodel.NewRoomCol(modelRoot, reGaDOM, modelService)
	vmodel.NewFunctionCol(modelRoot, reGaDOM, modelService)

	// startup device domain (starts handling of events)
	deviceCol.Interconnector = intercon
	deviceCol.ReGaDOM = reGaDOM
	deviceCol.ModelService = modelService
	deviceCol.Start()

	// startup interconnector
	// (an additional handler for XMLRPC is registered at the DefaultServeMux.)
	intercon.Start()
}

func shutdownApp() {
	intercon.Stop()
	deviceCol.Stop()
	reGaDOM.Stop()
	sysVarReader.Stop()
	mqttServer.Stop()
	prgCol.Stop()
	sysVarCol.Stop()
}

func run() error {
	// log message for shut down
	defer func() {
		log.Info("Shutting down")
	}()

	// setup configuration
	if err := configure(); err != nil {
		return err
	}
	// write configuration on shut down
	defer func() {
		store.Write()
		store.Close()
	}()

	// startup message
	message()

	// other setups
	if err := certificates(); err != nil {
		return err
	}

	// react on INT or TERM signal (to ensure that no signal is missed, the
	// buffer size must be 1)
	termSig := make(chan os.Signal, 1)
	signal.Notify(termSig, os.Interrupt, syscall.SIGTERM)

	// react on fatal serve errors
	serveErr := make(chan error)

	// startup base services
	startupBase(serveErr)
	defer shutdownBase()

	// startup application services
	startupApp(serveErr)
	defer shutdownApp()

	// wait for start up to complete (do not call Close() on the servers before
	// the start up is finished)
	time.Sleep(1 * time.Second)

	// wait for shutdown or error
	select {
	case err := <-serveErr:
		return err
	case <-termSig:
		log.Trace("Shutdown signal received")
		return nil
	}
}

func main() {
	err := run()
	// log fatal error
	if err != nil {
		log.Error(err)
	}
	// close log file, if present
	if logFile != nil {
		logFile.Close()
	}
	// exit with code
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
