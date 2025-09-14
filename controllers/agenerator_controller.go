package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	secretsv1alpha1 "github.com/example/another-secrets-operator/api/v1alpha1"
)

// AGeneratorReconciler reconciles a AGenerator object
type AGeneratorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=agenerators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=agenerators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=agenerators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *AGeneratorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("agenerator", req.NamespacedName)
	log.Info("Reconciling AGenerator")

	// Fetch the AGenerator instance
	var aGenerator secretsv1alpha1.AGenerator
	if err := r.Get(ctx, req.NamespacedName, &aGenerator); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// AGenerator is a passive object that is referenced by ASecret resources
	// No active reconciliation is needed besides validation

	// Validate the generator specification
	if err := validateGeneratorSpec(aGenerator.Spec); err != nil {
		log.Error(err, "Invalid generator specification")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AGeneratorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1alpha1.AGenerator{}).
		Complete(r)
}
