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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/Kyuzial/free5gc-k8s/api/v1alpha1"
)

const (
	finalizerName = "free5gc.core.free5gc.org/finalizer"
)

// Free5GCReconciler reconciles a Free5GC object
type Free5GCReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Helper function to create or update a deployment
func (r *Free5GCReconciler) reconcileDeployment(ctx context.Context, free5gc *corev1alpha1.Free5GC, component string, spec *corev1alpha1.ComponentSpec) error {
	log := log.FromContext(ctx)

	if spec == nil {
		log.Info("Component spec is nil, skipping", "component", component)
		return nil
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", free5gc.Name, component),
			Namespace: free5gc.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		if err := ctrl.SetControllerReference(free5gc, deploy, r.Scheme); err != nil {
			return err
		}

		replicas := int32(1)
		if spec.Replicas != nil {
			replicas = *spec.Replicas
		}

		deploy.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "free5gc",
					"component": component,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "free5gc",
						"component": component,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  component,
							Image: spec.Image,
							Resources: spec.Resources,
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile deployment for %s: %w", component, err)
	}

	log.Info("Reconciled deployment", "component", component, "operation", op)
	return nil
}

// Helper function to create or update a service
func (r *Free5GCReconciler) reconcileService(ctx context.Context, free5gc *corev1alpha1.Free5GC, component string) error {
	log := log.FromContext(ctx)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", free5gc.Name, component),
			Namespace: free5gc.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := ctrl.SetControllerReference(free5gc, svc, r.Scheme); err != nil {
			return err
		}

		svc.Spec = corev1.ServiceSpec{
			Selector: map[string]string{
				"app":       "free5gc",
				"component": component,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: corev1.ProtocolTCP,
					Port:     80,
				},
			},
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile service for %s: %w", component, err)
	}

	log.Info("Reconciled service", "component", component, "operation", op)
	return nil
}

// Helper function to update component status
func (r *Free5GCReconciler) updateComponentStatus(ctx context.Context, free5gc *corev1alpha1.Free5GC, component string) error {
	deploy := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", free5gc.Name, component),
		Namespace: free5gc.Namespace,
	}, deploy)

	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if free5gc.Status.Components == nil {
		free5gc.Status.Components = make(map[string]corev1alpha1.ComponentStatus)
	}

	status := corev1alpha1.ComponentStatus{
		Phase:         "Running",
		ReadyReplicas: deploy.Status.ReadyReplicas,
		Replicas:      deploy.Status.Replicas,
	}

	if deploy.Status.ReadyReplicas != deploy.Status.Replicas {
		status.Phase = "Pending"
		status.Message = fmt.Sprintf("Waiting for %d/%d replicas to be ready", deploy.Status.ReadyReplicas, deploy.Status.Replicas)
	}

	free5gc.Status.Components[component] = status
	return nil
}

// +kubebuilder:rbac:groups=core.free5gc.org,resources=free5gcs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.free5gc.org,resources=free5gcs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.free5gc.org,resources=free5gcs/finalizers,verbs=update

