package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"
	"github.com/testifysec/judge-k8s/cmd/options"
	"github.com/testifysec/judge-k8s/pkg/rules"

	"github.com/labstack/echo"
	admv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PostValidatingAdmission(o options.ServeOptions) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[panic] recovered in PostValidatingAdmission: %v", r)
				debug.PrintStack()
			}
		}()

		fmt.Println("PostValidatingAdmission")
		var admissionReviewReq admv1.AdmissionReview
		c.Bind(&admissionReviewReq)
		podRaw := admissionReviewReq.Request.Object.Raw
		fmt.Println("podRaw/unmarshal pod")
		pod := &v1.Pod{}
		if err := json.Unmarshal(podRaw, pod); err != nil {
			return err
		}
		log.Println("podName", pod.Name)
		log.Printf("podNamespace %s", pod.Namespace)

		annotations := pod.GetAnnotations()
		if annotations == nil {
			log.Println("annotation: its null")
			annotations = make(map[string]string)
		}

		rekorStrings := []string{}
		admissionResponse := admv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: "Unknown Erorr",
				Code:    http.StatusForbidden,
			},

			UID: admissionReviewReq.Request.UID,
		}

		admissionReviewReq.Response = &admissionResponse

		for _, container := range pod.Spec.Containers {
			wp, err := rules.New(&o) // important to look at the rules
			if err != nil {
				admissionResponse.Result.Message = err.Error()
				return c.JSON(http.StatusOK, admissionReviewReq)
			}
			fmt.Println("wp.Envelopes")
			log.Println(wp.Envelopes)
			fmt.Println("wp.Manifest")
			fmt.Println(wp.Manifest)

			log.Printf("container.Image %s", container.Image)
			rekorUIDs, err := wp.Verify(container.Image)
			if err != nil {
				admissionReviewReq.Response.Warnings = append(admissionReviewReq.Response.Warnings, fmt.Sprintf("failed to verify image: %v", err))
				admissionReviewReq.Response.Result.Message = err.Error()
				return c.JSON(http.StatusOK, admissionReviewReq)

			}
			rekorStrings = append(rekorStrings, rekorUIDs...)
		}

		for i, rekorUID := range rekorStrings {
			annotations[fmt.Sprintf("testifysec.io/rekoruid%d", i)] = rekorUID
		}

		patch, err := createPatch(pod, annotations)
		if err != nil {
			return err
		}

		pt := func() *admv1.PatchType {
			pt := admv1.PatchTypeJSONPatch
			return &pt
		}()

		admissionReviewReq.Response.Allowed = true
		admissionReviewReq.Response.PatchType = pt
		admissionReviewReq.Response.Patch = patch
		admissionReviewReq.Response.Result.Message = "OK"
		admissionReviewReq.Response.Result.Code = http.StatusOK

		return c.JSON(http.StatusOK, &admissionReviewReq)
	}
}

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	log.Println("updating Annotation...")
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

// create mutation patch for resoures
func createPatch(pod *v1.Pod, annotations map[string]string) ([]byte, error) {
	var patch []patchOperation
	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)
	spew.Dump(patch)
	return json.Marshal(patch)
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}
