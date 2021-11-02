# githooks

git 勾子golang版本实现, 勾子介绍官方文档：https://git-scm.com/docs/githooks

## gitlab docker 部署

```bash
docker run --detach \
    --publish 443:443 --publish 80:80 --publish 22:22 \
    --name gitlab \
    --restart always \
    gitlab/gitlab-ce:latest
```

默认账号： root

默认密码： 查看文件 /etc/gitlab/initial_root_password


## 服务端hooks部署路径

官方文档：https://docs.gitlab.com/ee/administration/server_hooks.html

全局勾子的设置路径说明（以pre-receive为例）：

> The default directory:
> 
>    * For an installation from source is usually /home/git/gitlab-shell/hooks. 
>
>    * For Omnibus GitLab installs is usually /opt/gitlab/embedded/service/gitlab-shell/hooks

docker 部署(镜像：gitlab/gitlab-ce), 路径为：

> /opt/gitlab/embedded/service/gitlab-shell/hooks/

 注：最后一级hooks目录不存在，需要手动创建，并且在hooks目录下创建目录pre-receive.d, 最终hooks的路径为:
 /opt/gitlab/embedded/service/gitlab-shell/hooks/pre-receive.d/pre-receive

## 勾子介绍

### pre-receive

1. 参数:

该输入来自标准输入(stdin), 每行按以下格式：
 ```
 <old-value> SP <new-value> SP <ref-name> LF
 ```

2. 检测逻辑(按执行顺序)

|  规则             |  执行逻辑    |
|  ----            | ----        | 
| 读取stdin        | 按行读取stdin, 解析old-value, new-value, ref-name |
| 解析环境变量      | 分隔环境变量:GL_PROJECT_PATH, 解析出当前推送的版本库和命名空间 |
| 创建临时目录      | 用于存储本次commit改变的文件, 按文件类型存储到不同的临时目录 |
| 加载配置文件      | 读取/etc/pre-receive.yaml, 不存在则读取当前目录的pre-receive.yaml |
| 打印header头     | 输出相关的调试信息: <br />code exemption: 豁免码提交规则 <br />repository: 版本库名称<br />namespace: 命名空间 <br />old_ref: 提交前的hash id <br /> new_ref: 提交后的hash id <br />branch: 分支信息(refs/heads/$branch_name) |
| 加载版本库        | 使用 github.com/go-git/go-git/v5 库读取当前版本库信息 |
| 检查email账号格式 | 参考pre-receive.yaml中的AllowEmail配置项, 配置支持的email后缀 |
| 判断是否超级账号   | 参考pre-receive.yaml SuperAccount配置项, 超级账号跳过检查 |
| 检查命名空间      | 参考pre-receive.yaml IgnoreNamespace配置项, 配置则跳过所有此命名空间下的库检查 |
| 检查版本库        | 参考pre-receive.yaml IgnoreRepos配置项, 存在则跳过此版本库的检查 |
| 检查豁免码        | 参考pre-receive.yaml CodeExemptionDir配置项, 配置豁免码的存储路径, commit message中携带了豁免码, 则判断 CodeExemptionDir/$code 文件是否存在, 存在则跳过检查, 豁免码在commit message中输入:[A]$code[/A]。豁免码的申请流程，暂未支持，需要自行将豁免码文件推送到指定目录下。 |
| 检查提交方式      | 跳过merge request 提交，否则在代码分支合并时commit message 不符合提交规范 |
| 检查Jira ID      | 检查commit message是否包涵jira ID, 格式参考：pre-receive.yaml RequireJiraIDRexp配置项, 留空则不检查 |
| 读取变更文件      | 将变更文件，根据变更文件类型，存储到不同的目录(取文件的扩展名), 存储到 $temp_dir/$file_extension/$file_path, 扩展名为空, 则认为不是常规则文件跳过检查 |
| 检验代码规范      | 当前只支持PHP代码风格检查, 需安装PHPCS, 参考pre-receive.yaml StyleCheck.PHP 配置项，配置phpcs路径以及相关参数 |
| 删除临时文件      | 删除临时文件目录 |


## License

See [LICENSE](./LICENSE).