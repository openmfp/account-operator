package subroutines_test

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openfga/openfga/pkg/server"
	"github.com/openfga/openfga/pkg/storage/memory"
	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

type StoreSubroutineSuite struct {
	suite.Suite
	testEnv       envtest.Environment
	testClient    client.Client
	testFGAClient openfgav1.OpenFGAServiceClient
	grpcServer    *grpc.Server
	openfgaServer *server.Server
}

func TestStoreSubroutineProcess(t *testing.T) {
	suite.Run(t, new(StoreSubroutineSuite))
}

func (s *StoreSubroutineSuite) SetupSuite() {

	scheme := runtime.NewScheme()
	corev1alpha1.AddToScheme(scheme)

	s.testEnv = envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "chart", "charts", "crds", "templates")},
		ErrorIfCRDPathMissing: true,
		Scheme:                scheme,
	}

	cfg, err := s.testEnv.Start()
	s.Require().NoError(err)

	cache, err := cache.New(cfg, cache.Options{
		Scheme: scheme,
	})
	s.Require().NoError(err)

	err = cache.IndexField(context.Background(), &corev1alpha1.AuthorizationModel{}, ".spec.storeRef.name", func(o client.Object) []string {
		store := o.(*corev1alpha1.AuthorizationModel).Spec.StoreRef.Name
		return []string{store}
	})
	s.Require().NoError(err)

	go func() {
		cache.Start(context.Background())
	}()

	syncCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if !cache.WaitForCacheSync(syncCtx) {
		s.T().Fatal("failed to wait for cache sync")
	}

	s.testClient, err = client.New(cfg, client.Options{
		Scheme: s.testEnv.Scheme,
		Cache: &client.CacheOptions{
			Reader: cache,
		},
	})
	s.Require().NoError(err)

	s.openfgaServer = server.MustNewServerWithOpts(server.WithDatastore(memory.New()))

	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)

	s.grpcServer = grpc.NewServer()
	openfgav1.RegisterOpenFGAServiceServer(s.grpcServer, s.openfgaServer)

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			panic(err)
		}
	}()

	resolver.SetDefaultScheme("passthrough")
	conn, err := grpc.NewClient("",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)

	s.testFGAClient = openfgav1.NewOpenFGAServiceClient(conn)
}

func (s *StoreSubroutineSuite) TearDownSuite() {
	s.openfgaServer.Close()
	s.grpcServer.Stop()
	s.testEnv.Stop()
}

func (s *StoreSubroutineSuite) TestStoreSubroutineProcess() {
	routine := subroutines.NewStoreSubroutine(s.testClient, s.testFGAClient)

	authzModel := &corev1alpha1.AuthorizationModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: corev1alpha1.AuthorizationModelSpec{
			StoreRef: corev1.LocalObjectReference{
				Name: "test",
			},
			Model: "module core\n\ntype user",
		},
	}
	err := s.testClient.Create(context.Background(), authzModel)
	s.Require().NoError(err)

	meta.SetStatusCondition(&authzModel.Status.Conditions, metav1.Condition{
		Type:   lifecycle.ConditionReady,
		Status: metav1.ConditionTrue,
		Reason: "Reason",
	})

	err = s.testClient.Status().Update(context.Background(), authzModel)
	s.Require().NoError(err)

	store := &corev1alpha1.Store{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: corev1alpha1.StoreSpec{
			CoreModule: corev1.LocalObjectReference{
				Name: "test",
			},
		},
	}
	_, operatorErr := routine.Process(context.Background(), store)

	s.Require().Nil(operatorErr)

	s.Require().NotEmpty(store.Status.StoreID)

	res, err := s.testFGAClient.GetStore(context.Background(), &openfgav1.GetStoreRequest{StoreId: store.Status.StoreID})
	s.Require().NoError(err)

	s.Require().Equal(store.Status.StoreID, res.Id)
	s.Require().Equal(store.Name, res.Name)
}
