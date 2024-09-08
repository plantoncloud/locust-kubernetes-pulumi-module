package pkg

import (
	"github.com/pkg/errors"
	"github.com/plantoncloud/pulumi-module-golang-commons/pkg/datatypes/stringmaps/mergestringmaps"
	"github.com/plantoncloud/pulumi-module-golang-commons/pkg/provider/kubernetes/containerresources"
	kubernetescorev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func locust(ctx *pulumi.Context, locals *Locals,
	createdNamespace *kubernetescorev1.Namespace, labels map[string]string) error {
	// Create a ConfigMap for the main.py file
	_, err := kubernetescorev1.NewConfigMap(ctx, "main-py", &kubernetescorev1.ConfigMapArgs{
		Metadata: metav1.ObjectMetaPtrInput(&metav1.ObjectMetaArgs{
			Name:      pulumi.String(vars.MainPyConfigMapName),
			Namespace: createdNamespace.Metadata.Name(),
			Labels:    pulumi.ToStringMap(labels),
		}),
		Data: pulumi.StringMap{
			"main.py": pulumi.String(locals.LocustKubernetes.Spec.LoadTest.MainPyContent),
		},
	}, pulumi.Timeouts(&pulumi.CustomTimeouts{Create: "10s", Update: "10s", Delete: "10s"}),
		pulumi.Parent(createdNamespace))
	if err != nil {
		return errors.Wrap(err, "failed to create main py configmap")
	}

	// Create a ConfigMap for the lib files
	_, err = kubernetescorev1.NewConfigMap(ctx, "lib-files", &kubernetescorev1.ConfigMapArgs{
		Metadata: metav1.ObjectMetaPtrInput(&metav1.ObjectMetaArgs{
			Name:      pulumi.String(vars.LibFilesConfigMapName),
			Namespace: createdNamespace.Metadata.Name(),
			Labels:    pulumi.ToStringMap(labels),
		}),
		Data: pulumi.ToStringMap(locals.LocustKubernetes.Spec.LoadTest.LibFilesContent),
	}, pulumi.Timeouts(&pulumi.CustomTimeouts{Create: "10s", Update: "10s", Delete: "10s"}),
		pulumi.Parent(createdNamespace))

	if err != nil {
		return errors.Wrap(err, "failed to create lib files configmap")
	}

	//https://github.com/deliveryhero/helm-charts/tree/master/stable/locust#values
	// helm values
	var helmValues = pulumi.Map{
		"fullnameOverride": pulumi.String(locals.LocustKubernetes.Metadata.Name),
		"master": pulumi.Map{
			"replicas":  pulumi.Int(locals.LocustKubernetes.Spec.MasterContainer.Replicas),
			"resources": containerresources.ConvertToPulumiMap(locals.LocustKubernetes.Spec.MasterContainer.Resources),
		},
		"worker": pulumi.Map{
			"replicas":  pulumi.Int(locals.LocustKubernetes.Spec.WorkerContainer.Replicas),
			"resources": containerresources.ConvertToPulumiMap(locals.LocustKubernetes.Spec.WorkerContainer.Resources),
		},
		"loadtest": pulumi.Map{
			"name":                        pulumi.String(locals.LocustKubernetes.Spec.LoadTest.Name),
			"locust_locustfile_configmap": pulumi.String(vars.MainPyConfigMapName),
			"locust_lib_configmap":        pulumi.String(vars.LibFilesConfigMapName),
		},
	}
	mergestringmaps.MergeMapToPulumiMap(helmValues, locals.LocustKubernetes.Spec.HelmValues)

	// Deploying a Locust Helm chart from the Helm repository.
	_, err = helmv3.NewChart(ctx, locals.LocustKubernetes.Metadata.Id, helmv3.ChartArgs{
		Chart:     pulumi.String("locust"),
		Version:   pulumi.String("0.31.5"), // Use the Helm chart version you want to install
		Namespace: pulumi.String(locals.Namespace),
		Values:    helmValues,
		//if you need to add the repository, you can specify `repo url`:
		FetchArgs: helmv3.FetchArgs{
			Repo: pulumi.String("https://charts.deliveryhero.io"), // The URL for the Helm chart repository
		},
	}, pulumi.Timeouts(&pulumi.CustomTimeouts{Create: "2m", Update: "2m", Delete: "2m"}), pulumi.Parent(createdNamespace))

	if err != nil {
		return errors.Wrap(err, "failed to create locust resource")
	}
	return nil
}
