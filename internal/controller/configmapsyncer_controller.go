/*
Copyright 2025.

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
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	syncv1alpha1 "github.com/devShahriar/configmap-sync-controller/api/v1alpha1"
)

const (
	// ConditionTypeReady is the type for the ready condition
	ConditionTypeReady = "Ready"

	// ConditionReasonSyncSuccess is the reason for a successful sync
	ConditionReasonSyncSuccess = "SyncSuccess"

	// ConditionReasonSyncFailed is the reason for a failed sync
	ConditionReasonSyncFailed = "SyncFailed"

	// ConditionReasonMasterConfigMapNotFound is the reason when the master ConfigMap is not found
	ConditionReasonMasterConfigMapNotFound = "MasterConfigMapNotFound"

	// SyncStatusPending indicates that the sync is pending
	SyncStatusPending = "Pending"

	// SyncStatusSynced indicates that the sync was successful
	SyncStatusSynced = "Synced"

	// SyncStatusFailed indicates that the sync failed
	SyncStatusFailed = "Failed"

	// MergeStrategyReplace replaces the target ConfigMap with the master ConfigMap
	MergeStrategyReplace = "Replace"

	// MergeStrategyMerge merges the master ConfigMap with the target ConfigMap
	MergeStrategyMerge = "Merge"

	// FinalizerName is the name of the finalizer
	FinalizerName = "configmapsyncer.conf-sync.com/finalizer"

	// SourceConfigMapLabel is the label key for the source ConfigMap
	SourceConfigMapLabel = "configmapsyncer.conf-sync.com/source"
)

// ConfigMapSyncerReconciler reconciles a ConfigMapSyncer object
type ConfigMapSyncerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=conf-sync.com,resources=configmapsyncers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=conf-sync.com,resources=configmapsyncers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=conf-sync.com,resources=configmapsyncers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ConfigMapSyncerReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling ConfigMapSyncer", "namespacedName", req.NamespacedName)

	// Fetch the ConfigMapSyncer instance
	configMapSyncer := &syncv1alpha1.ConfigMapSyncer{}
	if err := r.Get(ctx, req.NamespacedName, configMapSyncer); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			logger.Info(
				"ConfigMapSyncer resource not found. Ignoring since object must be deleted",
			)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get ConfigMapSyncer")
		return ctrl.Result{}, err
	}

	// Initialize status if it's empty
	if configMapSyncer.Status.Conditions == nil {
		configMapSyncer.Status.Conditions = []metav1.Condition{}
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(configMapSyncer, FinalizerName) {
		controllerutil.AddFinalizer(configMapSyncer, FinalizerName)
		if err := r.Update(ctx, configMapSyncer); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the ConfigMapSyncer is being deleted
	if !configMapSyncer.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, configMapSyncer)
	}

	// Get the master ConfigMap
	masterConfigMap := &corev1.ConfigMap{}
	masterConfigMapKey := types.NamespacedName{
		Name:      configMapSyncer.Spec.MasterConfigMap.Name,
		Namespace: configMapSyncer.Spec.MasterConfigMap.Namespace,
	}

	if err := r.Get(ctx, masterConfigMapKey, masterConfigMap); err != nil {
		if errors.IsNotFound(err) {
			// Master ConfigMap not found, update status and requeue
			logger.Info("Master ConfigMap not found", "configMap", masterConfigMapKey)
			r.setCondition(configMapSyncer, metav1.Condition{
				Type:    ConditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  ConditionReasonMasterConfigMapNotFound,
				Message: fmt.Sprintf("Master ConfigMap %s not found", masterConfigMapKey),
			})
			if err := r.Status().Update(ctx, configMapSyncer); err != nil {
				logger.Error(err, "Failed to update ConfigMapSyncer status")
				return ctrl.Result{}, err
			}
			// Requeue after 1 minute
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get master ConfigMap")
		return ctrl.Result{}, err
	}

	// Sync ConfigMaps
	syncResult, err := r.syncConfigMaps(ctx, configMapSyncer, masterConfigMap)
	if err != nil {
		logger.Error(err, "Failed to sync ConfigMaps")
		r.setCondition(configMapSyncer, metav1.Condition{
			Type:    ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  ConditionReasonSyncFailed,
			Message: fmt.Sprintf("Failed to sync ConfigMaps: %v", err),
		})
		if updateErr := r.Status().Update(ctx, configMapSyncer); updateErr != nil {
			logger.Error(updateErr, "Failed to update ConfigMapSyncer status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Update status
	now := metav1.NewTime(time.Now())
	configMapSyncer.Status.LastSyncTime = &now
	configMapSyncer.Status.SyncStatuses = syncResult

	r.setCondition(configMapSyncer, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  ConditionReasonSyncSuccess,
		Message: "Successfully synced ConfigMaps",
	})

	if err := r.Status().Update(ctx, configMapSyncer); err != nil {
		logger.Error(err, "Failed to update ConfigMapSyncer status")
		return ctrl.Result{}, err
	}

	// Use sync interval from spec, default to 300 seconds (5 minutes)
	interval := time.Duration(configMapSyncer.Spec.SyncInterval) * time.Second
	if interval == 0 {
		interval = 300 * time.Second
	}

	// Requeue after the specified interval
	return ctrl.Result{RequeueAfter: interval}, nil
}

// handleDeletion handles the deletion of the ConfigMapSyncer resource
func (r *ConfigMapSyncerReconciler) handleDeletion(
	ctx context.Context,
	configMapSyncer *syncv1alpha1.ConfigMapSyncer,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling deletion of ConfigMapSyncer", "name", configMapSyncer.Name)

	// Remove finalizer
	controllerutil.RemoveFinalizer(configMapSyncer, FinalizerName)
	if err := r.Update(ctx, configMapSyncer); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// syncConfigMaps syncs the master ConfigMap to target ConfigMaps
func (r *ConfigMapSyncerReconciler) syncConfigMaps(
	ctx context.Context,
	configMapSyncer *syncv1alpha1.ConfigMapSyncer,
	masterConfigMap *corev1.ConfigMap,
) ([]syncv1alpha1.SyncStatus, error) {
	logger := log.FromContext(ctx)
	var syncStatuses []syncv1alpha1.SyncStatus

	// Get target namespaces
	targetNamespaces := configMapSyncer.Spec.TargetNamespaces
	if len(targetNamespaces) == 0 {
		// If no target namespaces are specified, get all namespaces
		namespaceList := &corev1.NamespaceList{}
		if err := r.List(ctx, namespaceList); err != nil {
			return nil, fmt.Errorf("failed to list namespaces: %w", err)
		}
		for _, ns := range namespaceList.Items {
			// Skip the namespace of the master ConfigMap
			if ns.Name != masterConfigMap.Namespace {
				targetNamespaces = append(targetNamespaces, ns.Name)
			}
		}
	}

	// Process each target namespace
	for _, namespace := range targetNamespaces {
		// Skip the namespace of the master ConfigMap
		if namespace == masterConfigMap.Namespace {
			continue
		}

		var targetConfigMaps []corev1.ConfigMap

		// If targetSelector is specified, find ConfigMaps matching the selector
		if configMapSyncer.Spec.TargetSelector != nil {
			selector, err := metav1.LabelSelectorAsSelector(
				configMapSyncer.Spec.TargetSelector,
			)
			if err != nil {
				logger.Error(err, "Failed to parse label selector")
				continue
			}

			configMapList := &corev1.ConfigMapList{}
			if err := r.List(ctx, configMapList, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
				logger.Error(err, "Failed to list ConfigMaps", "namespace", namespace)
				continue
			}

			targetConfigMaps = configMapList.Items
		} else {
			// If no targetSelector is specified, create/update a ConfigMap with the same name as the master
			targetConfigMap := &corev1.ConfigMap{}
			err := r.Get(ctx, types.NamespacedName{Name: masterConfigMap.Name, Namespace: namespace}, targetConfigMap)
			if err != nil {
				if !errors.IsNotFound(err) {
					logger.Error(err, "Failed to get target ConfigMap", "namespace", namespace, "name", masterConfigMap.Name)
					continue
				}
				// ConfigMap doesn't exist, create a new one
				targetConfigMap = &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      masterConfigMap.Name,
						Namespace: namespace,
						Labels: map[string]string{
							SourceConfigMapLabel: fmt.Sprintf("%s.%s", masterConfigMap.Namespace, masterConfigMap.Name),
						},
					},
				}
				targetConfigMaps = append(targetConfigMaps, *targetConfigMap)
			} else {
				targetConfigMaps = append(targetConfigMaps, *targetConfigMap)
			}
		}

		// Process each target ConfigMap
		for _, targetConfigMap := range targetConfigMaps {
			syncStatus := syncv1alpha1.SyncStatus{
				ConfigMapName: targetConfigMap.Name,
				Namespace:     targetConfigMap.Namespace,
				Status:        SyncStatusPending,
			}

			// Create a copy of the target ConfigMap for updates
			updatedConfigMap := targetConfigMap.DeepCopy()

			// Set or update labels
			if updatedConfigMap.Labels == nil {
				updatedConfigMap.Labels = make(map[string]string)
			}
			updatedConfigMap.Labels[SourceConfigMapLabel] = fmt.Sprintf(
				"%s.%s",
				masterConfigMap.Namespace,
				masterConfigMap.Name,
			)

			// Apply merge strategy
			mergeStrategy := configMapSyncer.Spec.MergeStrategy
			if mergeStrategy == "" {
				mergeStrategy = MergeStrategyMerge
			}

			switch mergeStrategy {
			case MergeStrategyReplace:
				// Replace all data with master ConfigMap data
				updatedConfigMap.Data = make(map[string]string)
				for k, v := range masterConfigMap.Data {
					updatedConfigMap.Data[k] = v
				}
				updatedConfigMap.BinaryData = make(map[string][]byte)
				for k, v := range masterConfigMap.BinaryData {
					updatedConfigMap.BinaryData[k] = v
				}
			case MergeStrategyMerge:
				// Merge data with master ConfigMap data
				if updatedConfigMap.Data == nil {
					updatedConfigMap.Data = make(map[string]string)
				}
				for k, v := range masterConfigMap.Data {
					updatedConfigMap.Data[k] = v
				}
				if updatedConfigMap.BinaryData == nil {
					updatedConfigMap.BinaryData = make(map[string][]byte)
				}
				for k, v := range masterConfigMap.BinaryData {
					updatedConfigMap.BinaryData[k] = v
				}
			default:
				logger.Info(
					"Unknown merge strategy, using Merge",
					"strategy",
					mergeStrategy,
				)
				if updatedConfigMap.Data == nil {
					updatedConfigMap.Data = make(map[string]string)
				}
				for k, v := range masterConfigMap.Data {
					updatedConfigMap.Data[k] = v
				}
				if updatedConfigMap.BinaryData == nil {
					updatedConfigMap.BinaryData = make(map[string][]byte)
				}
				for k, v := range masterConfigMap.BinaryData {
					updatedConfigMap.BinaryData[k] = v
				}
			}

			// Check if the ConfigMap exists
			existingConfigMap := &corev1.ConfigMap{}
			err := r.Get(
				ctx,
				types.NamespacedName{
					Name:      updatedConfigMap.Name,
					Namespace: updatedConfigMap.Namespace,
				},
				existingConfigMap,
			)
			if err != nil {
				if errors.IsNotFound(err) {
					// ConfigMap doesn't exist, create it
					if err := r.Create(ctx, updatedConfigMap); err != nil {
						logger.Error(
							err,
							"Failed to create ConfigMap",
							"namespace",
							updatedConfigMap.Namespace,
							"name",
							updatedConfigMap.Name,
						)
						syncStatus.Status = SyncStatusFailed
						syncStatus.Message = fmt.Sprintf(
							"Failed to create ConfigMap: %v",
							err,
						)
					} else {
						logger.Info("Created ConfigMap", "namespace", updatedConfigMap.Namespace, "name", updatedConfigMap.Name)
						syncStatus.Status = SyncStatusSynced
						syncStatus.LastSyncTime = &metav1.Time{Time: time.Now()}
					}
				} else {
					logger.Error(err, "Failed to get ConfigMap", "namespace", updatedConfigMap.Namespace, "name", updatedConfigMap.Name)
					syncStatus.Status = SyncStatusFailed
					syncStatus.Message = fmt.Sprintf("Failed to get ConfigMap: %v", err)
				}
			} else {
				// ConfigMap exists, check if it needs to be updated
				if !reflect.DeepEqual(existingConfigMap.Data, updatedConfigMap.Data) || !reflect.DeepEqual(existingConfigMap.BinaryData, updatedConfigMap.BinaryData) {
					// Update the ConfigMap
					existingConfigMap.Data = updatedConfigMap.Data
					existingConfigMap.BinaryData = updatedConfigMap.BinaryData
					existingConfigMap.Labels = updatedConfigMap.Labels
					if err := r.Update(ctx, existingConfigMap); err != nil {
						logger.Error(err, "Failed to update ConfigMap", "namespace", existingConfigMap.Namespace, "name", existingConfigMap.Name)
						syncStatus.Status = SyncStatusFailed
						syncStatus.Message = fmt.Sprintf("Failed to update ConfigMap: %v", err)
					} else {
						logger.Info("Updated ConfigMap", "namespace", existingConfigMap.Namespace, "name", existingConfigMap.Name)
						syncStatus.Status = SyncStatusSynced
						syncStatus.LastSyncTime = &metav1.Time{Time: time.Now()}
					}
				} else {
					// ConfigMap is already in sync
					logger.Info("ConfigMap is already in sync", "namespace", existingConfigMap.Namespace, "name", existingConfigMap.Name)
					syncStatus.Status = SyncStatusSynced
					syncStatus.LastSyncTime = &metav1.Time{Time: time.Now()}
				}
			}

			syncStatuses = append(syncStatuses, syncStatus)
		}
	}

	return syncStatuses, nil
}

// setCondition sets a condition on the ConfigMapSyncer status
func (r *ConfigMapSyncerReconciler) setCondition(
	configMapSyncer *syncv1alpha1.ConfigMapSyncer,
	condition metav1.Condition,
) {
	// Set the transition time if it's not already set
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = metav1.NewTime(time.Now())
	}

	// Find and update existing condition or append new condition
	for i, c := range configMapSyncer.Status.Conditions {
		if c.Type == condition.Type {
			// Only update if the status has changed
			if c.Status != condition.Status {
				configMapSyncer.Status.Conditions[i] = condition
			}
			return
		}
	}

	// Condition doesn't exist, append it
	configMapSyncer.Status.Conditions = append(
		configMapSyncer.Status.Conditions,
		condition,
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapSyncerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1alpha1.ConfigMapSyncer{}).
		Owns(&corev1.ConfigMap{}).
		Named("configmapsyncer").
		Complete(r)
}
