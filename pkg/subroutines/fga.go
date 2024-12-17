package subroutines

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	"github.com/openmfp/golang-commons/fga/helpers"
	"github.com/openmfp/golang-commons/logger"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/service"
)

type FGASubroutine struct {
	client          openfgav1.OpenFGAServiceClient
	srv             service.Servicer
	rootNamespace   string
	objectType      string
	parentRelation  string
	creatorRelation string
}

func NewFGASubroutine(cl openfgav1.OpenFGAServiceClient, s service.Servicer, rootNamespace, creatorRelation, parentRealtion, objectType string) *FGASubroutine {
	return &FGASubroutine{
		client:          cl,
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

	if account.GetNamespace() != e.rootNamespace {
		parent, err := e.srv.GetAccountForNamespace(ctx, account.GetNamespace())
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

		_, err = e.client.Write(ctx, &openfgav1.WriteRequest{
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
		parent, err := e.srv.GetAccountForNamespace(ctx, account.GetNamespace())
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

		_, err = e.client.Write(ctx, &openfgav1.WriteRequest{
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
		a, err := e.srv.GetFirstLevelAccountForNamespace(ctx, account.Namespace)
		if err != nil {
			return "", err
		}

		firstLevelAccountName = a.Name
	}

	storeId, err := helpers.GetStoreIDForTenant(ctx, e.client, firstLevelAccountName)
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
