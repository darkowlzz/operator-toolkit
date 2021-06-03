package function

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

var _ = Describe("ValidateSingletonCreate", func() {

	gameObj := &tdv1alpha1.Game{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-game",
			Namespace: "default",
		},
	}

	Context("Create objects", func() {
		It("should create first new object", func() {
			go1 := gameObj.DeepCopy()
			go1.SetName("test-obj1")
			Expect(k8sClient.Create(context.TODO(), go1)).ToNot(HaveOccurred())
		})

		It("should fail creating a second new object", func() {
			go2 := gameObj.DeepCopy()
			go2.SetName("test-obj2")
			Expect(k8sClient.Create(context.TODO(), go2)).To(HaveOccurred())
		})
	})
})
