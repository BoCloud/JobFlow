package utils

const CreateByJobTemplate = "volcano.sh/createByJobTemplate"

// vcjob对jobTemplate引用的annotations的value值

func GetConnectionOfJobAndJobTemplate(namespace, name string) string {
	return namespace + "." + name
}
