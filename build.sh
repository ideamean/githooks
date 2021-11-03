#!/bin/sh

#useage: sh build.sh [online]

p=$(cd `dirname $0`;pwd)

rm -rf $p/output
mkdir -p $p/output/server $p/output/client

cd $p/server
for f in server;do
    for hook in $p/$f/*; do
        cd $hook
        hook_name=${hook##*/}
        GOOS=linux GOARCH=amd64 go build -o $p/output/$f/
        cp *.yaml $p/output/$f/
        is_upx=$(which upx)
        if [ $? -eq 0 ]; then
            upx -1 $p/output/$f/$hook_name
        fi
    done
done

echo "build succeeded!"
