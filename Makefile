default: build

build:
	go build -o podwise .

release:
	@$(if $(strip $(version)),:,$(error version is required: make release version=v1.2.3))
	git tag $(version) && git push origin $(version)
	goreleaser release --snapshot --clean

upload:
	@$(if $(strip $(version)),:,$(error version is required: make release version=v1.2.3))
	gh release create $(version) ./dist/*.tar.gz ./dist/*.zip checksums.txt --generate-notes --title "v$(version)" --latest