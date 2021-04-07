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
	docker build -t payments-api:1.0 -f deploy/Dockerfile .

push:
	docker push tamasbrandstadter/payments-api:1.0

# Add payments.example.com as a host for the ingress resource
add-host:
	echo "$$(minikube ip) payments.example.com" | sudo tee -a /etc/hosts

# Make sure minikube is started before running this
kube-up:
	kubectl create -f kubernetes/namespace.yaml
	kubectl create -f deploy/db/secret.yaml
	kubectl create -f deploy/db/configmap.yaml
	kubectl create -f deploy/db/volume.yaml
	kubectl create -f deploy/db/deployment.yaml
	kubectl create -f deploy/db/service.yaml
	kubectl create -f kubernetes/ingress.yaml

kube-down:
	kubectl delete -f deploy/db/service.yaml
	kubectl delete -f deploy/db/deployment.yaml
	kubectl delete -f deploy/db/volume.yaml
	kubectl delete -f deploy/db/configmap.yaml
	kubectl delete -f deploy/db/secret.yaml
	kubectl delete -f kubernetes/ingress.yaml
	kubectl delete -f kubernetes/namespace.yaml