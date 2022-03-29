package validate

import (
	"fmt"

	"k8s.io/api/admission/v1beta1"
	whv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/klog"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
	"jobflow/utils"
	"jobflow/webhooks/router"
	"jobflow/webhooks/schema"
)

func init() {
	_ = router.RegisterAdmission(service)
}

var service = &router.AdmissionService{
	Path: "/jobflows/validate",
	Func: AdmitJobFlows,

	ValidatingConfig: &whv1beta1.ValidatingWebhookConfiguration{
		Webhooks: []whv1beta1.ValidatingWebhook{{
			Name: "validatejobflow.volcano.sh",
			Rules: []whv1beta1.RuleWithOperations{
				{
					Operations: []whv1beta1.OperationType{whv1beta1.Create},
					Rule: whv1beta1.Rule{
						APIGroups:   []string{"flow.volcano.sh"},
						APIVersions: []string{"v1alpha1"},
						Resources:   []string{"jobflows"},
					},
				},
			},
		}},
	},
}

// AdmitJobFlows is to admit jobFlows and return response.
func AdmitJobFlows(ar v1beta1.AdmissionReview) error {
	klog.V(3).Infof("admitting jobflows -- %s", ar.Request.Operation)

	jobFlow, err := schema.DecodeJobFlow(ar.Request.Object, ar.Request.Resource)
	if err != nil {
		return err
	}

	switch ar.Request.Operation {
	case v1beta1.Create:
		err = validateJobFlowCreate(jobFlow)
	default:
		err = fmt.Errorf("only support 'CREATE' operation")
	}

	return err
}

func validateJobFlowCreate(jobFlow *jobflowv1alpha1.JobFlow) error {
	flows := jobFlow.Spec.Flows
	var msg string
	templateNames := map[string][]string{}
	vertexMap := make(map[string]*utils.Vertex)
	dag := &utils.DAG{}
	var duplicatedTemplate = false
	for _, template := range flows {
		if _, found := templateNames[template.Name]; found {
			// duplicate task name
			msg += fmt.Sprintf(" duplicated template name %s;", template.Name)
			duplicatedTemplate = true
			break
		} else {
			templateNames[template.Name] = template.DependsOn.Targets
			vertexMap[template.Name] = &utils.Vertex{Key: template.Name}
		}
	}
	// Skip closed-loop detection if there are duplicate templates
	if !duplicatedTemplate {
		// Build dag through dependencies
		for current, parents := range templateNames {
			if parents != nil && len(parents) > 0 {
				for _, parent := range parents {
					if _, found := vertexMap[parent]; !found {
						msg += fmt.Sprintf("cannot find the template: %s ", parent)
						vertexMap = nil
						break
					}
					dag.AddEdge(vertexMap[parent], vertexMap[current])
				}
			}
		}
		// Check if there is a closed loop
		for k := range vertexMap {
			if err := dag.BFS(vertexMap[k]); err != nil {
				msg += fmt.Sprintf("%v;", err)
				break
			}
		}
	}

	if msg != "" {
		return fmt.Errorf("failed to create jobFlow for: %s", msg)
	}

	return nil
}
