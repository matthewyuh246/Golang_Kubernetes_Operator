package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var defaultKubeConfigPath string
	if home := homedir.HomeDir(); home != "" { // ユーザーのホームディレクトリを取得
		defaultKubeConfigPath = filepath.Join(home, ".kube", "config") // それが空でなければ、~/.kube/config をデフォルトの kubeconfig パスとして設定
	}

	fmt.Println("デフォルト kubeconfig パスの設定", defaultKubeConfigPath)
	fmt.Println()

	kubeconfig := flag.String("kubeconfig", defaultKubeConfigPath, "kubeconfig file")
	flag.Parse() // 実際にコマンドライン引数を読み取り、kubeconfig ポインタへ設定

	fmt.Println("kubeconfig: ", kubeconfig)
	fmt.Println()

	// Kubernetes クライアント設定の構築
	// BuildConfigFromFlags(masterUrl, kubeconfigPath)
	// masterUrl を空文字にすると、kubeconfig の current-context を利用
	// 認証情報・API サーバーのアドレスを含む *rest.Config を生成
	config, _ := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	fmt.Println(config)
	fmt.Println()

	// Clientset（API クライアント）の生成
	// kubernetes.NewForConfig に rest.Config を渡すと、
	// CoreV1, AppsV1, BatchV1 など各種サブクライアントを持つ Clientset が返る
	clientset, _ := kubernetes.NewForConfig(config)
	fmt.Println(clientset)

	// Pods("") 空文字で全 Namespace を対象にする。特定 namespace のみなら "default" などを指定。
	// context.Background() タイムアウトやキャンセルを行わないベースコンテキスト。実運用ではタイムアウト付きコンテキストを使うと安全。
	// metav1.ListOptions{} ラベルセレクターやフィールドセレクターなどで絞り込み可能。空なら全件取得
	// 返り値 pods は *v1.PodList 型
	pods, _ := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})

	fmt.Println("INDEX\tNAMESPACE\tNAME")
	for i, pod := range pods.Items {
		fmt.Printf("%d\t%s\t%s\n", i, pod.GetNamespace(), pod.GetName())
	}
}
