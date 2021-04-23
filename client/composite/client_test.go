package composite

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// ctrl "sigs.k8s.io/controller-runtime"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Composite client", func() {
	var (
		dCli  client.Client
		cache fakeReader
	)

	BeforeEach(func() {
		// Create a fake cache.
		cache = fakeReader{}

		// Create a delegating client with a cache and uncached client.
		var err error
		dCli, err = client.NewDelegatingClient(client.NewDelegatingClientInput{
			CacheReader: &cache,
			Client:      k8sClient,
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should fetch created resource with composite client", func() {
		// Create a new composite client using cached and uncached clients.
		cCli := NewClient(dCli, k8sClient, Options{})

		// Create a resource.
		nsx := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "some-ns-for-comp-client"},
		}
		Expect(k8sClient.Create(context.Background(), &nsx)).To(Succeed())

		defer func() {
			Expect(k8sClient.Delete(context.Background(), &nsx)).To(Succeed())
		}()

		key := client.ObjectKeyFromObject(&nsx)

		By("Expecting not to get the object using cached client")
		Expect(dCli.Get(context.Background(), key, &nsx)).NotTo(Succeed())
		Expect(cache.Called).To(Equal(1))

		By("Expecting to get the object using composite client")
		Expect(cCli.Get(context.Background(), key, &nsx)).To(Succeed())
		Expect(cache.Called).To(Equal(2))
	})

	It("should not fetch non-existing resource with composite client", func() {
		cCli := NewClient(dCli, k8sClient, Options{})

		// Random object key that doesn't exist.
		key := client.ObjectKey{Name: "foo999"}
		nsx := corev1.Namespace{}

		By("Expecting not to get the object using cached client")
		Expect(dCli.Get(context.Background(), key, &nsx)).NotTo(Succeed())
		Expect(cache.Called).To(Equal(1))

		By("Expecting not to get the object using composite client")
		Expect(cCli.Get(context.Background(), key, &nsx)).NotTo(Succeed())
		Expect(cache.Called).To(Equal(2))
	})

	It("list from the cached client", func() {
		cCli := NewClient(dCli, k8sClient, Options{RawListing: false})

		nsl := corev1.NamespaceList{}
		Expect(cCli.List(context.Background(), &nsl, []client.ListOption{}...)).To(Succeed())
		Expect(cache.Called).To(Equal(1))
		Expect(len(nsl.Items)).To(Equal(0))
	})

	It("list from the uncached client", func() {
		cCli := NewClient(dCli, k8sClient, Options{RawListing: true})

		nsl := corev1.NamespaceList{}
		Expect(cCli.List(context.Background(), &nsl, []client.ListOption{}...)).To(Succeed())
		Expect(cache.Called).To(Equal(0))
		Expect(len(nsl.Items) > 0).To(BeTrue())
	})
})

// fakeReader is used with a delegating client as a fake cache.
type fakeReader struct {
	Called int
}

// Get implements the CacheReader interface Get method. It returns NOT FOUND
// error to enable testing when an object is not in cache.
func (f *fakeReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	f.Called = f.Called + 1
	return apierrors.NewNotFound(schema.GroupResource{}, "")
}

// List implements the CacheReader interface List method.
func (f *fakeReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	f.Called = f.Called + 1
	return nil
}
