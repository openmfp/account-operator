package v1alpha1_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestService(t *testing.T) {

	testEnv := envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "chart", "charts", "crds", "templates")},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := testEnv.Start()

	require.NoError(t, err)
	defer testEnv.Stop()

	v1alpha1.AddToScheme(scheme.Scheme)

	testClient, err := client.New(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	require.NoError(t, err)

	err = testClient.Create(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "root-namespace"}})
	require.NoError(t, err)

	tests := []struct {
		name string
	}{
		{
			name: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := v1alpha1.NewService(testClient, "root-namespace")

			_ = svc
		})
	}
}
