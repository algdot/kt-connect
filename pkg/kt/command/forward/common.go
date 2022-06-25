package forward

import (
	"fmt"
	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/alibaba/kt-connect/pkg/kt/service/cluster"
	"github.com/alibaba/kt-connect/pkg/kt/transmission"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func RedirectService(serviceName string, localPort, remotePort int) error {
	svc, err := cluster.Ins().GetService(serviceName, opt.Get().Global.Namespace)
	if err != nil {
		return err
	}
	targetPort := intstr.IntOrString { Type: -1 }
	for _, p := range svc.Spec.Ports {
		if int(p.Port) == remotePort {
			targetPort = p.TargetPort
		}
	}
	if targetPort.Type == -1 {
		return fmt.Errorf("port %d not available for service %s", remotePort, serviceName)
	}
	pods, err := cluster.Ins().GetPodsByLabel(svc.Spec.Selector, opt.Get().Global.Namespace)
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("no pod available for service %s", serviceName)
	}
	podPort := -1
	if targetPort.Type == intstr.Int {
		podPort = int(targetPort.IntVal)
	} else {
		containerLoop:
		for _, c := range pods.Items[0].Spec.Containers {
			for _, p := range c.Ports {
				if p.Name == targetPort.StrVal {
					podPort = int(p.ContainerPort)
					break containerLoop
				}
			}
		}
	}
	if podPort == -1 {
		return fmt.Errorf("port %d not fit for any pod of service %s", remotePort, serviceName)
	}
	return transmission.SetupPortForwardToLocal(pods.Items[0].Name, podPort, localPort)
}

func RedirectAddress(remoteAddress string, localPort, remotePort int) error {
	return fmt.Errorf("redirecting to an arbitrary address havn't been implemented yet")
}