cfn_cmd = aws --profile $(profile) cloudformation

clean:
	rm -rf build/update_name

deps:
	dep ensure -vendor-only

build: deps
	make clean
	GOARCH=amd64 GOOS=linux go build -o build/update_name

invoke: deps
	make clean
	GOARCH=amd64 GOOS=linux go build -o build/update_name
	sam local invoke -e event.json

deploy: deps
	GOARCH=amd64 GOOS=linux go build -o build/update_name
	$(cfn_cmd) package \
		--template-file template.yml \
		--output-template-file tmp_template.yml \
		--s3-bucket $(bucket) 
	$(cfn_cmd) deploy \
		--template-file tmp_template.yml \
		--stack-name update-name-stack \
		--capabilities CAPABILITY_NAMED_IAM
	rm tmp_template.yml

delete:
	$(cfn_cmd) delete-stack \
		--stack-name update-name-stack 