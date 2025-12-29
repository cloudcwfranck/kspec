package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	clientpkg "github.com/cloudcwfranck/kspec/pkg/client"
	"github.com/cloudcwfranck/kspec/pkg/enforcer"
	"github.com/cloudcwfranck/kspec/pkg/fleet"
)

// MultiClusterController coordinates enforcement across multiple clusters
type MultiClusterController struct {
	client.Client
	LocalKubeClient    *kubernetes.Clientset
	LocalDynamicClient dynamic.Interface
	ClientFactory      *clientpkg.ClusterClientFactory
	FleetAggregator    *fleet.FleetAggregator
}

// NewMultiClusterController creates a new multi-cluster controller
func NewMultiClusterController(
	client client.Client,
	localKube *kubernetes.Clientset,
	localDynamic dynamic.Interface,
	factory *clientpkg.ClusterClientFactory,
) *MultiClusterController {
	return &MultiClusterController{
		Client:             client,
		LocalKubeClient:    localKube,
		LocalDynamicClient: localDynamic,
		ClientFactory:      factory,
		FleetAggregator:    fleet.NewFleetAggregator(client, factory),
	}
}

// SyncEnforcementAcrossFleet synchronizes enforcement configuration across all clusters
func (m *MultiClusterController) SyncEnforcementAcrossFleet(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
) error {
	log := log.FromContext(ctx).WithValues("clusterSpec", clusterSpec.Name)

	log.Info("Syncing enforcement across fleet")

	// Get all cluster targets
	var clusterTargets kspecv1alpha1.ClusterTargetList
	if err := m.List(ctx, &clusterTargets); err != nil {
		return fmt.Errorf("failed to list cluster targets: %w", err)
	}

	if len(clusterTargets.Items) == 0 {
		log.Info("No remote clusters found, skipping fleet sync")
		return nil
	}

	// Create enforcers
	multiClusterEnforcer := enforcer.NewMultiClusterEnforcer(m.LocalKubeClient)
	policySynchronizer := enforcer.NewPolicySynchronizer(m.LocalDynamicClient)

	// Sync to each cluster
	syncErrors := make(map[string]error)
	for _, target := range clusterTargets.Items {
		if err := m.syncToSingleCluster(
			ctx,
			clusterSpec,
			&target,
			multiClusterEnforcer,
			policySynchronizer,
		); err != nil {
			log.Error(err, "Failed to sync to cluster", "cluster", target.Name)
			syncErrors[target.Name] = err
		}
	}

	if len(syncErrors) > 0 {
		log.Info("Fleet sync completed with errors", "errorCount", len(syncErrors))
		return fmt.Errorf("failed to sync to %d clusters", len(syncErrors))
	}

	log.Info("Successfully synced enforcement across fleet", "clusterCount", len(clusterTargets.Items))
	return nil
}

// syncToSingleCluster syncs enforcement to a single target cluster
func (m *MultiClusterController) syncToSingleCluster(
	ctx context.Context,
	clusterSpec *kspecv1alpha1.ClusterSpecification,
	target *kspecv1alpha1.ClusterTarget,
	multiClusterEnforcer *enforcer.MultiClusterEnforcer,
	policySynchronizer *enforcer.PolicySynchronizer,
) error {
	log := log.FromContext(ctx).WithValues("cluster", target.Name)

	// Create clients for target cluster
	kubeClient, dynamicClient, clusterInfo, err := m.ClientFactory.CreateClientsForClusterTarget(ctx, target)
	if err != nil {
		return fmt.Errorf("failed to create clients: %w", err)
	}

	// Skip if enforcement not allowed on this cluster
	if !clusterInfo.AllowEnforcement {
		log.Info("Enforcement not allowed on cluster, skipping")
		return nil
	}

	// Step 1: Sync enforcement infrastructure (webhooks, etc.)
	if err := multiClusterEnforcer.SyncEnforcementToCluster(
		ctx,
		clusterSpec,
		kubeClient,
		target.Name,
	); err != nil {
		return fmt.Errorf("failed to sync enforcement infrastructure: %w", err)
	}

	// Step 2: Sync policies (if using Kyverno)
	if clusterSpec.Spec.Enforcement != nil &&
		clusterSpec.Spec.Enforcement.Enabled &&
		(clusterSpec.Spec.Webhooks == nil || !clusterSpec.Spec.Webhooks.Enabled) {
		if err := policySynchronizer.SyncPolicyToCluster(
			ctx,
			clusterSpec,
			dynamicClient,
			target.Name,
		); err != nil {
			return fmt.Errorf("failed to sync policies: %w", err)
		}
	}

	log.Info("Successfully synced enforcement to cluster")
	return nil
}

// RemoveEnforcementFromFleet removes enforcement from all clusters
func (m *MultiClusterController) RemoveEnforcementFromFleet(
	ctx context.Context,
	clusterSpecName string,
) error {
	log := log.FromContext(ctx).WithValues("clusterSpec", clusterSpecName)

	log.Info("Removing enforcement from fleet")

	// Get all cluster targets
	var clusterTargets kspecv1alpha1.ClusterTargetList
	if err := m.List(ctx, &clusterTargets); err != nil {
		return fmt.Errorf("failed to list cluster targets: %w", err)
	}

	multiClusterEnforcer := enforcer.NewMultiClusterEnforcer(m.LocalKubeClient)
	policySynchronizer := enforcer.NewPolicySynchronizer(m.LocalDynamicClient)

	// Remove from each cluster
	for _, target := range clusterTargets.Items {
		kubeClient, dynamicClient, _, err := m.ClientFactory.CreateClientsForClusterTarget(ctx, &target)
		if err != nil {
			log.Error(err, "Failed to create clients", "cluster", target.Name)
			continue
		}

		// Remove webhook infrastructure
		if err := multiClusterEnforcer.RemoveEnforcementFromCluster(
			ctx,
			kubeClient,
			target.Name,
			clusterSpecName,
		); err != nil {
			log.Error(err, "Failed to remove enforcement", "cluster", target.Name)
		}

		// Remove policies
		policyName := fmt.Sprintf("kspec-%s", clusterSpecName)
		if err := policySynchronizer.RemovePolicyFromCluster(
			ctx,
			policyName,
			dynamicClient,
			target.Name,
		); err != nil {
			log.Error(err, "Failed to remove policy", "cluster", target.Name)
		}
	}

	log.Info("Enforcement removal from fleet complete")
	return nil
}

// AggregateFleetCompliance aggregates compliance across all clusters
func (m *MultiClusterController) AggregateFleetCompliance(ctx context.Context) (*fleet.FleetSummary, error) {
	return m.FleetAggregator.AggregateFleetCompliance(ctx)
}

// StartFleetMonitoring starts periodic fleet-wide monitoring
func (m *MultiClusterController) StartFleetMonitoring(ctx context.Context) error {
	log := log.FromContext(ctx)
	log.Info("Starting fleet-wide monitoring")

	// Start periodic aggregation (every 5 minutes)
	go m.FleetAggregator.StartPeriodicAggregation(ctx, 5*time.Minute)

	return nil
}

// SetupWithManager sets up the controller with the Manager
func (m *MultiClusterController) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(m)
}

// Start implements manager.Runnable
func (m *MultiClusterController) Start(ctx context.Context) error {
	return m.StartFleetMonitoring(ctx)
}
