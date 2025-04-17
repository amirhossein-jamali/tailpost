package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/amirhossein-jamali/tailpost/pkg/k8s/api/v1alpha1"
	"github.com/amirhossein-jamali/tailpost/pkg/k8s/operator"
	"github.com/amirhossein-jamali/tailpost/pkg/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// Register custom metrics
var (
	reconciliationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tailpost_operator_reconciliations_total",
			Help: "Total number of reconciliations per controller",
		},
		[]string{"controller", "result"},
	)

	reconciliationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tailpost_operator_reconciliation_duration_seconds",
			Help:    "Duration of reconciliation in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller"},
	)

	managedResources = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tailpost_operator_managed_resources",
			Help: "Number of resources managed by the operator",
		},
		[]string{"controller", "resource_type"},
	)
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.Register(scheme))

	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		reconciliationsTotal,
		reconciliationDuration,
		managedResources,
		collectors.NewBuildInfoCollector(),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enableTelemetry bool
	var telemetryEndpoint string
	var logFormat string
	var logLevel string

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-addr", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableTelemetry, "enable-telemetry", false, "Enable OpenTelemetry for operator metrics and tracing.")
	flag.StringVar(&telemetryEndpoint, "telemetry-endpoint", "http://localhost:4318", "OpenTelemetry exporter endpoint")
	flag.StringVar(&logFormat, "log-format", "json", "Log format (json or console)")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Configure logging with better options
	var level zapcore.Level
	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	opts := ctrlzap.Options{
		Development: false,
		Level:       level,
	}

	if logFormat == "console" {
		opts.EncoderConfigOptions = []ctrlzap.EncoderConfigOption{
			func(ec *zapcore.EncoderConfig) {
				ec.EncodeTime = zapcore.ISO8601TimeEncoder
			},
		}
	}

	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := ctrlzap.New(ctrlzap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	setupLog.Info("Starting operator",
		"version", "0.1.0",
		"metricsAddress", metricsAddr,
		"probeAddress", probeAddr,
		"enableLeaderElection", enableLeaderElection,
		"enableTelemetry", enableTelemetry)

	// Setup telemetry if enabled
	if enableTelemetry {
		telConfig := telemetry.Config{
			ServiceName:       "tailpost-operator",
			ServiceVersion:    "0.1.0",
			ExporterType:      "http",
			ExporterEndpoint:  telemetryEndpoint,
			ExporterTimeout:   30 * time.Second,
			SamplingRate:      1.0,
			PropagateContexts: true,
			Attributes: map[string]string{
				"k8s.namespace": os.Getenv("POD_NAMESPACE"),
				"k8s.pod":       os.Getenv("POD_NAME"),
				"component":     "operator",
			},
		}

		cleanup, err := telemetry.Setup(context.Background(), telConfig)
		if err != nil {
			setupLog.Error(err, "unable to setup telemetry")
		} else {
			setupLog.Info("telemetry setup successful")
			defer cleanup()
		}
	}

	// Setup manager with metrics
	shutdownTimeout := 30 * time.Second
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "tailpost-operator-leader-election",
		// Add graceful shutdown
		GracefulShutdownTimeout: &shutdownTimeout,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create and setup the reconciler
	reconciler, err := operator.NewTailpostAgentReconciler(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create reconciler")
		os.Exit(1)
	}

	// Note: metrics will be collected through Prometheus Registry registration

	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TailpostAgent")
		os.Exit(1)
	}

	// Add health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Add custom checks for dependent services
	if err := mgr.AddHealthzCheck("webhook", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up webhook health check")
		os.Exit(1)
	}

	// Add advanced custom health check for Kubernetes API connectivity
	if err := mgr.AddHealthzCheck("kubernetes-api", func(req *http.Request) error {
		c := mgr.GetClient()
		ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
		defer cancel()

		// Try to list some resources to check API connectivity
		var nodes corev1.NodeList
		listOpts := &client.ListOptions{Limit: 1}
		if err := c.List(ctx, &nodes, listOpts); err != nil {
			return err
		}
		return nil
	}); err != nil {
		setupLog.Error(err, "unable to set up kubernetes API health check")
		os.Exit(1)
	}

	setupLog.Info("starting manager", "version", "0.1.0")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
