这个项目中有两个部分： Trigger 和 Deployer，Trigger 的作用是解析 github 事件， 并提交 PipelineRun 定义。Deployer 的作用就是更新 Service 的镜像信息。
