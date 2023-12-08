package namespace

import (
	"context"
	"strconv"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	"github.com/denniskniep/provider-temporal/internal/features"
	temporal "github.com/denniskniep/provider-temporal/internal/temporal"
)

const (
	errNotNamespace = "managed resource is not a Namespace custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles Namespace managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.NamespaceGroupKind)
	o.Logger.Info("Controller.Setup - Namespace")
	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.NamespaceGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: temporal.NewTemporalService,
			logger:       o.Logger.WithValues("controller", name)}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Namespace{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	logger       logging.Logger
	newServiceFn func(creds []byte) (temporal.TemporalService, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Namespace)
	if !ok {
		return nil, errors.New(errNotNamespace)
	}

	c.logger.Info("Controller.Connect - Namespace")

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials

	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := c.newServiceFn(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc, logger: c.logger}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service temporal.TemporalService
	logger  logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	c.logger.Info("Controller.Observe - Namespace")
	cr, ok := mg.(*v1alpha1.Namespace)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotNamespace)
	}

	currentSpec := cr.Spec.ForProvider.DeepCopy()
	id := meta.GetExternalName(cr)

	if id == "" {
		return c.doesNotExist(cr), nil
	}

	observed, err := c.service.DescribeNamespaceById(ctx, id)

	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if observed == nil {
		return c.doesNotExist(cr), nil
	}

	// Update Status
	cr.Status.AtProvider = *observed

	diff := ""
	resourceUpToDate := cmp.Equal(cr.Spec.ForProvider, observed)

	// Compare Spec with observed
	if !resourceUpToDate {
		diff = cmp.Diff(cr.Spec.ForProvider, observed)

	}
	c.logger.Debug("Controller.Observe - Managed resource '" + cr.Name + "' upToDate: " + strconv.FormatBool(resourceUpToDate) + "")

	specUpdatedDuringObservation := !cmp.Equal(currentSpec, &cr.Spec.ForProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        resourceUpToDate,
		Diff:                    diff,
		ResourceLateInitialized: specUpdatedDuringObservation,
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

func (c *external) doesNotExist(cr *v1alpha1.Namespace) managed.ExternalObservation {
	c.logger.Debug("Controller.Observe - Managed resource '" + cr.Name + "' does not exist")
	return managed.ExternalObservation{
		ResourceExists:    false,
		ResourceUpToDate:  false,
		ConnectionDetails: managed.ConnectionDetails{},
	}
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	c.logger.Debug("Controller.Create")
	cr, ok := mg.(*v1alpha1.Namespace)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotNamespace)
	}

	created, err := c.service.CreateNamespace(ctx, &cr.Spec.ForProvider)

	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, created.Id)
	c.logger.Debug("Controller.Create - Managed resource '" + cr.Name + "' created")
	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	c.logger.Debug("Controller.Update")
	cr, ok := mg.(*v1alpha1.Namespace)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotNamespace)
	}

	id := meta.GetExternalName(cr)

	_, err := c.service.UpdateNamespaceById(ctx, id, &cr.Spec.ForProvider)

	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	c.logger.Debug("Controller.Update - Managed resource '" + cr.Name + "' updated")
	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	c.logger.Debug("Controller.Delete")
	cr, ok := mg.(*v1alpha1.Namespace)
	if !ok {
		return errors.New(errNotNamespace)
	}

	id := meta.GetExternalName(cr)

	err := c.service.DeleteNamespaceById(ctx, id)

	if err != nil {
		return err
	}

	c.logger.Debug("Controller.Create - Managed resource '" + cr.Name + "' deleted")
	return nil
}
