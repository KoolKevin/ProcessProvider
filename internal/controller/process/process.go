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

package process

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
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

	"github.com/crossplane/provider-processprovider/apis/mygroup/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-processprovider/apis/v1alpha1"
	"github.com/crossplane/provider-processprovider/internal/features"

	processClientType "github.com/crossplane/provider-processprovider/internal/controller/process/processClientType"
)

const (
	errNotProcess   = "managed resource is not a Process custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// Un ProcessService è un wrapper a un client http che manda le richieste di
// creazione/cancellazione di processi al mio server fuori dal cluster
type ProcessService struct {
	ProcessClient processClientType.ProcessClient
}

var (
	newProcessService = func(creds []byte) (*ProcessService, error) {
		processService := ProcessService{}
		//questa configurazione di default imposta coma url del server kk-crossplane3 e porta 12345
		processService.ProcessClient.ConfigureClientDefault()
		return &processService, nil
	}
)

// Setup adds a controller that reconciles Process managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ProcessGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ProcessGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: newProcessService}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Process{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(creds []byte) (*ProcessService, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Process)
	if !ok {
		return nil, errors.New(errNotProcess)
	}

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

	return &external{service: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service *ProcessService
}

/*
QUESTO METODO è FATTO MALE!!!
dovrei riscrivere il server ed aggiungere funzionalità che mi permettano di controllare se

	-un processo esiste o meno
	-se esiste in che stato si trova

per semplicità uso la mappa qua sotto e faccio degli hack
*/
var processiCreati []string

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Process)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProcess)
	}

	trovato := false
	for _, id := range processiCreati {
		if id == cr.Spec.ForProvider.Id {
			trovato = true

			break
		}
	}

	if !trovato {
		processiCreati = append(processiCreati, cr.Spec.ForProvider.Id)
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// questo imposta il flag READY = TRUE quando faccio kubectl get process
	// imposto che sono sempre ready a prescindere
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Process)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProcess)
	}

	//c.service.ProcessClient.ConfigureClient(); 	eventualmente da configurare con i parametri della cr

	//TODO: CreaProcesso per adesso stampa e basta, per aggiornare anche lo stato sarebbe meglio che ritornasse qualcosa -> refactor
	c.service.ProcessClient.CreaProcesso(cr.Spec.ForProvider.Id)

	cr.Status.AtProvider.StatoProcesso = v1alpha1.Ok //per adesso lo stato lo impostiamo sempre ad ok e basta

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// NON IMPLEMENTATO
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Process)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProcess)
	}

	fmt.Printf("Updating: %+v", cr)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Process)
	if !ok {
		return errors.New(errNotProcess)
	}

	c.service.ProcessClient.EliminaProcesso(cr.Spec.ForProvider.Id)

	// altro hack per observe, se non elimino l'elemento dallo slice continuo a mandare richieste di eliminazione
	indiceDaEliminare := -1
	for i, id := range processiCreati {
		if id == cr.Spec.ForProvider.Id {
			indiceDaEliminare = i

			break
		}
	}

	//assumo di trovare sempre il processo e quindi che indiceDaEliminare sia sempre != -1
	prima := processiCreati[:indiceDaEliminare]
	dopo := processiCreati[indiceDaEliminare+1:]
	processiCreati = append(prima, dopo...)

	return nil
}
