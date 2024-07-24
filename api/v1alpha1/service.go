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

func (s *service) GetFirstLevelAccountForAccount(ctx context.Context, accountKey client.ObjectKey) (*Account, error) {
	return s.GetAccountForNamespace(ctx, accountKey.Namespace)
}

func (s *service) GetFirstLevelAccountForNamespace(ctx context.Context, namespace string) (*Account, error) {
	var ns corev1.Namespace
	err := s.client.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
	if err != nil {
		return nil, err
	}

	if ns.Labels == nil {
		return nil, errors.New("namespace does not have a label and therefore no connected account")
	}

	accountNamespace, ok := ns.Labels[NamespaceAccountOwnerNamespaceLabel]
	if !ok || accountNamespace == "" {
		return nil, errors.New("namespace does not have an account-owner-namespace label and therefore no connected account")
	}

	if s.rootNamespace != accountNamespace {
		return s.GetFirstLevelAccountForNamespace(ctx, accountNamespace)
	}

	accountName, ok := ns.Labels[NamespaceAccountOwnerLabel]
	if !ok || accountName == "" {
		return nil, errors.New("namespace does not have an account-owner label and therefore no connected account")
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
	var ns corev1.Namespace
	err := s.client.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
	if err != nil {
		return nil, err
	}

	if ns.Labels == nil {
		return nil, errors.New("namespace does not have a label and therefore no connected account")
	}

	accountName, ok := ns.Labels[NamespaceAccountOwnerLabel]
	if !ok || accountName == "" {
		return nil, errors.New("namespace does not have an account-owner label and therefore no connected account")
	}

	accountNamespace, ok := ns.Labels[NamespaceAccountOwnerNamespaceLabel]
	if !ok || accountNamespace == "" {
		return nil, errors.New("namespace does not have an account-owner-namespace label and therefore no connected account")
	}

	var account Account
	err = s.client.Get(ctx, client.ObjectKey{Name: accountName, Namespace: accountNamespace}, &account)
	return &account, err
}
