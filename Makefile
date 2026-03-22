default: release

release:
	@$(if $(strip $(version)),:,$(error version is required: make release version=v1.2.3))
	git tag $(version) && git push origin $(version)
	goreleaser release --snapshot --clean