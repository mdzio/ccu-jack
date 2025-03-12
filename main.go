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
	"github.com/mdzio/ccu-jack/virtdev"
	"github.com/mdzio/ccu-jack/vmodel"
	"github.com/mdzio/go-hmccu/itf"
	"github.com/mdzio/go-hmccu/script"
	"github.com/mdzio/go-lib/httputil"
	"github.com/mdzio/go-logging"
	"github.com/mdzio/go-mqtt/auth"
	"github.com/mdzio/go-mqtt/service"
	"github.com/mdzio/go-veap"
	"github.com/mdzio/go-veap/model"
	veapsvr "github.com/mdzio/go-veap/server"
)

const (
	appDisplayName = "CCU-Jack"
	appName        = "ccu-jack"
	appDescription = "REST/MQTT-Interface for the HomeMatic CCU"
	appCopyright   = "(C)2019-2025"
	appVendor      = "info@ccu-historian.de"

	// wait time for ReGaHss before signaling an error
	reGaHssStartupTimeout = 3 * time.Minute
)

var (
	appVersion = "-dev-" // overwritten during build process

	// command line options
	configFile = flag.String("config", "ccu-jack.cfg", "configuration `file`")

	// global shutdown signals
	serveErr = make(chan error)
	// to ensure that no signal is missed, the buffer size must be 1
	termSig = make(chan os.Signal, 1)

	// base services
	log          = logging.Get("main")
	logFile      *os.File
	logBuffer    *LogBuffer
	store        rtcfg.Store
	httpServer   *httputil.Server
	modelRoot    *model.Root
	configVar    *vmodel.Config
	vendorCol    model.ChangeableCollection
	modelService *model.Service
	mqttServer   *mqtt.Server
	mqttBridge   *mqtt.Bridge

	// application services
	virtualDevices   *virtdev.VirtualDevices
	scriptClient     *script.Client
	sysVarCol        *vmodel.SysVarCol
	prgCol           *vmodel.ProgramCol
	reGaDOM          *script.ReGaDOM
	virtualDeviceCol *vmodel.VirtualDeviceCol
	deviceCol        *vmodel.DeviceCol
	intercon         *itf.Interconnector
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
	logBuffer = NewLogBuffer()
	logBuffer.Next = os.Stderr
	logging.SetWriter(logBuffer)
	if logCfg.FilePath != "" {
		logFile, err := os.OpenFile(logCfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Opening log file failed: %w", err)
		}
		// switch to file log
		logBuffer.Next = logFile
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
	if cfg.MQTT.Bridge.Enable {
		log.Info("  MQTT bridge address: ", cfg.MQTT.Bridge.Address)
		log.Info("  MQTT bridge port: ", cfg.MQTT.Bridge.Port)
		log.Info("  MQTT bridge TLS: ", cfg.MQTT.Bridge.UseTLS)
		log.Info("  MQTT bridge user name: ", cfg.MQTT.Bridge.Username)
		log.Info("  MQTT bridge client ID: ", cfg.MQTT.Bridge.ClientID)
	}
	log.Info("  Generate certificates: ", cfg.Certificates.AutoGenerate)
	log.Infof("  Certificate files: %s, %s, %s, %s", cfg.Certificates.CACertFile, cfg.Certificates.CAKeyFile,
		cfg.Certificates.ServerCertFile, cfg.Certificates.ServerKeyFile)
	log.Info("  CCU address: ", cfg.CCU.Address)
	log.Info("  Interfaces: ", cfg.CCU.Interfaces.String())
	log.Info("  Init ID: ", cfg.CCU.InitID)
	log.Info("  Virtual devices: ", cfg.VirtualDevices.Enable)
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

func newRoot(handlerStats *veapsvr.HandlerStats) *model.Root {
	// root domain
	r := new(model.Root)
	r.Identifier = "root"
	r.Title = "Root"
	r.Description = "Root of the CCU-Jack VEAP server"
	r.ItemRole = "domain"

	// vendor domain
	vendorCol = model.NewVendor(&model.VendorCfg{
		ServerName:        appDisplayName,
		ServerVersion:     appVersion,
		ServerDescription: appDescription,
		VendorName:        appVendor,
		Collection:        r,
	})
	configVar = vmodel.NewConfig(vendorCol, &store)
	NewDiagnostics(vendorCol)
	model.NewHandlerStats(vendorCol, handlerStats)
	return r
}

func runBase() error {
	// lock config for reading
	store.RLock()
	// find RUnlock at end of function, no intermediate returns in this function
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
	defer httpServer.Shutdown()

	// veap handler and model
	veapHandler := &veapsvr.Handler{}
	modelRoot = newRoot(&veapHandler.Stats)
	modelService = &model.Service{Root: modelRoot}
	veapHandler.Service = &veap.BasicMetaService{Service: modelService}

	// authentication for VEAP
	var handler http.Handler
	handler = &HTTPAuthHandler{
		Handler: veapHandler,
		Store:   &store,
		Realm:   "CCU-Jack VEAP-Server",
	}

	// CORS handler for VEAP
	allowedMethods := handlers.AllowedMethods([]string{http.MethodGet, http.MethodPut})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
	if len(cfg.HTTP.CORSOrigins) == 0 {
		handler = handlers.CORS(allowedMethods, allowedHeaders)(handler)
	} else {
		allowedOrigins := handlers.AllowedOrigins(cfg.HTTP.CORSOrigins)
		// only if origin is specified, credentials are allowed (CORS spec)
		allowCredentials := handlers.AllowCredentials()
		handler = handlers.CORS(allowedMethods, allowedOrigins, allowCredentials, allowedHeaders)(handler)
	}

	// register VEAP handler
	http.Handle(veapHandler.URLPrefix+"/", handler)

	// MQTT authentication handler
	mqttAuth := "configAuthHandler"
	auth.Register(mqttAuth, &mqtt.AuthHandler{Store: &store})

	// setup and start MQTT server
	mqttServer = &mqtt.Server{
		Addr:          "tcp://:" + strconv.Itoa(cfg.MQTT.Port),
		AddrTLS:       "tcp://:" + strconv.Itoa(cfg.MQTT.PortTLS),
		CertFile:      cfg.Certificates.ServerCertFile,
		KeyFile:       cfg.Certificates.ServerKeyFile,
		Authenticator: mqttAuth,
		BufferSize:    cfg.MQTT.BufferSize,
		ServeErr:      serveErr,
	}
	mqttServer.Start()
	defer mqttServer.Stop()

	// register websocket proxy for MQTT
	log.Infof("MQTT websocket path: " + cfg.MQTT.WebSocketPath)
	mqttWs := &service.WebsocketHandler{
		Addr: ":" + strconv.Itoa(cfg.MQTT.Port),
	}
	http.Handle(cfg.MQTT.WebSocketPath, mqttWs)

	// start MQTT bridge
	mqttBridge = &mqtt.Bridge{
		EmbeddedServer: mqttServer,
	}
	mqttBridge.Start(&cfg.MQTT.Bridge)
	defer mqttBridge.Stop()

	// release config before going to next run level
	store.RUnlock()

	// run the application
	return runApp()
}

func waitForReGaHss() (shutdown bool, err error) {
	log.Info("Waiting for ReGaHss")
	t := time.Now()
	l := true
	for {
		// test ReGaHss
		resp, err := scriptClient.Execute("WriteLine(\"Hello CCU-Jack!\");")
		if err == nil && len(resp) == 1 && resp[0] == "Hello CCU-Jack!" {
			return false, nil
		}

		// log error?
		if l && time.Since(t) >= reGaHssStartupTimeout {
			log.Error("ReGaHss not reachable")
			l = false
		}

		// wait for timer, shutdown or error
		select {
		case err := <-serveErr:
			return true, err
		case <-termSig:
			log.Trace("Shutdown signal received")
			return true, nil
		case <-time.After(10 * time.Second):
			// next try
		}
	}
}

func runApp() error {
	// lock config for reading
	store.RLock()
	// find RUnlock right below
	cfg := store.Config

	// configure HM script client
	useInternalPorts := cfg.CCU.Address == "127.0.0.1" || cfg.CCU.Address == "localhost"
	scriptClient = &script.Client{
		Addr:            cfg.CCU.Address,
		UseInternalPort: useInternalPorts,
	}

	// remember for later
	enableVirtualDevices := cfg.VirtualDevices.Enable

	// intermediate unlock
	store.RUnlock()

	// start virtual devices (store must be unlocked)
	if enableVirtualDevices {
		virtualDevices = &virtdev.VirtualDevices{
			Store:            &store,
			UseInternalPorts: useInternalPorts,
			EventPublisher: &mqtt.VirtDevEventReceiver{
				Server: mqttServer,
			},
			MQTTServer: mqttServer,
		}
		virtualDevices.Start()
		defer virtualDevices.Stop()
		// listen for configuration changes
		configVar.SetChangeListener(func(_ *rtcfg.Config) {
			virtualDevices.SynchronizeDevices()
		})
		defer configVar.SetChangeListener(nil)
	}

	// wait for ReGaHss to come online
	if shutdown, err := waitForReGaHss(); shutdown || err != nil {
		return err
	}

	// lock config for reading
	store.RLock()
	// find RUnlock at end of function

	// create device collection
	deviceCol = vmodel.NewDeviceCol(modelRoot)

	// create system variable collection
	sysVarCol = vmodel.NewSysVarCol(modelRoot)
	sysVarCol.ScriptClient = scriptClient
	sysVarCol.Start()
	defer sysVarCol.Stop()

	// create programs collection
	prgCol = vmodel.NewProgramCol(modelRoot)
	prgCol.ScriptClient = scriptClient
	prgCol.Start()
	defer prgCol.Stop()

	// setup and start MQTT/VEAP bridge
	mqttVeapBridge := &mqtt.VEAPBridge{
		Server:  mqttServer,
		Service: modelService,
	}
	mqttVeapBridge.Start()
	defer mqttVeapBridge.Stop()

	// CCU device event receiver for MQTT
	mqttReceiver := &mqtt.EventReceiver{
		Server: mqttServer,
		// forward events
		Next: deviceCol,
	}

	// system variable reader for MQTT
	sysVarReader := &mqtt.SysVarReader{
		Service:      modelService,
		ScriptClient: scriptClient,
		Server:       mqttServer,
	}
	sysVarReader.Start()
	defer sysVarReader.Stop()

	// configure interconnector
	intercon = &itf.Interconnector{
		CCUAddr:          cfg.CCU.Address,
		Types:            cfg.CCU.Interfaces,
		UseInternalPorts: useInternalPorts,
		IDPrefix:         cfg.CCU.InitID + "-",
		LogicLayer:       mqttReceiver,
		ServeErr:         serveErr,
		// for callbacks from CCU
		HostAddr:   cfg.Host.Address,
		XMLRPCPort: cfg.HTTP.Port,
		BINRPCPort: cfg.BINRPC.Port,
	}

	// start ReGa DOM explorer
	reGaDOM = script.NewReGaDOM(scriptClient)
	reGaDOM.Start()
	defer reGaDOM.Stop()

	// add variable for rereading meta info from CCU
	vmodel.NewRefreshVar(
		vendorCol,
		func() {
			sysVarCol.Refresh()
			prgCol.Refresh()
			reGaDOM.Refresh()
		},
	)

	// create room and function collections
	vmodel.NewRoomCol(modelRoot, reGaDOM, modelService)
	vmodel.NewFunctionCol(modelRoot, reGaDOM, modelService)

	// create virtual devices collection
	if enableVirtualDevices {
		virtualDeviceCol = vmodel.NewVirtualDeviceCol(modelRoot)
		virtualDeviceCol.Container = virtualDevices.Devices
		virtualDeviceCol.ModelService = modelService
		virtualDeviceCol.ReGaDOM = reGaDOM
	}

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

	// release config before going to next run level
	store.RUnlock()

	// following function blocks until shutdown request
	return waitForShutdown()
}

func waitForShutdown() error {
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

	// react on INT or TERM signal
	signal.Notify(termSig, os.Interrupt, syscall.SIGTERM)

	// run base services
	return runBase()
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
