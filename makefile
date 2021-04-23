run: stop up

mod:
	GO111MODULE=on go mod tidy
	GO111MODULE=on go mod vendor

up:
	docker-compose -f docker-compose.yml up -d --build

stop:
	docker-compose -f docker-compose.yml stop

down:
	docker-compose -f docker-compose.yml down

test:
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f docker-compose.test.yml down --volumes

tag:
	docker build -t payments-api:1.0.0 -f deploy/Dockerfile .

push:
	docker push tamasbrandstadter/payments-api:1.0.0

# Add payments.example.com as a host for the ingress resource
add-host:
	echo "$$(minikube ip) payments.example.com" | sudo tee -a /etc/hosts

# Make sure minikube is started before running these
kube-infra-up:
	kubectl apply -f kubernetes/namespace.yaml
	kubectl apply -f https://github.com/rabbitmq/cluster-operator/releases/latest/download/cluster-operator.yml
	kubectl apply -f deploy/mq/secret.yaml
	kubectl apply -f deploy/mq/cluster.yaml
	kubectl apply -f deploy/db/secret.yaml
	kubectl apply -f deploy/db/configmap.yaml
	kubectl apply -f deploy/db/volume.yaml
	kubectl apply -f deploy/db/deployment.yaml
	kubectl apply -f deploy/db/service.yaml
	kubectl apply -f deploy/cache/secret.yaml
	kubectl apply -f deploy/cache/deployment.yaml
	kubectl apply -f deploy/cache/service.yaml
	kubectl apply -f kubernetes/ingress.yaml

kube-api-up:
	kubectl apply -f deploy/api/deployment.yaml
	kubectl apply -f deploy/api/service.yaml
	kubectl apply -f deploy/api/hpa.yaml

mesh:
	curl https://run.linkerd.io/install | sh
	linkerd install | kubectl apply -f -
	linkerd check
	linkerd viz install | kubectl apply -f -
	linkerd jaeger install | kubectl apply -f -
	linkerd check
	kubectl get -n payments statefulset -o yaml | linkerd inject - | kubectl apply -f -
	kubectl get -n payments deploy -o yaml | linkerd inject - | kubectl apply -f -

kube-api-down:
	kubectl delete -f deploy/api/service.yaml
	kubectl delete -f deploy/api/deployment.yaml
	kubectl delete -f deploy/api/hpa.yaml

kube-infra-down:
	kubectl delete -f deploy/db/service.yaml
	kubectl delete -f deploy/db/deployment.yaml
	kubectl delete -f deploy/db/volume.yaml
	kubectl delete -f deploy/db/configmap.yaml
	kubectl delete -f deploy/db/secret.yaml
	kubectl delete -f deploy/mq/secret.yaml
	kubectl delete -f deploy/mq/cluster.yaml
	kubectl delete -f https://github.com/rabbitmq/cluster-operator/releases/latest/download/cluster-operator.yml
	kubectl delete -f deploy/cache/service.yaml
	kubectl delete -f deploy/cache/deployment.yaml
	kubectl delete -f deploy/cache/secret.yaml
	kubectl delete -f kubernetes/ingress.yaml
	kubectl delete -f kubernetes/namespace.yaml