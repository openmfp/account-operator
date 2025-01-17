package subroutines

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/kcp-dev/logicalcluster/v3"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	"github.com/openmfp/golang-commons/fga/helpers"
	"github.com/openmfp/golang-commons/logger"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/kontext"

	"github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/service"
)

type FGASubroutine struct {
	fgaClient       openfgav1.OpenFGAServiceClient
	client          client.Client
	srv             service.Servicer
	rootNamespace   string
	objectType      string
	parentRelation  string
	creatorRelation string
}

func NewFGASubroutine(cl client.Client, fgaClient openfgav1.OpenFGAServiceClient, s service.Servicer, rootNamespace, creatorRelation, parentRealtion, objectType string) *FGASubroutine {
	return &FGASubroutine{
		client:          cl,
		fgaClient:       fgaClient,
		srv:             s,
		rootNamespace:   rootNamespace,
		creatorRelation: creatorRelation,
		parentRelation:  parentRealtion,
		objectType:      objectType,
	}
}

func (e *FGASubroutine) Process(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	account := runtimeObj.(*v1alpha1.Account)

	log := logger.LoadLoggerFromContext(ctx)
	log.Debug().Msg("Starting creator subroutine process() function")

	if meta.IsStatusConditionTrue(account.Status.Conditions, fmt.Sprintf("%s_Ready", e.GetName())) {
		log.Debug().Msgf("Owner has already been written for account: %s", account.GetName())
		return ctrl.Result{}, nil
	}

	storeId, err := e.getStoreId(ctx, account)
	if err != nil {
		log.Error().Err(err).Msg("Couldn't get Store Id")
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	writes := []*openfgav1.TupleKey{}

	// Determine parent account to create parent relation
	if account.GetNamespace() != e.rootNamespace {
		parent, _, err := getParentAccount(ctx, e.client, account.GetNamespace())
		if err != nil {
			log.Error().Err(err).Msg("Couldn't get parent account")
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}

		writes = append(writes, &openfgav1.TupleKey{
			Object:   fmt.Sprintf("%s:%s", e.objectType, account.GetName()),
			Relation: e.parentRelation,
			User:     fmt.Sprintf("%s:%s", e.objectType, parent.GetName()),
		})
	}

	// Assign creator to the account
	if account.Spec.Creator != nil {
		if valid := validateCreator(*account.Spec.Creator); !valid {
			log.Error().Err(err).Str("creator", *account.Spec.Creator).Msg("creator string is in the protected service account prefix range")
			return ctrl.Result{}, errors.NewOperatorError(err, false, false)
		}
		creator := formatUser(*account.Spec.Creator)

		writes = append(writes, &openfgav1.TupleKey{
			Object:   fmt.Sprintf("role:%s/%s/owner", account.Spec.Type, account.Name),
			Relation: "assignee",
			User:     fmt.Sprintf("user:%s", creator),
		})

		writes = append(writes, &openfgav1.TupleKey{
			Object:   fmt.Sprintf("%s:%s", e.objectType, account.Name),
			Relation: e.creatorRelation,
			User:     fmt.Sprintf("role:%s/%s/owner#assignee", account.Spec.Type, account.Name),
		})
	}

	for _, writeTuple := range writes {

		_, err = e.fgaClient.Write(ctx, &openfgav1.WriteRequest{
			StoreId: storeId,
			Writes: &openfgav1.WriteRequestWrites{
				TupleKeys: []*openfgav1.TupleKey{writeTuple},
			},
		})

		if helpers.IsDuplicateWriteError(err) {
			log.Info().Err(err).Msg("Open FGA writeTuple failed due to invalid input (possible duplicate)")
			err = nil
		}

		if err != nil {
			log.Error().Err(err).Msg("Open FGA writeTuple failed")
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}
	}

	return ctrl.Result{}, nil
}

func (e *FGASubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	account := runtimeObj.(*v1alpha1.Account)
	log := logger.LoadLoggerFromContext(ctx)

	storeId, err := e.getStoreId(ctx, account)
	if err != nil {
		log.Error().Err(err).Msg("Couldn't get Store Id")
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	deletes := []*openfgav1.TupleKeyWithoutCondition{}

	if account.GetNamespace() != e.rootNamespace {
		parent, _, err := getParentAccount(ctx, e.client, account.GetNamespace())
		if err != nil {
			log.Error().Err(err).Msg("Couldn't get parent account")
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}

		deletes = append(deletes, &openfgav1.TupleKeyWithoutCondition{
			Object:   fmt.Sprintf("%s:%s", e.objectType, account.GetName()),
			Relation: e.parentRelation,
			User:     fmt.Sprintf("%s:%s", e.objectType, parent.GetName()),
		})
	}

	if account.Spec.Creator != nil {
		creator := formatUser(*account.Spec.Creator)
		deletes = append(deletes, &openfgav1.TupleKeyWithoutCondition{
			Object:   fmt.Sprintf("role:%s/%s/owner", account.Spec.Type, account.Name),
			Relation: "assignee",
			User:     fmt.Sprintf("user:%s", creator),
		})

		deletes = append(deletes, &openfgav1.TupleKeyWithoutCondition{
			Object:   fmt.Sprintf("%s:%s", e.objectType, account.Name),
			Relation: e.creatorRelation,
			User:     fmt.Sprintf("role:%s/%s/owner#assignee", account.Spec.Type, account.Name),
		})
	}

	for _, deleteTuple := range deletes {

		_, err = e.fgaClient.Write(ctx, &openfgav1.WriteRequest{
			StoreId: storeId,
			Deletes: &openfgav1.WriteRequestDeletes{
				TupleKeys: []*openfgav1.TupleKeyWithoutCondition{deleteTuple},
			},
		})

		if helpers.IsDuplicateWriteError(err) {
			log.Info().Err(err).Msg("Open FGA write failed due to invalid input (possibly trying to deleteTuple nonexisting entry)")
			err = nil
		}

		if err != nil {
			log.Error().Err(err).Msg("Open FGA write failed")
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}

	}

	return ctrl.Result{}, nil
}

func (e *FGASubroutine) getStoreId(ctx context.Context, account *v1alpha1.Account) (string, error) {
	firstLevelAccountName := account.Name

	if e.rootNamespace != account.Namespace {

		lookupNamespace := account.Namespace
		lookupCtx := ctx
		for {
			parent, newClusterContext, err := getParentAccount(lookupCtx, e.client, lookupNamespace)
			if errors.Is(err, ErrNoParentAvailable) {
				break
			}
			if err != nil {
				return "", err
			}

			if newClusterContext != nil {
				lookupCtx = kontext.WithCluster(lookupCtx, logicalcluster.Name(*newClusterContext))
			}

			lookupNamespace = parent.GetNamespace()
			firstLevelAccountName = parent.GetName()
		}
	}

	storeId, err := helpers.GetStoreIDForTenant(ctx, e.fgaClient, firstLevelAccountName)
	if err != nil {
		return "", err
	}

	return storeId, nil
}

func (e *FGASubroutine) GetName() string { return "CreatorSubroutine" }

func (e *FGASubroutine) Finalizers() []string { return []string{"account.core.openmfp.io/fga"} }

var saRegex = regexp.MustCompile(`^system:serviceaccount:[^:]*:[^:]*$`)

// formatUser formats the user string to be used in the FGA write request
// it replaces colons for users conforming to the kubernetes service account pattern with dots.
func formatUser(user string) string {
	if saRegex.MatchString(user) {
		return strings.ReplaceAll(user, ":", ".")
	}
	return user
}

// validateCreator validates the creator string to ensure if it is not in the service account prefix range
func validateCreator(creator string) bool {
	return !strings.HasPrefix(creator, "system.serviceaccount")
}
