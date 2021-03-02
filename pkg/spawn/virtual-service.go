package spawn

import (
	labsv1alpha1 "github.com/ishankhare07/launchpad/api/v1alpha1"
	istionetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateVirtualService(workload *labsv1alpha1.Workload) *istiov1alpha3.VirtualService {
	vs := &istiov1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.GetIstioResourceName(),
			Namespace: "default",
		},
	}

	routeDestination := []*istionetworkingv1alpha3.HTTPRouteDestination{}

	for _, target := range workload.Spec.Targets {
		// extract route from CRD
		route := &istionetworkingv1alpha3.HTTPRouteDestination{
			Destination: &istionetworkingv1alpha3.Destination{
				Host:   target.TrafficSplit.DestinationHost,
				Subset: target.TrafficSplit.SubsetName,
			},
			Weight: int32(target.TrafficSplit.Weight),
		}

		// append routes into route destination
		routeDestination = append(routeDestination, route)
	}

	// append route destination to VirtualService.Routes
	vs.Spec.Http = append(vs.Spec.Http, &istionetworkingv1alpha3.HTTPRoute{
		Route: routeDestination,
	})

	// set hosts
	vs.Spec.Hosts = workload.GetHosts()

	return vs
}
