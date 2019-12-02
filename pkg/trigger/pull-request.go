package trigger

import (
	"bytes"
	"text/template"

	"github.com/golang/glog"
	"github.com/knative-sample/tekton-serving/pkg/utils/kube"
	v1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"fmt"

	"github.com/ghodss/yaml"
	"k8s.io/api/rbac/v1beta1"
)

func (dp *Trigger) pullRequestMergedEvent(payload *EventInfo) error {
	glog.Infof("pullRequestMergedEvent, image: %s:%s ", payload.Repository.RepoFullName, payload.PushData.Tag)

	tmpl, err := template.ParseFiles(dp.TriggerConfig)
	if err != nil {
		glog.Errorf("Parse TriggerConfig error:%s ", err.Error())
		return err
	}
	args := &Args{}
	buf := &bytes.Buffer{}
	tmpl.Execute(buf, args)

	cfg, err := kube.GetKubeconfig()
	if err != nil {
		glog.Errorf("get kubeconfig error:%s ", err)
		return err
	}

	tektonClient, err := tektonclientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building Build clientset: %v", err)
	}

	jsonbts, _ := yaml.YAMLToJSON(buf.Bytes())
	u := &v1alpha1.PipelineRun{}
	if err := yaml.Unmarshal(jsonbts, u); err != nil {
		glog.Errorf("parse Build Object error:%s ", err.Error())
		return err
	}

	if u.Namespace == "" {
		u.Namespace = "default"
	}
	ps := make([]v1alpha1.Param, 0)
	for _, param := range u.Spec.Params {
		if param.Name == "imageTag" {
			param.Value = v1alpha1.ArrayOrString{
				Type:      v1alpha1.ParamTypeString,
				StringVal: payload.PushData.Tag,
			}
		}
		ps = append(ps, param)
	}
	u.Spec.Params = ps
	// bind role
	//if err := dp.bindServiceRole(fmt.Sprintf("%s-serving-role", u.Name), u.Namespace, u.Spec.ServiceAccount); err != nil {
	//	glog.Errorf("bindService Role error:%s ", err)
	//	return err
	//}

	if _, err := tektonClient.TektonV1alpha1().PipelineRuns(u.Namespace).Get(u.Name, metav1.GetOptions{}); err != nil {
		// The Build resource may not exist.
		if !errors.IsNotFound(err) {
			glog.Errorf("get build %s error:%s ", u.Name, err.Error())
			return err
		}
	} else {
		if err := tektonClient.TektonV1alpha1().PipelineRuns(u.Namespace).Delete(u.Name, &metav1.DeleteOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				glog.Errorf("delete build %s error:%s ", u.Name, err.Error())
				return err
			}
		}
	}

	if _, err := tektonClient.TektonV1alpha1().PipelineRuns(u.Namespace).Create(u); err != nil {
		glog.Errorf("create build %s error:%s ", u.Name, err.Error())
		return err
	}

	return nil
}

func (dp *Trigger) bindServiceRole(name, namespace string, serviceAccount string) error {
	newRole := &v1beta1.Role{
		Rules: []v1beta1.PolicyRule{
			{
				APIGroups: []string{"serving.knative.dev"},
				Resources: []string{"services"},
				Verbs:     []string{"get", "list", "create", "watch", "patch", "update"},
			},
		},
	}
	newRole.Name = name
	newRole.Namespace = namespace

	newRoleBind := &v1beta1.RoleBinding{
		Subjects: []v1beta1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: namespace,
			},
		},
		RoleRef: v1beta1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     newRole.Name,
		},
	}
	newRoleBind.Namespace = namespace
	newRoleBind.Name = fmt.Sprintf("%s-rolebinding", name)

	cfg, err := kube.GetKubeconfig()
	if err != nil {
		glog.Errorf("get kubeconfig error:%s ", err)
		return err
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building tekton clientset: %v", err)
	}

	// reconciler Role
	role, err := clientset.RbacV1beta1().Roles(namespace).Get(newRole.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			glog.Errorf("get Role: %s/%s error:%s ", newRole.Namespace, newRole.Name, err.Error())
			return err
		}

		if _, e := clientset.RbacV1beta1().Roles(namespace).Create(newRole); e != nil {
			glog.Errorf("create role:%s/%s error:%s", newRole.Namespace, newRole.Name, e.Error())
			return err
		}
	} else {
		role.Rules = newRole.Rules
		_, err := clientset.RbacV1beta1().Roles(namespace).Update(role)
		if err != nil {
			glog.Errorf("update Role: %s/%s error:%s ", role.Namespace, role.Name, err.Error())
			return err
		}
	}

	// reconciler RoleBinding
	rb, err := clientset.RbacV1beta1().RoleBindings(namespace).Get(newRoleBind.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			glog.Errorf("get RoleBinding: %s/%s error:%s ", newRoleBind.Namespace, newRoleBind.Name, err.Error())
			return err
		}

		if _, e := clientset.RbacV1beta1().RoleBindings(namespace).Create(newRoleBind); e != nil {
			glog.Errorf("create roleBinding:%s/%s error:%s", newRoleBind.Namespace, newRoleBind.Name, e.Error())
			return err
		}
	} else {
		rb.RoleRef = newRoleBind.RoleRef
		rb.Subjects = newRoleBind.Subjects
		_, err := clientset.RbacV1beta1().RoleBindings(namespace).Update(rb)
		if err != nil {
			glog.Errorf("update Role: %s/%s error:%s ", newRoleBind.Namespace, newRoleBind.Name, err.Error())
			return err
		}
	}

	return nil
}
