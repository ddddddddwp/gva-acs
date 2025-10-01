#!/bin/bash

# 批量修复导入路径
find . -name "*.go" -type f -exec sed -i 's|"root/demo/tr069/|"github.com/root/demo/tr069/|g' {} \;

echo "导入路径修复完成"