# Makefile for Thesis and Proposal Compilation

MAIN = practical-range-emptiness
PROPOSAL = proposal
OUT_DIR = build
FIG_DIR = figures
THESIS_ROOT = ..
BENCH_ROOT = ../..

# --- DIFF CONFIGURATION ---
# Set the commit hash here to highlight changes relative to it.
# You can use a hash (e.g., eecabc5) or relative (e.g., HEAD~1)
BASE_COMMIT = eecabc5
DIFF_OUTPUT = diff_highlighted.pdf
# ---------------------------

.PHONY: all clean watch force proposal thesis figures diff

# SVG → PDF mapping (source → figures/<name>.pdf)
# All sources are local to Thesis/text/ — no dependency on code modules.
SVG_MAP = \
	$(FIG_DIR)/soda_hash_mechanism.svg:soda_hash_mechanism \
	$(FIG_DIR)/soda_tradeoff_uniform.svg:soda_tradeoff_uniform \
	$(FIG_DIR)/soda_tradeoff_clustered.svg:soda_tradeoff_clustered \
	$(FIG_DIR)/phantom_comparison.svg:phantom_comparison \
	$(FIG_DIR)/trunc_tradeoff_uniform.svg:trunc_tradeoff_uniform \
	$(FIG_DIR)/trunc_tradeoff_osm.svg:trunc_tradeoff_osm \
	$(FIG_DIR)/cdf_mapping.svg:cdf_mapping \
	$(FIG_DIR)/cdf_redistribution.svg:cdf_redistribution \
	$(FIG_DIR)/gap_equidistant.svg:gap_equidistant \
	$(FIG_DIR)/segmentation.svg:segmentation \
	$(FIG_DIR)/universe_comparison.svg:universe_comparison \
	$(FIG_DIR)/fpr_bpk_books_L65536.svg:fpr_bpk_books_L65536 \
	$(FIG_DIR)/fpr_bpk_wiki_L65536.svg:fpr_bpk_wiki_L65536 \
	$(FIG_DIR)/hist_sosd_fb.svg:hist_sosd_fb \
	$(FIG_DIR)/hist_sosd_books.svg:hist_sosd_books \
	$(FIG_DIR)/hist_sosd_wiki.svg:hist_sosd_wiki \
	$(FIG_DIR)/hist_sosd_osm.svg:hist_sosd_osm

figures:
	mkdir -p $(FIG_DIR)
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

all: thesis proposal

diff: figures
	python3 diff_thesis.py $(BASE_COMMIT) --output $(DIFF_OUTPUT)

thesis: figures
	latexmk -pdf -interaction=nonstopmode $(MAIN).tex

proposal:
	latexmk -pdf -interaction=nonstopmode $(PROPOSAL).tex

clean:
	latexmk -C
	rm -rf $(OUT_DIR) $(MAIN).pdf $(PROPOSAL).pdf $(DIFF_OUTPUT) diff_tmp_simple

watch:
	latexmk -pvc -pdf -interaction=nonstopmode $(MAIN).tex

force: figures
	latexmk -pdf -g -interaction=nonstopmode $(MAIN).tex
