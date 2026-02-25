# Makefile for Thesis Compilation

MAIN = main
PDF = $(MAIN).pdf
SRC = $(MAIN).tex references.bib

.PHONY: all clean watch force

all: $(PDF)

$(PDF): $(SRC)
	latexmk -pdf -interaction=nonstopmode $(MAIN).tex

clean:
	latexmk -C
	rm -f *.bbl *.run.xml *.synctex.gz

watch:
	latexmk -pvc -pdf -interaction=nonstopmode $(MAIN).tex

force:
	latexmk -pdf -f $(MAIN).tex
