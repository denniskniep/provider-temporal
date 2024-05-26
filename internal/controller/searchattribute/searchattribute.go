/*
Copyright 2022 The Crossplane Authors.

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

package searchattribute

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/sync/syncmap"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/denniskniep/provider-temporal/apis/core/v1alpha1"
	apisv1alpha1 "github.com/denniskniep/provider-temporal/apis/v1alpha1"
	temporal "github.com/denniskniep/provider-temporal/internal/clients"
	"github.com/denniskniep/provider-temporal/internal/features"
)

const (
	errNotSearchAttribute = "managed resource is not a SearchAttribute custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errDescribe           = "failed to describe SearchAttribute resource"
	errNewClient          = "cannot create new Service"
	errMapping            = "failed to map SearchAttribute resource as comparable"
	errCreate             = "failed to create SearchAttribute resource"
	errUpdate             = "failed to update SearchAttribute resource"
	errDelete             = "failed to delete SearchAttribute resource"
)

// Setup adds a controller that reconciles SearchAttribute managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	o.Logger.Info("Setup Controller: SearchAttribute")
	name := managed.ControllerName(v1alpha1.SearchAttributeGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.SearchAttributeGroupVersionKind),
		managed.WithExternalConnectDisconnecter(&connector{
			externalClientsByCreds: syncmap.Map{},
			kube:                   mgr.GetClient(),
			usage:                  resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn:           temporal.NewSearchAttributeService,
			logger:                 o.Logger.WithValues("controller", name)}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithInitializers(),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.SearchAttribute{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                   client.Client
	usage                  resource.Tracker
	logger                 logging.Logger
	externalClientsByCreds syncmap.Map
	newServiceFn           func(creds []byte) (temporal.SearchAttributeService, error)
}

func hash(content []byte) string {
	h := sha256.New()
	h.Write(content)
	sha := h.Sum(nil)
	shaStr := hex.EncodeToString(sha)
	return shaStr
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	logger := c.logger.WithValues("method", "connect")
	logger.Debug("Start Connect")
	cr, ok := mg.(*v1alpha1.SearchAttribute)
	if !ok {
		return nil, errors.New(errNotSearchAttribute)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	creds, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	credHash := hash(creds)

	svc, err := c.newServiceFn(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	ext := &external{service: svc, logger: c.logger, id: uuid.New().String()}
	value, ok := c.externalClientsByCreds.LoadOrStore(credHash, ext)
	if ok {
		ext.service.Close()
		ext = value.(*external)
		logger.Debug("Use existing " + ext.id)
	} else {
		logger.Debug("Connected " + ext.id)
	}

	ext.IncrementUsageCounter()
	return ext, nil
}

func (c *connector) Disconnect(ctx context.Context) error {
	logger := c.logger.WithValues("method", "disconnect")
	logger.Debug("Start Disconnect")

	c.externalClientsByCreds.Range(func(key, value interface{}) bool {

		ext := value.(*external)
		ext.DecrementUsageCounter()
		if ext.GetUsageCounter() < 0 {
			ext.SetUsageCounter(0)
		}

		if ext.GetUsageCounter() == 0 && ext.service != nil {
			ext.service.Close()
			c.externalClientsByCreds.LoadAndDelete(key)
			logger.Debug("Disconnected " + ext.id)
		} else {
			logger.Debug("Keep connection " + ext.id)
		}

		// this will continue iterating
		return true
	})

	return nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service      temporal.SearchAttributeService
	logger       logging.Logger
	id           string
	usageCounter int
	sync.RWMutex
}

func (c *external) GetUsageCounter() int {
	c.RLock()
	defer c.RUnlock()
	return c.usageCounter
}

func (c *external) IncrementUsageCounter() {
	c.Lock()
	defer c.Unlock()
	c.usageCounter++
}

func (c *external) DecrementUsageCounter() {
	c.Lock()
	defer c.Unlock()
	c.usageCounter--
}

func (c *external) SetUsageCounter(usageCounter int) {
	c.Lock()
	defer c.Unlock()
	c.usageCounter = usageCounter
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	logger := c.logger.WithValues("method", "observe", "serviceId", c.id)
	logger.Debug("Start observe")
	cr, ok := mg.(*v1alpha1.SearchAttribute)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSearchAttribute)
	}

	externalName := meta.GetExternalName(cr)
	c.logger.Debug("ExternalName: '" + externalName + "'")

	if cr.Spec.ForProvider.TemporalNamespaceName == nil {
		return managed.ExternalObservation{}, errors.New("TemporalNamespaceName not set")
	}

	observed, err := c.service.DescribeSearchAttributeByName(ctx, *cr.Spec.ForProvider.TemporalNamespaceName, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errDescribe)
	}

	if observed == nil {
		c.logger.Debug("Managed resource '" + cr.Name + "' does not exist")
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	c.logger.Debug("Found '" + observed.Name + "' in namespace '" + observed.TemporalNamespaceName + "'")

	// Update Status
	cr.Status.AtProvider = *observed
	cr.SetConditions(xpv1.Available().WithMessage("SearchAttribute exists"))

	observedCompareable, err := c.service.MapToSearchAttributeCompare(observed)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errMapping)
	}

	specCompareable, err := c.service.MapToSearchAttributeCompare(&cr.Spec.ForProvider)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errMapping)
	}

	diff := ""
	resourceUpToDate := cmp.Equal(specCompareable, observedCompareable)

	// Compare Spec with observed
	if !resourceUpToDate {
		diff = cmp.Diff(specCompareable, observedCompareable)
	}
	c.logger.Debug("Managed resource '" + cr.Name + "' upToDate: " + strconv.FormatBool(resourceUpToDate) + "")

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        resourceUpToDate,
		Diff:                    diff,
		ResourceLateInitialized: false,
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	logger := c.logger.WithValues("method", "create", "serviceId", c.id)
	logger.Debug("Start create")
	cr, ok := mg.(*v1alpha1.SearchAttribute)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSearchAttribute)
	}

	err := c.service.CreateSearchAttribute(ctx, &cr.Spec.ForProvider)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreate)
	}

	meta.SetExternalName(cr, *cr.Spec.ForProvider.TemporalNamespaceName+"."+cr.Spec.ForProvider.Name)
	c.logger.Debug("Managed resource '" + meta.GetExternalName(cr) + "' created")

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	logger := c.logger.WithValues("method", "update", "serviceId", c.id)
	logger.Debug("Start update")
	cr, ok := mg.(*v1alpha1.SearchAttribute)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotSearchAttribute)
	}

	return managed.ExternalUpdate{}, errors.New("Search Attribute '" + meta.GetExternalName(cr) + "' can not be updated! All properties are immutable!")
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	logger := c.logger.WithValues("method", "delete", "serviceId", c.id)
	logger.Debug("Start delete")
	cr, ok := mg.(*v1alpha1.SearchAttribute)
	if !ok {
		return errors.New(errNotSearchAttribute)
	}

	err := c.service.DeleteSearchAttributeByName(ctx, *cr.Spec.ForProvider.TemporalNamespaceName, cr.Spec.ForProvider.Name)

	if err != nil {
		return errors.Wrap(err, errDelete)
	}

	c.logger.Debug("Managed resource '" + meta.GetExternalName(cr) + "' deleted")
	return nil
}
