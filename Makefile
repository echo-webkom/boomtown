run:
	@echo "Running in Docker"
	@docker build -t boomtown . 
	@docker run -p 8080:8080 -it --rm --env-file .env boomtown
