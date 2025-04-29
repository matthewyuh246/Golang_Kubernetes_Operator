package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GVR の定義
var gvr = schema.GroupVersionResource{
	Group:    "example.com", // Group: CRD の spec.group（例：example.com）
	Version:  "v1alpha1",    // Version: CRD のバージョン（例：v1alpha1）
	Resource: "foos",        // Resource: plural 形（例：foos）
}

// CR の Go 構造体定義
type Foo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	TestString string `json:"testString"`
	TestNum    int    `json:"testNum"`
}

type FooList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Foo `json:"items"`
}

// listFoos 関数：Dynamic Client での取得〜デシリアライズ
func listFoos(client dynamic.Interface, namespace string) (*FooList, error) {
	// 戻り値は *unstructured.UnstructuredList 型
	list, err := client.Resource(gvr).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// 取得した unstructured オブジェクト群を JSON バイト列に変換
	data, err := list.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// json.Unmarshal → FooList
	var fooList FooList
	if err := json.Unmarshal(data, &fooList); err != nil {
		return nil, err
	}
	return &fooList, nil
}

func createPod(clientset *kubernetes.Clientset, namespace, name string) error {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "busybox",
					Image: "registry.k8s.io/echoserver:1.10",
					Ports: []v1.ContainerPort{{ContainerPort: 8080}},
				},
			},
			RestartPolicy: v1.RestartPolicyAlways,
		},
	}
	_, err := clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("failed to create Pod %v\n", err)
		return err
	}
	fmt.Printf("Successfully created a Pod (%s)", name)
	return nil
}

func main() {
	var defaultKubeConfigPath string

	// ホームディレクトリ検出
	if home := homedir.HomeDir(); home != "" {
		defaultKubeConfigPath = filepath.Join(home, ".kube", "config")
	}

	kubeconfig := flag.String("kubeconfig", defaultKubeConfigPath, "kubeconfig config file")
	flag.Parse()

	// kubeconfig を読み込み、API サーバー接続情報 (rest.Config) を得る
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalf("failed to build kubeconfig: %v", err)
	}

	// Typed client の代わりに “動的” にあらゆる GVR を操作できるクライアントを生成
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create dynamic client: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create clientset: %v", err)
	}

	foos, err := listFoos(client, "")
	if err != nil {
		log.Fatalf("failed to list Foos: %v", err)
	}
	for {
		fmt.Println("INDEX\tNAMESPACE\tNAME")
		for i, foo := range foos.Items {
			namespace := foo.GetNamespace()
			name := foo.GetName()
			fmt.Printf("%d\t%s\t%s\n", i, namespace, name)

			// podが存在するか確認
			_, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					fmt.Println("Pod doesn't exist. Creating new Pod")
					createPod(clientset, namespace, name)
				} else {
					fmt.Printf("failed to get pod %v\n", err)
				}
			} else {
				fmt.Printf("successfully got pod %s\n", name)
			}
		}
		time.Sleep(1 * time.Second)
	}
}
