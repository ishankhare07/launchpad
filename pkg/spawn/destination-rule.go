package spawn

import (
	labsv1alpha1 "github.com/ishankhare07/launchpad/api/v1alpha1"
	istionetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateDestinationRule(workload *labsv1alpha1.Workload) *istiov1alpha3.DestinationRule {
	dr := &istiov1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.GetIstioResourceName(),
			Namespace: workload.GetNamespace(),
		},
	}

	for _, target := range workload.Spec.Targets {
		dr.Spec.Subsets = append(dr.Spec.Subsets, &istionetworkingv1alpha3.Subset{
			Name:   target.TrafficSplit.SubsetName,
			Labels: target.TrafficSplit.SubsetLabels,
		})
	}

	dr.Spec.Host = workload.Spec.Host

	return dr
}
