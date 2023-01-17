// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package manager

import (
    "fmt"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/onosproject/aether-roc-api/pkg/southbound"
	"github.com/onosproject/onos-api/go/onos/config/admin"
    "github.com/onosproject/onos-lib-go/pkg/certs"
    "github.com/onosproject/onos-lib-go/pkg/grpc/retry"
    "github.com/onosproject/onos-lib-go/pkg/logging"
    "google.golang.org/grpc"
    "time"
    server "github.com/tmp/out/roc-api-server-sca-0.1.x/pkg/sca_0_1_x"
)

var log = logging.GetLogger()

type Config struct {
    CaPath              string
    KeyPath             string
    CertPath            string
    GnmiEndpoint        string
    GnmiTimeout         time.Duration
    Port                uint
    ValidateResponses   bool
    AllowCorsOrigins    []string
    TopoEndpoint        string
    LogLevel            string
}

type Manager struct {
    Config      Config
    echoRouter  *echo.Echo
}

func NewManager(cfg Config) *Manager {
    log.Info("Creating Manager")
    mgr := Manager{
    Config: cfg,
    }
    return &mgr
}

func (m *Manager) Run() error {
    opts, err := certs.HandleCertPaths(m.Config.CaPath, m.Config.KeyPath, m.Config.CertPath, true)
    if err != nil {
        return err
    }

    optsWithRetry := []grpc.DialOption{
    		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithInterval(100 * time.Millisecond))),
    }
    optsWithRetry = append(opts, optsWithRetry...)
    gnmiConn, err := grpc.Dial(m.Config.GnmiEndpoint, optsWithRetry...)
    if err != nil {
        log.Error("Unable to connect to onos-config", err)
        return err
    }

    gnmiClient := new(southbound.GNMIProvisioner)
    err = gnmiClient.Init(gnmiConn)
    if err != nil {
        log.Error("Unable to setup GNMI provisioner", err)
        return err
    }

    serverImpl := &server.ServerImpl{
        GnmiClient:  gnmiClient,
        GnmiTimeout: m.Config.GnmiTimeout,
        TopoEndpoint: m.Config.TopoEndpoint,
   	   	ConfigAdminServiceClient: admin.NewConfigAdminServiceClient(gnmiConn),
    }
    m.echoRouter = echo.New()
    if m.Config.LogLevel == "DEBUG" {
        m.echoRouter.Use(middleware.Logger())
    }
    if len(m.Config.AllowCorsOrigins) > 0 {
    		m.echoRouter.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    			AllowOrigins: m.Config.AllowCorsOrigins,
    			AllowHeaders: []string{echo.HeaderAccessControlAllowOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
    		}))
    }
    m.echoRouter.File("/", "assets/index.html")
    m.echoRouter.Static("/", "assets")
    if err := server.RegisterHandlers(m.echoRouter, serverImpl, m.Config.ValidateResponses); err != nil {
    		return fmt.Errorf("server.RegisterHandlers()  %s", err)
    }

    log.Infof("Starting Manager on port %d", m.Config.Port)
    m.echoRouter.Logger.Fatal(m.echoRouter.Start(fmt.Sprintf(":%d", m.Config.Port)))

    return nil
}
