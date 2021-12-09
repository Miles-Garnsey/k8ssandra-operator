// Logic in this file reconciles telemetry resources into the state declared in the CRs.

package k8ssandra

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	api "github.com/k8ssandra/k8ssandra-operator/apis/k8ssandra/v1alpha1"
	"github.com/k8ssandra/k8ssandra-operator/pkg/telemetry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *K8ssandraClusterReconciler) reconcileCassandraDCTelemetry(
	ctx context.Context,
	kc *api.K8ssandraCluster,
	dcTemplate api.CassandraDatacenterTemplate,
	actualDc *cassdcapi.CassandraDatacenter,
	logger logr.Logger,
	remoteClient client.Client,
) (ctrl.Result, error) {
	logger.Info("reconciling telemetry", "cluster", kc.Name, "datacenter", actualDc.Name)
	mergedSpec := kc.Spec.Cassandra.CassandraTelemetry.Merge(dcTemplate.CassandraTelemetry)
	dcCfg := telemetry.CassTelemetryResourcer{
		CassandraNamespace: kc.Namespace,
		DataCenterName:     actualDc.Name,
		ClusterName:        kc.Name,
		Logger:             logger,
	}
	logger.Info("merged TelemetrySpec constructed", "mergedSpec", mergedSpec, "cluster", kc.Name, "datacenter", actualDc.Name)
	// Confirm telemetry config is valid (e.g. Prometheus is installed if it is requested.)
	validConfig, err := mergedSpec.IsValid(remoteClient, logger)
	if err != nil {
		return ctrl.Result{}, errors.New("could not determine if telemetry config is valid")
	}
	if !validConfig {
		return ctrl.Result{}, errors.New("Telemetry spec was invalid for this cluster - is Prometheus installed if you have requested it?")
	}
	// Determine if we want a cleanup or a resource update.
	switch {
	case mergedSpec == nil:
		logger.Info("Telemetry not present for CassDC, will delete resources", "datacenter", actualDc.Name)
		if err := dcCfg.CleanupResources(ctx, remoteClient); err != nil {
			return ctrl.Result{}, err
		}
	case mergedSpec.Prometheus == nil:
		logger.Info("Telemetry not present for CassDC, will delete resources", "datacenter", actualDc.Name)
		if err := dcCfg.CleanupResources(ctx, remoteClient); err != nil {
			return ctrl.Result{}, err
		}
	case mergedSpec.Prometheus.Enabled == nil:
		logger.Info("Telemetry not present for CassDC, will delete resources", "datacenter", actualDc.Name, "mergedSpec.Prometheus.Enabled", mergedSpec.Prometheus.Enabled)
		if err := dcCfg.CleanupResources(ctx, remoteClient); err != nil {
			return ctrl.Result{}, err
		}
	case *mergedSpec.Prometheus.Enabled:
		logger.Info("Prometheus config found", "datacenter", actualDc.Name, "mergedSpec.Prometheus.Enabled", mergedSpec.Prometheus.Enabled)
		dcCfg.TelemetrySpec = mergedSpec
		if err := dcCfg.UpdateResources(ctx, remoteClient, kc); err != nil {
			return ctrl.Result{}, err
		}
	default:
		logger.Info("Telemetry not present for CassDC, will delete resources", "datacenter", actualDc.Name)
		if err := dcCfg.CleanupResources(ctx, remoteClient); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{RequeueAfter: r.DefaultDelay}, nil
}

func (r *K8ssandraClusterReconciler) setCassandraTelemetryStatus() (ctrl.Result, error) {
	// TODO
	return ctrl.Result{}, nil
}
