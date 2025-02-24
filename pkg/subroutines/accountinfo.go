package subroutines

import (
	"context"
	"fmt"

	kcpcorev1alpha "github.com/kcp-dev/kcp/sdk/apis/core/v1alpha1"
	kcptenancyv1alpha "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	"github.com/kcp-dev/logicalcluster/v3"
	commonconfig "github.com/openmfp/golang-commons/config"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	"github.com/openmfp/golang-commons/logger"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/kontext"

	"github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
)

var _ lifecycle.Subroutine = (*AccountInfoSubroutine)(nil)

const (
	AccountInfoSubroutineName = "AccountLocationSubroutine"
)

type AccountInfoSubroutine struct {
	client client.Client
}

func NewAccountInfoSubroutine(client client.Client) *AccountInfoSubroutine {
	return &AccountInfoSubroutine{client: client}
}

func (r *AccountInfoSubroutine) GetName() string {
	return AccountInfoSubroutineName
}

func (r *AccountInfoSubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	return ctrl.Result{}, nil
}

func (r *AccountInfoSubroutine) Finalizers() []string { // coverage-ignore
	return []string{}
}

func (r *AccountInfoSubroutine) Process(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	instance := runtimeObj.(*v1alpha1.Account)
	cfg := commonconfig.LoadConfigFromContext(ctx).(config.Config)
	log := logger.LoadLoggerFromContext(ctx)

	// select workspace for account
	ws, err := r.retrieveWorkspace(ctx, instance, log)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	// Prepare context to work in workspace
	wsCtx := kontext.WithCluster(ctx, logicalcluster.Name(ws.Spec.Cluster))

	// Get FGA Store ID
	// For now this is hard coded, needs to be replaced with Store generation on Organization level
	storeId := cfg.FGA.StoreId

	// Retrieve logical cluster
	parentPath, err := r.retrieveCurrentWorkspacePath(ctx, err, log)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	wsPath := fmt.Sprintf("%s:%s", parentPath, ws.Name)
	selfAccountLocation := v1alpha1.AccountLocation{Name: instance.Name, ClusterId: ws.Spec.Cluster, Type: instance.Spec.Type, Path: wsPath}

	if instance.Spec.Type == v1alpha1.AccountTypeOrg {
		accountInfo := &v1alpha1.AccountInfo{ObjectMeta: v1.ObjectMeta{Name: "account"}}
		_, err = controllerutil.CreateOrUpdate(wsCtx, r.client, accountInfo, func() error {
			accountInfo.Spec.Account = selfAccountLocation
			accountInfo.Spec.ParentAccount = nil
			accountInfo.Spec.Organization = selfAccountLocation
			accountInfo.Spec.FGA.Store.Id = storeId
			return nil
		})

		return ctrl.Result{}, nil
	}

	parentAccountInfo, exists, err := r.retrieveAccountInfo(ctx, err, log)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	if !exists {
		return ctrl.Result{}, errors.NewOperatorError(fmt.Errorf("AccountInfo does not yet exist. Retry another time"), true, false)
	}

	accountInfo := &v1alpha1.AccountInfo{ObjectMeta: v1.ObjectMeta{Name: "account"}}
	_, err = controllerutil.CreateOrUpdate(wsCtx, r.client, accountInfo, func() error {
		accountInfo.Spec.Account = selfAccountLocation
		accountInfo.Spec.ParentAccount = &parentAccountInfo.Spec.Account
		accountInfo.Spec.Organization = parentAccountInfo.Spec.Organization
		accountInfo.Spec.FGA.Store.Id = storeId
		return nil
	})
	return ctrl.Result{}, nil
}

func (r *AccountInfoSubroutine) retrieveAccountInfo(ctx context.Context, err error, log *logger.Logger) (*v1alpha1.AccountInfo, bool, error) {
	accountInfo := &v1alpha1.AccountInfo{}
	err = r.client.Get(ctx, client.ObjectKey{Name: "account"}, accountInfo)
	if err != nil {
		if kerrors.IsNotFound(err) {
			log.Info().Msg("accountInfo does not yet exist, retry")
			return nil, false, nil
		}
		log.Error().Err(err).Msg("error retrieving accountInfo")
		return nil, false, err
	}
	return accountInfo, true, nil
}

func (r *AccountInfoSubroutine) retrieveCurrentWorkspacePath(ctx context.Context, err error, log *logger.Logger) (string, error) {
	lc := &kcpcorev1alpha.LogicalCluster{}
	err = r.client.Get(ctx, client.ObjectKey{Name: "cluster"}, lc)
	if err != nil {
		log.Error().Err(err).Msg("logicalCluster does not yet exist")
		return "", errors.Wrap(err, "logicalCluster does not yet exist")
	}
	selfPath, ok := lc.ObjectMeta.Annotations["kcp.io/path"]
	if !ok {
		log.Error().Msg("logicalCluster does not have a path annotation")
		return "", errors.New("logicalCluster does not have a path annotation")
	}
	return selfPath, nil
}

func (r *AccountInfoSubroutine) retrieveWorkspace(ctx context.Context, instance *v1alpha1.Account, log *logger.Logger) (*kcptenancyv1alpha.Workspace, error) {
	ws := &kcptenancyv1alpha.Workspace{}
	err := r.client.Get(ctx, client.ObjectKey{Name: instance.Name}, ws)
	if err != nil {
		log.Error().Msg("workspace does not exist")
		return nil, err
	}
	return ws, nil
}
