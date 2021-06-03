package singleton

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tkerror "github.com/darkowlzz/operator-toolkit/error"
	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

var _ = Describe("Singleton", func() {
	var c client.Client

	gameObj := &tdv1alpha1.Game{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-game",
			Namespace: "default",
		},
	}
	scheme := runtime.NewScheme()

	BeforeEach(func() {
		Expect(cfg).NotTo(BeNil())
		Expect(tdv1alpha1.AddToScheme(scheme)).To(Succeed())

		var err error
		c, err = client.New(cfg, client.Options{Scheme: scheme})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("When multiple instances exist", func() {
		game1 := gameObj.DeepCopy()
		game1.SetName("test-game1")
		game2 := gameObj.DeepCopy()
		game2.SetName("test-game2")
		existingObjects := []client.Object{game1, game2}
		gameSingleton, serr := GetInstance(&tdv1alpha1.GameList{}, scheme)
		Expect(serr).ToNot(HaveOccurred())

		BeforeEach(func() {
			createResources(c, existingObjects)
		})

		AfterEach(func() {
			deleteResources(c, existingObjects)
		})

		It("should return error", func() {
			By("Get singleton")
			o, e := gameSingleton(context.TODO(), c)
			Expect(e).To(HaveOccurred())
			Expect(o).To(BeNil())
			By("Checking the error to be multiple instances found")
			errResult, count := tkerror.IsMultipleInstancesFound(e)
			Expect(errResult).To(BeTrue())
			Expect(count).To(Equal(2))
		})
	})

	Context("When single instance exists", func() {
		game2 := gameObj.DeepCopy()
		game2.SetName("test-game2")
		existingObjects := []client.Object{game2}
		gameSingleton, serr := GetInstance(&tdv1alpha1.GameList{}, scheme)
		Expect(serr).ToNot(HaveOccurred())

		BeforeEach(func() {
			createResources(c, existingObjects)
		})

		AfterEach(func() {
			deleteResources(c, existingObjects)
		})

		It("should return the instance", func() {
			By("Get singleton")
			o, e := gameSingleton(context.TODO(), c)
			Expect(e).ToNot(HaveOccurred())
			Expect(o).ToNot(BeNil())
			Expect(o.GetName()).To(Equal("test-game2"))
		})
	})

	Context("When no instance exists", func() {
		gameSingleton, serr := GetInstance(&tdv1alpha1.GameList{}, scheme)
		Expect(serr).ToNot(HaveOccurred())

		It("should return nil", func() {
			By("Get singleton")
			o, e := gameSingleton(context.TODO(), c)
			Expect(e).ToNot(HaveOccurred())
			Expect(o).To(BeNil())
		})
	})

	Context("When bad singleton provider is created", func() {
		Context("with nil object list", func() {
			It("should return error", func() {
				gs, serr := GetInstance(nil, scheme)
				Expect(serr).To(HaveOccurred())
				Expect(gs).To(BeNil())
			})
		})

		Context("with nil scheme", func() {
			It("should return error", func() {
				gs, serr := GetInstance(&tdv1alpha1.GameList{}, nil)
				Expect(serr).To(HaveOccurred())
				Expect(gs).To(BeNil())
			})
		})
	})
})

func createResources(c client.Client, objs []client.Object) {
	for _, o := range objs {
		Expect(c.Create(context.TODO(), o)).To(Succeed())
	}
}

func deleteResources(c client.Client, objs []client.Object) {
	for _, o := range objs {
		Expect(c.Delete(context.TODO(), o)).To(Succeed())
	}
}
