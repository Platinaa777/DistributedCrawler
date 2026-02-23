```bash
minikube service headlamp -n headlamp

-- kubectl delete clusterrolebinding headlamp-admin

kubectl create clusterrolebinding headlamp-admin \
  --serviceaccount=headlamp:headlamp-admin \
  --clusterrole=cluster-admin

kubectl -n headlamp create token headlamp-admin
```