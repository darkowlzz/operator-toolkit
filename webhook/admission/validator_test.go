package admission

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	appv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

var _ = Describe("Validating Webhooks", func() {

	err := appv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	decoder, _ := admission.NewDecoder(scheme.Scheme)

	Context("when there's no validating function and target object is a native resource", func() {
		f := &fakeValidator{
			RequireValidityToReturn: true,
			NewObject:               &corev1.ConfigMap{},
		}

		handler := validatingHandler{validator: f, decoder: decoder}

		It("should return 200 in response when create succeeds", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))
		})

		It("should return 200 in response when update succeeds", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))
		})

		It("should return 200 in response when delete succeeds", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))
		})
	})

	Context("when the target object is a custom resource", func() {
		f := &fakeValidator{
			RequireValidityToReturn: true,
			NewObject:               &appv1alpha1.Game{},
		}

		handler := validatingHandler{validator: f, decoder: decoder}

		It("should return 200 in response when create succeeds", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))
		})

		It("should return 200 in response when update succeeds", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))
		})

		It("should return 200 in response when delete succeeds", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))
		})
	})

	Context("when validating functions are chained, no error", func() {
		// Create validate functions to chain together.
		validateFunc1 := fakeValidateFunc{ErrorToReturn: nil}
		validateFunc2 := fakeValidateFunc{ErrorToReturn: nil}
		validateFunc3 := fakeValidateFunc{ErrorToReturn: nil}

		f := &fakeValidator{
			RequireValidityToReturn: true,
			NewObject:               &corev1.ConfigMap{},
			CreateFuncs: []ValidateCreateFunc{
				validateFunc1.CreateFunc(),
				validateFunc2.CreateFunc(),
				validateFunc3.CreateFunc(),
			},
			UpdateFuncs: []ValidateUpdateFunc{
				validateFunc1.UpdateFunc(),
				validateFunc2.UpdateFunc(),
				validateFunc3.UpdateFunc(),
			},
			DeleteFuncs: []ValidateDeleteFunc{
				validateFunc1.DeleteFunc(),
				validateFunc2.DeleteFunc(),
				validateFunc3.DeleteFunc(),
			},
		}

		handler := validatingHandler{validator: f, decoder: decoder}

		BeforeEach(func() {
			// Reset all the validate funcs.
			validateFunc1.Reset()
			validateFunc2.Reset()
			validateFunc3.Reset()
		})

		It("should call all the chained functions in response to create request", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))

			callCount := validateFunc1.Count() + validateFunc2.Count() + validateFunc3.Count()
			Expect(callCount).Should(Equal(3))
		})

		It("should call all the chained functions in response to update request", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))

			callCount := validateFunc1.Count() + validateFunc2.Count() + validateFunc3.Count()
			Expect(callCount).Should(Equal(3))
		})

		It("should call all the chained functions in response to delete request", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))

			callCount := validateFunc1.Count() + validateFunc2.Count() + validateFunc3.Count()
			Expect(callCount).Should(Equal(3))
		})
	})

	Context("when validating functions return error", func() {
		// Create validate functions to chain together.
		validateFunc1 := fakeValidateFunc{ErrorToReturn: nil}
		validateFunc2 := fakeValidateFunc{ErrorToReturn: fmt.Errorf("fake error")}
		validateFunc3 := fakeValidateFunc{ErrorToReturn: nil}

		f := &fakeValidator{
			RequireValidityToReturn: true,
			NewObject:               &corev1.ConfigMap{},
			CreateFuncs: []ValidateCreateFunc{
				validateFunc1.CreateFunc(),
				validateFunc2.CreateFunc(),
				validateFunc3.CreateFunc(),
			},
			UpdateFuncs: []ValidateUpdateFunc{
				validateFunc1.UpdateFunc(),
				validateFunc2.UpdateFunc(),
				validateFunc3.UpdateFunc(),
			},
			DeleteFuncs: []ValidateDeleteFunc{
				validateFunc1.DeleteFunc(),
				validateFunc2.DeleteFunc(),
				validateFunc3.DeleteFunc(),
			},
		}

		handler := validatingHandler{validator: f, decoder: decoder}

		BeforeEach(func() {
			// Reset all the validate funcs.
			validateFunc1.Reset()
			validateFunc2.Reset()
			validateFunc3.Reset()
		})

		It("should not call all the chained functions in response to create request", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeFalse())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusForbidden)))

			callCount := validateFunc1.Count() + validateFunc2.Count() + validateFunc3.Count()
			Expect(callCount).ShouldNot(Equal(3))
		})

		It("should not call all the chained functions in response to update request", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeFalse())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusForbidden)))

			callCount := validateFunc1.Count() + validateFunc2.Count() + validateFunc3.Count()
			Expect(callCount).ShouldNot(Equal(3))
		})

		It("should not call all the chained functions in response to delete request", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeFalse())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusForbidden)))

			callCount := validateFunc1.Count() + validateFunc2.Count() + validateFunc3.Count()
			Expect(callCount).ShouldNot(Equal(3))
		})
	})

	Context("when require validating returns false", func() {
		// Create validate functions to chain together.
		validateFunc1 := fakeValidateFunc{ErrorToReturn: nil}
		validateFunc2 := fakeValidateFunc{ErrorToReturn: nil}
		validateFunc3 := fakeValidateFunc{ErrorToReturn: nil}

		f := &fakeValidator{
			RequireValidityToReturn: false,
			NewObject:               &corev1.ConfigMap{},
			CreateFuncs: []ValidateCreateFunc{
				validateFunc1.CreateFunc(),
				validateFunc2.CreateFunc(),
				validateFunc3.CreateFunc(),
			},
			UpdateFuncs: []ValidateUpdateFunc{
				validateFunc1.UpdateFunc(),
				validateFunc2.UpdateFunc(),
				validateFunc3.UpdateFunc(),
			},
			DeleteFuncs: []ValidateDeleteFunc{
				validateFunc1.DeleteFunc(),
				validateFunc2.DeleteFunc(),
				validateFunc3.DeleteFunc(),
			},
		}

		handler := validatingHandler{validator: f, decoder: decoder}

		BeforeEach(func() {
			// Reset all the validate funcs.
			validateFunc1.Reset()
			validateFunc2.Reset()
			validateFunc3.Reset()
		})

		It("should not call chained functions in response to create request", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))

			callCount := validateFunc1.Count() + validateFunc2.Count() + validateFunc3.Count()
			Expect(callCount).Should(Equal(0))
		})

		It("should not call chained functions in response to update request", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))

			callCount := validateFunc1.Count() + validateFunc2.Count() + validateFunc3.Count()
			Expect(callCount).Should(Equal(0))
		})

		It("should not call chained functions in response to delete request", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					OldObject: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.validator.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
			Expect(response.Result.Code).Should(Equal(int32(http.StatusOK)))

			callCount := validateFunc1.Count() + validateFunc2.Count() + validateFunc3.Count()
			Expect(callCount).Should(Equal(0))
		})
	})
})

