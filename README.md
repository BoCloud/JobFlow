# JobFlow

## 背景

### volcano

Volcano是CNCF 下首个也是唯一的基于Kubernetes的容器批量计算平台，主要用于高性能计算场景。
它提供了Kubernetes目前缺 少的一套机制，这些机制通常是机器学习大数据应用、科学计算、
特效渲染等多种高性能工作负载所需的。

现状：当前volcano开源社区对于complete依赖的作业的解决方案是引入argo workflow。

1. 但目前volcano中缺乏对vcjob的编排，缺乏vcjob之间的作业依赖运行实现。当出现complete依赖的task时则与gang调度算法相矛盾。
   虽然argo workflow也能很好的支持vcjob的编排，argo workflow的引入又太过庞大。

2. argo workflow更适用于广泛的场景，并没有针对某个场景的特定情况有细分的需求，例如使用argo workflow创建vcjob的任务依赖无法直接查看个vcjob的具体情况。
   JobFlow根据需求场景适配vcjob。更贴合vcjob的使用。

3. workflow目前只针对vcjob completed状态的场景有支持。
   jobflow 可以针对vcjob增加对task running状态以及探针方式的依赖等等，依赖方式多样化。

4. workflow目前可观察到下发的vcjob级别的进度。但当vcjob支持task级别的任务依赖后，workflow无法查看task级别的进度。
   JobFlow则可提供该场景。

## JobFlow

jobflow旨在实现volcano中vcjob之间的作业依赖运行。根据vcjob之间的依赖关系对vcjob进行下发。

### 字段定义

```
apiVersion: batch.volcano.sh/v1alpha1
kind: JobTemplate
metadata:
  name: A
spec:
  minAvailable: 3
  schedulerName: volcano
  priorityClassName: high-priority
  policies:
    - event: PodEvicted
      action: RestartJob
  plugins:
    ssh: []
    env: []
    svc: []
  maxRetry: 5
  queue: default
  volumes:
    - mountPath: "/myinput"
    - mountPath: "/myoutput"
      volumeClaimName: "testvolumeclaimname"
      volumeClaim:
        accessModes: [ "ReadWriteOnce" ]
        storageClassName: "my-storage-class"
        resources:
          requests:
            storage: 1Gi
  tasks:
    - replicas: 6
      name: "default-nginx"
      template:
        metadata:
          name: web
        spec:
          containers:
            - image: nginx
              imagePullPolicy: IfNotPresent
              name: nginx
              resources:
                requests:
                  cpu: "1"
          restartPolicy: OnFailure
---
apiVersion: batch.volcano.sh/v1alpha1
kind: JobTemplate
metadata:
  name: B
spec:
  minAvailable: 3
  schedulerName: volcano
  priorityClassName: high-priority
  policies:
    - event: PodEvicted
      action: RestartJob
  plugins:
    ssh: []
    env: []
    svc: []
  maxRetry: 5
  queue: default
  volumes:
    - mountPath: "/myinput"
    - mountPath: "/myoutput"
      volumeClaimName: "testvolumeclaimname"
      volumeClaim:
        accessModes: [ "ReadWriteOnce" ]
        storageClassName: "my-storage-class"
        resources:
          requests:
            storage: 1Gi
  tasks:
    - replicas: 6
      name: "default-nginx"
      template:
        metadata:
          name: web
        spec:
          containers:
            - image: nginx
              imagePullPolicy: IfNotPresent
              name: nginx
              resources:
                requests:
                  cpu: "1"
          restartPolicy: OnFailure
---
apiVersion: batch.volcano.sh/v1alpha1
kind: JobTemplate
metadata:
  name: C
spec:
  minAvailable: 3
  schedulerName: volcano
  priorityClassName: high-priority
  policies:
    - event: PodEvicted
      action: RestartJob
  plugins:
    ssh: []
    env: []
    svc: []
  maxRetry: 5
  queue: default
  volumes:
    - mountPath: "/myinput"
    - mountPath: "/myoutput"
      volumeClaimName: "testvolumeclaimname"
      volumeClaim:
        accessModes: [ "ReadWriteOnce" ]
        storageClassName: "my-storage-class"
        resources:
          requests:
            storage: 1Gi
  tasks:
    - replicas: 6
      name: "default-nginx"
      template:
        metadata:
          name: web
        spec:
          containers:
            - image: nginx
              imagePullPolicy: IfNotPresent
              name: nginx
              resources:
                requests:
                  cpu: "1"
          restartPolicy: OnFailure
---
apiVersion: batch.volcano.sh/v1alpha1
kind: JobTemplate
metadata:
  name: D
spec:
  minAvailable: 3
  schedulerName: volcano
  priorityClassName: high-priority
  policies:
    - event: PodEvicted
      action: RestartJob
  plugins:
    ssh: []
    env: []
    svc: []
  maxRetry: 5
  queue: default
  volumes:
    - mountPath: "/myinput"
    - mountPath: "/myoutput"
      volumeClaimName: "testvolumeclaimname"
      volumeClaim:
        accessModes: [ "ReadWriteOnce" ]
        storageClassName: "my-storage-class"
        resources:
          requests:
            storage: 1Gi
  tasks:
    - replicas: 6
      name: "default-nginx"
      template:
        metadata:
          name: web
        spec:
          containers:
            - image: nginx
              imagePullPolicy: IfNotPresent
              name: nginx
              resources:
                requests:
                  cpu: "1"
          restartPolicy: OnFailure
---
apiVersion: batch.volcano.sh/v1alpha1
kind: JobTemplate
metadata:
  name: E
spec:
  minAvailable: 3
  schedulerName: volcano
  priorityClassName: high-priority
  policies:
    - event: PodEvicted
      action: RestartJob
  plugins:
    ssh: []
    env: []
    svc: []
  maxRetry: 5
  queue: default
  volumes:
    - mountPath: "/myinput"
    - mountPath: "/myoutput"
      volumeClaimName: "testvolumeclaimname"
      volumeClaim:
        accessModes: [ "ReadWriteOnce" ]
        storageClassName: "my-storage-class"
        resources:
          requests:
            storage: 1Gi
  tasks:
    - replicas: 6
      name: "default-nginx"
      template:
        metadata:
          name: web
        spec:
          containers:
            - image: nginx
              imagePullPolicy: IfNotPresent
              name: nginx
              resources:
                requests:
                  cpu: "1"
          restartPolicy: OnFailure
```

