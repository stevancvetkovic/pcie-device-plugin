# ArgoCD apps installation

## Install and configure ArgoCD
```
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
brew install argocd

kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "LoadBalancer"}}'
kubectl port-forward svc/argocd-server -n argocd 8080:443
argocd admin initial-password -n argocd
argocd login localhost:8080
argocd account update-password
```

## Install PCIe Device Plugin app
```
kubectl create namespace builders
argocd app create pcie-device-plugin --repo https://github.com/stevancvetkovic/pcie-device-plugin --path templates --dest-server https://kubernetes.default.svc --dest-namespace builders
argocd app set pcie-device-plugin --sync-policy automated
```

## Install PCIe Device Plugin Sample app
```
argocd app create pcie-device-plugin-sample-app --repo https://github.com/stevancvetkovic/pcie-device-plugin --path templates/sample --dest-server https://kubernetes.default.svc --dest-namespace builders
argocd app set pcie-device-plugin-sample-app --sync-policy automated
```
