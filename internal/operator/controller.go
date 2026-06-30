package operator

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kriipke/chartpress/internal/apis"
	"github.com/kriipke/chartpress/internal/objectstore"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller wires a dynamic SharedInformer to a rate-limiting workqueue and a
// Reconciler. No controller-runtime, no leader election, one replica; the 30s
// resync re-delivers cached objects as the level-based safety net.
type Controller struct {
	informer cache.SharedIndexInformer
	queue    workqueue.TypedRateLimitingInterface[string]
	rec      *Reconciler
}

func NewController(client dynamic.Interface, rec *Reconciler, namespace string, resync time.Duration) (*Controller, error) {
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, resync, namespace, nil)
	informer := factory.ForResource(apis.GVR).Informer()
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
	c := &Controller{informer: informer, queue: queue, rec: rec}

	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.enqueue,
		UpdateFunc: func(_, newObj interface{}) { c.enqueue(newObj) },
		DeleteFunc: c.enqueue,
	}); err != nil {
		return nil, fmt.Errorf("add event handler: %w", err)
	}
	return c, nil
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("[ERROR] key for object: %v", err)
		return
	}
	c.queue.Add(key)
}

// Run starts the informer and worker loops, blocking until ctx is cancelled.
func (c *Controller) Run(ctx context.Context, workers int) error {
	defer c.queue.ShutDown()
	go c.informer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		return fmt.Errorf("informer cache sync failed")
	}
	for i := 0; i < workers; i++ {
		go wait.Until(func() {
			for c.processNext(ctx) {
			}
		}, time.Second, ctx.Done())
	}
	<-ctx.Done()
	return nil
}

func (c *Controller) processNext(ctx context.Context) bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil || !exists {
		c.queue.Forget(key)
		return true
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		c.queue.Forget(key)
		return true
	}
	if err := c.rec.Reconcile(ctx, u.DeepCopy()); err != nil {
		log.Printf("[ERROR] reconcile %s: %v", key, err)
		c.queue.AddRateLimited(key)
		return true
	}
	c.queue.Forget(key)
	return true
}

// Start wires the in-cluster operator and runs it (production entrypoint).
func Start() {
	log.Println("[INFO] starting chartpress operator")
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("[FATAL] in-cluster config: %v", err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("[FATAL] dynamic client: %v", err)
	}
	store, err := objectstore.New(objectstore.ConfigFromEnv())
	if err != nil {
		log.Fatalf("[FATAL] object store: %v", err)
	}
	ns := namespaceFromEnv()
	rec := &Reconciler{
		Client:    newDynamicCRClient(dyn),
		Renderer:  &chartRenderer{templatesDir: templatesDir()},
		Uploader:  store,
		Namespace: ns,
	}
	ctrl, err := NewController(dyn, rec, ns, 30*time.Second)
	if err != nil {
		log.Fatalf("[FATAL] controller: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Printf("[INFO] operator watching namespace %q", ns)
	if err := ctrl.Run(ctx, 1); err != nil {
		log.Fatalf("[FATAL] run: %v", err)
	}
}

func namespaceFromEnv() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return "default"
}

func templatesDir() string {
	if d := os.Getenv("CHARTPRESS_TEMPLATES_DIR"); d != "" {
		return d
	}
	return "templates"
}
