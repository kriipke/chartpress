package operator

import (
	"context"

	"github.com/kriipke/chartpress/internal/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// dynamicCRClient implements CRClient over a dynamic client for apis.GVR.
type dynamicCRClient struct{ client dynamic.Interface }

func newDynamicCRClient(client dynamic.Interface) *dynamicCRClient {
	return &dynamicCRClient{client: client}
}

func (c *dynamicCRClient) Update(ctx context.Context, ns string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.client.Resource(apis.GVR).Namespace(ns).Update(ctx, obj, metav1.UpdateOptions{})
}

func (c *dynamicCRClient) UpdateStatus(ctx context.Context, ns string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.client.Resource(apis.GVR).Namespace(ns).UpdateStatus(ctx, obj, metav1.UpdateOptions{})
}

func (c *dynamicCRClient) Delete(ctx context.Context, ns, name string) error {
	return c.client.Resource(apis.GVR).Namespace(ns).Delete(ctx, name, metav1.DeleteOptions{})
}
