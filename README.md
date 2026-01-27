# Distributor

HTTP 请求分发器，根据 Host 头将请求反向代理到对应的 worker 容器，并通过 Kubernetes API 动态管理 worker 部署。

## 功能

- 反向代理：根据 Host 头路由请求到对应 worker
- Worker 注册 API：接收 CI/CD 的注册请求，自动部署到 k8s
- 动态部署：通过 Kubernetes API 创建/更新/删除 Deployment 和 Service

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/register | 注册 worker |
| DELETE | /api/register/:domain | 删除 worker |
| GET | /api/workers | 获取所有 worker |

### 注册请求示例

```bash
curl -X POST https://distributor.app238.com/api/register \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "nginx.worker.app238.com",
    "worker_id": "nginx_01",
    "owner_id": "distributor_admin",
    "image": "nginx:latest",
    "port": 80
  }'

curl -X POST https://distributor.app238.com/api/register -H "Content-Type: application/json" -d "{\"domain\": \"nginx.worker.app238.com\",\"worker_id\": \"nginx_01\",\"owner_id\": \"distributor_admin\",\"image\": \"nginx:latest\",\"port\": 80}"
```

## 部署

### 1. 安装 cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
```

### 2. 下载部署文件

```bash
curl -o distributor-k3s-deployment.yaml https://raw.githubusercontent.com/jabberwocky238/distributor/main/scripts/k3s-deployment.yaml
```

### 3. 部署

```bash
export CLOUDFLARE_API_TOKEN=xxxx
export DOMAIN=example.com
envsubst < distributor-k3s-deployment.yaml > distributor-k3s-deployment-final.yaml
envsubst < distributor-k3s-deployment.yaml | kubectl apply -f -
kubectl apply -f distributor-k3s-deployment-final.yaml
kubectl delete -f distributor-k3s-deployment-final.yaml

kubectl get pods -n distributor
kubectl describe pod distributor-7ffcbc985-2bncj -n distributor
```

## Cloudflare API Token

创建 Token 时需要以下权限：

- Zone:DNS:Edit
- Zone:Zone:Read

## 本地开发

```bash
# 运行
make run

# 构建
make build
```
