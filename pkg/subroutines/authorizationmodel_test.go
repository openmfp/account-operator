package subroutines_test

import (
	"context"
	"testing"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/account-operator/pkg/subroutines/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAuthorizationModelProcess(t *testing.T) {

	scheme := runtime.NewScheme()
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	tests := []struct {
		name               string
		k8sMocks           func(*mocks.Client)
		authorizationModel corev1alpha1.AuthorizationModel
	}{
		{
			name: "should process authorization model and set owner reference",
			k8sMocks: func(mockClient *mocks.Client) {
				mockClient.EXPECT().
					Get(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
						store := o.(*corev1alpha1.Store)

						store.SetNamespace("default")
						store.SetName("store")

						return nil
					})
				mockClient.EXPECT().Scheme().Return(scheme)
			},
			authorizationModel: corev1alpha1.AuthorizationModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "authorizationmodel",
					Namespace: "default",
				},
				Spec: corev1alpha1.AuthorizationModelSpec{
					StoreRef: corev1.LocalObjectReference{
						Name: "store",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			mockClient := mocks.NewClient(t)
			if test.k8sMocks != nil {
				test.k8sMocks(mockClient)
			}

			subroutine := subroutines.NewAuthorizationModelSubroutine(mockClient)

			_, operatorErr := subroutine.Process(context.Background(), &test.authorizationModel)
			require.Nil(t, operatorErr)

			require.Len(t, test.authorizationModel.GetOwnerReferences(), 1)

			ownerReference := test.authorizationModel.GetOwnerReferences()[0]
			require.Equal(t, "Store", ownerReference.Kind)
			require.Equal(t, "store", ownerReference.Name)
		})
	}
}

func TestAuthorizationModelFinalize(t *testing.T) {

	scheme := runtime.NewScheme()
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	tests := []struct {
		name               string
		k8sMocks           func(*mocks.Client)
		authorizationModel corev1alpha1.AuthorizationModel
	}{
		{
			name: "should process authorization model and set owner reference",
			k8sMocks: func(mockClient *mocks.Client) {
				mockClient.EXPECT().
					Get(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
						store := o.(*corev1alpha1.Store)

						store.SetNamespace("default")
						store.SetName("store")

						return nil
					})
				mockClient.EXPECT().Scheme().Return(scheme)
			},
			authorizationModel: corev1alpha1.AuthorizationModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "authorizationmodel",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "core.openmfp.io/v1alpha1",
							Kind:       "Store",
							Name:       "store",
						},
					},
				},
				Spec: corev1alpha1.AuthorizationModelSpec{
					StoreRef: corev1.LocalObjectReference{
						Name: "store",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			mockClient := mocks.NewClient(t)
			if test.k8sMocks != nil {
				test.k8sMocks(mockClient)
			}

			subroutine := subroutines.NewAuthorizationModelSubroutine(mockClient)

			_, operatorErr := subroutine.Finalize(context.Background(), &test.authorizationModel)
			require.Nil(t, operatorErr)

			require.Len(t, test.authorizationModel.GetOwnerReferences(), 0)
		})
	}
}
