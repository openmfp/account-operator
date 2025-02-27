package subroutines

import (
	"context"
	"fmt"
	"strings"

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
	AccountInfoSubroutineName = "AccountInfoSubroutine"
	DefaultAccountInfoName    = "account"
)

type AccountInfoSubroutine struct {
	client   client.Client
	serverCA string
}

func NewAccountInfoSubroutine(client client.Client, serverCA string) *AccountInfoSubroutine {
	return &AccountInfoSubroutine{client: client, serverCA: serverCA}
}

func (r *AccountInfoSubroutine) GetName() string {
	return AccountInfoSubroutineName
}

func (r *AccountInfoSubroutine) Finalize(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
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
	accountWorkspace, err := retrieveWorkspace(ctx, instance, r.client, log)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	if accountWorkspace.Status.Phase != kcpcorev1alpha.LogicalClusterPhaseReady {
		log.Info().Msg("workspace is not ready yet, retry")
		return ctrl.Result{Requeue: true}, nil
	}

	// Prepare context to work in workspace
	wsCtx := kontext.WithCluster(ctx, logicalcluster.Name(accountWorkspace.Spec.Cluster))

	// Retrieve logical cluster
	currentWorkspacePath, currentWorkspaceUrl, err := r.retrieveCurrentWorkspacePath(accountWorkspace)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	selfAccountLocation := v1alpha1.AccountLocation{Name: instance.Name, ClusterId: accountWorkspace.Spec.Cluster, Type: instance.Spec.Type, Path: currentWorkspacePath, URL: currentWorkspaceUrl}

	if instance.Spec.Type == v1alpha1.AccountTypeOrg {
		accountInfo := &v1alpha1.AccountInfo{ObjectMeta: v1.ObjectMeta{Name: DefaultAccountInfoName}}
		_, err = controllerutil.CreateOrUpdate(wsCtx, r.client, accountInfo, func() error {
			accountInfo.Spec.Account = selfAccountLocation
			accountInfo.Spec.ParentAccount = nil
			accountInfo.Spec.Organization = selfAccountLocation
			// Get FGA Store ID
			// For now this is hard coded, needs to be replaced with Store generation on Organization level
			accountInfo.Spec.FGA.Store.Id = cfg.FGA.StoreId
			accountInfo.Spec.ClusterInfo.CA = r.serverCA
			return nil
		})
		if err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}

		return ctrl.Result{}, nil
	}

	parentAccountInfo, exists, err := r.retrieveAccountInfo(ctx, log)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	if !exists {
		return ctrl.Result{}, errors.NewOperatorError(fmt.Errorf("AccountInfo does not yet exist. Retry another time"), true, false)
	}

	accountInfo := &v1alpha1.AccountInfo{ObjectMeta: v1.ObjectMeta{Name: DefaultAccountInfoName}}
	_, err = controllerutil.CreateOrUpdate(wsCtx, r.client, accountInfo, func() error {
		accountInfo.Spec.Account = selfAccountLocation
		accountInfo.Spec.ParentAccount = &parentAccountInfo.Spec.Account
		accountInfo.Spec.Organization = parentAccountInfo.Spec.Organization
		accountInfo.Spec.FGA.Store.Id = parentAccountInfo.Spec.FGA.Store.Id
		accountInfo.Spec.ClusterInfo.CA = r.serverCA
		return nil
	})
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}
	return ctrl.Result{}, nil
}

func (r *AccountInfoSubroutine) retrieveAccountInfo(ctx context.Context, log *logger.Logger) (*v1alpha1.AccountInfo, bool, error) {
	accountInfo := &v1alpha1.AccountInfo{}
	err := r.client.Get(ctx, client.ObjectKey{Name: "account"}, accountInfo)
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

func (r *AccountInfoSubroutine) retrieveCurrentWorkspacePath(ws *kcptenancyv1alpha.Workspace) (string, string, error) {
	if ws.Spec.URL == "" {
		return "", "", fmt.Errorf("workspace URL is empty")
	}

	// Parse path from URL
	split := strings.Split(ws.Spec.URL, "/")
	if len(split) < 3 {
		return "", "", fmt.Errorf("workspace URL is invalid")
	}

	lastSegment := split[len(split)-1]
	if lastSegment == "" || strings.Trim(lastSegment, " ") == "" {
		return "", "", fmt.Errorf("workspace URL is empty")
	}
	return lastSegment, ws.Spec.URL, nil
}
