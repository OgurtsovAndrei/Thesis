.PHONY: all clean watch figures

DECK = defence
FIG_DIR = figures
BENCH_ROOT = ../../../..

# Headline + limitations SVGs → PDF
SVG_MAP = \
	$(BENCH_ROOT)/bench_results/plots/b6_N16777216/tradeoff_per_L/sosd_fb/L128.svg:headline_fb_L128 \
	$(BENCH_ROOT)/bench_results/plots/b6_N16777216_gap_heavy/tradeoff_per_L/uniform/L1.svg:limitations_uniform_L1 \
	$(BENCH_ROOT)/bench_results/plots/b6_N16777216/tradeoff_per_L/sosd_books/L128.svg:backup_books_L128

all: figures $(DECK).pdf

figures:
	@mkdir -p $(FIG_DIR)
	@for entry in $(SVG_MAP); do \
		src=$${entry%%:*}; \
		name=$${entry##*:}; \
		if [ -f "$$src" ]; then \
			echo "Converting $$name.svg → $$name.pdf"; \
			rsvg-convert -f pdf "$$src" -o $(FIG_DIR)/$$name.pdf; \
		else \
			echo "SKIP (not found): $$src"; \
		fi; \
	done

$(DECK).pdf: $(DECK).tex core/*.tex
	latexmk -pdf -interaction=nonstopmode $(DECK).tex

watch:
	latexmk -pdf -pvc -interaction=nonstopmode $(DECK).tex

clean:
	latexmk -C
	rm -rf build $(FIG_DIR) $(DECK).pdf
