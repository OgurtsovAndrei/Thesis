# Makefile for Thesis and Proposal Compilation

MAIN = main
PROPOSAL = proposal
OUT_DIR = out

.PHONY: all clean watch force proposal

all: thesis proposal

thesis:
	mkdir -p $(OUT_DIR)
	latexmk -pdf -interaction=nonstopmode -outdir=$(OUT_DIR) $(MAIN).tex
	cp $(OUT_DIR)/$(MAIN).pdf .

proposal:
	mkdir -p $(OUT_DIR)
	latexmk -pdf -interaction=nonstopmode -outdir=$(OUT_DIR) $(PROPOSAL).tex
	cp $(OUT_DIR)/$(PROPOSAL).pdf .

clean:
	latexmk -C -outdir=$(OUT_DIR)
	rm -rf $(OUT_DIR)
	rm -f $(MAIN).pdf $(PROPOSAL).pdf

watch:
	mkdir -p $(OUT_DIR)
	latexmk -pvc -pdf -interaction=nonstopmode -outdir=$(OUT_DIR) $(MAIN).tex

force:
	mkdir -p $(OUT_DIR)
	latexmk -pdf -f -outdir=$(OUT_DIR) $(MAIN).tex
