package temporalnamespace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
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
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/denniskniep/provider-temporal/apis/core/v1alpha1"
	apisv1alpha1 "github.com/denniskniep/provider-temporal/apis/v1alpha1"
	temporal "github.com/denniskniep/provider-temporal/internal/clients"
	"github.com/denniskniep/provider-temporal/internal/features"
)

const (
	errNotTemporalNamespace = "managed resource is not a TemporalNamespace custom resource"
	errTrackPCUsage         = "cannot track ProviderConfig usage"
	errGetPC                = "cannot get ProviderConfig"
	errGetCreds             = "cannot get credentials"

	errNewClient = "cannot create new Service"
	errDescribe  = "failed to describe Namespace resource"
	errCreate    = "failed to create Namespace resource"
	errUpdate    = "failed to update Namespace resource"
	errDelete    = "failed to delete Namespace resource"
	errMapping   = "failed to map Namespace resource"
)

// Setup adds a controller that reconciles TemporalNamespace managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	o.Logger.Info("Setup Controller: TemporalNamespace")
	name := managed.ControllerName(v1alpha1.TemporalNamespaceGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.TemporalNamespaceGroupVersionKind),
		managed.WithExternalConnectDisconnecter(&connector{
			externalClientsByCreds: syncmap.Map{},
			kube:                   mgr.GetClient(),
			usage:                  resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn:           temporal.NewNamespaceService,
			logger:                 o.Logger.WithValues("controller", name)}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithInitializers(),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.TemporalNamespace{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                   client.Client
	usage                  resource.Tracker
	logger                 logging.Logger
	externalClientsByCreds syncmap.Map
	newServiceFn           func(creds []byte) (temporal.NamespaceService, error)
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
	cr, ok := mg.(*v1alpha1.TemporalNamespace)
	if !ok {
		return nil, errors.New(errNotTemporalNamespace)
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

	ext.usageCounter++
	return ext, nil
}

func (c *connector) Disconnect(ctx context.Context) error {
	logger := c.logger.WithValues("method", "disconnect")
	logger.Debug("Start Disconnect")

	c.externalClientsByCreds.Range(func(key, value interface{}) bool {

		ext := value.(*external)
		ext.usageCounter--
		if ext.usageCounter < 0 {
			ext.usageCounter = 0
		}

		if ext.usageCounter == 0 && ext.service != nil {
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
	service      temporal.NamespaceService
	logger       logging.Logger
	id           string
	usageCounter int
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	logger := c.logger.WithValues("method", "observe", "serviceId", c.id)
	logger.Debug("Start observe")
	cr, ok := mg.(*v1alpha1.TemporalNamespace)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTemporalNamespace)
	}

	externalName := meta.GetExternalName(cr)
	c.logger.Debug("ExternalName: '" + externalName + "'")

	observed, err := c.service.DescribeNamespaceByName(ctx, cr.Spec.ForProvider.Name)
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

	c.logger.Debug("Found '" + observed.Name + "' with id '" + observed.Id + "'")

	// Update Status
	cr.Status.AtProvider = *observed

	if observed.State == "Registered" {
		cr.SetConditions(xpv1.Available().WithMessage("Namespace.State = " + observed.State))
	}

	if observed.State == "Unspecified" {
		cr.SetConditions(xpv1.Unavailable().WithMessage("Namespace.State = " + observed.State))
	}

	if observed.State == "Deleted" {
		cr.SetConditions(xpv1.Deleting().WithMessage("Namespace.State = " + observed.State))
	}

	observedCompareable, err := c.service.MapToNamespaceCompare(observed)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errMapping)
	}

	specCompareable, err := c.service.MapToNamespaceCompare(&cr.Spec.ForProvider)
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
	cr, ok := mg.(*v1alpha1.TemporalNamespace)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTemporalNamespace)
	}

	err := c.service.CreateNamespace(ctx, &cr.Spec.ForProvider)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreate)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.Name)
	c.logger.Debug("Managed resource '" + cr.Name + "' created")

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	logger := c.logger.WithValues("method", "update", "serviceId", c.id)
	logger.Debug("Start update")
	cr, ok := mg.(*v1alpha1.TemporalNamespace)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTemporalNamespace)
	}

	err := c.service.UpdateNamespaceByName(ctx, &cr.Spec.ForProvider)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdate)
	}

	c.logger.Debug("Managed resource '" + cr.Name + "' updated")
	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	logger := c.logger.WithValues("method", "delete", "serviceId", c.id)
	logger.Debug("Start delete")
	cr, ok := mg.(*v1alpha1.TemporalNamespace)
	if !ok {
		return errors.New(errNotTemporalNamespace)
	}

	_, err := c.service.DeleteNamespaceByName(ctx, cr.Spec.ForProvider.Name)

	if err != nil {
		return errors.Wrap(err, errDelete)
	}

	c.logger.Debug("Managed resource '" + cr.Name + "' deleted")
	return nil
}
