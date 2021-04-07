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

# Kubernetes Rules

# Build and tag containers
#tag:
#	docker build -t payments-api:1.0 -f deploy/Dockerfile .
#
## Add payments.example.com as a host for the ingress resource
#add-host:
#	echo "$$(minikube ip) payments.example.com" | sudo tee -a /etc/hosts
#
## Make sure minikube is started before running this
#kube-up:
#	kubectl create -f kubernetes/namespace.yaml
#	kubectl create -f deploy/postgres/deployment.yaml
#	kubectl create -f deploy/postgres/service.yaml
#	kubectl create -f deploy/mq/deployment.yaml
#	kubectl create -f deploy/mq/service.yaml
#	kubectl create -f deploy/deployment.yaml
#	kubectl create -f deploy/service.yaml
#	kubectl create -f kubernetes/ingress.yaml

kube-down:
#	kubectl delete -f kubernetes/ingress.yaml
#	kubectl delete -f deploy/service.yaml
#	kubectl delete -f deploy/deployment.yaml
#	kubectl delete -f deploy/postgres/service.yaml
#	kubectl delete -f deploy/postgres/deployment.yaml
#	kubectl delete -f kubernetes/namespace.yaml