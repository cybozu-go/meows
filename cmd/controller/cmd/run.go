package cmd

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"

	constants "github.com/cybozu-go/meows"
	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/controllers"
	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/meows/runner"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	k8sMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     config.metricsAddr,
		Host:                   host,
		Port:                   port,
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

	orgRegexp, repoRegexp, err := getValidationRule(ctx, mgr.GetAPIReader(), config.controllerNamespace, constants.OptionConfigMapName)
	if err != nil {
		setupLog.Error(err, "unable to read validation rule")
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

	if err = (&meowsv1alpha1.RunnerPool{}).SetupWebhookWithManager(mgr); err != nil {
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

func getValidationRule(ctx context.Context, reader client.Reader, namespace, name string) (*regexp.Regexp, *regexp.Regexp, error) {
	cm := new(corev1.ConfigMap)
	err := reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, cm)
	if apierrors.IsNotFound(err) {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get configmap; %w", err)
	}

	var orgRegexp *regexp.Regexp
	orgRegexStr := cm.Data[constants.OptionConfigMapDataOrganizationRule]
	if orgRegexStr != "" {
		re, err := regexp.Compile(orgRegexStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid %s key: %w", constants.OptionConfigMapDataOrganizationRule, err)
		}
		orgRegexp = re
	}

	var repoRegexp *regexp.Regexp
	repoRegexpStr := cm.Data[constants.OptionConfigMapDataRepositoryRule]
	if repoRegexpStr != "" {
		re, err := regexp.Compile(repoRegexpStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid %s key: %w", constants.OptionConfigMapDataRepositoryRule, err)
		}
		repoRegexp = re
	}

	return orgRegexp, repoRegexp, nil
}
