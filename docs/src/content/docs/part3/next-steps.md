---
title: 4. この講義のその先
description: 次に学ぶ道筋
---

ここまでで、コンテナの中身を作って、使い方や危なさを考えられるところまで来ました。コンテナの世界は、まだまだ広がっています。最後に、ここから先に何があるのかを、簡単に紹介します。

## 関連技術を俯瞰する

コンテナの周りの技術を、層に分けて並べると、次のようになります。

```mermaid
flowchart LR
    OCI["OCI の仕様<br/>(runtime / image / distribution)"] --> RT["ランタイム<br/>runc、youki、crun、gVisor、Kata など"]
    RT --> CD["containerd、CRI-O"]
    CD --> K8S["Kubernetes"]
    K8S --> CLOUD["クラウド<br/>EKS、GKE など"]
```

第2章で作ったのは、この図の左の方、ランタイムの部分です。そこから右へ進むと、たくさんのコンテナを管理する仕組みが出てきて、その先にクラウドがあります。

## コンテナランタイム youki

ランタイムの世界には、日本から生まれたプロジェクトもあります。youki は、Rust で書かれた OCI ランタイムで、`runc` と同じように、OCI の仕様に従ってコンテナを起動します。CNCF のプロジェクトの1つにもなっていて、ランタイムの実装を、Rust のコードとして読めます。第2章で作ったものが、実際のランタイムでは、どう書かれているのかを見てみるのも、面白いと思います。

## Kubernetes を触ってみる

![Kubernetes のロゴ](../../../assets/part3/kubernetes-logo.png)

たくさんのコンテナを、複数のマシンにまたがって管理するのが、Kubernetes です。Kubernetes は、コンテナを直接起動するのではなく、CRI (Container Runtime Interface) という標準のインターフェースを通して、`containerd` や CRI-O にお願いします。その先で、OCI ランタイム (`runc` など) が、第2章で作ったようなコンテナを起動しています。つまり、Kubernetes の中でも、最後にコンテナを動かしているのは、第2章と同じ仕組みです。AWS の EKS や Google Cloud の GKE のような、Kubernetes を管理してくれるサービスも、広く使われています。

## コミュニティに参加する

![CloudNative Days](../../../assets/part3/cloudnativedays-logo.png)

![KubeCon + CloudNativeCon](../../../assets/part3/kubecon-japan-2026.png)

コンテナやクラウドネイティブの技術は、コミュニティが中心になって育てています。CNCF (Cloud Native Computing Foundation) は、Kubernetes や containerd などのプロジェクトを預かっていて、Linux Foundation の下にあります。日本でも、CloudNative Days のようなカンファレンスが開かれていて、実際に使っている人や、開発している人の話を聞けます。興味があれば、覗いてみてください。

:::note[来月、CloudNative Days が開催されます！！]
2026年11月18日 (水)・19日 (木) に、有明セントラルタワーホール&カンファレンスで開催されます。現地とオンライン配信のハイブリッド開催で、参加は無料です (参加登録が必要です)。詳しくは、[公式サイト](https://cloudnativedays.jp/) をご覧ください。
:::

## ランタイムの自作をもっと続けたい方へ

コンテナランタイムの自作そのものに、もっと興味が湧いた方には、[コンテナランタイム自作講座](https://coding-container-runtime.logica0419.dev/) があります。ランタイムの自作を扱った講座なので、この講義の続きとして、読んでみてください。

ここまで、お疲れさまでした。この講義が、コンテナの中身を覗いてみるきっかけになっていたら、嬉しいです。
