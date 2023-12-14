package medusa

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	k8ssandraapi "github.com/k8ssandra/k8ssandra-operator/apis/k8ssandra/v1alpha1"
	"github.com/k8ssandra/k8ssandra-operator/pkg/reconciliation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RefreshSecrets(dc *cassdcapi.CassandraDatacenter, ctx context.Context, client client.Client, logger logr.Logger, requeueDelay time.Duration) error {
	logger.Info(fmt.Sprintf("Restore complete for DC %#v, Refreshing secrets", dc.ObjectMeta))
	userSecrets := []string{}
	for _, user := range dc.Spec.Users {
		userSecrets = append(userSecrets, user.SecretName)
	}
	if dc.Spec.SuperuserSecretName == "" {
		userSecrets = append(userSecrets, cassdcapi.CleanupForKubernetes(dc.Spec.ClusterName)+"-superuser") //default SU secret
	} else {
		userSecrets = append(userSecrets, dc.Spec.SuperuserSecretName)
	}
	logger.Info(fmt.Sprintf("refreshing user secrets for %v", userSecrets))
	//  Both Reaper and medusa secrets go into the userSecrets, so they don't need special handling.
	for _, i := range userSecrets {
		secret := &corev1.Secret{}
		err := client.Get(ctx, types.NamespacedName{Name: i, Namespace: dc.Namespace}, secret)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to get secret %s", i))
			return err
		}
		if secret.ObjectMeta.Annotations == nil {
			secret.ObjectMeta.Annotations = make(map[string]string)
		}
		secret.ObjectMeta.Annotations[k8ssandraapi.RefreshAnnotation] = time.Now().String()
		if err := reconciliation.ReconcileObject(ctx, client, requeueDelay, *secret); err != nil {
			return errors.New(err.GetError().Error())
		}
	}
	return nil

}
