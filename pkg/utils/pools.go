package utils

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/rand"
	"sort"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

// GetFittingPools returns a list of pools that have enough resources to satisfy the resource requirements.
// The list is sorted by the sum of the resource usage of the pool. The pool with the least resource usage is first.
func GetFittingPools(lease *v1.Lease, pools []*v1.Pool) []*v1.Pool {
	var fittingPools []*v1.Pool
	for _, pool := range pools {
		nameMatch := len(lease.Spec.RequiredPool) > 0 && lease.Spec.RequiredPool == pool.ObjectMeta.Name
		if !nameMatch && pool.Spec.Exclude {
			continue
		}
		if len(lease.Spec.RequiredPool) > 0 && !nameMatch {
			continue
		}
		if int(pool.Status.VCpusAvailable) >= lease.Spec.VCpus &&
			int(pool.Status.MemoryAvailable) >= lease.Spec.Memory {
			fittingPools = append(fittingPools, pool)
		}
	}
	sort.Slice(fittingPools, func(i, j int) bool {
		iPool := fittingPools[i]
		jPool := fittingPools[j]
		return iPool.Status.VCpusAvailable+iPool.Status.MemoryAvailable+iPool.Status.DatastoreAvailable+iPool.Status.NetworkAvailable <
			jPool.Status.VCpusAvailable+jPool.Status.MemoryAvailable+jPool.Status.DatastoreAvailable+jPool.Status.NetworkAvailable
	})
	return fittingPools
}

func shuffleFittingPools(pools []*v1.Pool) {
	rand.Shuffle(len(pools), func(i, j int) {
		pools[i], pools[j] = pools[j], pools[i]
	})
}

// GetPoolWithStrategy returns a pool that has enough resources to satisfy the lease requirements.
func GetPoolWithStrategy(lease *v1.Lease, pools []*v1.Pool, strategy v1.AllocationStrategy) (*v1.Pool, error) {
	fittingPools := GetFittingPools(lease, pools)

	if len(fittingPools) == 0 {
		return nil, fmt.Errorf("no pools available")
	}
	switch strategy {
	case v1.RESOURCE_ALLOCATION_STRATEGY_RANDOM:
		shuffleFittingPools(fittingPools)
		fallthrough
	case v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED:
		fallthrough
	default:
		pool := fittingPools[0]
		lease.OwnerReferences = append(lease.OwnerReferences, metav1.OwnerReference{
			APIVersion: pool.APIVersion,
			Kind:       pool.Kind,
			Name:       pool.Name,
			UID:        pool.UID,
		})
		return pool, nil
	}
}
