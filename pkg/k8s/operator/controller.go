package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/amirhossein-jamali/tailpost/pkg/k8s/api/v1alpha1"
	"github.com/amirhossein-jamali/tailpost/pkg/k8s/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// ConditionTypeAvailable represents the Available condition type
	ConditionTypeAvailable = "Available"
	// ConditionTypeDegraded represents the Degraded condition type
	ConditionTypeDegraded = "Degraded"

	// DefaultImage is the default TailPost image to use
	DefaultImage = "tailpost:latest"
	// DefaultImagePullPolicy is the default image pull policy
	DefaultImagePullPolicy = "IfNotPresent"
	// DefaultReplicas is the default number of replicas
	DefaultReplicas = 1
	// DefaultBatchSize is the default batch size
	DefaultBatchSize = 10
	// DefaultFlushInterval is the default flush interval
	DefaultFlushInterval = "5s"
)

// TailpostAgentReconciler reconciles a TailpostAgent object
type TailpostAgentReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	KubeClient    *kubernetes.Clientset
	DefaultImage  string
	ResyncPeriod  time.Duration
	RequeuePeriod time.Duration
}

// NewTailpostAgentReconciler creates a new reconciler for TailpostAgent resources
func NewTailpostAgentReconciler(mgr manager.Manager) (*TailpostAgentReconciler, error) {
	kubeClient, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	reconciler := &TailpostAgentReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Recorder:      mgr.GetEventRecorderFor("tailpostagent-controller"),
		KubeClient:    kubeClient,
		DefaultImage:  DefaultImage,
		ResyncPeriod:  time.Minute * 10,
		RequeuePeriod: time.Second * 30,
	}

	return reconciler, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *TailpostAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TailpostAgent{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}

// Reconcile reconciles the state of a TailpostAgent resource
// +kubebuilder:rbac:groups=tailpost.elastic.co,resources=tailpostagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tailpost.elastic.co,resources=tailpostagents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
func (r *TailpostAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx).WithValues("tailpostagent", req.NamespacedName)
	log.Info("Reconciling TailpostAgent")

	// Fetch the TailpostAgent instance
	instance := &v1alpha1.TailpostAgent{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return
			return ctrl.Result{}, nil
		}
		// Error reading the object
		return ctrl.Result{}, err
	}

	// Set default values if they're not specified
	if err := r.setDefaults(ctx, instance); err != nil {
		log.Error(err, "Failed to set defaults")
		return ctrl.Result{RequeueAfter: r.RequeuePeriod}, err
	}

	// Reconcile ConfigMap
	if err := r.reconcileConfigMap(ctx, instance); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		r.setCondition(ctx, instance, ConditionTypeDegraded, "True", "ConfigMapReconcileFailed", err.Error())
		return ctrl.Result{RequeueAfter: r.RequeuePeriod}, err
	}

	// Reconcile StatefulSet
	if err := r.reconcileStatefulSet(ctx, instance); err != nil {
		log.Error(err, "Failed to reconcile StatefulSet")
		r.setCondition(ctx, instance, ConditionTypeDegraded, "True", "StatefulSetReconcileFailed", err.Error())
		return ctrl.Result{RequeueAfter: r.RequeuePeriod}, err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, instance); err != nil {
		log.Error(err, "Failed to reconcile Service")
		r.setCondition(ctx, instance, ConditionTypeDegraded, "True", "ServiceReconcileFailed", err.Error())
		return ctrl.Result{RequeueAfter: r.RequeuePeriod}, err
	}

	// Update status
	if err := r.updateStatus(ctx, instance); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: r.RequeuePeriod}, err
	}

	// Set agent as available
	r.setCondition(ctx, instance, ConditionTypeAvailable, "True", "AgentAvailable", "The agent is available")
	// Remove degraded condition if it exists
	r.removeCondition(ctx, instance, ConditionTypeDegraded)

	return ctrl.Result{RequeueAfter: r.ResyncPeriod}, nil
}

