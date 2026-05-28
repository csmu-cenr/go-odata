.PHONY: clean default version build run
APP := go-odata-v1
FILES := /usr/local/bin/$(APP)

default: version run

build:
	@echo "Building '$(APP)' with 'go build'"
	@go build -o $(APP)

clean:
	@for file in $(FILES); do \
		if [ -e "$$file" ]; then \
			echo "Deleting $$file..."; \
			rm "$$file"; \
		else \
			echo "$$file does not exist."; \
		fi \
	done

copy:
	@echo "Copying $(APP) to /usr/local/bin/..."
	@cp $(APP) /usr/local/bin/

finished:
	@echo "Finished."

install: starting build clean copy remove finished

push:
	@echo "Pushing to repository using 'git push'"
	@git push
	@echo "Pushing tags to repository using 'git push origin --tags'"
	@git push origin --tags

run:
	@echo "Generating code with './$(APP) config.json'"
	@./$(APP) config.json

remove:
	@echo "Removing local $(APP)"
	@rm $(APP)

starting:
	@echo "Starting."

version:
	@echo "Showing versions using 'git tag -n1'"
	@git tag -n1