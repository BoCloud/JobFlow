package utils

const CreateByJobTemplate = "volcano.sh/createByJobTemplate"

func GetConnectionOfJobAndJobTemplate(namespace, name string) string {
	return namespace + "." + name
}
