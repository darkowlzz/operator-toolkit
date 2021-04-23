package admission

import (
	"context"

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

var _ = Describe("Defaulter Webhooks", func() {

	err := appv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	decoder, _ := admission.NewDecoder(scheme.Scheme)

	Context("when there's no defaulting function and the target object is a native resource", func() {
		f := &fakeMutator{
			RequireDefaultingToReturn: true,
			NewObject:                 &corev1.ConfigMap{},
		}

		handler := mutatingHandler{defaulter: f, decoder: decoder}

		It("should succeed", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.defaulter.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
		})
	})

	Context("when the target object is a custom resource", func() {
		f := &fakeMutator{
			RequireDefaultingToReturn: true,
			NewObject:                 &appv1alpha1.Game{},
		}

		handler := mutatingHandler{defaulter: f, decoder: decoder}

		It("should succeed", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.defaulter.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())
		})
	})

	Context("when the defaulting functions are chained", func() {
		defFunc1 := fakeDefaultFunc{}
		defFunc2 := fakeDefaultFunc{}
		defFunc3 := fakeDefaultFunc{}

		f := &fakeMutator{
			RequireDefaultingToReturn: true,
			NewObject:                 &corev1.ConfigMap{},
			DefaultFuncs: []DefaultFunc{
				defFunc1.MutateFunc(),
				defFunc2.MutateFunc(),
				defFunc3.MutateFunc(),
			},
		}

		handler := mutatingHandler{defaulter: f, decoder: decoder}

		BeforeEach(func() {
			// Reset all the defaulting funcs.
			defFunc1.Reset()
			defFunc2.Reset()
			defFunc3.Reset()
		})

		It("should call all the chained functions", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.defaulter.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())

			callCount := defFunc1.Count() + defFunc2.Count() + defFunc3.Count()
			Expect(callCount).Should(Equal(3))
		})
	})

	Context("when require defaulting returns false", func() {
		defFunc1 := fakeDefaultFunc{}
		defFunc2 := fakeDefaultFunc{}
		defFunc3 := fakeDefaultFunc{}

		f := &fakeMutator{
			RequireDefaultingToReturn: false,
			NewObject:                 &corev1.ConfigMap{},
			DefaultFuncs: []DefaultFunc{
				defFunc1.MutateFunc(),
				defFunc2.MutateFunc(),
				defFunc3.MutateFunc(),
			},
		}

		handler := mutatingHandler{defaulter: f, decoder: decoder}

		BeforeEach(func() {
			// Reset all the defaulting funcs.
			defFunc1.Reset()
			defFunc2.Reset()
			defFunc3.Reset()
		})

		It("should not call the chained functions", func() {
			response := handler.Handle(context.TODO(), admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw:    []byte("{}"),
						Object: handler.defaulter.GetNewObject(),
					},
				},
			})
			Expect(response.Allowed).Should(BeTrue())

			callCount := defFunc1.Count() + defFunc2.Count() + defFunc3.Count()
			Expect(callCount).Should(Equal(0))
		})
	})
})

type fakeMutator struct {
	DefaultFuncs              []DefaultFunc
	RequireDefaultingToReturn bool
	NewObject                 client.Object
}

var _ Defaulter = &fakeMutator{}

func (m *fakeMutator) Default() []DefaultFunc {
	return m.DefaultFuncs
}

func (m *fakeMutator) RequireDefaulting(obj client.Object) bool {
	return m.RequireDefaultingToReturn
}

func (m *fakeMutator) GetNewObject() client.Object {
	return m.NewObject
}

// fakeDefaultFunc is a fake default function with a call counter.
type fakeDefaultFunc struct {
	callCount int
}

func (f *fakeDefaultFunc) MutateFunc() DefaultFunc {
	return func(ctx context.Context, obj client.Object) {
		f.callCount++
	}
}

func (f *fakeDefaultFunc) Count() int {
	return f.callCount
}

func (f *fakeDefaultFunc) Reset() {
	f.callCount = 0
}
