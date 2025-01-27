package service

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmfp/account-operator/api/v1alpha1"
)

type Servicer interface {
	GetFirstLevelAccountForAccount(ctx context.Context, accountKey client.ObjectKey) (*v1alpha1.Account, error)
	GetFirstLevelAccountForNamespace(ctx context.Context, namespace string) (*v1alpha1.Account, error)

	GetAccount(ctx context.Context, accountKey client.ObjectKey) (*v1alpha1.Account, error)
	GetAccountForNamespace(ctx context.Context, namespace string) (*v1alpha1.Account, error)
}

var _ Servicer = (*Service)(nil)

type Service struct {
	client        client.Client
	rootNamespace string
}

func NewService(client client.Client, rootNamespace string) *Service {
	return &Service{
		client:        client,
		rootNamespace: rootNamespace,
	}
}

func (s *Service) getAccountOwnerAndNamespaceForNamespace(ctx context.Context, namespace string) (string, string, error) {
	var ns corev1.Namespace
	err := s.client.Get(ctx, client.ObjectKey{Name: namespace}, &ns)
	if err != nil {
		return "", "", err
	}

	if ns.Labels == nil {
		return "", "", errors.New("namespace does not have a label and therefore no connected account")
	}

	accountNamespace, ok := ns.Labels[v1alpha1.NamespaceAccountOwnerNamespaceLabel]
	if !ok || accountNamespace == "" {
		return "", "", errors.New("namespace does not have an account-owner-namespace label and therefore no connected account")
	}

	accountName, ok := ns.Labels[v1alpha1.NamespaceAccountOwnerLabel]
	if !ok || accountName == "" {
		return "", "", errors.New("namespace does not have an account-owner label and therefore no connected account")
	}

	return accountName, accountNamespace, nil
}

func (s *Service) GetFirstLevelAccountForAccount(ctx context.Context, accountKey client.ObjectKey) (*v1alpha1.Account, error) {
	return s.GetFirstLevelAccountForNamespace(ctx, accountKey.Namespace)
}

func (s *Service) GetFirstLevelAccountForNamespace(ctx context.Context, namespace string) (*v1alpha1.Account, error) {

	accountName, accountNamespace, err := s.getAccountOwnerAndNamespaceForNamespace(ctx, namespace)
	if err != nil {
		return nil, err
	}

	if s.rootNamespace != accountNamespace {
		return s.GetFirstLevelAccountForNamespace(ctx, accountNamespace)
	}

	var account v1alpha1.Account
	err = s.client.Get(ctx, client.ObjectKey{Name: accountName, Namespace: accountNamespace}, &account)
	return &account, err
}

func (s *Service) GetAccount(ctx context.Context, accountKey client.ObjectKey) (*v1alpha1.Account, error) {
	var account v1alpha1.Account
	err := s.client.Get(ctx, accountKey, &account)
	return &account, err
}

func (s *Service) GetAccountForNamespace(ctx context.Context, namespace string) (*v1alpha1.Account, error) {
	accountName, accountNamespace, err := s.getAccountOwnerAndNamespaceForNamespace(ctx, namespace)
	if err != nil {
		return nil, err
	}

	var account v1alpha1.Account
	err = s.client.Get(ctx, client.ObjectKey{Name: accountName, Namespace: accountNamespace}, &account)
	return &account, err
}
