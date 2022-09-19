package transformer

import (
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// Constants for FunctionConfig `SetNamespace`
	fnConfigGroup   = "fn.kpt.dev"
	fnConfigVersion = "v1alpha1"
	fnConfigKind    = "SetValue"
	// The ConfigMap name generated from variant constructor
	//builtinConfigMapName = "kptfile.kpt.dev"
)

// SetValue contains the information to perform the mutator function on a package
type SetValue struct {
	Spec []*SetValueSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
	// sWebhookResults is used internally to track which resources were updated
	Results setValueResults
}

type SetValueSpec struct {
	Data    string                  `json:"data,omitempty" yaml:"data,omitempty"`
	Targets []*types.TargetSelector `json:"targets,omitempty" yaml:"targets,omitempty"`
}

// webhookResultKey is used as a unique identifier for webhook results
type setValueKey struct {
	ResourceRef fn.ResourceRef
	// FilePath is the file path of the resource
	FilePath string
	// FileIndex is the file index of the resource
	FileIndex int
	// FieldPath is field path of the serviceaccount field
	FieldPath string
}

// setValueresult tracks the operation
type setValueresult struct {
	Operation string
}

// setValueResults tracks the operation matching the key
type setValueResults map[setValueKey]setValueresult

func Run(rl *fn.ResourceList) (bool, error) {
	tc := SetValue{}
	if err := tc.config(rl.FunctionConfig); err != nil {
		rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, rl.FunctionConfig))
	}
	tc.Transform(rl)
	return true, nil
}

func (t *SetValue) config(fc *fn.KubeObject) error {
	switch {
	case fc.IsEmpty():
		return fmt.Errorf("FunctionConfig is missing. Expect `%s.%s.%s`",
			fnConfigKind, fnConfigVersion, fnConfigGroup)
	case fc.IsGVK(fnConfigGroup, fnConfigVersion, fnConfigKind):
		fc.AsOrDie(&t)
	default:
		return fmt.Errorf("unknown functionConfig Kind=%v ApiVersion=%v, expect `ConfigMap.v1` or `%s.%s.%s`",
			fc.GetKind(), fc.GetAPIVersion(), fnConfigKind, fnConfigVersion, fnConfigGroup)
	}
	return nil
}

func (t *SetValue) Transform(rl *fn.ResourceList) {
	for _, sv := range t.Spec {
		data, err := yaml.Parse(sv.Data)
		if err != nil {
			rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, rl.FunctionConfig))
		}

		for _, target := range sv.Targets {
			for i, o := range rl.Items {
				if target.Select.Kind == o.GetKind() && target.Select.Name == o.GetName() {
					node, err := yaml.Parse(o.String())
					if err != nil {
						rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, o))
					}

					// per fieldPath apply the replacement
					for _, fieldPath := range target.FieldPaths {
						_, err = node.Pipe(
							yaml.LookupCreate(yaml.SequenceNode, strings.Split(fieldPath, ".")...),
							yaml.Append(data.YNode().Content...),
						)
						if err != nil {
							rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, o))
						}
					}
					str, err := node.String()
					if err != nil {
						rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, o))
					}
					newObj, err := fn.ParseKubeObject([]byte(str))
					if err != nil {
						rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, o))
					}
					rl.Items[i] = newObj
				}
			}
		}
	}
}
