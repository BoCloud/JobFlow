package router

import (
	"fmt"
	"net/http"
	"sync"
)

type AdmissionHandler func(w http.ResponseWriter, r *http.Request)

var admissionMap = make(map[string]*AdmissionService)
var admissionMutex sync.Mutex

func RegisterAdmission(service *AdmissionService) error {
	admissionMutex.Lock()
	defer admissionMutex.Unlock()

	if _, found := admissionMap[service.Path]; found {
		return fmt.Errorf("duplicated admission service for %s", service.Path)
	}

	// Also register handler to the service.
	service.Handler = func(w http.ResponseWriter, r *http.Request) {
		Serve(w, r, service.Func)
	}

	admissionMap[service.Path] = service

	return nil
}

func ForEachAdmission(admissions []string, handler func(*AdmissionService)) {
	for _, admission := range admissions {
		if service, found := admissionMap[admission]; found {
			handler(service)
		}
	}
}