type fakeValidator struct {
	CreateFuncs             []ValidateCreateFunc
	UpdateFuncs             []ValidateUpdateFunc
	DeleteFuncs             []ValidateDeleteFunc
	RequireValidityToReturn bool
	NewObject               client.Object
}

var _ Validator = &fakeValidator{}

func (v *fakeValidator) ValidateCreate() []ValidateCreateFunc {
	return v.CreateFuncs
}

func (v *fakeValidator) ValidateUpdate() []ValidateUpdateFunc {
	return v.UpdateFuncs
}

func (v *fakeValidator) ValidateDelete() []ValidateDeleteFunc {
	return v.DeleteFuncs
}

func (v *fakeValidator) RequireValidating(obj client.Object) bool {
	return v.RequireValidityToReturn
}

func (v *fakeValidator) GetNewObject() client.Object {
	return v.NewObject
}

// fakeValidateFunc is a fake validate function with a call counter. It has
// methods for create, update and delete validate functions which enables it to
// be used as any of the types of validating functions.
type fakeValidateFunc struct {
	callCount     int
	ErrorToReturn error
}

func (f *fakeValidateFunc) CreateFunc() ValidateCreateFunc {
	return func(ctx context.Context, obj client.Object) error {
		f.callCount++
		return f.ErrorToReturn
	}
}

func (f *fakeValidateFunc) UpdateFunc() ValidateUpdateFunc {
	return func(ctx context.Context, obj client.Object, old client.Object) error {
		f.callCount++
		return f.ErrorToReturn
	}
}

func (f *fakeValidateFunc) DeleteFunc() ValidateDeleteFunc {
	return func(ctx context.Context, obj client.Object) error {
		f.callCount++
		return f.ErrorToReturn
	}
}

func (f *fakeValidateFunc) Count() int {
	return f.callCount
}

func (f *fakeValidateFunc) Reset() {
	f.callCount = 0
}
