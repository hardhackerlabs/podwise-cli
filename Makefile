default: build

build:
	go build -o podwise .

release:
	@$(if $(strip $(version)),:,$(error version is required: make release version=v1.2.3))
	git tag $(version) && git push origin $(version)
	goreleaser release --clean

upload:
	@$(if $(strip $(version)),:,$(error version is required: make release version=v1.2.3))
	gh release create $(version) ./dist/*.tar.gz ./dist/*.zip ./dist/checksums.txt --generate-notes --title "$(version)" --latest