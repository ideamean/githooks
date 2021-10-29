#!/usr/bin/env bash
# 创建临时目录
TMP_DIR=$(mktemp -d)
# 空hash
EMPTY_REF='0000000000000000000000000000000000000000'
# Colors
PURPLE='\033[35m'
RED='\033[31m'
RED_BOLD='\033[1;31m'
YELLOW='\033[33m'
YELLOW_BOLD='\033[1;33m'
GREEN='\033[32m'
GREEN_BOLD='\033[1;32m'
BLUE='\033[34m'
BLUE_BOLD='\033[1;34m'
COLOR_END='\033[0m'

cwd=$(cd `dirname $0`; pwd)
repos=$(echo $GL_PROJECT_PATH|awk -F'/' '{print $2}')
namespace=$(echo $GL_PROJECT_PATH|awk -F'/' '{print $1}')
ignoreRepos="ysphp,readoo,readwith,xhgui,monitor,ys_editor, PbCard"

while read oldrev newrev ref
do
    echo -e "申请豁免: ${GREEN_BOLD} 紧急情况可申请豁免，请在钉钉[审批]->[代码豁免] 申请豁免码,审批通过后豁免码会发送到钉钉,在commit message中增加: [A]code[/A]即可免检查。 ${COLOR_END}"
    echo -e "版本库名: ${GREEN_BOLD} $repos ${COLOR_END}"
    echo -e "命名空间: ${GREEN_BOLD} $namespace ${COLOR_END}"
    echo -e "修改版本: ${GREEN_BOLD} $oldrev ${COLOR_END}"
    echo -e "最新版本: ${GREEN_BOLD} $newrev ${COLOR_END}"
    echo -e "推送分支: ${GREEN_BOLD} $ref ${COLOR_END}"

    if [[ $oldrev == "$EMPTY_REF" || $newrev == "$EMPTY_REF" ]]; then
        echo -e "未修改的提交: ${GREEN_BOLD} 已跳过检查。 ${COLOR_END}"
        break
    fi

    author_email=$(git cat-file commit $newrev | grep author|awk -F' ' '{print $3}'|sed 's/[<>]//g')
    author_email_count=$(echo $author_email|grep '@youshu\.cc'|wc -l)

    message=$(git cat-file commit $newrev | sed '1,/^$/d')
    messageLine=$(echo $message|tr "\n" " ")
    echo -e "$author_email\t$repos\t$oldrev\t$newrev\t$messageLine\n" >> /tmp/gitlab-hook.log

    if [ $author_email_count -le 0 ]; then
        echo -e "${RED_BOLD}  git账号配置的邮箱不正确($author_email)，请使用公司邮箱。${COLOR_END}"
        echo -e "${GREEN_BOLD} 请在项目根目录使用: git config --list 检查 user.email邮箱, 并使用命令配置: git config user.email 'your email' ${COLOR_END}"
        echo -e "${GREEN_BOLD} 注：已经commit，请使用此命令修改: git commit --amend --author='author <email>' ${COLOR_END}"
        exit 1
    fi

    # 检测用户名是否为邮箱前缀
    #is_valid_username=$(git cat-file commit $newrev|grep author|awk '{ if("<"$2"@youshu.cc>" == $3){print 1}else{print 0}}')
    #if [ "$is_valid_username" == "0" ]; then
    #    echo -e "${RED_BOLD}  git账号配置的用户名不正确, 需使用邮箱前缀: git config user.name 'your name'${COLOR_END}"
    #fi

    #code
    code=$(echo $message|grep -o -E '\[A\]([0-9\]+)\[/A\]'|sed 's/[^0-9]//g')
    if [ -f "/alidata/gitlab-hook-code/${code}" ]; then
        echo -e "豁免码: ${GREEN_BOLD} $code, 已为您跳过检查。 ${COLOR_END}"
        rm -f /alidata/gitlab-hook-code/${code}
        break
    fi

    skipReposCount=$(echo $ignoreRepos|tr "," "\n"|grep $repos|wc -l)
    if [ $skipReposCount -gt 0 ]; then
        echo -e "${GREEN_BOLD} 提示: 该项目配置为($repos)跳过检查。 ${COLOR_END}"
        break
    fi
    #skip some namespace
    skip_count=$(echo $namespace|grep -E 'go|lix'|wc -l)

    if [ $skip_count -gt 0 ]; then
        echo -e "${GREEN_BOLD} 提示: 该命名空间配置为($namespace)跳过检查。 ${COLOR_END}"
        break
    fi


    jira_count=$(echo $message|tr "\n" " "|grep -o -E '[a-zA-Z]+\-[0-9]+|合并分支|Merge'|wc -l)

    echo  -e "提交内容: ${GREEN_BOLD} $message ${COLOR_END}"

    if [ $jira_count -le 0 ]; then
        echo -e "检查失败: ${RED_BOLD} 请在commit message 中携带jira号, 格式: PA-{yourid} ${COLOR_END}"
        exit 1
    fi

    echo -e  "${GREEN_BOLD} 基本检查通过。${COLOR_END}"

    skip_count=$(echo $message|tr "\n" " "|grep -o -E '合并分支|Merge'|wc -l)
    if [ $skip_count -gt 0 ]; then
        break
    fi

    if [ "$ref" == "refs/heads/master"  ]; then
        echo -e "${RED_BOLD} 不允许直接push master分支, 请通过合并请求提交。 ${COLOR_END}"
        exit 1
    fi
