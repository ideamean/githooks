#!/bin/bash
sh build.sh
contianer="ec2fb938b478"
docker cp output/server/pre-receive $contianer:/opt/gitlab/embedded/service/gitlab-shell/hooks/pre-receive.d/
docker cp output/server/pre-receive.yaml $contianer:/opt/gitlab/embedded/service/gitlab-shell/hooks/pre-receive.d/
docker exec -it $contianer chown -R git:git /opt/gitlab/embedded/service/gitlab-shell/hooks/pre-receive.d/