func (r *Free5GCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Free5GC instance
	free5gc := &corev1alpha1.Free5GC{}
	err := r.Get(ctx, req.NamespacedName, free5gc)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(free5gc, finalizerName) {
		controllerutil.AddFinalizer(free5gc, finalizerName)
		if err := r.Update(ctx, free5gc); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle deletion
	if !free5gc.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(free5gc, finalizerName) {
			// Perform cleanup
			if err := r.cleanupResources(ctx, free5gc); err != nil {
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(free5gc, finalizerName)
			if err := r.Update(ctx, free5gc); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Reconcile MongoDB if specified
	if free5gc.Spec.MongoDB != nil && !free5gc.Spec.MongoDB.External {
		if err := r.reconcileMongoDB(ctx, free5gc); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile core components
	components := map[string]*corev1alpha1.ComponentSpec{
		"nrf":    free5gc.Spec.NRF,
		"amf":    free5gc.Spec.AMF,
		"smf":    free5gc.Spec.SMF,
		"ausf":   free5gc.Spec.AUSF,
		"nssf":   free5gc.Spec.NSSF,
		"pcf":    free5gc.Spec.PCF,
		"udm":    free5gc.Spec.UDM,
		"udr":    free5gc.Spec.UDR,
		"n3iwf":  free5gc.Spec.N3IWF,
		"webui":  free5gc.Spec.WebUI,
	}

	for component, spec := range components {
		if err := r.reconcileDeployment(ctx, free5gc, component, spec); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.reconcileService(ctx, free5gc, component); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.updateComponentStatus(ctx, free5gc, component); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile UPF separately due to its special configuration
	if free5gc.Spec.UPF != nil {
		if err := r.reconcileUPF(ctx, free5gc); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update status
	if err := r.Status().Update(ctx, free5gc); err != nil {
		log.Error(err, "Failed to update Free5GC status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Free5GCReconciler) reconcileMongoDB(ctx context.Context, free5gc *corev1alpha1.Free5GC) error {
	log := log.FromContext(ctx)

	if free5gc.Spec.MongoDB == nil {
		return nil
	}

	// Create PVC if storage is specified
	if free5gc.Spec.MongoDB.Storage != nil {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-mongodb", free5gc.Name),
				Namespace: free5gc.Namespace,
			},
		}

		op, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
			if err := ctrl.SetControllerReference(free5gc, pvc, r.Scheme); err != nil {
				return err
			}

			quantity, err := resource.ParseQuantity(free5gc.Spec.MongoDB.Storage.Size)
			if err != nil {
				return fmt.Errorf("invalid storage size: %w", err)
			}

			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: quantity,
					},
				},
			}

			if free5gc.Spec.MongoDB.Storage.StorageClassName != "" {
				pvc.Spec.StorageClassName = &free5gc.Spec.MongoDB.Storage.StorageClassName
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to reconcile MongoDB PVC: %w", err)
		}
		log.Info("Reconciled MongoDB PVC", "operation", op)
	}

	// Create MongoDB deployment
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-mongodb", free5gc.Name),
			Namespace: free5gc.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		if err := ctrl.SetControllerReference(free5gc, deploy, r.Scheme); err != nil {
			return err
		}

		replicas := int32(1)
		deploy.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "mongodb",
					"free5gc":   free5gc.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "mongodb",
						"free5gc":   free5gc.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "mongodb",
							Image: free5gc.Spec.MongoDB.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "mongodb",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 27017,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "mongodb-data",
									MountPath: "/data/db",
								},
							},
						},
					},
				},
			},
		}

		// Add volume if storage is configured
		if free5gc.Spec.MongoDB.Storage != nil {
			deploy.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "mongodb-data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("%s-mongodb", free5gc.Name),
						},
					},
				},
			}
		} else {
			deploy.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "mongodb-data",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile MongoDB deployment: %w", err)
	}
	log.Info("Reconciled MongoDB deployment", "operation", op)

	// Create MongoDB service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-mongodb", free5gc.Name),
			Namespace: free5gc.Namespace,
		},
	}

	op, err = controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := ctrl.SetControllerReference(free5gc, svc, r.Scheme); err != nil {
			return err
		}

		svc.Spec = corev1.ServiceSpec{
			Selector: map[string]string{
				"app":       "mongodb",
				"free5gc":   free5gc.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "mongodb",
					Protocol: corev1.ProtocolTCP,
					Port:     27017,
					TargetPort: intstr.FromInt(27017),
				},
			},
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile MongoDB service: %w", err)
	}
	log.Info("Reconciled MongoDB service", "operation", op)

	// Update MongoDB status
	deploy = &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-mongodb", free5gc.Name),
		Namespace: free5gc.Namespace,
	}, deploy)

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		status := corev1alpha1.ComponentStatus{
			Phase:         "Running",
			ReadyReplicas: deploy.Status.ReadyReplicas,
			Replicas:      deploy.Status.Replicas,
		}

		if deploy.Status.ReadyReplicas != deploy.Status.Replicas {
			status.Phase = "Pending"
			status.Message = fmt.Sprintf("Waiting for %d/%d replicas to be ready", deploy.Status.ReadyReplicas, deploy.Status.Replicas)
		}

		free5gc.Status.MongoDB = status
	}

	return nil
}

