package v1alpha1

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Service interface {
	GetFirstLevelAccountForAccount(ctx context.Context, accountKey client.ObjectKey) (*Account, error)
	GetFirstLevelAccountForNamespace(ctx context.Context, namespace string) (*Account, error)

	GetAccount(ctx context.Context, accountKey client.ObjectKey) (*Account, error)
	GetAccountForNamespace(ctx context.Context, namespace string) (*Account, error)
}

var _ Service = (*service)(nil)

type service struct {
	client        client.Client
	rootNamespace string
}

func NewService(client client.Client, rootNamespace string) *service {
	return &service{
		client:        client,
		rootNamespace: rootNamespace,
	}
}

func (s *service) getAccountOwnerAndNamespaceForNamespace(ctx context.Context, namespace string) (string, string, error) {
	var ns corev1.Namespace
	err := s.client.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
	if err != nil {
		return "", "", err
	}

	if ns.Labels == nil {
		return "", "", errors.New("namespace does not have a label and therefore no connected account")
	}

	accountNamespace, ok := ns.Labels[NamespaceAccountOwnerNamespaceLabel]
	if !ok || accountNamespace == "" {
		return "", "", errors.New("namespace does not have an account-owner-namespace label and therefore no connected account")
	}

	accountName, ok := ns.Labels[NamespaceAccountOwnerLabel]
	if !ok || accountName == "" {
		return "", "", errors.New("namespace does not have an account-owner label and therefore no connected account")
	}

	return accountName, accountNamespace, nil
}

func (s *service) GetFirstLevelAccountForAccount(ctx context.Context, accountKey client.ObjectKey) (*Account, error) {
	return s.GetFirstLevelAccountForNamespace(ctx, accountKey.Namespace)
}

func (s *service) GetFirstLevelAccountForNamespace(ctx context.Context, namespace string) (*Account, error) {

	accountName, accountNamespace, err := s.getAccountOwnerAndNamespaceForNamespace(ctx, namespace)
	if err != nil {
		return nil, err
	}

	if s.rootNamespace != accountNamespace {
		return s.GetFirstLevelAccountForNamespace(ctx, accountNamespace)
	}

	var account Account
	err = s.client.Get(ctx, client.ObjectKey{Name: accountName, Namespace: accountNamespace}, &account)
	return &account, err
}

func (s *service) GetAccount(ctx context.Context, accountKey client.ObjectKey) (*Account, error) {
	var account Account
	err := s.client.Get(ctx, accountKey, &account)
	return &account, err
}

func (s *service) GetAccountForNamespace(ctx context.Context, namespace string) (*Account, error) {
	accountName, accountNamespace, err := s.getAccountOwnerAndNamespaceForNamespace(ctx, namespace)
	if err != nil {
		return nil, err
	}

	var account Account
	err = s.client.Get(ctx, client.ObjectKey{Name: accountName, Namespace: accountNamespace}, &account)
	return &account, err
}
