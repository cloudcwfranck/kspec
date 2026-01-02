/*
Copyright 2025 kspec contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/controllers"
	"github.com/cloudcwfranck/kspec/pkg/alerts"
	clientpkg "github.com/cloudcwfranck/kspec/pkg/client"
	"github.com/cloudcwfranck/kspec/pkg/webhooks"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kspecv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var enableWebhooks bool
	var probeAddr string
	var leaderElectionNamespace string
	var leaseDuration time.Duration
	var renewDeadline time.Duration
	var retryPeriod time.Duration

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableWebhooks, "enable-webhooks", true,
		"Enable admission webhooks for real-time validation")
	flag.StringVar(&leaderElectionNamespace, "leader-election-namespace", "",
		"Namespace where the leader election resource will be created. Defaults to the same namespace where the manager runs.")
	flag.DurationVar(&leaseDuration, "leader-election-lease-duration", 15*time.Second,
		"Duration that non-leader candidates will wait to force acquire leadership")
	flag.DurationVar(&renewDeadline, "leader-election-renew-deadline", 10*time.Second,
		"Duration that the acting leader will retry refreshing leadership before giving up")
	flag.DurationVar(&retryPeriod, "leader-election-retry-period", 2*time.Second,
		"Duration the LeaderElector clients should wait between tries of actions")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              "kspec-operator-lock",
		LeaderElectionNamespace:       leaderElectionNamespace,
		LeaderElectionReleaseOnCancel: true,
		LeaseDuration:                 &leaseDuration,
		RenewDeadline:                 &renewDeadline,
		RetryPeriod:                   &retryPeriod,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Get config for multi-cluster support
	config := ctrl.GetConfigOrDie()

	// Create Client Factory for multi-cluster support
	clientFactory := clientpkg.NewClusterClientFactory(config, mgr.GetClient())

	// Setup ClusterTarget controller
	if err = controllers.NewClusterTargetReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		config,
		clientFactory,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterTarget")
		os.Exit(1)
	}

	// Create alert manager for notification handling
	alertManager := alerts.NewManager(ctrl.Log.WithName("alerts"))

	// Setup ClusterSpecification controller (multi-cluster enabled)
	if err = controllers.NewClusterSpecReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		config,
		clientFactory,
		alertManager,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterSpecification")
		os.Exit(1)
	}

	// Setup AlertConfig controller
	if err = controllers.NewAlertConfigReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		alertManager,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AlertConfig")
		os.Exit(1)
	}

	// Start webhook server (v0.3.0 Phase 3)
	if enableWebhooks {
		setupLog.Info("Starting admission webhook server")
		webhookServer := webhooks.NewServer(mgr.GetClient(), 9443)
		if err := mgr.Add(webhookServer); err != nil {
			setupLog.Error(err, "unable to start webhook server")
			// Don't exit - allow operator to run without webhooks
			setupLog.Info("Webhooks disabled - continuing without real-time validation")
		} else {
			setupLog.Info("Webhook server started successfully on port 9443")
		}
	} else {
		setupLog.Info("Webhooks disabled via flag")
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager", "leaderElection", enableLeaderElection,
		"leaseDuration", leaseDuration,
		"renewDeadline", renewDeadline,
		"retryPeriod", retryPeriod)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