func (r *Free5GCReconciler) reconcileUPF(ctx context.Context, free5gc *corev1alpha1.Free5GC) error {
	log := log.FromContext(ctx)

	if free5gc.Spec.UPF == nil {
		return nil
	}

	// Handle ULCL-enabled configuration
	if free5gc.Spec.UPF.ULCL != nil && free5gc.Spec.UPF.ULCL.Enabled {
		// Create UPF instances for ULCL
		for _, instance := range free5gc.Spec.UPF.ULCL.Instances {
			if err := r.reconcileUPFInstance(ctx, free5gc, instance); err != nil {
				return err
			}
		}
		return nil
	}

	// Handle standard UPF configuration
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-upf", free5gc.Name),
			Namespace: free5gc.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		if err := ctrl.SetControllerReference(free5gc, deploy, r.Scheme); err != nil {
			return err
		}

		replicas := int32(1)
		if free5gc.Spec.UPF.Replicas != nil {
			replicas = *free5gc.Spec.UPF.Replicas
		}

		deploy.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "free5gc",
					"component": "upf",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "free5gc",
						"component": "upf",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      "upf",
							Image:     free5gc.Spec.UPF.Image,
							Resources: free5gc.Spec.UPF.Resources,
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_ADMIN"},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Add network interfaces if specified
		if free5gc.Spec.Network.N3Network != nil {
			deploy.Spec.Template.Annotations = map[string]string{
				"k8s.v1.cni.cncf.io/networks": free5gc.Spec.Network.N3Network.Name,
			}
		}
		if free5gc.Spec.Network.N4Network != nil {
			if deploy.Spec.Template.Annotations == nil {
				deploy.Spec.Template.Annotations = make(map[string]string)
			}
			if _, ok := deploy.Spec.Template.Annotations["k8s.v1.cni.cncf.io/networks"]; ok {
				deploy.Spec.Template.Annotations["k8s.v1.cni.cncf.io/networks"] += "," + free5gc.Spec.Network.N4Network.Name
			} else {
				deploy.Spec.Template.Annotations["k8s.v1.cni.cncf.io/networks"] = free5gc.Spec.Network.N4Network.Name
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile UPF deployment: %w", err)
	}
	log.Info("Reconciled UPF deployment", "operation", op)

	// Create UPF service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-upf", free5gc.Name),
			Namespace: free5gc.Namespace,
		},
	}

	op, err = controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := ctrl.SetControllerReference(free5gc, svc, r.Scheme); err != nil {
			return err
		}

		svc.Spec = corev1.ServiceSpec{
			Selector: map[string]string{
				"app":       "free5gc",
				"component": "upf",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "pfcp",
					Protocol:   corev1.ProtocolUDP,
					Port:       8805,
					TargetPort: intstr.FromInt(8805),
				},
			},
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile UPF service: %w", err)
	}
	log.Info("Reconciled UPF service", "operation", op)

	return nil
}

