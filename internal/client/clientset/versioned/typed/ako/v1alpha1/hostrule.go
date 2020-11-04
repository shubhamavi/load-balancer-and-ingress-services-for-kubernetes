/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"time"

	v1alpha1 "github.com/shubhamavi/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1alpha1"
	scheme "github.com/shubhamavi/load-balancer-and-ingress-services-for-kubernetes/internal/client/clientset/versioned/scheme"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// HostRulesGetter has a method to return a HostRuleInterface.
// A group's client should implement this interface.
type HostRulesGetter interface {
	HostRules(namespace string) HostRuleInterface
}

// HostRuleInterface has methods to work with HostRule resources.
type HostRuleInterface interface {
	Create(*v1alpha1.HostRule) (*v1alpha1.HostRule, error)
	Update(*v1alpha1.HostRule) (*v1alpha1.HostRule, error)
	UpdateStatus(*v1alpha1.HostRule) (*v1alpha1.HostRule, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.HostRule, error)
	List(opts v1.ListOptions) (*v1alpha1.HostRuleList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.HostRule, err error)
	HostRuleExpansion
}

// hostRules implements HostRuleInterface
type hostRules struct {
	client rest.Interface
	ns     string
}

// newHostRules returns a HostRules
func newHostRules(c *AkoV1alpha1Client, namespace string) *hostRules {
	return &hostRules{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the hostRule, and returns the corresponding hostRule object, and an error if there is any.
func (c *hostRules) Get(name string, options v1.GetOptions) (result *v1alpha1.HostRule, err error) {
	result = &v1alpha1.HostRule{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("hostrules").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of HostRules that match those selectors.
func (c *hostRules) List(opts v1.ListOptions) (result *v1alpha1.HostRuleList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.HostRuleList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("hostrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested hostRules.
func (c *hostRules) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("hostrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a hostRule and creates it.  Returns the server's representation of the hostRule, and an error, if there is any.
func (c *hostRules) Create(hostRule *v1alpha1.HostRule) (result *v1alpha1.HostRule, err error) {
	result = &v1alpha1.HostRule{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("hostrules").
		Body(hostRule).
		Do().
		Into(result)
	return
}

// Update takes the representation of a hostRule and updates it. Returns the server's representation of the hostRule, and an error, if there is any.
func (c *hostRules) Update(hostRule *v1alpha1.HostRule) (result *v1alpha1.HostRule, err error) {
	result = &v1alpha1.HostRule{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("hostrules").
		Name(hostRule.Name).
		Body(hostRule).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *hostRules) UpdateStatus(hostRule *v1alpha1.HostRule) (result *v1alpha1.HostRule, err error) {
	result = &v1alpha1.HostRule{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("hostrules").
		Name(hostRule.Name).
		SubResource("status").
		Body(hostRule).
		Do().
		Into(result)
	return
}

// Delete takes name of the hostRule and deletes it. Returns an error if one occurs.
func (c *hostRules) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("hostrules").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *hostRules) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("hostrules").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched hostRule.
func (c *hostRules) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.HostRule, err error) {
	result = &v1alpha1.HostRule{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("hostrules").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
