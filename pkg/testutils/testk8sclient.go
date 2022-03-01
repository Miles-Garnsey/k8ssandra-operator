package testutils

import (
	"context"
	"sync"
	"testing"
	"time"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	k8ssandraapi "github.com/k8ssandra/k8ssandra-operator/apis/k8ssandra/v1alpha1"
	reaperapi "github.com/k8ssandra/k8ssandra-operator/apis/reaper/v1alpha1"
	stargateapi "github.com/k8ssandra/k8ssandra-operator/apis/stargate/v1alpha1"
	promapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestK8sClient struct {
	client.Client
	TestState      *testing.T
	UnsafeGetSync  func(ctx context.Context, key client.ObjectKey, obj client.Object) error
	UnsafeListSync func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
	timeout        time.Duration
	tick           time.Duration
}

// Get IS AN ASYNC SAFE client.Get(), it is not the normal `Get()`! Use UnsafeListSync() if you want regular Get()'s behaviour.
func (my TestK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	var err error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-time.After(my.tick): // Poll
				err = my.UnsafeGetSync(ctx, key, obj)
				if err == nil {
					return
				}
			case <-time.After(my.timeout): // Timeout
				err = errors.NewTimeoutError("timed out trying to get resource", 0)
				return
			}
		}
	}()
	wg.Wait()
	return err
}

// List IS AN ASYNC SAFE client.List(), it is not the normal `List()`! Use UnsafeListSync() if you want regular List()'s behaviour.
func (my TestK8sClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	var err error
	assert.Eventually(my.TestState, func() bool {
		if err := my.UnsafeListSync(ctx, list, opts...); err != nil {
			return false
		}
		return true
	}, my.timeout, my.tick)
	return err
}

// NewTestk8sClient takes a configured client and returns a client which overrides certain methods to ensure their safety in testing.
func NewTestk8sClient(t *testing.T, configuredClient client.Client, timeout time.Duration, tick time.Duration) TestK8sClient {
	testScheme := configuredClient.Scheme()
	utilruntime.Must(promapi.AddToScheme(testScheme))
	utilruntime.Must(cassdcapi.AddToScheme(testScheme))
	utilruntime.Must(k8ssandraapi.AddToScheme(testScheme))
	utilruntime.Must(reaperapi.AddToScheme(testScheme))
	utilruntime.Must(stargateapi.AddToScheme(testScheme))
	testClient := TestK8sClient{
		Client:         configuredClient,
		TestState:      t,
		UnsafeGetSync:  configuredClient.Get,
		UnsafeListSync: configuredClient.List,
		timeout:        timeout,
		tick:           tick,
	}
	return testClient
}
