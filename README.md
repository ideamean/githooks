# githooks

git 勾子golang版本实现, 勾子介绍官方文档：https://git-scm.com/docs/githooks

## gitlab docker 部署

```bash
docker run \
    --publish 443:443 --publish 80:80 --publish 22:22 \
    --name gitlab \
    gitlab/gitlab-ce
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

|  勾子名称 |  类型     | 参数         |
|  ----   | ----      | ----        |
| pre-receive  | server端 |<old-value> SP <new-value> SP <ref-name> LF <br /> 注：该输入来自标准输入(stdin) 参考： |


## License

See [LICENSE](./LICENSE).