#!/bin/bash

# 切换到 build.sh 脚本所在的目录
cd $(dirname $0) || exit 1
projectpath=`pwd`

# 设置阿里云代理
export GOPROXY="https://mirrors.aliyun.com/goproxy/"

# 编译信息
moduleName=$(go list -m)
commit=$(git rev-parse --short HEAD)
branch=$(git rev-parse --abbrev-ref HEAD)
buildTime=$(date +%Y%m%d%H%M)

# environ.go 包名
environPkgName="environ"

flags="-X '${moduleName}/${environPkgName}.BuildGitBranch=${branch}' -X '${moduleName}/${environPkgName}.BuildGitCommit=${commit}'  -X '${moduleName}/${environPkgName}.BuildDateTime=${buildTime}' "

program=$(basename ${moduleName})

cd ${projectpath}

go build -v -ldflags "$flags" -o ${projectpath}/bin/$program  || exit 1

exit 0