// setDefaults sets default values for TailpostAgent if they're not specified
func (r *TailpostAgentReconciler) setDefaults(ctx context.Context, instance *v1alpha1.TailpostAgent) error {
	needsUpdate := false

	// Set default image
	if instance.Spec.Image == "" {
		instance.Spec.Image = r.DefaultImage
		needsUpdate = true
	}

	// Set default image pull policy
	if instance.Spec.ImagePullPolicy == "" {
		instance.Spec.ImagePullPolicy = DefaultImagePullPolicy
		needsUpdate = true
	}

	// Set default replicas
	if instance.Spec.Replicas == nil {
		replicas := int32(DefaultReplicas)
		instance.Spec.Replicas = &replicas
		needsUpdate = true
	}

	// Set default batch size
	if instance.Spec.BatchSize == nil {
		batchSize := int32(DefaultBatchSize)
		instance.Spec.BatchSize = &batchSize
		needsUpdate = true
	}

	// Set default flush interval
	if instance.Spec.FlushInterval == "" {
		instance.Spec.FlushInterval = DefaultFlushInterval
		needsUpdate = true
	}

	// Update the instance if needed
	if needsUpdate {
		if err := r.Update(ctx, instance); err != nil {
			return fmt.Errorf("failed to update instance with defaults: %w", err)
		}
	}

	return nil
}

// reconcileConfigMap reconciles the ConfigMap for the TailpostAgent
func (r *TailpostAgentReconciler) reconcileConfigMap(ctx context.Context, instance *v1alpha1.TailpostAgent) error {
	configMap, err := resources.CreateConfigMap(instance)
	if err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	// Set controller reference
	if err := ctrl.SetControllerReference(instance, configMap, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on ConfigMap: %w", err)
	}

	// Check if ConfigMap exists
	found := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create ConfigMap
			if err := r.Create(ctx, configMap); err != nil {
				return fmt.Errorf("failed to create ConfigMap: %w", err)
			}
			r.Recorder.Eventf(instance, corev1.EventTypeNormal, "ConfigMapCreated", "Created ConfigMap %s", configMap.Name)
			return nil
		}
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Update ConfigMap if needed
	if resources.ConfigMapNeedsUpdate(found, configMap) {
		found.Data = configMap.Data
		if err := r.Update(ctx, found); err != nil {
			return fmt.Errorf("failed to update ConfigMap: %w", err)
		}
		r.Recorder.Eventf(instance, corev1.EventTypeNormal, "ConfigMapUpdated", "Updated ConfigMap %s", configMap.Name)
	}

	return nil
}

// reconcileStatefulSet reconciles the StatefulSet for the TailpostAgent
func (r *TailpostAgentReconciler) reconcileStatefulSet(ctx context.Context, instance *v1alpha1.TailpostAgent) error {
	statefulSet, err := resources.CreateStatefulSet(instance)
	if err != nil {
		return fmt.Errorf("failed to create StatefulSet: %w", err)
	}

	// Set controller reference
	if err := ctrl.SetControllerReference(instance, statefulSet, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on StatefulSet: %w", err)
	}

	// Check if StatefulSet exists
	found := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: statefulSet.Name, Namespace: statefulSet.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create StatefulSet
			if err := r.Create(ctx, statefulSet); err != nil {
				return fmt.Errorf("failed to create StatefulSet: %w", err)
			}
			r.Recorder.Eventf(instance, corev1.EventTypeNormal, "StatefulSetCreated", "Created StatefulSet %s", statefulSet.Name)
			return nil
		}
		return fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	// Update StatefulSet if needed
	if resources.StatefulSetNeedsUpdate(found, statefulSet) {
		found.Spec = statefulSet.Spec
		if err := r.Update(ctx, found); err != nil {
			return fmt.Errorf("failed to update StatefulSet: %w", err)
		}
		r.Recorder.Eventf(instance, corev1.EventTypeNormal, "StatefulSetUpdated", "Updated StatefulSet %s", statefulSet.Name)
	}

	return nil
}

