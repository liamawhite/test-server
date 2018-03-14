## Copyright 2017 Zack Butcher.
##
## Licensed under the Apache License, Version 2.0 (the "License");
## you may not use this file except in compliance with the License.
## You may obtain a copy of the License at
##
##     http://www.apache.org/licenses/LICENSE-2.0
##
## Unless required by applicable law or agreed to in writing, software
## distributed under the License is distributed on an "AS IS" BASIS,
## WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
## See the License for the specific language governing permissions and
## limitations under the License.

HUB := gcr.io/google.com/zbutcher-test
SHELL := /bin/bash

default: build

##### Go

build:
	go build .

test-server.linux:
	GOOS=linux go build -a --ldflags '-extldflags "-static"' -tags netgo -installsuffix netgo -o test-server .

##### Docker

docker.build: test-server.linux
	docker build -t ${HUB}/test-server -f Dockerfile .

docker.run: docker.build
	docker run ${HUB}/test-server

docker.push: docker.build
	gcloud docker -- push ${HUB}/test-server

##### Kube Deploy

deploy:
	kubectl apply -f <(./istio-0.5.1/bin/istioctl kube-inject -f kubernetes/deployment.yaml)
	kubectl apply -f kubernetes/service.yaml
	kubectl apply -f kubernetes/ingress.yaml

deploy.istio:
	kubectl apply -f ./istio-0.5.1/install/kubernetes/istio.yaml

deploy-all: deploy.istio deploy

##### Kube Delete

remove:
	kubectl delete -f kubernetes/ || true

remove.istio:
	kubectl delete -f ./istio-0.5.1/install/kubernetes/istio.yaml || true

remove-all: remove remove.istio
