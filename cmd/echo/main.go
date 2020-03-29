package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hpifu/go-kit/hconf"
	"github.com/hpifu/go-kit/hdef"
	"github.com/hpifu/go-kit/henv"
	"github.com/hpifu/go-kit/hflag"
	"github.com/hpifu/go-kit/hgrpc"
	"github.com/hpifu/go-kit/hrule"
	"github.com/hpifu/go-kit/logger"
	"github.com/hpifu/tpl-go-grpc/api"
	"github.com/hpifu/tpl-go-grpc/internal/echo"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"gopkg.in/sohlich/elogrus.v7"
)

// AppVersion name
var AppVersion = "unknown"

type Options struct {
	Service struct {
		Port int `hflag:"usage: service port" hdef:"7060"`
	}
	Es struct {
		Uri string `hflag:"usage: elasticsearch address"`
	}
	Logger struct {
		Info   logger.Options
		Warn   logger.Options
		Access logger.Options
	}
}

func main() {
	version := hflag.Bool("v", false, "print current version")
	configfile := hflag.String("c", "configs/echo.json", "config file path")
	if err := hflag.Bind(&Options{}); err != nil {
		panic(err)
	}
	if err := hflag.Parse(); err != nil {
		panic(err)
	}
	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	options := &Options{}
	if err := hdef.SetDefault(options); err != nil {
		panic(err)
	}
	config, err := hconf.New("json", "local", *configfile)
	if err != nil {
		panic(err)
	}
	if err := config.Unmarshal(options); err != nil {
		panic(err)
	}
	if err := henv.NewHEnv("ECHO").Unmarshal(options); err != nil {
		panic(err)
	}
	if err := hflag.Unmarshal(options); err != nil {
		panic(err)
	}
	if err := hrule.Evaluate(options); err != nil {
		panic(err)
	}

	// init logger
	logs, err := logger.NewLoggerGroup([]*logger.Options{
		&options.Logger.Info, &options.Logger.Warn, &options.Logger.Access,
	})
	if err != nil {
		panic(err)
	}
	infoLog := logs[0]
	warnLog := logs[1]
	accessLog := logs[2]

	client, err := elastic.NewClient(
		elastic.SetURL(options.Es.Uri),
		elastic.SetSniff(false),
	)
	if err != nil {
		panic(err)
	}
	hook, err := elogrus.NewAsyncElasticHook(client, "echo", logrus.InfoLevel, "echo")
	if err != nil {
		panic(err)
	}
	accessLog.Hooks.Add(hook)

	infoLog.Infof("%v init success, config\n %#v", os.Args[0], options)

	interceptor := hgrpc.NewGrpcInterceptor(infoLog, warnLog, accessLog)
	// run server
	var kaep = keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}
	var kasp = keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
		MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
		MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
		Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
	}
	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.Interceptor),
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
	)
	go func() {
		// init services
		svc := echo.NewService()
		svc.SetLogger(infoLog, warnLog, accessLog)
		api.RegisterServiceServer(server, svc)
		address, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", options.Service.Port))
		if err != nil {
			panic(err)
		}

		if err := server.Serve(address); err != nil {
			panic(err)
		}
	}()

	// graceful quit
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	infoLog.Infof("%v shutdown ...", os.Args[0])
	server.GracefulStop()
	infoLog.Infof("%v shutdown success", os.Args[0])

	// close loggers
	for _, log := range logs {
		_ = log.Out.(*rotatelogs.RotateLogs).Close()
	}
}