#while read oldrev newrev ref
#do
    # 当push新分支的时候oldrev会不存在，删除时newrev就不存在
    if [[ $oldrev != $EMPTY_REF && $newrev != $EMPTY_REF ]]; then
        echo -e "\n${GREEN_BOLD}执行代码风格检查:${COLOR_END}"
        echo -e "ref_name: $ref"
        pwd
        echo
        # 找出哪些文件被更新了
        #for line in $(git diff-tree -r $oldrev..$newrev | grep -oP '.*\.(js|php)' | awk '{print $5$6}')
        #git diff-tree -r $oldrev..$newrev
        for line in $(git diff-tree -r $oldrev..$newrev | grep -oP '.*\.(php)' | awk '{print $5$6}')
        do
            # 文件状态
            # D: deleted
            # A: added
            # M: modified
            status=$(echo $line | grep -o '^.')

            # 不检查被删除的文件
            if [[ $status == 'D' ]]; then
                continue
            fi

            # 文件名
            file=$(echo $line | sed 's/^.//')

            # 为文件创建目录
            mkdir -p $(dirname $TMP_DIR/$file)
            # 保存文件内容
            git show $newrev:$file > $TMP_DIR/$file

            if [[ $(echo $file | grep -e '.go') ]]; then
               /usr/bin/golangci-lint -j 1000 -D errcheck -E stylecheck -E gofmt -E misspell -E whitespace -E maligned run $TMP_DIR/$file
               if [ $? != 0 ]; then
                  exit 1
               fi
               continue
            fi

            if [[ $(echo $file | grep -e '.php') ]]; then
                standard='PSR2'
            fi

            if [[ $(echo $file | grep -e '.js') ]]; then
                standard='ClosureLinter'
            fi

            #log check
             if grep -n 'Ys_Log_App::' $TMP_DIR/$file|grep -v 'getInstance'; then
                 echo -e "${RED_BOLD}日志类调用错误, 正确调用: \Ys_Log_App::getInstance()->write(SEASLOG_DEBUG, 'message');${COLOR_END}"
                 exit 1
             fi

            #output=$(phpcs --report=summary --standard=$standard $TMP_DIR/$file)
            #output=$(/usr/local/bin/phpcs --standard=$standard $TMP_DIR/$file)
            /usr/local/bin/phpcs -n --exclude=PSR1.Methods.CamelCapsMethodName --report-width=240 --standard=$standard --colors --encoding=utf-8 -p $TMP_DIR/$file

            if [ $? != 0 ]; then
                echo -e "${RED_BOLD} 代码风格检查失败, 请修复。${COLOR_END}"
                echo -e "${GREEN_BOLD} 代码格式化工具参考: http://devbook.youshu.cc/docs/phpcs.html ${COLOR_END}"
                exit 1
            fi
        done
    fi
done

rm -rf $TMP_DIR
exit 0