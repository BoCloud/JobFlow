package utils

const CreateByJobTemplate = "createByJobTemplate"

func GetCreateByJobTemplateValue(name, namespace string) string {
	return name + "." + namespace
}
