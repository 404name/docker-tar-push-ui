build:
	go build -o ./docker-tar-push-ui ./

docker-build: 
	docker build -t docker-tar-push-ui:latest  .

docker-run: 
	docker run -d --name docker-tar-push-ui -p 8088:8088 docker-tar-push-ui:latest