#!/bin/bash

# Flutter 聊天室服務器測試執行腳本
# 使用方式：./run_tests.sh 或 bash run_tests.sh

echo "🚀 開始執行 Flutter 聊天室服務器完整測試套件"
echo "============================================================"

# 執行所有測試檔案
go test -v main_test.go main.go api.go auth.go config.go models.go routes.go websocket.go

echo "============================================================"
echo "測試執行完成！"
