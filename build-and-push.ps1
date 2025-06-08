#!/usr/bin/env pwsh

# GoMailAPI2 Docker 镜像构建和推送脚本
# 作者: mufeng888
# 描述: 自动构建适用于 Ubuntu 的 Docker 镜像并推送到 DockerHub

param(
    [string]$Tag = "latest",
    [string]$Platform = "linux/amd64",
    [string]$ImageName = "mufeng888/gomailapi2-server"
)

Write-Host "========================================" -ForegroundColor Green
Write-Host "Docker 镜像构建和推送脚本" -ForegroundColor Green
Write-Host "镜像名称: $ImageName" -ForegroundColor Yellow
Write-Host "标签: $Tag" -ForegroundColor Yellow
Write-Host "平台: $Platform" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Green

# 检查 Docker 是否运行
Write-Host "检查 Docker 状态..." -ForegroundColor Blue
try {
    docker version | Out-Null
    Write-Host "✓ Docker 正在运行" -ForegroundColor Green
} catch {
    Write-Host "✗ Docker 未运行，请启动 Docker Desktop" -ForegroundColor Red
    exit 1
}

# 构建镜像
Write-Host "`n开始构建 Docker 镜像..." -ForegroundColor Blue
Write-Host "执行命令: docker build --platform $Platform -t ${ImageName}:$Tag ." -ForegroundColor Gray

$buildResult = docker build --platform $Platform -t "${ImageName}:$Tag" .
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ 镜像构建失败" -ForegroundColor Red
    exit 1
}
Write-Host "✓ 镜像构建成功" -ForegroundColor Green

# 检查登录状态
Write-Host "`n检查 DockerHub 登录状态..." -ForegroundColor Blue
$loginCheck = docker info 2>&1 | Select-String "Username"
if ($loginCheck) {
    Write-Host "✓ 已登录到 DockerHub" -ForegroundColor Green
} else {
    Write-Host "需要登录到 DockerHub..." -ForegroundColor Yellow
    docker login
    if ($LASTEXITCODE -ne 0) {
        Write-Host "✗ DockerHub 登录失败" -ForegroundColor Red
        exit 1
    }
}

# 推送镜像
Write-Host "`n开始推送镜像到 DockerHub..." -ForegroundColor Blue
Write-Host "执行命令: docker push ${ImageName}:$Tag" -ForegroundColor Gray

$pushResult = docker push "${ImageName}:$Tag"
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ 镜像推送失败" -ForegroundColor Red
    exit 1
}

Write-Host "✓ 镜像推送成功" -ForegroundColor Green

# 显示镜像信息
Write-Host "`n镜像信息:" -ForegroundColor Blue
docker images "${ImageName}:$Tag" --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}\t{{.CreatedAt}}"

Write-Host "`n========================================" -ForegroundColor Green
Write-Host "✓ 构建和推送完成！" -ForegroundColor Green
Write-Host "镜像地址: ${ImageName}:$Tag" -ForegroundColor Yellow
Write-Host "可以使用以下命令拉取镜像:" -ForegroundColor Blue
Write-Host "docker pull ${ImageName}:$Tag" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Green 