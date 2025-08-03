#!/bin/bash

# Kubernetes 배포 스크립트
set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 함수 정의
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 환경 변수 설정
NAMESPACE="trace-system"
MONITORING_NAMESPACE="trace-monitoring"

# 네임스페이스 생성
log_info "Creating namespaces..."
kubectl apply -f namespace.yaml

# ConfigMap과 Secret 생성
log_info "Creating ConfigMap and Secret..."
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml

# PersistentVolume 생성
log_info "Creating PersistentVolume..."
kubectl apply -f persistent-volume.yaml

# Deployment 생성
log_info "Creating Deployment..."
kubectl apply -f deployment.yaml

# Service 생성
log_info "Creating Service..."
kubectl apply -f service.yaml

# Ingress 생성 (선택사항)
if [ "$1" = "--with-ingress" ]; then
    log_info "Creating Ingress..."
    kubectl apply -f ingress.yaml
fi

# HPA 생성
log_info "Creating HorizontalPodAutoscaler..."
kubectl apply -f hpa.yaml

# 모니터링 리소스 생성 (선택사항)
if [ "$1" = "--with-monitoring" ]; then
    log_info "Creating monitoring resources..."
    kubectl apply -f monitoring.yaml
fi

# 배포 상태 확인
log_info "Checking deployment status..."
kubectl get pods -n $NAMESPACE
kubectl get services -n $NAMESPACE
kubectl get ingress -n $NAMESPACE

# 로그 확인
log_info "Checking application logs..."
kubectl logs -n $NAMESPACE -l app=trace --tail=50

log_info "Deployment completed successfully!"
log_info "Access the application:"
log_info "  - ClusterIP: kubectl port-forward -n $NAMESPACE svc/trace-service 8080:80"
log_info "  - NodePort: http://localhost:30080"
log_info "  - Ingress: http://trace.example.com (if configured)" 