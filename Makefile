.PHONY: clean default version build run
FILES := /usr/local/bin/go-odata

default: version run
	
build:
	@echo "Building 'go-odata' with 'go build'"
	@go build
	
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
	@echo "Copying go-data to /usr/local/bin/..."
	@cp go-odata /usr/local/bin/
	
finished:
	@echo "Finished."

install: starting build clean copy finished

push:
	@echo "Pushing to repository using 'git push'"
	@git push
	@echo "Pushing tags to repository using 'git push origin --tags'"
	@git push origin --tags

run:
	@echo "Generating code with './go-odata config.json'"
	@./go-odata config.json		

starting:
	@echo "Starting."
	
version:
	@echo "Showing versions using 'git tag -n1'"
	@git tag -n1
	
	