// reconcileService reconciles the Service for the TailpostAgent
func (r *TailpostAgentReconciler) reconcileService(ctx context.Context, instance *v1alpha1.TailpostAgent) error {
	service := resources.CreateService(instance)

	// Set controller reference
	if err := ctrl.SetControllerReference(instance, service, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on Service: %w", err)
	}

	// Check if Service exists
	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create Service
			if err := r.Create(ctx, service); err != nil {
				return fmt.Errorf("failed to create Service: %w", err)
			}
			r.Recorder.Eventf(instance, corev1.EventTypeNormal, "ServiceCreated", "Created Service %s", service.Name)
			return nil
		}
		return fmt.Errorf("failed to get Service: %w", err)
	}

	// Update Service if needed (we only update the selector and ports)
	if resources.ServiceNeedsUpdate(found, service) {
		found.Spec.Selector = service.Spec.Selector
		found.Spec.Ports = service.Spec.Ports
		if err := r.Update(ctx, found); err != nil {
			return fmt.Errorf("failed to update Service: %w", err)
		}
		r.Recorder.Eventf(instance, corev1.EventTypeNormal, "ServiceUpdated", "Updated Service %s", service.Name)
	}

	return nil
}

// updateStatus updates the status of the TailpostAgent
func (r *TailpostAgentReconciler) updateStatus(ctx context.Context, instance *v1alpha1.TailpostAgent) error {
	// Get the StatefulSet
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      resources.GetStatefulSetName(instance),
		Namespace: instance.Namespace,
	}, statefulSet)
	if err != nil {
		if errors.IsNotFound(err) {
			// StatefulSet not found, set available replicas to 0
			instance.Status.AvailableReplicas = 0
		} else {
			return fmt.Errorf("failed to get StatefulSet: %w", err)
		}
	} else {
		// Update available replicas
		instance.Status.AvailableReplicas = statefulSet.Status.ReadyReplicas
	}

	// Update last update time
	instance.Status.LastUpdateTime = metav1.Now()

	// Update status
	if err := r.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// setCondition sets a condition on the TailpostAgent
func (r *TailpostAgentReconciler) setCondition(ctx context.Context, instance *v1alpha1.TailpostAgent, condType, status, reason, message string) {
	now := metav1.Now()
	condition := v1alpha1.TailpostAgentCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	// Check if the condition already exists
	existingCondition := r.findCondition(instance, condType)
	if existingCondition != nil {
		// Update existing condition if status changed
		if existingCondition.Status != status {
			existingCondition.Status = status
			existingCondition.LastTransitionTime = now
			existingCondition.Reason = reason
			existingCondition.Message = message
		}
	} else {
		// Add new condition
		instance.Status.Conditions = append(instance.Status.Conditions, condition)
	}

	// Update status
	if err := r.Status().Update(ctx, instance); err != nil {
		klog.Errorf("Failed to update status with condition %s: %v", condType, err)
	}
}

// removeCondition removes a condition from the TailpostAgent
func (r *TailpostAgentReconciler) removeCondition(ctx context.Context, instance *v1alpha1.TailpostAgent, condType string) {
	// Check if the condition exists
	foundIdx := -1
	for i, cond := range instance.Status.Conditions {
		if cond.Type == condType {
			foundIdx = i
			break
		}
	}

	if foundIdx >= 0 {
		// Remove condition
		instance.Status.Conditions = append(instance.Status.Conditions[:foundIdx], instance.Status.Conditions[foundIdx+1:]...)

		// Update status
		if err := r.Status().Update(ctx, instance); err != nil {
			klog.Errorf("Failed to remove condition %s: %v", condType, err)
		}
	}
}

// findCondition finds a condition by type
func (r *TailpostAgentReconciler) findCondition(instance *v1alpha1.TailpostAgent, condType string) *v1alpha1.TailpostAgentCondition {
	for i := range instance.Status.Conditions {
		if instance.Status.Conditions[i].Type == condType {
			return &instance.Status.Conditions[i]
		}
	}
	return nil
}
