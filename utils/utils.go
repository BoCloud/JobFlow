package utils

import "k8s.io/apimachinery/pkg/util/json"

func GetDeepCopy(req interface{}) (interface{}, error) {
	by, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(by, &req); err != nil {
		return nil, err
	}
	return req, nil
}
