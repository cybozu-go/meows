package cmd

import (
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	"github.com/cybozu-go/github-actions-controller/controllers"
	"github.com/cybozu-go/github-actions-controller/github"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(actionsv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

}

func run() error {
	ctrl.SetLogger(zap.New(zap.StacktraceLevel(zapcore.DPanicLevel)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     config.metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: config.probeAddr,
		LeaderElection:         config.enableLeaderElection,
		LeaderElectionID:       "6bee5a22.cybozu.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	c, err := github.NewClient(
		config.appID,
		config.appInstallationID,
		config.appPrivateKeyPath,
		config.organizationName,
		config.repositoryName,
	)
	if err != nil {
		setupLog.Error(err, "unable to create github client", "controller", "RunnerPool")
		return err
	}

	rpr := controllers.NewRunnerPoolReconciler(
		mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("RunnerPool"),
		mgr.GetScheme(),
		config.organizationName,
	)
	if err = rpr.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RunnerPool")
		return err
	}

	atr := controllers.NewOldTokenSweeper(
		ctrl.Log.WithName("actions-token-updator"),
		mgr.GetEventRecorderFor("actions-token-updator"),
		config.tokenSweepInterval,
		mgr.GetClient(),
		c,
	)
	if err := mgr.Add(atr); err != nil {
		setupLog.Error(err, "unable to add runner to manager", "actions-token-runner")
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
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}
