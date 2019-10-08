package deployer

import (
	"github.com/golang/glog"
	"github.com/knative-sample/tekton-serving/pkg/utils/kube"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving/v1beta1"
	servingclientset "github.com/knative/serving/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
	"fmt"
)

type Deployer struct {
	Image       string
	Namespace   string
	ServiceName string
	Port        string
}

func (dp *Deployer) Run() error {
	cfg, err := kube.GetKubeconfig()
	if err != nil {
		glog.Errorf("get kubeconfig error:%s ", err)
		return err
	}

	servingClient, err := servingclientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building Serving clientset: %v", err)
	}

	if svc, err := servingClient.ServingV1alpha1().Services(dp.Namespace).Get(dp.ServiceName, metav1.GetOptions{}); err != nil {
		// The Build resource may not exist.
		if !errors.IsNotFound(err) {
			glog.Errorf("get Serving %s/%s error:%s ", dp.Namespace, dp.ServiceName, err.Error())
			return err
		}

		// create Serving
		newSvc := &v1alpha1.Service{}
		newSvc.Namespace = dp.Namespace
		newSvc.Name = dp.ServiceName
		newSvc.Spec.Template = &v1alpha1.RevisionTemplateSpec{
			Spec: v1alpha1.RevisionSpec{
				RevisionSpec: v1beta1.RevisionSpec{
					PodSpec: v1beta1.PodSpec{
						Containers: []corev1.Container{
							{
								Image:           dp.Image,
							},
						},
					},
				},
			},
		}
		if _, err := servingClient.ServingV1alpha1().Services(dp.Namespace).Create(newSvc); err != nil {
			glog.Errorf("create serving: %s/%s error:%s", dp.Namespace, dp.ServiceName, err.Error())
			return err
		}
	} else {
		// Update Serving
		if svc.Spec.Template.Annotations == nil {
			svc.Spec.Template.Annotations = map[string]string{}
		}
		svc.Spec.Template.Name = ""
		svc.Spec.Template.Annotations["updated"] = fmt.Sprintf("%v", time.Now().Unix())
		svc.Spec.Template.Spec.Containers[0].Image = dp.Image
		traffics := make([]v1alpha1.TrafficTarget, 0 )
		hasLatestRevision := false
		//for _, traffic := range svc.Spec.Traffic  {
		//	if *traffic.LatestRevision == true {
		//		traffic.Tag = fmt.Sprintf("test-%v", time.Now().Unix())
		//		hasLatestRevision = true
		//	}
		//	traffics = append(traffics, traffic)
		//}
		for _, traffic := range svc.Status.Traffic  {
			traffic.URL = nil
			if *traffic.LatestRevision == true {
				//traffic.Tag = fmt.Sprintf("test-%v", time.Now().Unix())
				//hasLatestRevision = true
				latestRevision := false
				traffic.LatestRevision = &latestRevision
			}
			traffics = append(traffics, traffic)
		}
		if !hasLatestRevision {
			version := fmt.Sprintf("%s-%v", dp.ServiceName,time.Now().Unix())
			svc.Spec.Template.Name = version
			tt := v1alpha1.TrafficTarget{}
			tt.RevisionName = version
			tt.Tag = fmt.Sprintf("test-%v", time.Now().Unix())
			latestRevision := false
			tt.LatestRevision = &latestRevision
			traffics = append(traffics, tt)
		}
		svc.Spec.Traffic = traffics
		if _, err := servingClient.ServingV1alpha1().Services(dp.Namespace).Update(svc); err != nil {
			glog.Errorf("create serving: %s/%s error:%s", dp.Namespace, dp.ServiceName, err.Error())
			return err
		}
	}

	return nil
}
