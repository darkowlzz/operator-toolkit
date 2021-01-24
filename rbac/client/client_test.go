package client

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

var _ = Describe("RBAC client recording", func() {
	var count uint64 = 0
	var ns *corev1.Namespace
	var gameObj *tdv1alpha1.Game
	var gameList *tdv1alpha1.GameList
	ctx := context.TODO()

	BeforeEach(func(done Done) {
		atomic.AddUint64(&count, 1)

		ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("testns-%v", count)}}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred(), "failed to create namespace")

		// Create instances of the target object.
		gameObj = &tdv1alpha1.Game{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-game",
				Namespace: ns.Name,
				Labels: map[string]string{
					"test-key": "test-val",
				},
			},
		}

		gameList = &tdv1alpha1.GameList{}

		close(done)
	}, 10)

	AfterEach(func(done Done) {
		err := k8sClient.Delete(ctx, ns)
		Expect(err).NotTo(HaveOccurred(), "failed to delete namespace")

		close(done)
	}, 10)

	Describe("Use RBAC Client", func() {

		Context("to record API calls", func() {
			It("should record the RBAC permission", func() {
				err := cli.Create(ctx, gameObj)
				Expect(err).NotTo(HaveOccurred(), "failed to create game obj")

				err = cli.Get(ctx, client.ObjectKeyFromObject(gameObj), gameObj)
				Expect(err).NotTo(HaveOccurred(), "failed to get game obj")

				gameObj.SetAnnotations(map[string]string{"nnn": "jjj"})
				err = cli.Update(ctx, gameObj)
				Expect(err).NotTo(HaveOccurred(), "failed to update game obj")

				err = cli.List(ctx, gameList, &client.ListOptions{Namespace: "test-ns"})
				Expect(err).NotTo(HaveOccurred(), "failed to list game obj")

				err = cli.Status().Update(ctx, gameObj)
				Expect(err).NotTo(HaveOccurred(), "failed to update game obj status")

				defer func() {
					err := cli.Delete(ctx, gameObj)
					Expect(err).NotTo(HaveOccurred(), "failed to delete game obj")
				}()

				Expect(Result(cli, os.Stdout, os.Stdout)).NotTo(HaveOccurred(), "failed to get RBAC result")
			})
		})

	})
})
