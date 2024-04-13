/*
Copyright 2024 Sourav Patnaik.

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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	securityv1beta1 "github.com/sourav977/stale-secrets-watch/api/v1beta1"
)

// StaleSecretWatchReconciler reconciles a StaleSecretWatch object
type StaleSecretWatchReconciler struct {
	client.Client
	Log             logr.Logger
	RequeueInterval time.Duration
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
}

const (
	typeAvailable             = "Available"
	typeDegraded              = "Degraded"
	typeUnavailable           = "Unavailable"
	errGetSSW                 = "could not get StaleSecretWatch"
	errNSnotEmpty             = "staleSecretToWatch.namespace cannot be empty"
	stalesecretwatchFinalizer = "security.stalesecretwatch.io/finalizer"
)

//+kubebuilder:rbac:groups=security.stalesecretwatch.io,resources=stalesecretwatches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=security.stalesecretwatch.io,resources=stalesecretwatches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=security.stalesecretwatch.io,resources=stalesecretwatches/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets/status,verbs=get
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the StaleSecretWatch object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *StaleSecretWatchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("stalesecretwatch", req.NamespacedName)

	logger.Info("Reconcile called")

	var staleSecretWatch securityv1beta1.StaleSecretWatch
	r.Recorder.Event(&staleSecretWatch, "Normal", "ReconcileStarted", "Reconciliation process started")

	// Fetch the StaleSecretWatch instance
	// The purpose is check if the Custom Resource for the Kind StaleSecretWatch
	// is applied on the cluster if not we return nil to stop the reconciliation
	if err := r.Get(ctx, req.NamespacedName, &staleSecretWatch); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("StaleSecretWatch resource not found. Ignoring since StaleSecretWatch object must be deleted. Exit Reconcile.")
			r.Recorder.Event(&staleSecretWatch, "Normal", "NotFound", "StaleSecretWatch resource not found. Ignoring and exiting reconcile loop.")
			// Object not found, return without requeueing
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		// Error fetching the StaleSecretWatch instance, requeue the request
		logger.Error(err, errGetSSW)
		return ctrl.Result{}, err
	}

	if err := r.updateStatusCondition(ctx, &staleSecretWatch, "ReconcileStarted", metav1.ConditionTrue, "ReconcileInitiated", "Reconciliation process has started"); err != nil {
		return ctrl.Result{}, err
	}

	// Check if the instance is marked to be deleted
	if staleSecretWatch.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(&staleSecretWatch, stalesecretwatchFinalizer) {
			logger.Info("Performing Finalizer Operations")
			r.Recorder.Event(&staleSecretWatch, "Normal", "FinalizerOpsStarted", "Starting finalizer operations")
			r.doFinalizerOperationsForStaleSecretWatch(&staleSecretWatch)
			r.Recorder.Event(&staleSecretWatch, "Normal", "FinalizerOpsComplete", "Finalizer operations completed")
			if err := r.updateStatusCondition(ctx, &staleSecretWatch, "FinalizerProcessing", metav1.ConditionTrue, "FinalizerStarted", "Performing finalizer operations before deleting resource"); err != nil {
				return ctrl.Result{}, err
			}

			// Refetch latest before final removal of finalizer
			if err := r.Get(ctx, req.NamespacedName, &staleSecretWatch); err != nil {
				logger.Error(err, "Failed to fetch StaleSecretWatch before finalizer removal")
				return ctrl.Result{}, err
			}

			logger.Info("Removing Finalizer for staleSecretWatch after successfully perform the operations")
			controllerutil.RemoveFinalizer(&staleSecretWatch, stalesecretwatchFinalizer)
			if err := r.Update(ctx, &staleSecretWatch); err != nil {
				logger.Error(err, "Failed to remove finalizer")
				r.Recorder.Event(&staleSecretWatch, "Warning", "FinalizerRemovalFailed", "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
			r.Recorder.Event(&staleSecretWatch, "Normal", "FinalizerRemoved", "Finalizer removed, resource cleanup complete"+"custom resource "+staleSecretWatch.Name+" deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}

	// Add a finalizer if it does not exist
	if !controllerutil.ContainsFinalizer(&staleSecretWatch, stalesecretwatchFinalizer) {
		logger.Info("Adding Finalizer for staleSecretWatch")
		controllerutil.AddFinalizer(&staleSecretWatch, stalesecretwatchFinalizer)
		if err := r.Update(ctx, &staleSecretWatch); err != nil {
			logger.Error(err, "Failed to add finalizer")
			r.Recorder.Event(&staleSecretWatch, "Warning", "FinalizerAdditionFailed", "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&staleSecretWatch, "Normal", "FinalizerAdded", "Finalizer added to CR successfully")
		if err := r.updateStatusCondition(ctx, &staleSecretWatch, "FinalizerAdded", metav1.ConditionTrue, "FinalizerAdded", "Finalizer added successfully to StaleSecretWatch"); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the ConfigMap already exists, if not create a new one
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Name: "hashed-secrets-stalesecretwatch", Namespace: "default"}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("ConfigMap not found, creating new one")
			newCM, err := r.configMapForStaleSecretWatch(&staleSecretWatch)
			if err != nil {
				logger.Error(err, "Failed to define new ConfigMap for StaleSecretWatch")
				r.Recorder.Event(&staleSecretWatch, "Warning", "ConfigMapCreationFailed", "Failed to define new ConfigMap")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, newCM); err != nil {
				logger.Error(err, "Failed to create new ConfigMap")
				r.Recorder.Event(&staleSecretWatch, "Warning", "ConfigMapCreationFailed", "Failed to create new ConfigMap")
				return ctrl.Result{}, err
			}
			r.Recorder.Event(&staleSecretWatch, "Normal", "ConfigMapCreated", "New ConfigMap created successfully")
			if err := r.updateStatusCondition(ctx, &staleSecretWatch, "ConfigMapCreated", metav1.ConditionTrue, "ConfigMapCreated", "New ConfigMap created successfully for StaleSecretWatch"); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "Failed to get ConfigMap")
		r.Recorder.Event(&staleSecretWatch, "Warning", "ConfigMapFetchFailed", "Failed to fetch ConfigMap")
		return ctrl.Result{}, err
	}

	//now prepare the namespace and secret list to watch
	secretsToWatch, err := r.prepareWatchList(ctx, logger, &staleSecretWatch)
	if err != nil {
		logger.Error(err, "Failed to prepare watch list")
		r.Recorder.Event(&staleSecretWatch, "Warning", "PrepareWatchlistFailed", "Failed to prepare watch list")
		return ctrl.Result{}, err
	}
	if err := r.updateStatusCondition(ctx, &staleSecretWatch, "PrepareWatchlist", metav1.ConditionTrue, "PrepareWatchlist", "Successfully prepared watch list for StaleSecretWatch"); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Monitoring namespaces and secrets", "secrets", secretsToWatch)

	// Refetch the ConfigMap right before updating it to ensure it's the latest version
	err = r.Get(ctx, types.NamespacedName{Name: "hashed-secrets-stalesecretwatch", Namespace: "default"}, cm)
	if err != nil {
		logger.Error(err, "Failed to refetch the ConfigMap")
		return ctrl.Result{}, err
	}
	herr := r.calculateAndStoreHashedSecrets(ctx, logger, secretsToWatch, cm)
	if herr != nil {
		logger.Error(herr, "calculateAndStoreHashedSecrets error")
		r.Recorder.Event(&staleSecretWatch, "Warning", "HashedSecretsCalculationFailed", "Failed to calculate and store hashed secrets")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&staleSecretWatch, "Normal", "ReconcileComplete", "Reconciliation complete")
	if err := r.updateStatusCondition(ctx, &staleSecretWatch, "ReconcileComplete", metav1.ConditionTrue, "Success", "Reconciliation process completed successfully"); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 2 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StaleSecretWatchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("stale-secret-watch")
	// For defines the type of Object being *reconciled*, and configures the ControllerManagedBy to respond to create / delete /
	// update events by *reconciling the object*.
	// This is the equivalent of calling
	// Watches(&source.Kind{Type: apiType}, &handler.EnqueueRequestForObject{}).
	return ctrl.NewControllerManagedBy(mgr).
		For(&securityv1beta1.StaleSecretWatch{}).
		Owns(&corev1.ConfigMap{}, builder.OnlyMetadata).
		Watches(&corev1.Secret{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

func (r *StaleSecretWatchReconciler) updateStatusCondition(ctx context.Context, staleSecretWatch *securityv1beta1.StaleSecretWatch, conditionType string, status metav1.ConditionStatus, reason, message string) error {
	updateFunc := func() error {
		// Fetch the latest version of the resource
		latest := &securityv1beta1.StaleSecretWatch{}
		if err := r.Get(ctx, client.ObjectKey{Name: staleSecretWatch.Name, Namespace: staleSecretWatch.Namespace}, latest); err != nil {
			return err
		}

		// Update status conditions on the latest version of the resource
		meta.SetStatusCondition(&latest.Status.Conditions, metav1.Condition{
			Type:    conditionType,
			Status:  status,
			Reason:  reason,
			Message: message,
		})

		// Try to update
		return r.Status().Update(ctx, latest)
	}

	// Attempt to update with a retry on conflict
	err := retry.OnError(retry.DefaultRetry, errors.IsConflict, updateFunc)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to update StaleSecretWatch status with retry")
		return err
	}

	return nil
}

// doFinalizerOperationsForStaleSecretWatch will perform the required operations before delete the CR.
func (r *StaleSecretWatchReconciler) doFinalizerOperationsForStaleSecretWatch(ssw *securityv1beta1.StaleSecretWatch) {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	// The following implementation will raise an event
	r.Recorder.Event(ssw, "Warning", "Deleting", fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s", ssw.Name, ssw.Namespace))

}

// configMapForStaleSecretWatch returns a new stalesecretwatch ConfigMap object
func (r *StaleSecretWatchReconciler) configMapForStaleSecretWatch(ssw *securityv1beta1.StaleSecretWatch) (*corev1.ConfigMap, error) {
	ls := LabelsForStaleSecretWatchConfigMap(ssw.Name)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hashed-secrets-stalesecretwatch", // Name of the ConfigMap
			Namespace: "default",
			Labels:    ls,
		},
		Data: map[string]string{},
	}
	// this CR will manage/owner of this configmap resource
	if err := ctrl.SetControllerReference(ssw, cm, r.Scheme); err != nil {
		return nil, err
	}
	return cm, nil
}

// calculateAndStoreHashedSecrets retrives the secret data, calculate the hash and update that into configmap
func (r *StaleSecretWatchReconciler) calculateAndStoreHashedSecrets(ctx context.Context, logger logr.Logger, secretsToWatch map[string][]string, cm *corev1.ConfigMap) error {
	var configData ConfigData

	// Check if ConfigMap exists and load existing data if it does
	err := r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, cm)
	if err == nil && len(cm.BinaryData["data"]) > 0 {
		// Unmarshal existing data into configData
		if err := json.Unmarshal(cm.BinaryData["data"], &configData); err != nil {
			logger.Error(err, "Failed to decode ConfigMap data")
			return fmt.Errorf("failed to decode ConfigMap data: %v", err)
		}
	} else if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Failed to get existing ConfigMap")
		return fmt.Errorf("failed to get ConfigMap: %v", err)
	}

	// Iterate over namespaces and secrets to calculate new data
	updated := false
	for namespaceName, secrets := range secretsToWatch {
		for _, secretName := range secrets {
			secret, err := r.getSecret(ctx, namespaceName, secretName)
			if err != nil {
				logger.Error(err, "Failed to get secret", "namespace", namespaceName, "name", secretName)
				return fmt.Errorf("failed to get secret %s in namespace %s: %v", secretName, namespaceName, err)
			}

			newHash := CalculateHash(secret.Data)
			newLastModified := RetrieveModifiedTime(secret.ObjectMeta)

			if existingSecret := secretDataExists(&configData, namespaceName, secretName); existingSecret != nil {
				updateOrAppendSecretData(existingSecret, newHash, newLastModified)
				r.Recorder.Event(cm, "Normal", "SecretUpdated", fmt.Sprintf("Updated existing secret data: %s/%s", namespaceName, secretName))
				updated = true
			} else {
				addSecretData(&configData, namespaceName, secretName, newHash, secret.CreationTimestamp.Time.UTC().Format(time.RFC3339), newLastModified)
				r.Recorder.Event(cm, "Normal", "SecretAdded", fmt.Sprintf("Added new secret data: %s/%s", namespaceName, secretName))
				updated = true
			}
		}
	}

	if updated {
		// Encode updated ConfigData to JSON
		jsonData, err := json.Marshal(configData)
		if err != nil {
			logger.Error(err, "Failed to encode ConfigData to JSON")
			return fmt.Errorf("failed to encode ConfigData to JSON: %v", err)
		}

		// Store updated JSON data in ConfigMap
		cm.BinaryData = map[string][]byte{"data": jsonData}
		if err := r.createOrUpdateConfigMap(ctx, cm); err != nil {
			logger.Error(err, "Failed to create or update ConfigMap")
			return fmt.Errorf("failed to create or update ConfigMap: %v", err)
		}
		r.Recorder.Event(cm, "Normal", "ConfigMapUpdated", "ConfigMap updated successfully")
		logger.Info("ConfigMap updated successfully")
	} else {
		logger.Info("No updates necessary for ConfigMap")
		r.Recorder.Event(cm, "Normal", "NoUpdateNeeded", "No updates made to ConfigMap")
	}

	return nil
}

// updateOrAppendSecretData will append new data hash to history
func updateOrAppendSecretData(secret *Secret, newHash, lastModified string) {
	found := false
	for _, h := range secret.History {
		if h.Data == newHash {
			found = true
			break
		}
	}
	if !found {
		secret.History = append(secret.History, History{Data: newHash})
	}
	secret.LastModified = lastModified
}

// secretDataExists checks whether hash data for perticular secret data exists or not
func secretDataExists(configData *ConfigData, namespace, name string) *Secret {
	for i, ns := range configData.Namespaces {
		if ns.Name == namespace {
			for j := range ns.Secrets {
				if ns.Secrets[j].Name == name {
					return &configData.Namespaces[i].Secrets[j]
				}
			}
		}
	}
	return nil
}

// addSecretData adds new secret data hash to history
func addSecretData(configData *ConfigData, namespace, name string, newHash, created, lastModified string) {
	newSecret := Secret{
		Name:         name,
		Created:      created,
		LastModified: lastModified,
		History:      []History{{Data: newHash}},
	}

	for i, ns := range configData.Namespaces {
		if ns.Name == namespace {
			configData.Namespaces[i].Secrets = append(configData.Namespaces[i].Secrets, newSecret)
			return
		}
	}

	// If namespace not found, add new namespace with the secret
	configData.Namespaces = append(configData.Namespaces, Namespace{
		Name:    namespace,
		Secrets: []Secret{newSecret},
	})
}

// getSecret gets secret data
func (r *StaleSecretWatchReconciler) getSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// createOrUpdateConfigMap creates or updates configmap
func (r *StaleSecretWatchReconciler) createOrUpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	found := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// ConfigMap does not exist, create it
		err = r.Create(ctx, cm)
		if err != nil {
			return fmt.Errorf("failed to create ConfigMap: %v", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check if ConfigMap exists: %v", err)
	}

	// ConfigMap exists, update it
	found = found.DeepCopy()
	found.BinaryData = cm.BinaryData
	err = r.Update(ctx, found)
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap: %v", err)
	}

	return nil
}

// prepareWatchList prepares the list of secret resources present inside the namespace, based on the data provided in yaml file
func (r *StaleSecretWatchReconciler) prepareWatchList(ctx context.Context, logger logr.Logger, ssw *securityv1beta1.StaleSecretWatch) (map[string][]string, error) {
	var namespacesToWatch []string
	secretsToWatch := make(map[string][]string)

	// If watching all namespaces, list all namespaces and add them to namespacesToWatch
	if ssw.Spec.StaleSecretToWatch.Namespace == "all" {
		nsList := &corev1.NamespaceList{}
		if err := r.List(ctx, nsList); err != nil {
			return nil, fmt.Errorf("failed to list namespaces: %v", err)
		}
		for _, ns := range nsList.Items {
			namespacesToWatch = append(namespacesToWatch, ns.Name)
		}
	} else {
		ns := strings.Split(ssw.Spec.StaleSecretToWatch.Namespace, ",")
		for _, n := range ns {
			namespacesToWatch = append(namespacesToWatch, strings.TrimSpace(n))
		}
	}

	// Now, list secrets in each namespace and filter based on the excludeList
	for _, ns := range namespacesToWatch {
		secretList := &corev1.SecretList{}
		if err := r.List(ctx, secretList, client.InNamespace(ns)); err != nil || apierrors.IsNotFound(err) {
			logger.Info("secret resources not found in " + ns)
			return nil, fmt.Errorf("failed to list secrets in namespace %s: %v", ns, err)
		}
		for _, secret := range secretList.Items {
			// Check if this secret is in the excludeList for its namespace
			excluded := false
			for _, excludeEntry := range ssw.Spec.StaleSecretToWatch.ExcludeList {
				if excludeEntry.Namespace == ns && Contains(excludeEntry.SecretName, secret.Name) {
					excluded = true
					break
				}
			}
			if !excluded {
				secretsToWatch[ns] = append(secretsToWatch[ns], secret.Name)
			}
		}
	}
	r.Recorder.Event(ssw, "Normal", "prepared list of secrets", "list of secrets present in different namespaces prepared for watch")

	return secretsToWatch, nil

}
