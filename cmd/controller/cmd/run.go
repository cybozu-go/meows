package cmd

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"

	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/controllers"
	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/meows/runner"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	k8sMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(meowsv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func run() error {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&config.zapOpts)))
	ctx := ctrl.SetupSignalHandler()

	host, p, err := net.SplitHostPort(config.webhookAddr)
	if err != nil {
		return fmt.Errorf("invalid webhook address: %s, %v", config.webhookAddr, err)
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return fmt.Errorf("invalid webhook address: %s, %v", config.webhookAddr, err)
	}

	webHookServer := webhook.NewServer(webhook.Options{
		Host: host,
		Port: port,
	})
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		WebhookServer:          webHookServer,
		Metrics:                metricsserver.Options{BindAddress: config.metricsAddr},
		HealthProbeBindAddress: config.probeAddr,
		LeaderElection:         true,
		LeaderElectionID:       "6bee5a22.cybozu.com",
	})

	if err != nil {
		setupLog.Error(err, "unable to create manager")
		return err
	}
	metrics.InitControllerMetrics(k8sMetrics.Registry)

	log := ctrl.Log.WithName("controllers")
	factory := github.NewFactory()

	runnerManager := controllers.NewRunnerManager(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		factory,
		runner.NewClient(),
		config.runnerManagerInterval,
	)
	defer runnerManager.StopAll()

	secretUpdater := controllers.NewSecretUpdater(
		log,
		mgr.GetClient(),
		factory,
	)
	defer secretUpdater.StopAll()

	orgRegexp, repoRegexp, err := loadValidationRuleFromFile(config.configFile)
	if err != nil {
		setupLog.Error(err, "unable to read validation rule from config file")
		return err
	}

	reconciler := controllers.NewRunnerPoolReconciler(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		config.runnerImage,
		runnerManager,
		secretUpdater,
		orgRegexp,
		repoRegexp,
	)

	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "runner-pool-reconciler")
		return err
	}

	if err = meowsv1alpha1.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "RunnerPool")
		return err
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}

type validationRuleConfig struct {
	OrganizationRule string `yaml:"organization-rule"`
	RepositoryRule   string `yaml:"repository-rule"`
}

func loadValidationRuleFromFile(path string) (*regexp.Regexp, *regexp.Regexp, error) {
	if path == "" {
		return nil, nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	var cfg validationRuleConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	setupLog.Info("validation rule loaded",
		"organization-rule", cfg.OrganizationRule,
		"repository-rule", cfg.RepositoryRule,
	)

	var orgRegexp *regexp.Regexp
	if cfg.OrganizationRule != "" {
		re, err := regexp.Compile(cfg.OrganizationRule)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid organization-rule: %w", err)
		}
		orgRegexp = re
	}

	var repoRegexp *regexp.Regexp
	if cfg.RepositoryRule != "" {
		re, err := regexp.Compile(cfg.RepositoryRule)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid repository-rule: %w", err)
		}
		repoRegexp = re
	}

	return orgRegexp, repoRegexp, nil
}
