package main

import (
	"context"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/spf13/pflag"
	secretsv1alpha1 "github.com/yaso/yet-another-secrets-operator/api/v1alpha1"
	"github.com/yaso/yet-another-secrets-operator/pkg/controllers"

	//+kubebuilder:scaffold:imports
	awsclient "github.com/yaso/yet-another-secrets-operator/pkg/providers/aws/client"
	awsconfig "github.com/yaso/yet-another-secrets-operator/pkg/providers/aws/config"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(secretsv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	// Create default config
	operatorConfig := awsconfig.NewDefaultConfig()

	// Add flags to pflag
	operatorConfig.AddFlags(pflag.CommandLine)

	// Parse flags
	pflag.Parse()

	// Set the global logger
	opts := zap.Options{
		Development: operatorConfig.Debug,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	// From this point, setupLog will work properly
	setupLog.Info("Starting the operator")

	// Load environment variables
	operatorConfig.LoadFromEnv()

	// Create AWS config
	awsConfig := operatorConfig.ToAWSConfig()

	// Force AWS client for now
	awsClient := awsclient.NewClient(awsConfig)

	// Test AWS connectivity at startup
	ctx := context.Background()

	setupLog.Info("Testing AWS connectivity...")
	if err := awsClient.TestConnection(ctx, setupLog); err != nil {
		setupLog.Error(err, "Failed to connect to AWS Secrets Manager")
		os.Exit(1)
	} else {
		setupLog.Info("Successfully connected to AWS Secrets Manager")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: operatorConfig.Health.ProbeBindAddress,
		LeaderElection:         operatorConfig.Leader.Enabled,
		LeaderElectionID:       operatorConfig.Leader.ID,
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ASecretReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Log:       log.Log.WithName("controllers").WithName("ASecret"),
		AwsClient: awsClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ASecret")
		os.Exit(1)
	}

	if err = (&controllers.AGeneratorReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    log.Log.WithName("controllers").WithName("AGenerator"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AGenerator")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
