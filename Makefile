build:
	go build -o ./image-upload-portal ./

docker-build: 
	docker build -t image-upload-portal:latest  .

docker-run: 
	docker run -d --name image-upload-portal -p 8088:8088 image-upload-portal:latest