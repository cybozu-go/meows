package cmd

import (
	"context"
	"fmt"
	"net"
	"strconv"

	constants "github.com/cybozu-go/meows"
	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/controllers"
	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/meows/runner"
	corev1 "k8s.io/api/core/v1"
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

	reader := mgr.GetAPIReader()
	orgName, err := getTargetOrganization(ctx, reader, config.controllerNamespace, constants.OptionConfigMapName)
	if err != nil {
		setupLog.Error(err, "unable to read target organization")
		return err
	}
	cred, err := getGitHubCredential(ctx, reader, config.controllerNamespace, constants.CredentialSecretName)
	if err != nil {
		setupLog.Error(err, "unable to read GitHub Credential")
		return err
	}

	githubClient, err := github.NewFactory(orgName).New(ctx, cred)
	if err != nil {
		setupLog.Error(err, "unable to create github client")
		return err
	}

	log := ctrl.Log.WithName("controllers")
	runnerManager := controllers.NewRunnerManager(
		log,
		mgr.GetClient(),
		githubClient,
		runner.NewClient(),
		config.runnerManagerInterval,
	)
	secretUpdater := controllers.NewSecretUpdater(
		log,
		mgr.GetClient(),
		githubClient,
	)
	reconciler := controllers.NewRunnerPoolReconciler(
		log,
		mgr.GetClient(),
		mgr.GetScheme(),
		orgName,
		config.runnerImage,
		runnerManager,
		secretUpdater,
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

func getTargetOrganization(ctx context.Context, reader client.Reader, namespace, name string) (string, error) {
	cm := new(corev1.ConfigMap)
	err := reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, cm)
	if err != nil {
		return "", fmt.Errorf("failed to get configmap; %w", err)
	}
	org, ok := cm.Data[constants.OptionConfigMapDataOrganization]
	if !ok || org == "" {
		return "", fmt.Errorf("missing %s key", constants.OptionConfigMapDataOrganization)
	}
	return org, nil
}

func getGitHubCredential(ctx context.Context, reader client.Reader, namespace, name string) (*github.ClientCredential, error) {
	s := new(corev1.Secret)
	err := reader.Get(ctx, types.NamespacedName{Namespace: config.controllerNamespace, Name: constants.CredentialSecretName}, s)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret; %w", err)
	}

	if pat, ok := s.Data[constants.CredentialSecretDataPATToken]; ok {
		return &github.ClientCredential{
			PersonalAccessToken: string(pat),
		}, nil
	}

	return readAppKeySecret(s)
}

func readAppKeySecret(s *corev1.Secret) (*github.ClientCredential, error) {
	appIDstr, ok := s.Data[constants.CredentialSecretDataAppID]
	if !ok {
		return nil, fmt.Errorf("missing %s key", constants.CredentialSecretDataAppID)
	}
	appID, err := strconv.Atoi(string(appIDstr))
	if err != nil {
		return nil, fmt.Errorf("invalid %s value; %w", constants.CredentialSecretDataAppID, err)
	}

	insIDstr, ok := s.Data[constants.CredentialSecretDataAppInstallationID]
	if !ok {
		return nil, fmt.Errorf("missing %s key", constants.CredentialSecretDataAppInstallationID)
	}
	insID, err := strconv.Atoi(string(insIDstr))
	if err != nil {
		return nil, fmt.Errorf("invalid %s value; %w", constants.CredentialSecretDataAppInstallationID, err)
	}

	key, ok := s.Data[constants.CredentialSecretDataAppPrivateKey]
	if !ok {
		return nil, fmt.Errorf("missing %s key", constants.CredentialSecretDataAppPrivateKey)
	}

	return &github.ClientCredential{
		AppID:             int64(appID),
		AppInstallationID: int64(insID),
		PrivateKey:        key,
	}, nil
}
