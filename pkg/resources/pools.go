package resources

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"

	"github.com/openshift-splat-team/vsphere-capacity-manager/data"
)

var (
	Pools  data.Pools
	Leases data.Leases
	mu     sync.Mutex
)

func init() {
	Pools = append(Pools, &data.Pool{
		Spec: data.PoolSpec{
			Name:       "pool1",
			VCenter:    "vcs8e-vc.ocp2.dev.cluster.com",
			Datacenter: "datacenter1",
			Cluster:    "cluster1",
			Datastore:  "datastore1",
			ResourceSpec: data.ResourceSpec{
				VCpus:   120,
				Memory:  1600,
				Storage: 10000,
			},
		}})
	Pools = append(Pools, &data.Pool{
		Spec: data.PoolSpec{
			Name:       "pool1",
			VCenter:    "v8c-2-vcenter.ocp2.dev.cluster.com",
			Datacenter: "datacenter1",
			Cluster:    "cluster1",
			Datastore:  "datastore1",
			ResourceSpec: data.ResourceSpec{
				VCpus:   120,
				Memory:  1600,
				Storage: 10000,
			},
		}})
	Pools = append(Pools, &data.Pool{
		Spec: data.PoolSpec{
			Name:       "pool2",
			VCenter:    "vcenter.ibmc.devcluster.openshift.com",
			Datacenter: "datacenter2",
			Cluster:    "cluster2",
			Datastore:  "datastore2",
			ResourceSpec: data.ResourceSpec{
				VCpus:   60,
				Memory:  800,
				Storage: 5000,
			},
		}})
	Pools = append(Pools, &data.Pool{
		Spec: data.PoolSpec{
			Name:       "pool3",
			VCenter:    "vcenter.devqe.ibmc.devcluster.openshift.com",
			Datacenter: "datacenter3",
			Cluster:    "cluster3",
			Datastore:  "datastore3",
			ResourceSpec: data.ResourceSpec{
				VCpus:   40,
				Memory:  600,
				Storage: 1000,
			},
		}})
	reconcileSubnets(Pools)
}

func calculateResourceUsage() {
	log.Printf("calculating pool resource usage")
	for _, pool := range Pools {
		pool.Status.VCpusAvailable = 0
		pool.Status.MemoryAvailable = 0
		pool.Status.DatastoreAvailable = 0
		pool.Status.NetworkAvailable = 0
		for _, lease := range pool.Status.Leases {
			pool.Status.VCpusAvailable += float64(lease.Spec.ResourceSpec.VCpus)
			pool.Status.MemoryAvailable += float64(lease.Spec.ResourceSpec.Memory)
			pool.Status.DatastoreAvailable += float64(lease.Spec.ResourceSpec.Storage)
			pool.Status.NetworkAvailable += float64(lease.Spec.ResourceSpec.Networks)
		}
		pool.Status.VCpusAvailable = float64(pool.Spec.VCpus) - pool.Status.VCpusAvailable
		pool.Status.MemoryAvailable = float64(pool.Spec.Memory) - pool.Status.MemoryAvailable
		pool.Status.DatastoreAvailable = float64(pool.Spec.Storage) - pool.Status.DatastoreAvailable
		pool.Status.NetworkAvailable = float64(len(pool.Status.PortGroups) - len(pool.Status.ActivePortGroups))
		log.Printf("Pool %s Usage: vcpu-available: %f, memory-available: %f, storage-available: %f, network-available: %f",
			pool.Spec.Name, pool.Status.VCpusAvailable, pool.Status.MemoryAvailable, pool.Status.DatastoreAvailable, pool.Status.NetworkAvailable)
	}
}

// GetPools returns a list of pools
func GetPools() data.Pools {
	mu.Lock()
	defer mu.Unlock()
	calculateResourceUsage()
	pools := make(data.Pools, len(Pools))
	copy(pools, Pools)
	return pools
}

// getFittingPools returns a list of pools that have enough resources to satisfy the resource requirements.
// The list is sorted by the sum of the resource usage of the pool. The pool with the least resource usage is first.
func getFittingPools(resource *data.Resource) data.Pools {
	var fittingPools data.Pools
	for _, pool := range Pools {
		if int(pool.Status.VCpusAvailable) >= resource.Spec.VCpus &&
			int(pool.Status.MemoryAvailable) >= resource.Spec.Memory &&
			int(pool.Status.DatastoreAvailable) >= resource.Spec.Storage &&
			int(pool.Status.NetworkAvailable) >= resource.Spec.Networks {
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

func shuffleFittingPools(pools data.Pools) {
	rand.Shuffle(len(pools), func(i, j int) {
		pools[i], pools[j] = pools[j], pools[i]
	})
}

func getPoolsWithStrategy(resource *data.Resource, strategy data.AllocationStrategy) (data.Pools, error) {
	fittingPools := getFittingPools(resource)

	if len(fittingPools) == 0 {
		return nil, fmt.Errorf("no pools with enough resources")
	}
	if len(fittingPools) < resource.Spec.VCenters {
		return nil, fmt.Errorf("required number of vCenters exceeds the number of fitting pools")
	}
	switch strategy {
	case data.RESOURCE_ALLOCATION_STRATEGY_RANDOM:
		shuffleFittingPools(fittingPools)
		return fittingPools[:resource.Spec.VCenters], nil
	case data.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED:
		fallthrough
	default:
		return fittingPools[:resource.Spec.VCenters], nil
	}
}
