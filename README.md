# operator
k8s中自定义资源(CRD)+自定义Controller
## client-go
是 Kubernetes 提供的一个官方 Go 语言客户端库，用于与 Kubernetes API 进行交互。开发者可以使用 client-go 来编写与 Kubernetes 集群交互的 Go 语言程序，比如创建、更新、删除、查询 Kubernetes 资源，管理集群中的对象等。
## Kubebuilder
Kubebuilder 是一个基于 client-go 的开发框架，用来简化 Kubernetes 控制器和 Operator 的开发。它通过提供工具和脚手架，使开发者可以更快地创建自定义资源（CRD）及其控制器，而不必手动处理所有的 client-go 细节。
## 核心机制
* Informer: Kubebuilder 使用 client-go 的 Informer 机制来监听 Kubernetes 资源事件，但 Kubebuilder 将它封装成更易用的 API。
* Client: Kubebuilder 通过 controller-runtime 提供了简化的客户端，用于创建、更新、删除、获取 Kubernetes 资源。这个客户端是对 client-go 的进一步封装，简化了 API 请求的流程。
* Reconciler：控制器的核心是 Reconcile 循环，它根据资源的当前状态与期望状态之间的差异，执行相应的操作。Kubebuilder 使用 client-go 进行这些资源状态的变更和同步操作。

## 开发流程
* 借助CustomResourceDefinition自定义资源
* 创建空项目
```shell
go mode init myopertor
```
* 初始化kubebuilder
```shell
kubebuilder init --domain xxx.com
```
* 创建api
```shell
kubebuilder create api --group myapp --version v1 --kind MyApp
```
## 发布
```shell
# 自定义Controller是以deploy的形式在k8s中运行的,因此开发完成后打包成docker镜像推送至远程仓库后即可使用
docker build -t xxxx.xxx-xxx/myapp-controller:xxx .
docker push xxxx.xxx-xxx/myapp-controller:xxx
```