func (r *Free5GCReconciler) reconcileUPFInstance(ctx context.Context, free5gc *corev1alpha1.Free5GC, instance corev1alpha1.UPFInstance) error {
	log := log.FromContext(ctx)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-upf-%s", free5gc.Name, instance.Name),
			Namespace: free5gc.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		if err := ctrl.SetControllerReference(free5gc, deploy, r.Scheme); err != nil {
			return err
		}

		replicas := int32(1)
		if instance.Replicas != nil {
			replicas = *instance.Replicas
		}

		deploy.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "free5gc",
					"component": "upf",
					"instance":  instance.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "free5gc",
						"component": "upf",
						"instance":  instance.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      "upf",
							Image:     instance.Image,
							Resources: instance.Resources,
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_ADMIN"},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "UPF_NAME",
									Value: instance.Name,
								},
							},
						},
					},
				},
			},
		}

		// Add network interfaces if specified
		if free5gc.Spec.Network.N3Network != nil {
			deploy.Spec.Template.Annotations = map[string]string{
				"k8s.v1.cni.cncf.io/networks": free5gc.Spec.Network.N3Network.Name,
			}
		}
		if free5gc.Spec.Network.N4Network != nil {
			if deploy.Spec.Template.Annotations == nil {
				deploy.Spec.Template.Annotations = make(map[string]string)
			}
			if _, ok := deploy.Spec.Template.Annotations["k8s.v1.cni.cncf.io/networks"]; ok {
				deploy.Spec.Template.Annotations["k8s.v1.cni.cncf.io/networks"] += "," + free5gc.Spec.Network.N4Network.Name
			} else {
				deploy.Spec.Template.Annotations["k8s.v1.cni.cncf.io/networks"] = free5gc.Spec.Network.N4Network.Name
			}
		}
		if free5gc.Spec.Network.N9Network != nil {
			if deploy.Spec.Template.Annotations == nil {
				deploy.Spec.Template.Annotations = make(map[string]string)
			}
			if _, ok := deploy.Spec.Template.Annotations["k8s.v1.cni.cncf.io/networks"]; ok {
				deploy.Spec.Template.Annotations["k8s.v1.cni.cncf.io/networks"] += "," + free5gc.Spec.Network.N9Network.Name
			} else {
				deploy.Spec.Template.Annotations["k8s.v1.cni.cncf.io/networks"] = free5gc.Spec.Network.N9Network.Name
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile UPF instance deployment: %w", err)
	}
	log.Info("Reconciled UPF instance deployment", "instance", instance.Name, "operation", op)

	// Create UPF instance service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-upf-%s", free5gc.Name, instance.Name),
			Namespace: free5gc.Namespace,
		},
	}

	op, err = controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		if err := ctrl.SetControllerReference(free5gc, svc, r.Scheme); err != nil {
			return err
		}

		svc.Spec = corev1.ServiceSpec{
			Selector: map[string]string{
				"app":       "free5gc",
				"component": "upf",
				"instance":  instance.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "pfcp",
					Protocol:   corev1.ProtocolUDP,
					Port:       8805,
					TargetPort: intstr.FromInt(8805),
				},
			},
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile UPF instance service: %w", err)
	}
	log.Info("Reconciled UPF instance service", "instance", instance.Name, "operation", op)

	return nil
}

func (r *Free5GCReconciler) cleanupResources(ctx context.Context, free5gc *corev1alpha1.Free5GC) error {
	log := log.FromContext(ctx)

	// Delete all deployments
	if err := r.Client.DeleteAllOf(
		ctx,
		&appsv1.Deployment{},
		client.InNamespace(free5gc.Namespace),
		client.MatchingLabels(map[string]string{"app": "free5gc"}),
		client.PropagationPolicy(metav1.DeletePropagationForeground),
	); err != nil {
		return fmt.Errorf("failed to cleanup deployments: %w", err)
	}

	// Delete all services
	if err := r.Client.DeleteAllOf(
		ctx,
		&corev1.Service{},
		client.InNamespace(free5gc.Namespace),
		client.MatchingLabels(map[string]string{"app": "free5gc"}),
		client.PropagationPolicy(metav1.DeletePropagationForeground),
	); err != nil {
		return fmt.Errorf("failed to cleanup services: %w", err)
	}

	// Delete MongoDB resources
	if err := r.Client.DeleteAllOf(
		ctx,
		&appsv1.Deployment{},
		client.InNamespace(free5gc.Namespace),
		client.MatchingLabels(map[string]string{
			"app":     "mongodb",
			"free5gc": free5gc.Name,
		}),
		client.PropagationPolicy(metav1.DeletePropagationForeground),
	); err != nil {
		return fmt.Errorf("failed to cleanup mongodb deployment: %w", err)
	}

	if err := r.Client.DeleteAllOf(
		ctx,
		&corev1.Service{},
		client.InNamespace(free5gc.Namespace),
		client.MatchingLabels(map[string]string{
			"app":     "mongodb",
			"free5gc": free5gc.Name,
		}),
		client.PropagationPolicy(metav1.DeletePropagationForeground),
	); err != nil {
		return fmt.Errorf("failed to cleanup mongodb service: %w", err)
	}

	if err := r.Client.DeleteAllOf(
		ctx,
		&corev1.PersistentVolumeClaim{},
		client.InNamespace(free5gc.Namespace),
		client.MatchingLabels(map[string]string{
			"app":     "mongodb",
			"free5gc": free5gc.Name,
		}),
		client.PropagationPolicy(metav1.DeletePropagationForeground),
	); err != nil {
		return fmt.Errorf("failed to cleanup mongodb pvc: %w", err)
	}

	log.Info("Cleaned up all resources")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Free5GCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Free5GC{}).
		Complete(r)
}
