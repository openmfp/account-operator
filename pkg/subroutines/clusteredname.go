package subroutines

import (
	"github.com/kcp-dev/logicalcluster/v3"
	"k8s.io/apimachinery/pkg/types"
)

type ClusteredName struct {
	types.NamespacedName
	ClusterID logicalcluster.Name
}
