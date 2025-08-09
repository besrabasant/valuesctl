package patcher

import (
	jsonpatch "github.com/evanphx/json-patch/v5"
	"sigs.k8s.io/yaml"
)

// MergePatchYAML computes an RFC 7396 merge patch from old->desired and applies it.
func MergePatchYAML(oldYAML, desiredYAML []byte) ([]byte, error) {
	oldJSON, err := yaml.YAMLToJSON(oldYAML)
	if err != nil {
		return nil, err
	}
	desiredJSON, err := yaml.YAMLToJSON(desiredYAML)
	if err != nil {
		return nil, err
	}
	patch, err := jsonpatch.CreateMergePatch(oldJSON, desiredJSON)
	if err != nil {
		return nil, err
	}
	newJSON, err := jsonpatch.MergePatch(oldJSON, patch)
	if err != nil {
		return nil, err
	}
	return yaml.JSONToYAML(newJSON)
}