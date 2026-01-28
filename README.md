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
    "worker_id": "nginx01",
    "owner_id": "distributoradmin",
    "image": "nginx:latest",
    "port": 80
  }'

curl -X POST https://distributor.app238.com/api/register -H "Content-Type: application/json" -d "{\"worker_id\": \"nginx01\",\"owner_id\": \"distributoradmin\",\"image\": \"nginx:latest\",\"port\": 80}"
curl -X POST https://distributor.app238.com/api/register -H "Content-Type: application/json" -d "{\"worker_id\": \"nginx02\",\"owner_id\": \"distributoradmin\",\"image\": \"nginx:latest\",\"port\": 80}"
curl "nginx01.distributoradmin.worker.app238.com"
```

## 部署

### 1. 安装 cert-manager

```bash
kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
```

### 2. 下载部署文件

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml 
curl -o distributor-k3s-deployment.yaml https://raw.githubusercontent.com/jabberwocky238/distributor/main/scripts/k3s-deployment.yaml
curl -O https://raw.githubusercontent.com/jabberwocky238/distributor/main/scripts/zerossl-issuer.yaml
```

### 3. 部署

```bash
crictl rmi ghcr.io/jabberwocky238/distributor:latest
export ZEROSSL_EAB_KID=your_eab_kid
export ZEROSSL_EAB_HMAC_KEY=your_eab_hmac_key
export CLOUDFLARE_API_TOKEN=xxxx
export DOMAIN=app238.com
envsubst < distributor-k3s-deployment.yaml > distributor-k3s-deployment-final.yaml
envsubst < distributor-k3s-deployment.yaml | kubectl apply -f -
envsubst < zerossl-issuer.yaml > zerossl-issuer-final.yaml
envsubst < zerossl-issuer.yaml | kubectl apply -f -

kubectl apply -f distributor-k3s-deployment-final.yaml
kubectl delete -f distributor-k3s-deployment-final.yaml
kubectl apply -f zerossl-issuer-final.yaml
kubectl delete -f zerossl-issuer-final.yaml

kubectl get ingressroute -A

kubectl create secret tls distributor-tls --cert=cert.pem --key=key.pem -n distributor
kubectl delete secret distributor-tls -n distributor
kubectl delete certificate distributor-cert -n distributor
envsubst < zerossl-issuer.yaml | kubectl apply -f 
kubectl describe certificate distributor-cert -n distributor
kubectl get certificaterequest -n distributor
kubectl get order -n distributor

kubectl delete namespace worker
kubectl get pods -n distributor
kubectl describe pod distributor-7ffcbc985-2bncj -n distributor
kubectl rollout restart deployment/distributor -n distributor
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
