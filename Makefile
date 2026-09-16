compose-up:
	docker compose up -d

compose-down:
	docker compose down

image-build:
	docker build -t yvv4docker/browser-firefox .

image-push:
	docker push yvv4docker/browser-firefox:latest

image-pull:
	docker pull yvv4docker/browser-firefox:latest

image-remove:
	docker rmi yvv4docker/browser-firefox:latest

check:
	@curl -sf http://localhost:9222/ -o /dev/null -w '%{http_code}\n'