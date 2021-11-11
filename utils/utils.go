package utils

const CreateByJobTemplate = "volcano.sh/createByJobTemplate"

func GetCreateByJobTemplateValue(namespace, name string) string {
	return namespace + "." + name
}
