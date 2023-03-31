.PHONY: man
man:
	mmark -man gitopper.8.md > gitopper.8
	mmark -man cmd/gitopperctl/gitopperctl.8.md > cmd/gitopperctl/gitopperctl.8
	mmark -man cmd/gitopperhdr/gitopperhdr.1.md > cmd/gitopperhdr/gitopperhdr.1