### JobTemplate

希望可以生成vcctl直接转换template和vcjob

jobtemplate和vcjob可以互相转换，区别是jobtemplate不会被job controller下发，jobflow可以直接引用该JobTemplate名称，来实现vcjob的下发。

```
apiVersion: batch.volcano.sh/v1alpha1
kind: JobFlow
metadata:
  name: test
  namespace: default
  creationTimestamp: "2021-10-18T07:01:24Z"
spec:
  JobRetainPolicy: retain/delete   jobflow运行结束后，保留产生的job. 否则删除。
  flows:
    - name: A
    - name: B
      dependsOn:
        targets: [‘A’]
    - name: C
      dependsOn:
        targets: [‘B’]
    - name: D
      dependsOn:
        targets: [‘B’]
    - name: E
      dependsOn:
        targets: [‘C’,‘D’]
status:
  jobStatusList: []
  pendingJobs: []
  runningJobs: []
  failedJobs: []
  completedJobs: []
  terminatedJobs: []
  unKnowJobs: []
  conditions: 
  - A: {
    phase: completed
    message: ''
    createTime: "2021-08-25T02:04:20Z"
    RunningDuration: "22s"
    taskStatusCount: 
    - nginx:  
      - phase:
        - Running: 1
    }
  - B: {
      phsae: running
      message: ''
      createTime: "2021-08-25T02:04:20Z"
      RunningDuration: "",
     }
  - C: {
      phsae: waiting
      message: ''
      createTime: ""
      RunningDuration: ""
     }
  - D: {
      phsae: waiting
      message: ''
      createTime: ""
      RunningDuration: ""
      }
  - E: {
      phsae: waiting
      message: ''
      createTime: ""
      RunningDuration: ""
      }
  state:
    phase: successed/terminating/terminated/failed/pending

```

jobStatusList 

```
type JobStatus struct {
    Name string
    State string // running/failed
    StartTimestamp Time
    EndTimestamp Time
    RestartCount int
    RunningHistories []JobRunningHistory
}

type JobRunningHistory struct {
    StartTimestamp Time
    EndTimestamp time
    State string // failded/succeeded ....
}

```

根据JobFlow创建的名称遵循JobFlowName-JobTemplateName,

### JobFlow解释

- pendingJobs 处于pending状态的vcjob
- runningJobs 处于running状态的vcjob
- failedJobs 处于failedJobs状态的vcjob
- completed 处于completed状态的vcjob
- terminated 处于terminated状态的vcjob
- metadata: 描述JobFlow的元数据信息
- flow：定义了vcjob之间的依赖关系，没有依赖项的vcjob即位入口。支持多入口和多出口。当前只支持了complete依赖
- depends.target: 指定了依赖的vcjob
- jobStatusList 拆分出来的所有vcjob的状态信息
- successfulJobList 已经成功complete的vcjob
- conditions 用于描述所有vcjob当前的状态，创建时间，完成时间以及信息，该处的vcjob状态额外增加waiting状态用于描述依赖项没有达到要求的vcjob。
- phase JobFlow的状态
```
Succeed： 所有的vcjob都已达到completed状态。
Terminating：jobflow正在删除。
Failed： flow中某个vcjob处于failed状态导致flow中的vcjob无法继续下发。
Running： flow中包含处于Running状态vcjob。
Pending: flow中包不含处于Running状态vcjob。
```


### JobFlow webhook 校验

对JobFlow的crd字段进行校验和默认值设置。

### JobFlow controller

1. 根据namespace加载JobFlow 资源。
2. 根据namespace加载jobflow。
3. 下发入口的vcjob。
4. 根据vcjob依赖下发vcjob。遍历判断非入口的vcjob的依赖是否满足条件，若满足则进行下发。
5. 将当前的状态信息写入jobflow的status。
6. 监听jobflow下发的vcjob。

## 开发
通过make generateDeployYaml 生成部署文件到deploy目录下

需要更改yaml文件时需要更改kustomize相关配置模板和patch文件，最后通过make generateDeployYaml生成对应yaml文件。不建议直接修改deploy下的yaml文件

## 部署
```
make build   #生成二进制执行文件./bin/manager

make docker-build     #构建镜像

kubectl apply -f ./deploy/     #部署

```
